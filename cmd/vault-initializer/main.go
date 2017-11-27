package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/richardcase/vault-initializer/inject"
	"github.com/richardcase/vault-initializer/model"

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
	defaultInitializerName = "vault.initializer.kubernetes.io"
	defaultConfigmap       = "vault-initializer"
	defaultSecret          = "vault-initializer"
)

var (
	initializerName string
	namespace       string
	outsideCluster  bool
	kubeconfig      string
	configmap       string
	secretName      string
	vaultConfig     *vault.Config
	vaultClient     *vault.Client
	secrets         map[string]string
	config          *model.Config
)

func main() {
	flag.StringVar(&initializerName, "initializer-name", defaultInitializerName, "The initializer name")
	flag.StringVar(&namespace, "namespace", corev1.NamespaceDefault, "The configuration namespace")
	flag.StringVar(&configmap, "configmap", defaultConfigmap, "The vault initializer configuration configmap")
	flag.StringVar(&secretName, "secret", defaultSecret, "The vault initializer secret")
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

	config, err = inject.GetInitializerConfig(clientset, namespace, configmap)
	if err != nil {
		log.Fatal(err)
	}

	secrets, err = inject.GetInitializerSecret(clientset, namespace, secretName)
	if err != nil {
		log.Fatal(err)
	}

	restClient := clientset.AppsV1beta1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "deployments", metav1.NamespaceAll, fields.Everything())

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

func createPath(deployment *v1beta1.Deployment, pathTemplate string) (string, error) {
	pc := model.PathConfig{Namespace: deployment.Namespace, DeploymentName: deployment.Name, ContainerName: deployment.Spec.Template.Spec.Containers[0].Name}
	tmpl, err := template.New("pathTemplate").Parse(pathTemplate)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, pc)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func initializeDeployment(deployment *v1beta1.Deployment, clientset *kubernetes.Clientset) error {

	//TODO: Move this else where
	vaultConfig := vault.DefaultConfig()
	if config.VaultAddress != "" {
		vaultConfig.Address = config.VaultAddress
	}
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

			if config.IgnoreSystemNamespaces && deployment.Namespace == "kube-system" {
				log.Printf("Ignoring deployments in kube-system namespace")
				_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				return err
			}

			if config.RequireAnnotation {
				a := deployment.ObjectMeta.GetAnnotations()
				_, ok := a[config.AnnotatioName]
				if !ok {
					log.Printf("Required '%s' annotation missing; skipping vault injection", config.AnnotatioName)
					_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
					return err
				}
			}

			// Add environment variables from vault
			vaultPath, err := createPath(initializedDeployment, config.VaultPathPattern)
			if err != nil {
				return err
			}
			log.Printf("Querying vault with path: %s", vaultPath)
			request := vaultClient.NewRequest("GET", vaultPath)
			if config.VaultAuthMode == "Token" {
				request.ClientToken = secrets["vaultToken"]
			}
			resp, err := vaultClient.RawRequest(request)
			if err != nil {
				return err
			}

			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if resp != nil && resp.StatusCode == 404 {
				log.Printf("No secrets in vault for path %s", vaultPath)
				_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				return err
			}
			secret, err := vault.ParseSecret(resp.Body)
			if err != nil {
				return err
			}
			secrets = make(map[string]string)
			for key, value := range secret.Data {
				secrets[key] = value.(string)
			}
			publisher, err := inject.CreatePublisher(config.SecretsPublisher)
			if err != nil {
				return err
			}
			err = publisher.PublishSecrets(clientset, initializedDeployment, secrets)
			if err != nil {
				return err
			}

			oldData, err := json.Marshal(deployment)
			if err != nil {
				return err
			}

			// Flag that this container has vault secrets
			if initializedDeployment.Spec.Template.Annotations == nil {
				annotations := make(map[string]string)
				initializedDeployment.Spec.Template.SetAnnotations(annotations)
			}
			initializedDeployment.Spec.Template.Annotations["vault-secrets-initialized"] = "true"

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
