## Introduction

This repo is used to create presto cluster in Kubernetes.

### Basic Functions

1. Create Presto Coordinator, Worker, Service and Configmap automatically according to the user's CR.
2. Support dynamic args(`dynamicArgs`) and dynamic configs(`dynamicConfigs`). Please Refer to the CR located in config/samples/prestooperator_v1alpha1_prestocluster.yaml

### Run and Deploy

```
# Run in local
make install
make run

# Deploy to the Kubernetes
make install
kubectl apply -f setup/operator.yaml

```
