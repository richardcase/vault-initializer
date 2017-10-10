 # Vault Initializer #
 
Vault Initializer is a [Kubernetes Initializer](https://kubernetes.io/docs/admin/extensible-admission-controllers/#what-are-initializers) that injects secrets from Vault into new environment variables for a container when a deployment is created.

> This is a proof-of-concept and isn't production ready yet.

> This is built based on the sample provided by Kelsey Hightower [here](https://github.com/kelseyhightower/kubernetes-initializer-tutorial).

## Getting Started

You need Kubernetes 1.7.0+. If you want to use minikube you can use the following to spin up a cluster:

```
minikube start --extra-config=apiserver.Admission.PluginNames="Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota" --kubernetes-version=v1.7.5
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

For all the secrets in the following path it will inject an enviroment variable into the container with the name of the secret and who's value is the value of the secret.

For example, if we create a secret using the following:
```
vault write secret/default/envprinter mysecret=Password123
```
An envieonment variable named *mysecret* will be injected into a container named envprinter when the deployment namespace is *defaul*.

> This needs to change to allow customization

## Development
The initializer can run outside of Kubernetes to help aid debugging. You can do this by:
```
make install
$GOPATH/bin/vault-initialzer --outside --kubeconfig ~/.kube/config
```