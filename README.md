# overwhelm
Overwhelm is an application that facilitate application deployment on Kubernetes.

## Development 
This repository uses [operator-sdk](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/).

To create a new CRD with the corresponding Go files:
```console
operator-sdk create api --group core --version v1alpha1 --kind Application --resource=true --controller=true --namespaced=true
```
