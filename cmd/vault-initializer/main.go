package main

import (
	"flag"
	"time"

	clientset "github.com/richardcase/vault-initializer/pkg/client/clientset/versioned"
	informers "github.com/richardcase/vault-initializer/pkg/client/informers/externalversions"
	"github.com/richardcase/vault-initializer/pkg/initializer"
	"github.com/richardcase/vault-initializer/pkg/signals"
	"github.com/richardcase/vault-initializer/pkg/version"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultInitializerName = "vault.initializer.kubernetes.io"
	defaultConfigmap       = "vault-initializer"
	defaultSecret          = "vault-initializer"
)

var (
	initializerName string
	namespace       string
	kubeconfig      string
	configmap       string
	secretName      string
	masterURL       string
)

func main() {
	flag.Parse()

	glog.Info("Starting the Kubernetes initializer...")
	version.OutputVersion()
	glog.V(2).Infof("Initializer name set to: %s", initializerName)
	glog.V(2).Infof("Using kubeconfig: %s", kubeconfig)

	stopCH := signals.SetupSignalHandler()

	clusterConfig, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		glog.Fatal(err)
	}

	mapClient, err := clientset.NewForConfig(clusterConfig)
	if err != nil {
		glog.Fatalf("Error build vault map clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	mapInformerFactory := informers.NewSharedInformerFactory(mapClient, time.Second*30)

	initializer := initializer.NewInitializer(
		kubeClient,
		mapClient,
		kubeInformerFactory,
		mapInformerFactory,
		namespace,
		configmap,
		secretName,
		initializerName,
		stopCH)

	go kubeInformerFactory.Start(stopCH)
	go mapInformerFactory.Start(stopCH)

	if err = initializer.Run(1, stopCH); err != nil {
		glog.Fatalf("Error running initializer: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&initializerName, "initializer-name", defaultInitializerName, "The initializer name")
	flag.StringVar(&namespace, "namespace", corev1.NamespaceDefault, "The configuration namespace")
	flag.StringVar(&configmap, "configmap", defaultConfigmap, "The vault initializer configuration configmap")
	flag.StringVar(&secretName, "secret", defaultSecret, "The vault initializer secret")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
