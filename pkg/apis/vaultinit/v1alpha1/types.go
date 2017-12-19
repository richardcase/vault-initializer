package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VaultMap defines a vault map resource
type VaultMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata, omitempty"`
	Spec              MapSpec   `json:"spec"`
	Status            MapStatus `json:"status"`
}

// MapSpec is the spec for a VaultMap resource
type MapSpec struct {
	VaultPathPattern       string `json:"vaultPathPattern"`
	SecretsPublisher       string `json:"secretsPublisher"`
	SecretsFilePathPattern string `json:"secretsFilePathPattern"`
	SecretsFileNamePattern string `json:"secretsFileNamePattern"`
	SecretNamePattern      string `json:"secretNamePattern"`
}

// MapStatus is the status fro the the VaultMap resource
type MapStatus struct {
	PodsWithSecrets []string `json:"podsWithSecrets"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VaultMapList is a list if VaultMap resources
type VaultMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VaultMap `json"items"`
}
