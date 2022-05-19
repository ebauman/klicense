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

	licensed, err := client.Standalone("", "stigatron.compliance.cattle.io", "nodes", 5)

	if err != nil {
		logrus.Fatal(err)
	}

	if licensed {
		logrus.Info("licensed")
		os.Exit(0)
	} else {
		logrus.Errorf("unlicensed: %s", err.Error())
		os.Exit(1)
	}
}