package controllers

import (
	licensingv1 "github.com/ebauman/klicense/api/v1"
	"github.com/ebauman/klicense/kubernetes"
	v1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	"github.com/sirupsen/logrus"
)

type RequestHandler struct {
	requestCache v1.RequestCache
	requestClient v1.RequestClient
	entitlementCache v1.EntitlementCache
	entitlementClient v1.EntitlementClient
}

func (r *RequestHandler) OnRequestChanged(key string, request *licensingv1.Request) (*licensingv1.Request, error) {
	if !request.DeletionTimestamp.IsZero() {
		// request is to be deleted
		// we can free up the grant
		entitlement, err := r.entitlementCache.Get(request.Spec.Entitlement.Namespace, request.Spec.Entitlement.Name)
		if err != nil {
			logrus.Error(err, "unable to fetch entitlement")
			return nil, err
		}

		if eStat, ok := entitlement.Status.Grants[request.Status.Grant]; ok {
			eStat.Status = licensingv1.GrantStatusFree
			eStat.Request = kubernetes.NamespacedName{}
		}

		_, err = r.entitlementClient.UpdateStatus(entitlement)
		if err != nil {
			return nil, err
		}

		return nil, nil
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
		entitlement, err := r.entitlementCache.Get(request.Spec.Entitlement.Namespace, request.Spec.Entitlement.Name)
		if err != nil {
			logrus.Error(err, "unable to fetch entitlement")
			return nil, err
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

			_, err := r.requestClient.UpdateStatus(request)
			if err != nil {
				logrus.Error(err, "error updating request")
				return nil, err
			}

			grant.Status = licensingv1.GrantStatusPending
			_, err = r.entitlementClient.UpdateStatus(entitlement)
			if err != nil {
				logrus.Error(err, "error updating entitlement")
				return nil, err
			}

			break // once we've found our grant, don't continue
		}

		// if we get to this point, there is no matching grant currently
		// update the request and say that
		request.Status.Message = "no matching grant found for specified amount and type"
		_, err = r.requestClient.UpdateStatus(request)
		if err != nil {
			logrus.Error(err, "error updating request")
			return nil, err
		}
	case licensingv1.UsageRequestStatusAcknowledged:
		entitlement, err := r.entitlementCache.Get(request.Spec.Entitlement.Namespace, request.Spec.Entitlement.Name)
		if err != nil {
			logrus.Error(err, "unable to fetch entitlement")
			return nil, err
		}

		if grant, ok := entitlement.Status.Grants[request.Status.Grant]; ok {
			grant.Status = licensingv1.GrantStatusInUse
			grant.Request = kubernetes.NamespacedName{
				Name:      request.Name,
				Namespace: request.Namespace,
			}
		}

		_, err = r.entitlementClient.UpdateStatus(entitlement)
		if err != nil {
			logrus.Error(err, "error updating entitlement")
			return nil, err
		}
	}

	return nil, nil
}