package inject

import (
	"errors"

	"github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/client-go/kubernetes"
)

// Publisher is an interface that defines what publishers need to implement.
type Publisher interface {
	PublishSecrets(vaultmap *v1alpha1.VaultMap, clientset *kubernetes.Clientset, deployment *v1beta1.Deployment, secrets map[string]string) error
}

// CreatePublisher create a new secrets publisher
func CreatePublisher(publisherType string) (Publisher, error) {
	switch publisherType {
	case "env":
		return new(EnvironmentPublisher), nil
	case "volume":
		return new(VolumePublisher), nil
	default:
		return nil, errors.New("Invalid Publisher Type")
	}
}
