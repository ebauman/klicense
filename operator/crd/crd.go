package crd

import (
	"context"
	v1 "github.com/ebauman/klicense/api/v1"
	"github.com/rancher/wrangler/pkg/crd"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

func Create(ctx context.Context, cfg *rest.Config) error {
	factory, err := crd.NewFactoryFromClient(cfg)
	if err != nil {
		return err
	}

	return factory.BatchCreateCRDs(ctx, List()...).BatchWait()
}

func List() []crd.CRD {
	return []crd.CRD{
		newCRD(&v1.Request{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("Kind", ".spec.kind").
				WithColumn("Unit", ".spec.unit").
				WithColumn("Amount", ".spec.amount").
				WithColumn("Status", ".status.status")
		}),
		newCRD(&v1.Entitlement{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("Licenses", ".status.licenses").
				WithColumn("Units", ".status.units").
				WithColumn("Earliest Expiration", ".status.earliestExpiration")
		}),
	}
}


func newCRD(obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	crd := crd.CRD{
		GVK: schema.GroupVersionKind{
			Group: "licensing.cattle.io",
			Version: "v1",
		},
		Status: true,
		SchemaObject: obj,
	}

	if customize != nil {
		crd = customize(crd)
	}

	return crd
}