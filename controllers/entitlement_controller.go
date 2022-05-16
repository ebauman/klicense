/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/ebauman/klicense/kubernetes"
	license2 "github.com/ebauman/klicense/license"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	licensingv1 "github.com/ebauman/klicense/api/v1"
)

// EntitlementReconciler reconciles a Entitlement object
type EntitlementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=licensing.cattle.io,resources=entitlements,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=licensing.cattle.io,resources=entitlements/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=licensing.cattle.io,resources=entitlements/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Entitlement object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *EntitlementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// on entitlements we calculate usage vs available
	// and confirm the ongoing validity of licenses
	var entitlement = &licensingv1.Entitlement{}
	err := r.Get(ctx, req.NamespacedName, entitlement)

	if errors.IsNotFound(err) {
		log.Info("ignoring not found entitlement")
		return ctrl.Result{}, nil // probably doesn't fully exist yet
	}

	if err != nil {
		// invalid entitlement
		log.Error(err, "entitlement not found")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, g := range entitlement.Status.Grants {
		// get the secret associated with the grant
		licenseSecret := &corev1.Secret{}
		err = r.Get(ctx, g.LicenseSecret.ToK8sNamespacedName(), licenseSecret)

		if err != nil {
			// couldn't get the license secret for some reason
			// let's removeNamespacedName it from entitlement
			log.Error(err, "couldn't get license secret from entitlement ref",
				"licenseSecret", g.LicenseSecret,
				"entitlement", entitlement.Name)
			err = r.processGrantDeletion(ctx, g.Id, entitlement)
			if err != nil {
				log.Error(err, "couldn't removeNamespacedName grant from entitlement")
			}
		}

		if _, ok := licenseSecret.Labels[LicensingLabel]; !ok {
			log.Error(err, "license secret not labeled as such, bailing")
			return ctrl.Result{}, err
		}

		license, err := license2.ValidateSecret(licenseSecret)
		if err != nil {
			log.Error(err, "error validating license secret")
			return ctrl.Result{}, err
		}

		// if the license is not expired, then that's all we need to check
		if license.NotAfter.Before(time.Now()) || license.NotBefore.After(time.Now()) {
			log.Info("license expired or not yet valid")
			err = r.processGrantDeletion(ctx, g.Id, entitlement)
			if err != nil {
				log.Error(err, "couldn't removeNamespacedName grant from entitlement")
			}
		}

		// if we get here the license is valid and non-expired.
		// now we just update the grant
		// this is to mostly prevent someone from manually editing the entitlement and changing the amounts
		// in the case of an updated license value, the secret controller will handle that
		g.NotBefore = v1.NewTime(license.NotBefore)
		g.NotAfter = v1.NewTime(license.NotAfter)
		g.Amount = license.Grants[entitlement.Name]
	}

	// update at the end
	err = r.Update(ctx, entitlement)
	if err != nil {
		log.Error(err, "error updating entitlement")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EntitlementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&licensingv1.Entitlement{}).
		Complete(r)
}

func (r *EntitlementReconciler) processGrantDeletion(ctx context.Context, key string, entitlement *licensingv1.Entitlement) error {
	// when a grant is deleted we both need to removeNamespacedName it from the entitlement
	// but also return corresponding requests to "Pending" for evaluation by the request controller
	// (so we don't break anything if there is another license that can be used)

	if entitlement.Status.Grants[key].Status == licensingv1.GrantStatusInUse {
		// now we need to notify the request, place it into request mode for now
		if grant, ok := entitlement.Status.Grants[key]; ok {
			request := &licensingv1.Request{}
			err := r.Get(ctx, grant.Request.ToK8sNamespacedName(), request)
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

			err = r.Update(ctx, request)
			if err != nil {
				return err
			}
		}
	}

	delete(entitlement.Status.Grants, key)

	return nil
}

func removeNamespacedName(s []kubernetes.NamespacedName, i int) []kubernetes.NamespacedName {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
