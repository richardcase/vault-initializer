package initializer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	vault "github.com/hashicorp/vault/api"
	clientset "github.com/richardcase/vault-initializer/pkg/client/clientset/versioned"
	informers "github.com/richardcase/vault-initializer/pkg/client/informers/externalversions"
	listers "github.com/richardcase/vault-initializer/pkg/client/listers/vaultinit/v1alpha1"
	"github.com/richardcase/vault-initializer/pkg/inject"
	"github.com/richardcase/vault-initializer/pkg/model"
	"k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1beta2"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Initializer is the implemntation for the vault initializer
type Initializer struct {
	kubeclientset kubernetes.Interface
	mapclientset  clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	mapsLister        listers.VaultMapLister
	mapsSynced        cache.InformerSynced

	namespace       string
	secrets         map[string]string
	config          *model.Config
	initializerName string

	workqueue workqueue.RateLimitingInterface
}

// NewInitializer returns a new vault initializer
func NewInitializer(
	kubeclientset kubernetes.Interface,
	mapclientset clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	mapsInformerFactory informers.SharedInformerFactory,
	namespace string,
	configmapName string,
	secretName string,
	initializerName string) *Initializer {

	config, err := inject.GetInitializerConfig(kubeclientset, namespace, configmapName)
	if err != nil {
		glog.Fatal(err)
	}

	secrets, err := inject.GetInitializerSecret(kubeclientset, namespace, secretName)
	if err != nil {
		glog.Fatal(err)
	}

	deploymentInformer := kubeInformerFactory.Apps().V1beta2().Deployments()
	mapsInformer := mapsInformerFactory.Vaultinit().V1alpha1().VaultMaps()

	//TODO: event braodcaster?????

	initializer := &Initializer{
		kubeclientset:     kubeclientset,
		mapclientset:      mapclientset,
		namespace:         namespace,
		config:            config,
		secrets:           secrets,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		mapsLister:        mapsInformer.Lister(),
		mapsSynced:        mapsInformer.Informer().HasSynced,
		initializerName:   initializerName,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "InitDeployments"),
	}

	glog.Info("Setting up event handlers")
	// Setup event handler for when Deployments resources change
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: initializer.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1beta2.Deployment)
			oldDepl := old.(*appsv1beta2.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				return
			}
			initializer.handleObject(new)
		},
		DeleteFunc: initializer.handleObject,
	})

	//TODO: resyncPeriod := 30 * time.Second

	return initializer
}

func (i *Initializer) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer i.workqueue.ShutDown()

	glog.Info("Starting vault initializer")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, i.deploymentsSynced, i.mapsSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	for it := 0; it < threadiness; it++ {
		go wait.Until(i.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

func (i *Initializer) runWorker() {
	for i.processNextWorkItem() {
	}
}

func (i *Initializer) processNextWorkItem() bool {
	obj, shutdown := i.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer i.workqueue.Done(obj)

		var depl *appsv1beta2.Deployment
		var ok bool

		if depl, ok = obj.(*appsv1beta2.Deployment); !ok {
			i.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("error processing deployment"))
			return nil
		}

		if err := i.initializeDeployment(depl); err != nil {
			return fmt.Errorf("Error initializing deployment. %v", err)
		}

		i.workqueue.Forget(obj)
		glog.Info("Successfully initialized deployment %s", depl.Name)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return false

}

func (i *Initializer) handleObject(obj interface{}) {
	i.workqueue.AddRateLimited(obj)
}

func (i *Initializer) initializeDeployment(deployment *appsv1beta2.Deployment) error {

	//TODO: Move this else where
	vaultConfig := vault.DefaultConfig()
	if i.config.VaultAddress != "" {
		vaultConfig.Address = i.config.VaultAddress
	}
	vaultClient, err := vault.NewClient(vaultConfig)
	if err != nil {
		glog.Fatal(err.Error())
	}

	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if i.initializerName == pendingInitializers[0].Name {
			glog.Infof("Initializing deployment: %s", deployment.Name)

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

			if i.config.IgnoreSystemNamespaces && deployment.Namespace == "kube-system" {
				glog.Infof("Ignoring deployments in kube-system namespace")
				_, err = i.kubeclientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				return err
			}

			if i.config.RequireAnnotation {
				a := deployment.ObjectMeta.GetAnnotations()
				_, ok := a[i.config.AnnotatioName]
				if !ok {
					glog.V(2).Infof("Required '%s' annotation missing; skipping vault injection", i.config.AnnotatioName)
					_, err = i.kubeclientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
					return err
				}
			}

			vaultPath, err := inject.ResolveTemplate(initializedDeployment, i.config.VaultPathPattern)
			if err != nil {
				return err
			}
			glog.V(2).Infof("Querying vault with path: %s", vaultPath)
			request := vaultClient.NewRequest("GET", vaultPath)
			if i.config.VaultAuthMode == "Token" {
				request.ClientToken = i.secrets["vaultToken"]
			}
			resp, err := vaultClient.RawRequest(request)
			if err != nil {
				glog.Errorf("Error querying vault for secrets for %s: %v", vaultPath, err)
				return err
			}

			defer func() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
			}()

			if resp != nil && resp.StatusCode == 404 {
				glog.Infof("No secrets in vault for path %s", vaultPath)
				_, err = i.kubeclientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				return err
			}
			secret, err := vault.ParseSecret(resp.Body)
			if err != nil {
				return err
			}
			secrets := make(map[string]string)
			for key, value := range secret.Data {
				i.secrets[key] = value.(string)
			}
			publisher, err := inject.CreatePublisher(i.config.SecretsPublisher)
			if err != nil {
				return err
			}
			err = publisher.PublishSecrets(i.config, i.kubeclientset.(*kubernetes.Clientset), initializedDeployment, secrets)
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

			_, err = i.kubeclientset.AppsV1beta1().Deployments(deployment.Namespace).Patch(deployment.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}
			glog.Infof("Patched Deployment: %s\n", deployment.Name)
		}
	}
	return nil
}
