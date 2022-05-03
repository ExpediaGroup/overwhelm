# overwhelm
![test](https://github.com/ExpediaGroup/overwhelm/workflows/test/badge.svg?branch=master)

Overwhelm is an operator that facilitate application deployment on Kubernetes.

Traditionally, deploying an application on Kubernetes consists of creating multiple resources.
Rather than having to create multiple resources and monitor them separately, Overwhelm aims to
instead define an entire "application" in a single Kubernetes object, and expose all status
information through that same object, effectively making it possible to manage and monitor an
entire application by using a single resource.


## Development 
This repository uses [operator-sdk](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/).

To create a new CRD with the corresponding Go files:
```console
operator-sdk create api --group core --version v1alpha1 --kind Application --resource=true --controller=true --namespaced=true
```
