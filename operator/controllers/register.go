package controllers

import (
	"context"
	v1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	wranglerCore "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
)

func Register(
	ctx context.Context,
	entitlementClient v1.EntitlementClient,
	entitlementCache v1.EntitlementCache,
	requestClient v1.RequestClient,
	requestCache v1.RequestCache,
	secretCache wranglerCore.SecretCache,
	entitlementController v1.EntitlementController,
	requestController v1.RequestController,
	secretController wranglerCore.SecretController,
	) {

	entitlementHandler := &EntitlementHandler{
		entitlementClient: entitlementClient,
		entitlementCache: entitlementCache,
		requestClient: requestClient,
		requestCache: requestCache,
		secretCache: secretCache,
	}

	requestHandler := &RequestHandler{
		requestCache:      requestCache,
		requestClient:     requestClient,
		entitlementCache:  entitlementCache,
		entitlementClient: entitlementClient,
	}

	secretHandler := &SecretHandler{
		entitlementCache:  entitlementCache,
		entitlementClient: entitlementClient,
	}

	entitlementController.OnChange(ctx, "entitlement-handler", entitlementHandler.OnEntitlementChanged)
	requestController.OnChange(ctx, "request-handler", requestHandler.OnRequestChanged)
	secretController.OnChange(ctx, "secret-handler", secretHandler.OnSecretChanged)
}
