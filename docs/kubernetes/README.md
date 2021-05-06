# kubernetes provider

Provider Maturity: alpha

Basic implementation is in place, simple tests for HA, Scale Up and Scale Down have been performed.
Backups are supported only via the file provider for now and a simple helm chart is provided for
easy installation (NOT production ready).

The current implementation of the kubernetes provider is also determined not suitable for self hosted etcd installations backing kubernetes clusters like: https://coreos.com/tectonic/docs/latest/admin/self-hosted-etcd.html
Please consider other solutions when self hosting etcd clusters on kubernetes.


## Overview

The kubernetes(sts) provider is an opinionated framework for running the etcd-cloud-operator in any
given Kubernetes cluster (version >= 1.10).

The provider assumes it is running inside a k8s cluster (will error out if run outside) and within a
statefulset (will error out if run in a Kubernetes deployment for example).

It supports the exact same configuration options as the AWS provider (see:
[example](../../config.example.yaml)).

This project also provides an accompanying `helm` [chart](../../chart/etcd-cloud-operator) to make
it easy to deploy into your cluster. The helm chart can be deployed via all the regular methods:

* Tiller installed within cluster (see helm [docs](https://docs.helm.sh/using_helm/))
* Tiller running locally (see tillerless helm [docs](https://github.com/rimusz/helm-tiller))
* Or simply render the manifests using `helm template` and `kubectl apply`  (see helm template [docs](https://docs.helm.sh/helm/#helm-template))

If you already have tiller installed in your cluster, you can get up and running by simply:

```
git clone https://github.com/Quentin-M/etcd-cloud-operator.git
cd chart
helm install etcd-cloud-operator
```
This will install a basic 3 replica etcd cluster with persistent storage but without client TLS. You can
customize these settings using the variables available in the
[values](../../chart/etcd-cloud-operator/values.yaml) file. These variables directly map to various
configuration options for the operator.

TODO List:

* Add security context in helm chart
* Add E2E Tests for Kubernetes
* Support S3 Snapshot provider (for non AWS deployments)
* Make helm chart available via official helm repository
