package inject

import (
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// EnvironmentPublisher is a secrets publisher that makes secrets available as environment variabls
type EnvironmentPublisher struct{}

// PublishSecrets publishes secrets as environment variables.
func (p EnvironmentPublisher) PublishSecrets(clientset *kubernetes.Clientset, deployment *v1beta1.Deployment, secrets map[string]string) error {
	for key, value := range secrets {
		env := corev1.EnvVar{Name: key, Value: value}
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, env)
	}

	return nil
}
