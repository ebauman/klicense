package main

import (
	"flag"
	"github.com/ebauman/klicense/operator/controllers"
	"github.com/ebauman/klicense/operator/crd"
	"github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io"
	wranglerCore "github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
)

var (
	kubeconfigFile string
	installCRD bool
)

func init() {
	flag.StringVar(&kubeconfigFile, "kubeconfig", "", "Path to a kubeconfig file. Only required if out-of-cluster")
	flag.BoolVar(&installCRD, "installcrd", true, "Install new version of CRD")
	flag.Parse()
}

func main() {
	ctx := signals.SetupSignalContext()

	cfg, err := kubeconfig.GetNonInteractiveClientConfig(kubeconfigFile).ClientConfig()
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	licensingFactory := licensing.NewFactoryFromConfigOrDie(cfg)

	wrangler := wranglerCore.NewFactoryFromConfigOrDie(cfg)

	if installCRD {
		err = crd.Create(ctx, cfg)
		if err != nil {
			logrus.Fatalf("error installing crd: %s", err.Error())
		}
	}

	controllers.RegisterSecretHandler(ctx,
		licensingFactory.Licensing().V1().Entitlement(),
		licensingFactory.Licensing().V1().Request(),
		wrangler.Core().V1().Secret())


	controllers.Register(
		ctx,
		licensingFactory.Licensing().V1().Entitlement(),
		licensingFactory.Licensing().V1().Request(),
		wrangler.Core().V1().Secret(),
		)


	if err := start.All(ctx, 2, licensingFactory, wrangler); err != nil {
		logrus.Fatalf("error starting: %s", err.Error())
	}

	<-ctx.Done()
}