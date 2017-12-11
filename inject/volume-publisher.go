package inject

import (
	"encoding/json"
	"log"
	"path"

	"github.com/richardcase/vault-initializer/model"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// VolumePublisher is a secrets publisher that makes secrets available as a volumne
type VolumePublisher struct{}

// PublishSecrets publishes secrets as a volume.
func (p VolumePublisher) PublishSecrets(config *model.Config, clientset *kubernetes.Clientset, deployment *v1beta1.Deployment, secrets map[string]string) error {
	namespace := deployment.Namespace

	// Resolve templates
	secretName, err := ResolveTemplate(deployment, config.SecretNamePattern)
	secretFilePath, err := ResolveTemplate(deployment, config.SecretsFilePathPattern)
	secretFileName, err := ResolveTemplate(deployment, config.SecretsFileNamePattern)

	// Create full path to secret file
	secretFullPath := path.Join(secretFilePath, secretFileName)

	// Create json in  k8s secrets secrets
	jsonSecrets, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	secret := corev1.Secret{}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[secretFileName] = []byte(jsonSecrets)
	secret.Type = corev1.SecretTypeOpaque
	secret.Name = secretName

	exists, err := secretExists(clientset, namespace, secretName)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("Secret %s already exists in namespace %s", secretName, namespace)
	} else {
		log.Printf("Creating secret %s in namespace %s", secretName, namespace)
		_, err = clientset.CoreV1().Secrets(namespace).Create(&secret)
		if err != nil {
			return err
		}
	}

	// Create volume pointing to secrets
	volume := corev1.Volume{}
	volume.Name = "secrets"
	volume.Secret = &corev1.SecretVolumeSource{SecretName: secretName}
	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

	// Add volume to container

	mount := corev1.VolumeMount{}
	mount.Name = "secrets"
	mount.MountPath = secretFullPath
	mount.ReadOnly = true
	mount.SubPath = secretFileName
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, mount)

	return nil
}

func secretExists(clientset *kubernetes.Clientset, namespace string, secretName string) (bool, error) {
	secretsList, err := clientset.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, secret := range secretsList.Items {
		if secret.ObjectMeta.Namespace == namespace && secret.ObjectMeta.Name == secretName {
			return true, nil
		}
	}
	return false, nil
}
