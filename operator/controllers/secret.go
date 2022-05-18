package controllers

import (
	"context"
	v1 "github.com/ebauman/klicense/api/v1"
	"github.com/ebauman/klicense/kubernetes"
	license2 "github.com/ebauman/klicense/license"
	cattleLicensingv1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	"github.com/ebauman/klicense/remove"
	wranglerCorev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"strings"
	"time"
)

func RegisterSecretHandler(ctx context.Context,
	entitlementController cattleLicensingv1.EntitlementController,
	requestController cattleLicensingv1.RequestController,
	secretController wranglerCorev1.SecretController) {
	secretHandler := &SecretHandler{
		entitlementCache:  entitlementController.Cache(),
		entitlementClient: entitlementController,
		requestClient: requestController,
		requestCache: requestController.Cache(),
	}

	remove.RegisterScopedOnRemoveHandler(ctx, secretController, "on-license-secret-remove",
		func(key string, obj runtime.Object) (bool, error) {
			if obj == nil {
				return false, nil
			}

			secret, ok := obj.(*corev1.Secret)
			if !ok {
				return false, nil
			}

			return secretHandler.shouldManage(secret)
		},
		wranglerCorev1.FromSecretHandlerToHandler(secretHandler.OnRemove),
		)

	secretController.OnChange(ctx, "secret-on-change", secretHandler.OnSecretChanged)
}

type SecretHandler struct {
	entitlementCache  cattleLicensingv1.EntitlementCache
	entitlementClient cattleLicensingv1.EntitlementClient
	requestCache cattleLicensingv1.RequestCache
	requestClient cattleLicensingv1.RequestClient
}

func (s *SecretHandler) shouldManage(secret *corev1.Secret) (bool, error) {
	if secret == nil {
		return false, nil
	}

	// really the only requirement is to have a license label
	if _, ok := secret.Labels[LicensingLabel]; !ok {
		return false, nil
	}

	return true, nil
}

func (s *SecretHandler) OnSecretChanged(key string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret == nil {
		return nil, nil
	}

	if !secret.DeletionTimestamp.IsZero() {
		// there is another onremove handler that takes care of this
		return nil, nil
	}

	if _, ok := secret.Labels[LicensingLabel]; !ok {
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
		cachedEntitlement, err := s.entitlementCache.Get(secret.Namespace, url[0])
		var entitlement = &v1.Entitlement{}
		if errors.IsNotFound(err) {
			entitlement.Name = url[0]
			entitlement.Namespace = secret.Namespace

			if entitlement, err = s.entitlementClient.Create(entitlement); err != nil {
				logrus.Error(err, "error creating entitlement")
				return nil, err
			}
		} else {
			cachedEntitlement.DeepCopyInto(entitlement)
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
			Unit:      url[1],
			Status:    v1.GrantStatusFree,
			NotBefore: metav1.NewTime(license.NotBefore),
			NotAfter:  metav1.NewTime(license.NotAfter),
			LicenseSecret: kubernetes.NamespacedName{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = s.entitlementClient.UpdateStatus(entitlement)
			return err
		})

		if err != nil {
			logrus.Errorf("error updating entitlement status: %s", err.Error())
			return nil, err
		}
	}

	return nil, nil
}

func (s *SecretHandler) OnRemove(key string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret == nil {
		return nil, nil
	}

	license, err := license2.ValidateSecret(secret)
	if err != nil {
		return nil, err
	}

	// get all entitlements referenced in this license
	// in each entitlement, remove the corresponding grant
	for grantName, _ := range license.Grants {
		// pull out the entitlement name from the license
		entitlementName := strings.Split(grantName, "/")[0]

		// get the corresponding entitlement
		cachedEntitlement, err := s.entitlementCache.Get(secret.Namespace, entitlementName)
		if errors.IsNotFound(err) {
			continue // nothing to do about an entitlement we can't find
		}

		if err != nil {
			logrus.Errorf("error retrieving entitlement from kubernetes: %s", err.Error())
			return nil, err
		}

		// once we have the entitlement, copy it and remove the corresponding grant
		entitlement := cachedEntitlement.DeepCopy()
		if _, ok := entitlement.Status.Grants[license.Id]; ok {
			err = s.processGrantDeletion(license.Id, entitlement)
			if err != nil {
				logrus.Errorf("error notifying deleting grant from entitlement: %s", err.Error())
			}
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = s.entitlementClient.UpdateStatus(entitlement)
			return err
		})
		if err != nil {
			logrus.Errorf("error updating entitlement: %s", err.Error())
			return nil, err
		}
	}

	return secret, nil
}

func (s *SecretHandler) processGrantDeletion(key string, entitlement *v1.Entitlement) error {
	return ProcessGrantDeletion(s.requestCache.Get, s.requestClient.UpdateStatus, key, entitlement)
}