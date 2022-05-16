package controllers

import (
	licensingv1 "github.com/ebauman/klicense/api/v1"
	v12 "github.com/ebauman/klicense/operator/generated/controllers/core/v1"
	v1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	"github.com/sirupsen/logrus"
)

type RequestHandler struct {
	entitlementCache v1.EntitlementCache
	namespace string
	secretCache v12.SecretCache
	notifiers map[string]chan<- bool
}

func (r *RequestHandler) OnRequestChanged(key string, request *licensingv1.Request) (*licensingv1.Request, error) {
	if !request.DeletionTimestamp.IsZero() {
		// this request has been deleted, either by us or by another user
		// find the corresponding requester and notify of unlicensed status
		if notify, ok := r.notifiers[string(request.UID)]; ok {
			notify <- false
		}

		// nothing else to do
		return nil, nil
	}

	switch request.Status.Status {
	case licensingv1.UsageRequestStatusDiscover:
		// this is the initial creation of the request
		// there is curently nothing for us to do, because Discover status
		// means that the request is awaiting fulfillment from the operator
		return nil, nil
	case licensingv1.UsageRequestStatusOffer:
		// if there is an offer, we need to verify the license
		// start by fetching the entitlement for the request
		entitlement, err := r.entitlementCache.Get(request.Spec.Entitlement.Namespace, request.Spec.Entitlement.Name)
		if err != nil {
			logrus.Errorf("error retrieving entitlement from request: %s", err.Error())
			return nil, err
		}

		// once we have the entitlement, get the secret which contains the license
		secret, err := r.secretCache.Get(entitlement.)

	case licensingv1.UsageRequestStatusAcknowledged:
		// the license is ours, tell someone!
		if notify, ok := r.notifiers[string(request.UID)]; ok {
			notify <- true
		}

		return nil, nil
	}
}