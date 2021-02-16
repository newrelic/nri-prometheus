package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/util/homedir"

	"github.com/sirupsen/logrus"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
)

var kubeConfigFile = flag.String("kubeconfig", "", "location of the kube config file. Defaults to ~/.kube/config")

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println("")
		fmt.Println("k8s-target-retriever is a simple helper program to run the KubernetesTargetRetriever logic on your own machine, for debugging purposes.")
		fmt.Println("")
		flag.PrintDefaults()
	}
}
func main() {
	flag.Parse()

	if *kubeConfigFile == "" {
		*kubeConfigFile = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	kubeconf := endpoints.WithKubeConfig(*kubeConfigFile)
	ktr, err := endpoints.NewKubernetesTargetRetriever("prometheus.io/scrape", false, kubeconf)
	if err != nil {
		logrus.Fatalf("could not create KubernetesTargetRetriever: %v", err)
	}

	if err := ktr.Watch(); err != nil {
		logrus.Fatalf("could not watch for events: %v", err)
	}

	logrus.Infoln("connected to cluster, watching for targets")

	for range time.Tick(time.Second * 7) {
		targets, err := ktr.GetTargets()
		logrus.Infof("###################################")

		if err != nil {
			logrus.Fatalf("could not get targets: %v", err)
		}
		for _, b := range targets {
			logrus.Infof("%s[%s] %s", b.Name, b.Object.Kind, b.URL.String())
		}
		logrus.Infof("###################################")

		logrus.Println()
	}
}
