package controllers

import (
	licensingv1 "github.com/ebauman/klicense/api/v1"
	"github.com/ebauman/klicense/common"
	license2 "github.com/ebauman/klicense/license"
	v1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	wranglerCore "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	for _, g := range entitlement.Status.Grants {
		// get the secret associated with the grant
		licenseSecret, err := h.secretCache.Get(g.LicenseSecret.Namespace, g.LicenseSecret.Name)
		if err != nil {
			logrus.Error(err, "couldn't get license secret from entitlement ref")
			err = h.processGrantDeletion(g.Id, entitlement)
			if err != nil {
				logrus.Error(err, "couldnt remove grant from entitlement")
			}
		}

		if _, ok := licenseSecret.Labels[common.LicensingLabel]; !ok {
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

	_, err := h.entitlementClient.UpdateStatus(entitlement)
	if err != nil {
		logrus.Error(err, "error updating entitlement")
		return nil, err
	}

	return nil, nil
}

func (h *EntitlementHandler) processGrantDeletion(key string, entitlement *licensingv1.Entitlement) error {
	// when a grant is deleted we both need to removeNamespacedName it from the entitlement
	// but also return corresponding requestCache to "Pending" for evaluation by the request controller
	// (so we don't break anything if there is another license that can be used)

	if entitlement.Status.Grants[key].Status == licensingv1.GrantStatusInUse {
		// now we need to notify the request, place it into request mode for now
		if grant, ok := entitlement.Status.Grants[key]; ok {
			request, err := h.requestCache.Get(entitlement.Namespace, grant.Request.Name)
			if errors.IsNotFound(err) {
				// request doesn't exist, just delete it and move on!
				delete(entitlement.Status.Grants, key)
				return nil
			}

			if err != nil {
				return err // something else went wrong, err out
			}

			// no err here, we have a valid request
			request.Status.Status = licensingv1.UsageRequestStatusDiscover
			request.Status.Grant = ""
			request.Status.Message = "prior grant deleted"

			_, err = h.requestClient.UpdateStatus(request)
			if err != nil {
				return err
			}
		}
	}

	delete(entitlement.Status.Grants, key)

	return nil
}