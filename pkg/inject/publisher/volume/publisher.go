package volume

import (
	"encoding/json"
	"log"
	"path"

	"github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	"github.com/richardcase/vault-initializer/pkg/inject/template"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// VolumePublisher is a secrets publisher that makes secrets available as a volume
type VolumePublisher struct{}

// PublishSecrets publishes secrets as a volume.
func (p VolumePublisher) PublishSecrets(vaultmap *v1alpha1.VaultMap, client kubernetes.Interface, deployment *v1beta1.Deployment, secrets map[string]string) error {
	namespace := deployment.Namespace

	// Resolve templates
	secretName, err := template.ResolveTemplate(deployment, vaultmap.Spec.SecretNamePattern)
	if err != nil {
		return err
	}
	secretFilePath, err := template.ResolveTemplate(deployment, vaultmap.Spec.SecretsFilePathPattern)
	if err != nil {
		return err
	}
	secretFileName, err := template.ResolveTemplate(deployment, vaultmap.Spec.SecretsFileNamePattern)
	if err != nil {
		return err
	}

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

	exists, err := secretExists(client, namespace, secretName)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("Secret %s already exists in namespace %s", secretName, namespace)
	} else {
		log.Printf("Creating secret %s in namespace %s", secretName, namespace)
		_, err = client.CoreV1().Secrets(namespace).Create(&secret)
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

func secretExists(client kubernetes.Interface, namespace string, secretName string) (bool, error) {
	secretsList, err := client.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
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
