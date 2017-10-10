package model

// Config represents the configuration of the initilaizer
type Config struct {
	RequireAnnotation      bool   `yaml:"requireAnnotation"`
	AnnotatioName          string `yaml:"annotationName"`
	IgnoreSystemNamespaces bool   `yaml:"ignoreSystemNamespaces"`
	VaultAuthMode          string `yaml:"vaultAuthMode"` //TODO: enum??
	VaultAddress           string `yaml:"vaultAddress"`
	VaultPathPattern       string `yaml:"vaultPathPattern"`
}
