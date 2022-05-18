package controllers

import (
	"context"
	v1 "github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io/v1"
	wranglerCore "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
)

func Register(
	ctx context.Context,
	entitlementController v1.EntitlementController,
	requestController v1.RequestController,
	secretController wranglerCore.SecretController) {

	entitlementHandler := &EntitlementHandler{
		entitlementClient: entitlementController,
		entitlementCache:  entitlementController.Cache(),
		requestClient:     requestController,
		requestCache:      requestController.Cache(),
		secretCache:       secretController.Cache(),
	}

	requestHandler := &RequestHandler{
		requestCache:      requestController.Cache(),
		requestClient:     requestController,
		entitlementCache:  entitlementController.Cache(),
		entitlementClient: entitlementController,
	}

	entitlementController.OnChange(ctx, "entitlement-handler", entitlementHandler.OnEntitlementChanged)
	requestController.OnChange(ctx, "request-handler", requestHandler.OnRequestChanged)
}