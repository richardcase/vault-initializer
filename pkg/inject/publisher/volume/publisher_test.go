package volume

import (
	"encoding/json"
	"testing"

	vi "github.com/richardcase/vault-initializer/pkg/apis/vaultinit/v1alpha1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPublishSecretsNoSecretExisting(t *testing.T) {
	secrets := make(map[string]string)
	secrets["secret1"] = "my secret1"
	secrets["secret2"] = "my secret2"

	vm := vaultmap("ns", "name")
	depl := deployment("ns", "depname", "contname")
	fakeClient := fake.NewSimpleClientset()

	volpublisher := &VolumePublisher{}
	err := volpublisher.PublishSecrets(&vm, fakeClient, &depl, secrets)

	if err != nil {
		t.Errorf("Got unexpected error publishing secrets as volume: %v", err)
	}

	// Check new secret has been created
	createdSecret, err := fakeClient.CoreV1().Secrets("ns").Get("ns.contname", metav1.GetOptions{})
	if createdSecret == nil {
		t.Error("Expected a secret to be created but none created")
	}
	//actualsecret := string(createdSecret.Data["config.json"])
	secretsMap := make(map[string]interface{})
	err = json.Unmarshal(createdSecret.Data["config.json"], &secretsMap)
	if err != nil {
		t.Errorf("Error unmarshalling JSON secret data: %v", err)
	}
	if len(secretsMap) != 2 {
		t.Errorf("Got unexpected number of secrets")
	}
	if secretsMap["secret1"] != "my secret1" {
		t.Errorf("Got unexpected value for secret: %s", secretsMap["secret1"])
	}
	if secretsMap["secret2"] != "my secret2" {
		t.Errorf("Got unexpected value for secret: %s", secretsMap["secret2"])
	}

	// Check the volume has been created
	numVolumes := len(depl.Spec.Template.Spec.Volumes)
	if numVolumes != 1 {
		t.Errorf("Got unexpected number of volumes. Got %d but expected 1", numVolumes)
	}
	secretsVolume := depl.Spec.Template.Spec.Volumes[0]
	if secretsVolume.Name != "secrets" {
		t.Errorf("Got unexpected name for volume: %s", secretsVolume.Name)
	}
	if secretsVolume.Secret.SecretName != "ns.contname" {
		t.Errorf("Got unexpected secret name for volume: %s", secretsVolume.Secret.SecretName)
	}

	// Check the volume mount
	//deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	numMounts := len(depl.Spec.Template.Spec.Containers[0].VolumeMounts)
	if numMounts != 1 {
		t.Errorf("Got unexpected number of volume mounts. Got %d but expected 1", numMounts)
	}

}

func vaultmap(namespace string, name string) vi.VaultMap {
	return vi.VaultMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: vi.MapSpec{
			SecretNamePattern:      "{{.Namespace}}.{{.ContainerName}}",
			SecretsFileNamePattern: "config.json",
			SecretsFilePathPattern: "/",
			SecretsPublisher:       "volume",
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
