package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultAnnotation      = "initializer.kubernetes.io/vault"
	defaultInitializerName = "vault.initializer.kubernetes.io"
	defaultConfigmap       = "vault-initializer"
	defaultNamespace       = "default"
)

var (
	annotation        string
	configmap         string
	initializerName   string
	namespace         string
	requireAnnotation bool
)

/*type config struct {
	Containers []corev1.Container
	Volumes    []corev1.Volume
}*/

func main() {
	outsideCluster := flag.Bool("outside", false, "Indicates this is running outside cluster")
	var kubeconfig *string
	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")

	flag.Parse()

	log.Println("Starting the Kubernetes initializer...")

	clusterConfig, err := getConfig(outsideCluster, kubeconfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatal(err)
	}

	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	restClient := clientset.AppsV1beta1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "deployments", corev1.NamespaceAll, fields.Everything())

	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchlist.Watch(options)
		},
	}

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &v1beta1.Deployment{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				err := initializeDeployment(obj.(*v1beta1.Deployment), clientset)
				if err != nil {
					log.Println(err)
				}
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")
	close(stop)
}

func getConfig(runningOutside *bool, kubeconfig *string) (*rest.Config, error) {
	if *runningOutside {
		return clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	return rest.InClusterConfig()
}

// func initializeDeployment(deployment *v1beta1.Deployment, c *config, clientset *kubernetes.Clientset) error {
func initializeDeployment(deployment *v1beta1.Deployment, clientset *kubernetes.Clientset) error {
	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {

			o, err := runtime.NewScheme().DeepCopy(deployment)
			if err != nil {
				return err
			}
			initializeDeployment := o.(*v1beta1.Deployment)

			// Remove self from the list of pending Initializers whilse preserving order
			if len(pendingInitializers) == 1 {
				initializeDeployment.ObjectMeta.Initializers = nil
			} else {
				initializeDeployment.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
			}

			//TODO: if annotation required logic

			// Modify the Deployments Pod template to add environment variable
			container := initializeDeployment.Spec.Template.Spec.Containers[0]

			containerStr, _ := json.Marshal(container)
			fmt.Println(string(containerStr))

		}

	}
	return nil
}

/*func configmapToConfig(configmap *corev1.ConfigMap) (*config, error) {
	var c config
	err := yaml.Unmarshal([]byte(configmap.Data["config"]), &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}*/
