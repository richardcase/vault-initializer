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

	vault "github.com/hashicorp/vault/api"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
	outsideCluster    bool
	kubeconfig        string
	vaultConfig       *vault.Config
	vaultClient       *vault.Client
)

/*type config struct {
	Containers []corev1.Container
	Volumes    []corev1.Volume
}*/

func main() {
	flag.StringVar(&annotation, "annotation", defaultAnnotation, "The annotation to trigger initialization")
	flag.StringVar(&configmap, "configmap", defaultConfigmap, "The envoy initializer configuration configmap")
	flag.StringVar(&initializerName, "initializer-name", defaultInitializerName, "The initializer name")
	flag.StringVar(&namespace, "namespace", "default", "The configuration namespace")
	flag.BoolVar(&requireAnnotation, "require-annotation", false, "Require annotation for initialization")
	flag.BoolVar(&outsideCluster, "outside", false, "Indicates this is running outside cluster")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")

	flag.Parse()

	log.Println("Starting the Kubernetes initializer...")
	log.Printf("Initializer name set to: %s", initializerName)
	log.Printf("Using kubeconfig: %s", kubeconfig)

	clusterConfig, err := getConfig(outsideCluster, kubeconfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatal(err)
	}

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

func getConfig(runningOutside bool, kubeconfig string) (*rest.Config, error) {
	if runningOutside {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return rest.InClusterConfig()
}

// func initializeDeployment(deployment *v1beta1.Deployment, c *config, clientset *kubernetes.Clientset) error {
func initializeDeployment(deployment *v1beta1.Deployment, clientset *kubernetes.Clientset) error {

	//TODO: allow configuration to be overidden
	vaultConfig := vault.DefaultConfig()
	vaultClient, err := vault.NewClient(vaultConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {
			log.Printf("Initializing deployment: %s", deployment.Name)

			o, err := runtime.NewScheme().DeepCopy(deployment)
			if err != nil {
				return err
			}
			initializedDeployment := o.(*v1beta1.Deployment)

			// Remove self from the list of pending Initializers while preserving ordering.
			if len(pendingInitializers) == 1 {
				initializedDeployment.ObjectMeta.Initializers = nil
			} else {
				initializedDeployment.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
			}

			//TODO: if annotation required logic
			if deployment.Namespace == "kube-system" {
				log.Printf("Ignoring deployments in kube-system namespace")
				_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				if err != nil {
					return err
				}
				return nil
			}

			// Modify the Deployments Pod template to add environment variable

			log.Printf("Existing Environment Variables for container '%s':", initializedDeployment.Spec.Template.Spec.Containers[0].Name)
			for _, element := range initializedDeployment.Spec.Template.Spec.Containers[0].Env {
				log.Printf("\t%s = %s\n", element.Name, element.Value)
			}

			// Add environment variables from vault
			containerName := initializedDeployment.Spec.Template.Spec.Containers[0].Name
			vaultPath := fmt.Sprintf("/v1/secret/%s/%s", initializedDeployment.Namespace, containerName)
			log.Printf("Querying vault with path: %s", vaultPath)
			request := vaultClient.NewRequest("GET", vaultPath)
			request.ClientToken = "bfb01bd1-c27c-102c-5f3a-919de99853c5" //TODO: work out how to ghet this
			//request.Params.Set("list", "true")
			resp, err := vaultClient.RawRequest(request)
			if resp != nil {
				defer resp.Body.Close()
			}
			if err != nil {
				return err
			}
			if resp != nil && resp.StatusCode == 404 {
				log.Printf("No secrets in vault for path %s", vaultPath)
				_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				if err != nil {
					return err
				}
				return nil
			}
			secret, err := vault.ParseSecret(resp.Body)
			if err != nil {
				return err
			}
			for key, value := range secret.Data {
				env := corev1.EnvVar{Name: key, Value: value.(string)}
				initializedDeployment.Spec.Template.Spec.Containers[0].Env = append(initializedDeployment.Spec.Template.Spec.Containers[0].Env, env)
			}

			// Adding a hard coded environment variables
			//env := corev1.EnvVar{Name: "CTM_TEST", Value: "HELLO"}
			//initializedDeployment.Spec.Template.Spec.Containers[0].Env = append(initializedDeployment.Spec.Template.Spec.Containers[0].Env, env)

			log.Printf("Modified Environment Variables for container '%s':", initializedDeployment.Spec.Template.Spec.Containers[0].Name)
			for _, element := range initializedDeployment.Spec.Template.Spec.Containers[0].Env {
				log.Printf("\t%s = %s\n", element.Name, element.Value)
			}

			oldData, err := json.Marshal(deployment)
			if err != nil {
				return err
			}

			newData, err := json.Marshal(initializedDeployment)
			if err != nil {
				return err
			}

			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1beta1.Deployment{})
			if err != nil {
				return err
			}

			_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Patch(deployment.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}
			log.Printf("Patched Deployment: %s\n", deployment.Name)
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
