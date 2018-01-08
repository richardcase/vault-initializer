package environment

import (
	"testing"

	vi "github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPublishSecrets(t *testing.T) {
	secrets := make(map[string]string)
	secrets["secret1"] = "my secret1"
	secrets["secret2"] = "my secret2"
	vm := vaultmap("ns", "name")
	depl := deployment("ns", "depname", "contname")
	fakeClient := fake.NewSimpleClientset()

	envpublisher := &EnvironmentPublisher{}
	err := envpublisher.PublishSecrets(&vm, fakeClient, &depl, secrets)

	if err != nil {
		t.Errorf("Got unexpected error publishing secrets as environment variables: %v", err)
	}

	container := depl.Spec.Template.Spec.Containers[0]
	if len(container.Env) != 2 {
		t.Errorf("Unexpected number of environment variables. Got %d but expected %d", len(container.Env), 2)
	}

	//TODO: check the values of the environment variables

}

func vaultmap(namespace string, name string) vi.VaultMap {
	return vi.VaultMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: vi.MapSpec{
			SecretNamePattern:      "",
			SecretsFileNamePattern: "{{.Namespace}}.{{.ContainerName}}",
			SecretsFilePathPattern: "",
			SecretsPublisher:       "env",
			VaultPathPattern:       "/v1/secret/{{.Namespace}}/{{.ContainerName}}",
		},
	}
}

func deployment(namespace string, deploymentname string, containername string) v1beta1.Deployment {
	container := v1.Container{Name: containername}

	return v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      deploymentname,
		},
		Spec: v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{container},
				},
			},
		},
	}
}
