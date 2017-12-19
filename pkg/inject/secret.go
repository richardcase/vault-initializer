package inject

import (
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetInitializerSecret gets the initializer secrets from a Kubernetes secret
func GetInitializerSecret(kube kubernetes.Interface, namespace, secretName string) (map[string]string, error) {
	glog.V(2).Infof("Reading secret %s in namespace %s", secretName, namespace)
	secret, err := kube.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]string)
	for key, value := range secret.Data {
		secrets[key] = string(value)
	}

	return secrets, nil
}
