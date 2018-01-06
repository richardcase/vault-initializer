package inject

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetVaultSecret(t *testing.T) {
	fakeSecret := secrets("default", "vault-initializer")
	fakeClient := fake.NewSimpleClientset(&fakeSecret)

	vaultSecrets, err := GetInitializerSecret(fakeClient, "default", "vault-initializer")
	if err != nil {
		t.Errorf("Getting secrets resulted in an unexpected error: %v.", err)
	}

	numSecrets := len(vaultSecrets)
	if numSecrets != 2 {
		t.Errorf("Expected 2 secrets but got %d", numSecrets)
	}
}

func TestGetUnknownSecret(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	_, err := GetInitializerSecret(fakeClient, "default", "vault-initializer")
	if err == nil {
		t.Error("Getting secrets resulted in no error when one was expected")
	}
}

func secrets(namespace, name string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			"secret1": []byte("My secret"),
			"secret2": []byte("This is the second secret"),
		},
	}
}
