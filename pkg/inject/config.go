package inject

import (
	"github.com/golang/glog"
	"github.com/richardcase/vault-initializer/pkg/model"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultAnnotation = "initializer.kubernetes.io/vault"
)

// GetInitializerConfig gets the initializer configuration from a Kubernetes configmap
func GetInitializerConfig(kube kubernetes.Interface, namespace, configName string) (*model.Config, error) {
	glog.V(2).Infof("Reading config  %s in namespace %s", configName, namespace)
	cm, err := kube.CoreV1().ConfigMaps(namespace).Get(configName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	config, err := configmapToConfig(cm)
	if err != nil {
		return nil, err
	}

	if config.AnnotatioName == "" {
		config.AnnotatioName = defaultAnnotation
	}

	return config, nil
}

func configmapToConfig(configmap *corev1.ConfigMap) (*model.Config, error) {
	var c model.Config
	err := yaml.Unmarshal([]byte(configmap.Data["config"]), &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
