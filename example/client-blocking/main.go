package main

import (
	"flag"
	"github.com/ebauman/klicense/client"
	"github.com/sirupsen/logrus"
	"os"
)

var (
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to a valid kubeconfig")
	flag.Parse()
}

func main() {
	logrus.Infof("starting blocking licensing client")
	logrus.Infof("this client will request a license and wait for completion")

	licenseClient, err := client.NewLicenseClient(kubeconfig)
	if err != nil {
		logrus.Fatalf("error creating license client: %s", err.Error())
	}

	logrus.Infof("calling license client async")

	result := licenseClient.License("stigatron.compliance.cattle.io", "sdfsdfsdf", 5, "")
	if result {
		logrus.Info("success! licensed")
	} else {
		logrus.Errorf("error licensing")
	}

	logrus.Info("completed")
	os.Exit(0)
}