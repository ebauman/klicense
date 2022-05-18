package controllers

import (
	licensingv1 "github.com/ebauman/klicense/api/v1"
	license2 "github.com/ebauman/klicense/license"
	v1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	wranglerCore "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"strings"
	"time"
)

type EntitlementHandler struct {
	entitlementClient v1.EntitlementClient
	entitlementCache v1.EntitlementCache
	requestClient v1.RequestClient
	requestCache v1.RequestCache
	secretCache  wranglerCore.SecretCache
}

func (h *EntitlementHandler) OnEntitlementChanged(key string, entitlement *licensingv1.Entitlement) (*licensingv1.Entitlement, error) {
	if entitlement == nil {
		return nil, nil
	}

	licenses := map[string]bool{}
	unitMap := map[string]bool{}
	var earliestExpiration time.Time

	for _, g := range entitlement.Status.Grants {
		// count things
		licenses[g.Id] = true
		if earliestExpiration.IsZero() {
			earliestExpiration = g.NotAfter.Time
		} else {
			if g.NotAfter.Time.Before(earliestExpiration) {
				earliestExpiration = g.NotAfter.Time
			}
		}

		unitMap[g.Unit] = true

		cachedSecret, err := h.secretCache.Get(g.LicenseSecret.Namespace, g.LicenseSecret.Name)
		if errors.IsNotFound(err) {
			err = h.processGrantDeletion(g.Id, entitlement)
			if err != nil {
				logrus.Errorf("couldn't remove grant from entitlement: %s", err.Error())
				return nil, err
			}
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err = h.entitlementClient.UpdateStatus(entitlement)
				return err
			})
			if err != nil {
				logrus.Errorf("error updating entitlement: %s", err.Error())
				return nil, err
			}
			return nil, nil
		}

		if err != nil {
			// something bad happened, and it wasn't us not finding the secret
			logrus.Errorf("error getting secret from kubernetes: %s", err.Error())
			return nil, err
		}

		licenseSecret := cachedSecret.DeepCopy()

		if _, ok := licenseSecret.Labels[LicensingLabel]; !ok {
			logrus.Error(err, "license secret not labeled as such, bailing")
			return nil, err
		}

		license, err := license2.ValidateSecret(licenseSecret)
		if err != nil {
			logrus.Error("error validating license secret")
			return nil, err
		}

		// if the license is not expired, then that's all we need to check
		if license.NotAfter.Before(time.Now()) || license.NotBefore.After(time.Now()) {
			logrus.Info("license expired or not yet valid")
			err = h.processGrantDeletion(g.Id, entitlement)
			if err != nil {
				logrus.Error(err, "couldn't remove grant from entitlement")
			}
		}

		// if we get here the license is valid and non-expired.
		// now we just update the grant
		// this is to mostly prevent someone from manually editing the entitlement and changing the amounts
		// in the case of an updated license value, the secret controller will handle that
		g.NotBefore = metav1.NewTime(license.NotBefore)
		g.NotAfter = metav1.NewTime(license.NotAfter)
		g.Amount = license.Grants[entitlement.Name]
	}

	entitlement.Status.Licenses = len(licenses)
	entitlement.Status.EarliestExpiration = metav1.NewTime(earliestExpiration)
	var units []string
	for k := range unitMap {
		units = append(units, k)
	}
	entitlement.Status.Units = strings.Join(units, ",")


	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := h.entitlementClient.UpdateStatus(entitlement)
		return err
	})

	if err != nil {
		logrus.Error(err, "error updating entitlement")
		return nil, err
	}

	return nil, nil
}

func (h *EntitlementHandler) processGrantDeletion(key string, entitlement *licensingv1.Entitlement) error {
	return ProcessGrantDeletion(h.requestCache.Get, h.requestClient.UpdateStatus, key, entitlement)
}

