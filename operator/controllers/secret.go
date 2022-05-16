package controllers

import (
	v1 "github.com/ebauman/klicense/api/v1"
	"github.com/ebauman/klicense/common"
	"github.com/ebauman/klicense/kubernetes"
	license2 "github.com/ebauman/klicense/license"
	cattleLicensingv1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

type SecretHandler struct {
	entitlementCache  cattleLicensingv1.EntitlementCache
	entitlementClient cattleLicensingv1.EntitlementClient
}

func (s *SecretHandler) OnSecretChanged(key string, secret *corev1.Secret) (*corev1.Secret, error) {
	if _, ok := secret.Labels[common.LicensingLabel]; !ok {
		return nil, nil
	}

	license, err := license2.ValidateSecret(secret)
	if err != nil {
		return nil, err
	}

	// if we have a valid license at this point, convert its contents into grants
	for k, v := range license.Grants {
		// first, try and get an entitlement in the cluster
		var url = strings.Split(k, "/")
		entitlement, err := s.entitlementCache.Get(secret.Namespace, url[0])
		if errors.IsNotFound(err) {
			entitlement.Name = url[0]
			entitlement.Namespace = secret.Namespace

			if entitlement, err = s.entitlementClient.Create(entitlement); err != nil {
				logrus.Error(err, "error creating entitlement")
				return nil, err
			}
		}

		if license.NotAfter.Before(time.Now()) || license.NotBefore.After(time.Now()) {
			// license is expired, don't add it to the entitlement.
			logrus.Info("license expired or not yet valid")
			return nil, nil
		}

		if entitlement.Status.Grants == nil {
			entitlement.Status.Grants = make(map[string]v1.Grant, 0)
		}
		entitlement.Status.Grants[license.Id] = v1.Grant{
			Amount:    v,
			Id:        license.Id,
			Type:      url[1],
			Status:    v1.GrantStatusFree,
			NotBefore: metav1.NewTime(license.NotBefore),
			NotAfter:  metav1.NewTime(license.NotAfter),
			LicenseSecret: kubernetes.NamespacedName{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		}

		if _, err = s.entitlementClient.UpdateStatus(entitlement); err != nil {
			logrus.Error("error updating entitlement status")
			return nil, err
		}
	}

	return nil, nil
}