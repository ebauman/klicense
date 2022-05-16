package main

import (
	v1 "github.com/ebauman/klicense/api/v1"
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
	"os"
)

func main() {
	_ = os.Unsetenv("GOPATH")

	controllergen.Run(args.Options{
		OutputPackage: "github.com/ebauman/klicense/client/generated",
		Boilerplate: "hack/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"licensing.cattle.io": {
				Types: []interface{}{
					v1.Request{},
				},
			},
		},
	})
}