package main

import (
	v1 "github.com/ebauman/klicense/api/v1"
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
	corev1 "k8s.io/api/core/v1"
	"os"
)

func main() {
	_ = os.Unsetenv("GOPATH")

	controllergen.Run(args.Options{
		OutputPackage: "github.com/ebauman/klicense/operator/generated",
		Boilerplate: "hack/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"licensing.cattle.io": {
				Types: []interface{}{
					v1.Entitlement{},
					v1.Request{},
				},
				GenerateTypes: true,
			},
			"": {
				Types: []interface{}{
					corev1.Secret{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage: "k8s.io/cilent-go/listers",
			},
		},
	})
}