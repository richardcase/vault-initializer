package publisher

import (
	"github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	"github.com/richardcase/vault-initializer/pkg/inject/publisher/environment"
	"github.com/richardcase/vault-initializer/pkg/inject/publisher/volume"
	"k8s.io/api/apps/v1beta1"
	clientset "k8s.io/client-go/kubernetes"
)

// Publisher is an interface that defines what publishers need to implement.
type Publisher interface {
	PublishSecrets(vaultmap *v1alpha1.VaultMap, client clientset.Interface, deployment *v1beta1.Deployment, secrets map[string]string) error
}

// CreatePublisher create a new secrets publisher
func CreatePublisher(publisherType string) (Publisher, error) {
	switch publisherType {
	case "env":
		return new(environment.EnvironmentPublisher), nil
	case "volume":
		return new(volume.VolumePublisher), nil
	default:
		return nil, NewInvalidPublisherError(publisherType)
	}
}
