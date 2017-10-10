package inject

import (
	"log"

	"github.com/richardcase/k8sinit/model"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetInitializerConfig gets the initializer configuration from a Kubernetes configmap
func GetInitializerConfig(kube kubernetes.Interface, namespace, configName string) (*model.Config, error) {
	log.Printf("Reading config  %s in namespace %s", configName, namespace)
	cm, err := kube.CoreV1().ConfigMaps(namespace).Get(configName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	config, err := configmapToConfig(cm)
	if err != nil {
		return nil, err
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