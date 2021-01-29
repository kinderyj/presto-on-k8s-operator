## Introduction

This repo is used to create presto cluster in Kubernetes.

### Basic Functions

1. Create Presto Coordinator, Worker, Service and Configmap automatically according to the user's CR.

2. Presto's etc configs，catalog configs and core-site.xml are all integrated in the CR of PrestoClusters, please refer to the samplae  [prestooperator_v1alpha1_prestocluster.yaml](config/samples/prestooperator_v1alpha1_prestocluster.yaml).

### Run and Deploy

```
# Run in local
make install
make run

# Deploy to the Kubernetes
kubectl apply -f config/crd/bases/prestooperator.k8s.io_prestoclusters.yaml
kubectl apply -f setup/operator.yaml

```

### Quick Start

```
kubectl apply -f config/samples/prestooperator_v1alpha1_prestocluster.yaml

kubectl get pcs
NAME                  AGE
presto-test-cluster   40m

kubectl describe pcs presto-test-cluster
Name:         presto-test-cluster
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  prestooperator.k8s.io/v1alpha1
Kind:         PrestoCluster
Metadata:
  Creation Timestamp:  2021-01-29T15:02:15Z
  Generation:          1
  Resource Version:    5272151740
  Self Link:           /apis/prestooperator.k8s.io/v1alpha1/namespaces/default/prestoclusters/presto-test-cluster
  UID:                 f97254ae-6242-11eb-8098-6e638ff50718
Spec:
  Catalog Config:
  ......
Status:
  Available Workers:  2
Events:
  Type    Reason                 Age   From             Message
  ----    ------                 ----  ----             -------
  Normal  workers counts update  40m   presto-operator  presto-test-cluster workers counts changed: 0 -> 2

kubectl get pod
presto-test-cluster-coordinator-7c8458bb79-sl4lr   1/1     Running     0          42m
presto-test-cluster-worker-54bd79b57c-k6pz6        1/1     Running     0          42m
presto-test-cluster-worker-54bd79b57c-pchdx        1/1     Running     0          42m
```

### Notice about the PrestoClusters's Yaml(onfig/samples/prestooperator_v1alpha1_prestocluster.yaml)
```
All the configs under the spec.coordinatorConfig.etcConfig will be mounted to /usr/lib/presto/etc in the Presto Coordinator's container.
All the configs under the spec.workerConfig.etcConfig will be mounted to /usr/lib/presto/etc in the Presto Workers's container.
All the configs under the spec.catalogConfig will be mounted to /usr/lib/presto/catalog/etc/catalog both in the
Presto Coordinator's container and Workers's container.
All the configs under the spec.coresit will be mounted to the /tmp both in the Presto Coordinator's container and Workers's container.
```