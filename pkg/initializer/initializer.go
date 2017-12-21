package initializer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	vault "github.com/hashicorp/vault/api"
	clientset "github.com/richardcase/vault-initializer/pkg/client/clientset/versioned"
	mapscheme "github.com/richardcase/vault-initializer/pkg/client/clientset/versioned/scheme"
	informers "github.com/richardcase/vault-initializer/pkg/client/informers/externalversions"
	listers "github.com/richardcase/vault-initializer/pkg/client/listers/vaultinit/v1alpha1"
	"github.com/richardcase/vault-initializer/pkg/inject"
	"github.com/richardcase/vault-initializer/pkg/model"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1beta2"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const agentName = "vault-initializer"

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
	recorder  record.EventRecorder
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
	initializerName string,
	stopCh <-chan struct{}) *Initializer {

	config, err := inject.GetInitializerConfig(kubeclientset, namespace, configmapName)
	if err != nil {
		glog.Fatal(err)
	}

	secrets, err := inject.GetInitializerSecret(kubeclientset, namespace, secretName)
	if err != nil {
		glog.Fatal(err)
	}

	//TODO: with the current version (v1.8) this doesn't pick up unitialized deployments
	// see: https://github.com/kubernetes/kubernetes/pull/51247
	//deploymentInformer := kubeInformerFactory.Apps().V1beta2().Deployments()
	mapsInformer := mapsInformerFactory.Vaultinit().V1alpha1().VaultMaps()

	// TODO: Remove this when the above is true
	restClient := kubeclientset.AppsV1beta1().RESTClient()
	watchList := cache.NewListWatchFromClient(restClient, "deployments", metav1.NamespaceAll, fields.Everything())
	includeUninitializwdWatchList := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchList.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchList.Watch(options)
		},
	}
	//**** End Temp Code

	mapscheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: agentName})

	initializer := &Initializer{
		kubeclientset: kubeclientset,
		mapclientset:  mapclientset,
		namespace:     namespace,
		config:        config,
		secrets:       secrets,
		//deploymentsLister: deploymentInformer.Lister(),
		//deploymentsSynced: deploymentInformer.Informer().HasSynced,
		deploymentsLister: nil,
		mapsLister:        mapsInformer.Lister(),
		mapsSynced:        mapsInformer.Informer().HasSynced,
		initializerName:   initializerName,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "InitDeployments"),
		recorder:          recorder,
	}

	glog.Info("Setting up event handlers")
	mapsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			glog.Infof("New map %v", obj)
		},
	})

	// Setup event handler for when Deployments resources change
	//deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	_, deploymentInfomer := cache.NewInformer(includeUninitializwdWatchList, &v1beta1.Deployment{}, time.Second*30, cache.ResourceEventHandlerFuncs{
		AddFunc: initializer.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*v1beta1.Deployment)
			oldDepl := old.(*v1beta1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				glog.V(2).Infof("Skipping deployment %s as old and new versions are the same %s", newDepl.Name, newDepl.ResourceVersion)
				return
			}
			initializer.handleObject(new)
		},
		DeleteFunc: initializer.handleObject,
	})

	//NOTE: These are temporary
	initializer.setDeploymentCache(deploymentInfomer.HasSynced)
	go deploymentInfomer.Run(stopCh)

	return initializer
}

func (i *Initializer) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer i.workqueue.ShutDown()

	glog.Info("Starting vault initializer")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	//if ok := cache.WaitForCacheSync(stopCh, i.deploymentsSynced, i.mapsSynced); !ok {
	if ok := cache.WaitForCacheSync(stopCh, i.mapsSynced); !ok {
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
	glog.V(2).Info("Enter processNextWorkItem")
	obj, shutdown := i.workqueue.Get()

	if shutdown {
		return false
	}

	glog.V(2).Info("Starting processing queue item")
	err := func(obj interface{}) error {
		defer i.workqueue.Done(obj)

		var depl *v1beta1.Deployment
		var ok bool

		if depl, ok = obj.(*v1beta1.Deployment); !ok {
			i.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("Could cast deployment work item to v1beta1.Deployment"))
			return nil
		}

		if err := i.initializeDeployment(depl); err != nil {
			i.recorder.Event(depl, corev1.EventTypeWarning, "Error initialising deploymemnt", err.Error())
			return nil
		}

		i.workqueue.Forget(obj)
		glog.Info("Successfully initialized deployment %s", depl.Name)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	glog.V(2).Info("Exiting processNextWorkItem")
	return true
}

func (i *Initializer) handleObject(obj interface{}) {
	glog.V(2).Info("In handle object")
	i.workqueue.AddRateLimited(obj)
}

func (i *Initializer) initializeDeployment(deployment *v1beta1.Deployment) error {

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

			maps, err := i.mapsLister.VaultMaps(initializedDeployment.Namespace).List(labels.NewSelector())
			if err != nil {
				return err
			}
			if len(maps) == 0 {
				glog.V(2).Infof("No VaultMap for namespace %s; skipping vault injection", initializedDeployment.Namespace)
				_, err = i.kubeclientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				return err
			}
			vaultmap := maps[0]

			vaultPath, err := inject.ResolveTemplate(initializedDeployment, vaultmap.Spec.VaultPathPattern)
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
				glog.Errorf("Error querying vault for secrets for %s: %v", vaultPath, err.Error())
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
			publisher, err := inject.CreatePublisher(vaultmap.Spec.SecretsPublisher)
			if err != nil {
				return err
			}
			err = publisher.PublishSecrets(vaultmap, i.kubeclientset.(*kubernetes.Clientset), initializedDeployment, secrets)
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

// NOTE: This is a function only until the deployment informer supports unitiilaized
func (i *Initializer) setDeploymentCache(synced cache.InformerSynced) {
	i.deploymentsSynced = synced
}
