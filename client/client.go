package client

import (
	"fmt"
	klicensev1 "github.com/ebauman/klicense/api/v1"
	"github.com/ebauman/klicense/client/controllers"
	"github.com/ebauman/klicense/client/generated/controllers/licensing.cattle.io"
	v1 "github.com/ebauman/klicense/client/generated/controllers/licensing.cattle.io/v1"
	license2 "github.com/ebauman/klicense/license"
	"github.com/google/uuid"
	wranglerCore "github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	wranglerKubeconfig "github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

type LicenseStatus string

type LicenseClient struct {
	requestClient v1.RequestClient
	namespace     string
	notifiers     map[string]chan<- bool
}

const licenseUsedAnnotation string = "licensing.cattle.io/used-by"
const licenseAmountAnnotation string = "licensing.cattle.io/used-amount"

func NewLicenseClient(kubeconfig string) (*LicenseClient, error) {
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

	wrangler, err := wranglerCore.NewFactoryFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	ns, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, fmt.Errorf("error attempting to obtain namespace of running workload")
	}

	l := &LicenseClient{
		requestClient: licensingFactory.Licensing().V1().Request(),
		notifiers:     make(map[string]chan<- bool),
		namespace:     ns,
	}

	controllers.Register(
		ctx,
		licensingFactory.Licensing().V1().Request(),
		ns,
		wrangler.Core().V1().Secret().Cache(),
		l.notifiers,
		licensingFactory.Licensing().V1().Request())

	if err = start.All(ctx, 2, licensingFactory, wrangler); err != nil {
		return nil, fmt.Errorf("error starting controllers: %s", err.Error())
	}

	return l, nil
}

// License submits a request for licensing of the calling code application.
// A particular entitlement is identified by kind and unit.
// A request will be created with these properties as well as the amount.
// Passing a non-nil applicationIdentifier will use that value to search for a Request object.
// This method blocks until the software is licensed.
func (l *LicenseClient) License(kind string, unit string, amount int, applicationIdentifier string) bool {
	req := l.setupLicense(kind, unit, amount, applicationIdentifier)
	if req == nil {
		return false
	}

	notify := make(chan bool, 1)

	l.notifiers[string(req.UID)] = notify

	for {
		status := <-notify
		if status {
			return true // blocking until licensed
		}
	}
}

// LicenseAsync submits a request for licensing of the calling code application.
// Arguments are the same as License, with the exception of notify.
// Upon successful licensure, notify will emit a bool:true value.
// If the software becomes unlicensed, notify will emit a bool:false value.
func (l *LicenseClient) LicenseAsync(kind string, unit string, amount int, notify chan<- bool, applicationIdentifier string) {
	req := l.setupLicense(kind, unit, amount, applicationIdentifier)
	if req == nil {
		notify <- false
		return
	}

	l.notifiers[string(req.UID)] = notify
}

// Standalone looks for a license in the application's namespace that fulfills the kind, unit and amount parameters.
// This method does not create a request object, nor does it require the presence of any custom resources.
// Secrets with the label of licensing.cattle.io/license: "true" will be queried until a satisfactory license is found
// If no satisfactory license is located, this method returns false
func Standalone(kubeconfig string, kind string, unit string, amount int, applicationIdentifier string) (bool, error) {
	clientConfig := wranglerKubeconfig.GetNonInteractiveClientConfig(kubeconfig)

	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return false, fmt.Errorf("error obtaining client config: %s", err.Error())
	}

	wrangler, err := wranglerCore.NewFactoryFromConfig(cfg)
	if err != nil {
		return false, fmt.Errorf("error building wrangler factory: %s", err.Error())
	}

	ns, _, err := clientConfig.Namespace()

	secrets, err := wrangler.Core().V1().Secret().List(ns, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", "licensing.cattle.io/license", "true"),
	})
	if errors.IsNotFound(err) {
		return false, fmt.Errorf("no licensing secrets found")
	}

	if err != nil {
		return false, fmt.Errorf("error listing licensing secrets: %s", err.Error())
	}

	for _, s := range secrets.Items {
		// check if the license is in use
		if a, ok := s.Annotations[licenseUsedAnnotation]; ok {
			if a != applicationIdentifier {
				continue
			}
		}

		license, err := license2.ValidateSecret(&s)
		if err != nil {
			continue
		}

		for grantName, grantAmount := range license.Grants {
			if grantName == fmt.Sprintf("%s/%s", kind, unit) {
				if grantAmount >= amount {
					// this license satisfies
					// annotate that the license is in use
					sCopy := s.DeepCopy()
					sCopy.Annotations[licenseUsedAnnotation] = applicationIdentifier
					sCopy.Annotations[licenseAmountAnnotation] = string(rune(amount))
					sCopy, err := wrangler.Core().V1().Secret().Update(sCopy)
					if err != nil {
						return false, fmt.Errorf("error reserving license for use: %s", err.Error())
					}

					return true, nil
				}
			}
		}
	}

	return false, fmt.Errorf("no license found that satisfies request")
}

func (l *LicenseClient) setupLicense(kind string, unit string, amount int, applicationIdentifier string) *klicensev1.Request {
	if applicationIdentifier == "" {
		hostname, err := os.Hostname()
		if err != nil {
			// couldn't get hostname, just use a random string
			applicationIdentifier = uuid.NewString()
		}
		applicationIdentifier = hostname
	}

	// first, see if there is an existing request
	req, err := l.requestClient.Get(l.namespace, applicationIdentifier, metav1.GetOptions{})
	create := false
	if errors.IsNotFound(err) {
		create = true
		// need to create the request
		req = &klicensev1.Request{}
	} else if err != nil {
		// something else went wrong
		logrus.Errorf("error retrieving request from kubernetes: %s", err.Error())
		return nil
	}

	req.Spec.Kind = kind
	req.Spec.Unit = unit
	req.Spec.Amount = amount
	req.Name = applicationIdentifier
	req.Namespace = l.namespace

	if create {
		req, err = l.requestClient.Create(req)
		if err != nil {
			logrus.Errorf("error creating request in kubernetes: %s", err.Error())
			return nil
		}
	} else {
		req, err = l.requestClient.Update(req)
		if err != nil {
			logrus.Errorf("error updating request in kubernetes: %s", err.Error())
		}
	}

	req.Status.Status = klicensev1.UsageRequestStatusDiscover

	req, err = l.requestClient.UpdateStatus(req)
	if err != nil {
		logrus.Errorf("error updating request status in kubernetes: %s", err.Error())
		return nil
	}

	return req
}
