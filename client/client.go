package client

import (
	"fmt"
	v1 "github.com/ebauman/klicense/client/generated/controllers/licensing.cattle.io/v1"
	"github.com/ebauman/klicense/operator/generated/controllers/licensing.cattle.io"
	wranglerKubeconfig "github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

var (
	LicenseStatusLicensed LicenseStatus = "Licensed"
	LicenseStatusUnlicensed LicenseStatus = "Unlicensed"
)

type LicenseStatus string

type LicenseClient struct {
	requestClient v1.RequestClient
	namespace string
	requesters []*requester
}

type requester struct {
	licenseName string
	licenseType string
	licenseAmount int
	notify chan<- bool
}

func (l LicenseClient) NewLicenseClient(kubeconfig string) (*LicenseClient, error) {
	ctx := signals.SetupSignalContext()

	clientConfig := wranglerKubeconfig.GetNonInteractiveClientConfig(kubeconfig)

	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	licensingFactory, err := licensing.NewFactoryFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	ns, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, fmt.Errorf("error attempting to obtain namespace of running workload")
	}


	return &LicenseClient{
		requestClient: licensingFactory.Licensing().V1().Request(),
		namespace: ns,
	}, nil
}

// License submits a request for licensing of the calling code application.
// A particular entitlement is identified by licenseName and licenseType.
// A request will be created with these properties as well as the amount.
// Passing a non-nil applicationIdentifier will use that value to search for a Request object.
// This method blocks until the software is licensed.
func (l *LicenseClient) License(licenseName string, licenseType string, amount int, applicationIdentifier string) {

}

// LicenseAsync submits a request for licensing of the calling code application.
// Arguments are the same as License, with the exception of notify.
// Upon successful licensure, notify will emit a bool:true value.
// If the software becomes unlicensed, notify will emit a bool:false value.
func (l *LicenseClient) LicenseAsync(licenseName string, licenseType string, amount int, notify chan<- bool, applicationIdentifier string) {

}