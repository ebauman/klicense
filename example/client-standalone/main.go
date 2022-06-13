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
	logrus.Infof("starting standalone licensing client")
	logrus.Infof("this client will search for a license secret and become either licensed or not (or error)")

	licensed, err := client.Standalone("", "my.app.domain", "nodes", 5, "standlone-example")

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
