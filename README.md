 # Vault Initializer [![Build Status](https://travis-ci.org/richardcase/vault-initializer.svg?branch=master)](https://travis-ci.org/richardcase/vault-initializer) [![Go Report Card](https://goreportcard.com/badge/github.com/richardcase/vault-initializer)](https://goreportcard.com/report/github.com/richardcase/vault-initializer) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) #
 
Vault Initializer is a [Kubernetes Initializer](https://kubernetes.io/docs/admin/extensible-admission-controllers/#what-are-initializers) that injects secrets from Vault into a container when a deployment is created. It currently supports 2 ways to publish secrets into a container:
- Environment Variables
- A Kubernetes secret (automatically created) which is then mounted as a volume automatically into the POD.

> This isn't production ready yet. If you would like to help making in production production ready then see the [contributing guide](CONTRIBUTING.md)

## Getting Started

You need Kubernetes 1.7.0+. If you want to use minikube you can use the following to spin up a cluster:

```
minikube start --extra-config=apiserver.Admission.PluginNames="Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota" --kubernetes-version=v1.8.0
```

Edit and deploy the initializer config:
```
kubectl create -f kube/configmaps/vault-initializer.yaml
```

Edit the vault token in the secrets file and deploy:
```
kubectl create -f kube/secrets/vault-initializer.yaml
```
> Only the token authentication backend is currently supported for Vault
 
The Vault Initializer controller needs to be deployed to the cluster:

```
kubectl create -f kube/deployments/vault-initializer.yaml
```

Now create the Kuberenetes initialzer for deployments:
```
kubectl create -f kube/initializer-config/vault.yaml
```

Now when you create a deployment the Vault initializer will be invoked. For example you can deploy a test app that dumps environment variables to the logs:
```
kubectl create -f kube/deployments/envprinter.yaml
```

## Vault Naming Conventions
When the initializer runs it will look for secrets using the following convention:

secret/{deploymentnamespace}/{containername}

For all the secrets in the following path it will inject an enviroment variable or an entry in a JSON config file into the container with the name of the secret and who's value is the value of the secret.

This is controlled using the following template:
```
vaultPathPattern: /v1/secret/{{.Namespace}}/{{.ContainerName}}
```

For example, if we create a secret using the following:
```
vault write secret/default/envprinter mysecret=Password123
```
An environment variable named *mysecret* will be injected into a container named envprinter when the deployment namespace is *defaul*.

## Contributing

If you would like to contribute see the [guide](CONTRIBUTING.md).
