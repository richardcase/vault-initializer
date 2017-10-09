 minikube start --kubernetes-version v1.7.5 --feature-gates=AllAlpha=true


  minikube start --extra-config=apiserver.Admission.PluginNames="Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota" --feature-gates=AllAlpha=true --kubernetes-version=v1.7.5

  minikube start --extra-config=apiserver.Admission.PluginNames="Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota" --kubernetes-version=v1.7.5

 $GOPATH/bin/k8sinit --outside --kubeconfig ~/.kube/config

 vault write secret/default/envprinter mysecret=Password123