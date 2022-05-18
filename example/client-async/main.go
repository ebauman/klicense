package main

import (
	"flag"
	"github.com/ebauman/klicense/client"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

var (
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to a valid kubeconfig")
	flag.Parse()
}

func main() {
	logrus.Infof("starting async licensing client")
	logrus.Infof("this client will request a license and sleep for 10s")
	logrus.Infof("the async license operation _should_ return before that")

	licenseClient, err := client.NewLicenseClient(kubeconfig)
	if err != nil {
		logrus.Fatalf("error creating license client: %s", err.Error())
	}

	logrus.Infof("calling license client async")

	notify := make(chan bool, 1)
	licenseClient.LicenseAsync("stigatron.compliance.cattle.io", "nodes", 5, notify, "")

	go func() {
		licensed := <-notify

		if licensed {
			logrus.Info("success! licensed")
		} else {
			logrus.Error("license failed!")
		}
	}()

	time.Sleep(10 * time.Second)

	logrus.Info("completed")
	os.Exit(0)
}