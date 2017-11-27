package inject

import (
	"encoding/json"

	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// VolumePublisher is a secrets publisher that makes secrets available as a volumne
type VolumePublisher struct{}

// PublishSecrets publishes secrets as a volume.
func (p VolumePublisher) PublishSecrets(clientset *kubernetes.Clientset, deployment *v1beta1.Deployment, secrets map[string]string) error {
	namespace := deployment.Namespace
	secretName := namespace + "." + deployment.Spec.Template.Spec.Containers[0].Name //TODO: make this a template
	container := deployment.Spec.Template.Spec.Containers[0]

	// Create json in  k8s secrets secrets
	jsonSecrets, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	//encoded := base64.StdEncoding.EncodeToString(jsonSecrets)

	secret := corev1.Secret{}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	//secret.Data["secrets.json"] = []byte(encoded)
	secret.Data["secrets.json"] = []byte(jsonSecrets)
	secret.Type = corev1.SecretTypeOpaque
	secret.Name = secretName

	//TODO: check if secret already exists
	_, err = clientset.CoreV1().Secrets(namespace).Create(&secret)
	if err != nil {
		return err
	}

	// Create volume pointing to secrets
	volume := corev1.Volume{}
	volume.Name = "secrets"
	volume.Secret = &corev1.SecretVolumeSource{SecretName: secretName}
	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

	// Add volume to container

	mount := corev1.VolumeMount{}
	mount.Name = "secrets"
	mount.MountPath = "/"
	mount.ReadOnly = true
	container.VolumeMounts = append(container.VolumeMounts, mount)

	return nil
}
