package controllers

import (
	v1 "github.com/ebauman/klicense/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type requestCacheClient interface {
	Get(namespace string, name string) (*v1.Request, error)
	UpdateStatus(*v1.Request) (*v1.Request, error)
}

func ProcessGrantDeletion(requestCacheGet func (namespace string, name string) (*v1.Request, error),
	requestUpdateStatus func (request *v1.Request) (*v1.Request, error),
	key string, entitlement *v1.Entitlement) error {
	// when a grant is deleted we both need to removeNamespacedName it from the entitlement
	// but also return corresponding requestCache to "Pending" for evaluation by the request controller
	// (so we don't break anything if there is another license that can be used)

	if entitlement.Status.Grants[key].Status == v1.GrantStatusInUse {
		// now we need to notify the request, place it into request mode for now
		if grant, ok := entitlement.Status.Grants[key]; ok {
			cachedRequest, err := requestCacheGet(entitlement.Namespace, grant.Request.Name)
			request := cachedRequest.DeepCopy()
			if errors.IsNotFound(err) {
				// request doesn't exist, just delete it and move on!
				delete(entitlement.Status.Grants, key)
				return nil
			}

			if err != nil {
				return err // something else went wrong, err out
			}

			// no err here, we have a valid request
			request.Status.Status = v1.UsageRequestStatusDiscover
			request.Status.Grant = ""
			request.Status.Message = "prior grant deleted"

			_, err = requestUpdateStatus(request)
			if err != nil {
				return err
			}
		}
	}

	delete(entitlement.Status.Grants, key)

	return nil
}
