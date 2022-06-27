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

## Debugging
Since this project is a Kubernetes operator triggered by resource creation, debugging can be done using a remote debugger.

### Steps to achieve remote debugging:

* Make sure you install `kind`
* Create a kind cluster with the command in your terminal, `kind create cluster --name overwhelm`
* Use this cluster for all your debugging needs
* To build and deploy your operator run, `make deploy`
* Once the operator resources are deployed and running successfully in the cluster, run the `make run-delve` command
* In GoLand use remote debugger by selecting the `GO Remote` configuration. Within the configuration, host is `localhost` and port is `2345`
* Now you can debug your code. To test the operator logic, deploy whatever resources you need from the config/samples folder.
* Run `make undeploy` to remove your kubernetes resources once you are done

## Testing
* Kubebuilder uses @Ginkgo and @Gomega test suites which are configured in controllers/suite_test.go
* Run `make test` to test your changes. 
* Reference materials: [Gomega](https://onsi.github.io/gomega/) and [Ginkgo](https://onsi.github.io/ginkgo/)
