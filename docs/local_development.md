## Development
The initializer can run outside of Kubernetes to help aid debugging. You can do this by:
```
make install
$GOPATH/bin/vault-initialzer --outside --kubeconfig ~/.kube/config
```