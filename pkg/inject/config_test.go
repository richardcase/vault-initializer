package inject

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var initConfig = `
    requireAnnotation: false
    annotationName: anannotationname
    ignoreSystemNamespaces: true
    vaultAuthMode: Token 
    vaultAddress: http://127.0.0.1:8200
    vaultPathPattern: /v1/secret/{{.Namespace}}/{{.ContainerName}}
    secretsPublisher: volume
    secretsFilePathPattern: /
    secretsFileNamePattern: "config.json"
    secretNamePattern: "{{.Namespace}}.{{.ContainerName}}"
`

var initConfigNoAnnotation = `
    requireAnnotation: false
    ignoreSystemNamespaces: true
    vaultAuthMode: Token 
    vaultAddress: http://127.0.0.1:8200
    vaultPathPattern: /v1/secret/{{.Namespace}}/{{.ContainerName}}
    secretsPublisher: volume
    secretsFilePathPattern: /
    secretsFileNamePattern: "config.json"
    secretNamePattern: "{{.Namespace}}.{{.ContainerName}}"
`

func TestGetVaultConfigMap(t *testing.T) {
	cm := configMap("default", "vault-initializer", initConfig)
	fakeClient := fake.NewSimpleClientset(&cm)

	config, err := GetInitializerConfig(fakeClient, "default", "vault-initializer")
	if err != nil {
		t.Errorf("Getting config resulted in an error: %v.", err)
	}

	if config.RequireAnnotation != false {
		t.Errorf("Got unexpected RequireAnnotation flag: %t", config.RequireAnnotation)
	}
	if config.AnnotatioName != "anannotationname" {
		t.Errorf("Got unexpected AnnotationName: %s", config.AnnotatioName)
	}
	if config.IgnoreSystemNamespaces != true {
		t.Errorf("Got unexpected IgnoreSystemNamespaces flag: %t", config.IgnoreSystemNamespaces)
	}
	if config.VaultAuthMode != "Token" {
		t.Errorf("Got unexpected VaultAuthMode: %s", config.VaultAuthMode)
	}
	if config.VaultAddress != "http://127.0.0.1:8200" {
		t.Errorf("Got unexpected VaultAuthAddress: %s", config.VaultAddress)
	}
	if config.VaultPathPattern != "/v1/secret/{{.Namespace}}/{{.ContainerName}}" {
		t.Errorf("Got unexpected VaultPathPattern: %s", config.VaultPathPattern)
	}
	if config.SecretsPublisher != "volume" {
		t.Errorf("Got unexpected SecretsPublisher: %s", config.SecretsPublisher)
	}
	if config.SecretsFilePathPattern != "/" {
		t.Errorf("Got unexpected SecretsFilePathPattern: %s", config.SecretsFilePathPattern)
	}
	if config.SecretsFileNamePattern != "config.json" {
		t.Errorf("Got unexpected SecretsFileNamePattern: %s", config.SecretsFileNamePattern)
	}
	if config.SecretNamePattern != "{{.Namespace}}.{{.ContainerName}}" {
		t.Errorf("Got unexpected SecretNamePattern: %s", config.SecretNamePattern)
	}
}

func TestGetVaultConfigMapWithDefaultAnnotation(t *testing.T) {
	cm := configMap("default", "vault-initializer", initConfigNoAnnotation)
	fakeClient := fake.NewSimpleClientset(&cm)

	config, err := GetInitializerConfig(fakeClient, "default", "vault-initializer")
	if err != nil {
		t.Errorf("Getting config resulted in an error: %v.", err)
	}

	if config.AnnotatioName != "initializer.kubernetes.io/vault" {
		t.Errorf("Got unexpected AnnotationName: %s", config.AnnotatioName)
	}
}

func TestMissingVaultConfigMap(t *testing.T) {

	fakeClient := fake.NewSimpleClientset()

	_, err := GetInitializerConfig(fakeClient, "default", "vault-initializer")
	if err == nil {
		t.Error("Getting config resulted in no error where an error was expected")
	}
	if reflect.TypeOf(err).String() != "*errors.StatusError" {
		t.Errorf("Got unexpected error type: %s", reflect.TypeOf(err).String())
	}
}

func configMap(namespace, name string, configData string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string]string{
			"config": configData,
		},
	}
}
