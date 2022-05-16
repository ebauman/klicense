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
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	licensingv1 "github.com/ebauman/klicense/api/v1"
)

// RequestReconciler reconciles a Request object
type RequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=licensing.cattle.io,resources=requests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=licensing.cattle.io,resources=requests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=licensing.cattle.io,resources=requests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Request object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *RequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	request := &licensingv1.Request{}

	err := r.Get(ctx, req.NamespacedName, request)
	if err != nil {
		log.Error(err, "unable to fetch request")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !request.DeletionTimestamp.IsZero() {
		// request is to be deleted
		// we can free up the grant
		entitlement := &licensingv1.Entitlement{}
		err := r.Get(ctx, request.Spec.Entitlement.ToK8sNamespacedName(), entitlement)
		if err != nil {
			log.Error(err, "unable to fetch entitlement")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		if eStat, ok := entitlement.Status.Grants[request.Status.Grant]; ok {
			eStat.Status = licensingv1.GrantStatusFree
			eStat.Request = kubernetes.NamespacedName{}
		}
	}

	// we only take action on Request and Acknowledged
	// because Offer is the status _we_ put the request into,
	// it requires no action from us
	switch request.Status.Status {
	case licensingv1.UsageRequestStatusDiscover:
		// client is requesting usage of an entitlement. can we give it to them?
		// 1 - there must be a grant available
		// 2 - the grant must meet the usage requirements for the client
		// (ignoring things like invalid grants since other controllers handle that)
		entitlement := &licensingv1.Entitlement{}
		err = r.Get(ctx, request.Spec.Entitlement.ToK8sNamespacedName(), entitlement)
		if err != nil {
			log.Error(err, "unable to fetch entitlement")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		// if we have a valid entitlement at this point, let's check if the requested
		// entitlement has a grant they can use
		for _, grant := range entitlement.Status.Grants {
			// if the grant is used, continue
			if grant.Status != licensingv1.GrantStatusFree {
				continue
			}

			if grant.Type != request.Spec.Type {
				continue
			}

			if grant.Amount < request.Spec.Amount {
				continue
			}

			// if we get here, we have a grant that matches the request
			// let's offer it to the client
			request.Status.Status = licensingv1.UsageRequestStatusOffer
			request.Status.Grant = grant.Id

			err = r.Update(ctx, request)
			if err != nil {
				log.Error(err, "error updating request")
				return ctrl.Result{}, err
			}

			grant.Status = licensingv1.GrantStatusPending
			err = r.Update(ctx, entitlement)
			if err != nil {
				log.Error(err, "error updating entitlement")
				return ctrl.Result{}, err
			}

			break // once we've found our grant, don't continue
		}

		// if we get to this point, there is no matching grant currently
		// update the request and say that
		request.Status.Message = "no matching grant found for specified amount and type"
		err = r.Update(ctx, request)
		if err != nil {
			log.Error(err, "error updating request")
			return ctrl.Result{}, err
		}
	case licensingv1.UsageRequestStatusAcknowledged:
		entitlement := &licensingv1.Entitlement{}
		err = r.Get(ctx, request.Spec.Entitlement.ToK8sNamespacedName(), entitlement)
		if err != nil {
			log.Error(err, "unable to fetch entitlement")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		if grant, ok := entitlement.Status.Grants[request.Status.Grant]; ok {
			grant.Status = licensingv1.GrantStatusInUse
			grant.Request = kubernetes.NamespacedName{
				Name:      request.Name,
				Namespace: request.Namespace,
			}
		}

		err = r.Update(ctx, entitlement)
		if err != nil {
			log.Error(err, "error updating entitlement")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&licensingv1.Request{}).
		Complete(r)
}
