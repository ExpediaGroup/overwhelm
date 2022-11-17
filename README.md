# overwhelm
![test](https://github.com/ExpediaGroup/overwhelm/workflows/test/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/ExpediaGroup/overwhelm)](https://goreportcard.com/report/github.com/ExpediaGroup/overwhelm)

Overwhelm is an operator that facilitates application deployment on Kubernetes.

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

## Running Overwhelm on Kind
> ⚠ Make sure you have the required binaries installed. If you don't have them, execute `make kind-install-deps`
- Create the Kind cluster by executing `make kind-create-cluster`
- Deploy overwhelm by executing `make kind-deploy`
- (optional) Install a dummy Application by executing `make kind-install-dummy-app`


To clean the Kind cluster, execute `make kind-clean`.

## Helm Charts
- The .charts folder contains the `overwhelm` helm chart.
- `make helmManifests` will update the chart templates if any changes are made to the helm resources. 
- To debug using helm install of the resources execute `make kind-helm-debug` once you create a Kind cluster.


## Prerequisite
This project is based on golang 1.17. Make sure your GoRoot is configured to 1.17. If any other version then you may face issues.


## Testing
* Kubebuilder uses @Ginkgo and @Gomega test suites which are configured in controllers/suite_test.go
* Run `make test` to test your changes. 
* Reference materials: [Gomega](https://onsi.github.io/gomega/) and [Ginkgo](https://onsi.github.io/ginkgo/)


