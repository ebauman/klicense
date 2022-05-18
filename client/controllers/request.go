package controllers

import (
	"context"
	"fmt"
	licensingv1 "github.com/ebauman/klicense/api/v1"
	v1 "github.com/ebauman/klicense/client/generated/controllers/licensing.cattle.io/v1"
	v13 "github.com/ebauman/klicense/client/generated/controllers/licensing.cattle.io/v1"
	license2 "github.com/ebauman/klicense/license"
	v14 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"time"
)

func Register(ctx context.Context,
	requestClient v1.RequestClient,
	namespace string,
	secretCache v14.SecretCache,
	notifiers map[string]chan<- bool,
	requestController v13.RequestController) {

	handler := RequestHandler{
		requestClient:    requestClient,
		namespace:        namespace,
		secretCache:      secretCache,
		notifiers:        notifiers,
	}

	requestController.OnChange(ctx, "request-handler", handler.OnRequestChanged)
}

type RequestHandler struct {
	requestClient v1.RequestClient
	namespace string
	secretCache v14.SecretCache
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
		// let's get the secret that the license is int
		cachedSecret, err := r.secretCache.Get(r.namespace, request.Status.LicenseSecret)
		secret := cachedSecret.DeepCopy()
		if err != nil {
			logrus.Errorf("error retrieving secret for license: %s", err.Error())
			return nil, err
		}

		// once we have the secret, pull out the license
		license, err := license2.ValidateSecret(secret)
		if err != nil {
			logrus.Errorf("error validating license for grant: %s", err.Error())
			return nil, err
		}

		// if we have gotten here, the license is valid
		// now just check start/end times and amounts
		if license.NotAfter.Before(time.Now()) || license.NotBefore.After(time.Now()) {
			logrus.Errorf("license expired or not yet valid")
			return nil, err
		}

		grantName := fmt.Sprintf("%s/%s", request.Spec.Kind, request.Spec.Unit)
		amount, ok := license.Grants[grantName]
		if !ok {
			logrus.Errorf("could not locate grant in license with name %s", grantName)
		}

		if amount < request.Spec.Amount {
			// requesting too much
			logrus.Error("amount requested is higher than offered grant")
			return nil, err
		}

		// at this point, we have a license with the grant requested, in the amount requested (at least)
		// that isn't expired or not yet valid. we can acknowledge and accept it!
		request.Status.Status = licensingv1.UsageRequestStatusAcknowledged

		_, err = r.requestClient.UpdateStatus(request)
		if err != nil {
			logrus.Errorf("error updating status of request object in kubernetes: %s", err.Error())
			return nil, err
		}

		if notify, ok := r.notifiers[string(request.UID)]; ok {
			notify <- true
		}

		return nil, nil

	case licensingv1.UsageRequestStatusAcknowledged:
		// the license is ours, tell someone!
		if notify, ok := r.notifiers[string(request.UID)]; ok {
			notify <- true
		}

		return nil, nil
	}

	return nil, nil
}