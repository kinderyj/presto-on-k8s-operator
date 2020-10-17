/*
Copyright 2020 yujunwang.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	prestooperatorv1alpha1 "github.com/kinderyj/presto-on-k8s-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStateInspector gets the Inspected state of the cluster.
type ClusterStateInspector struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
}

// InspectedClusterState holds Inspected state of a cluster.
type InspectedClusterState struct {
	cluster               *prestooperatorv1alpha1.PrestoCluster
	prestoConfigMap       *corev1.ConfigMap
	catalogConfigMap      *corev1.ConfigMap
	coordinatorDeployment *appsv1.Deployment
	coordinatorService    *corev1.Service
	workerDeployment      *appsv1.Deployment
}

// Inspect the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (inspector *ClusterStateInspector) inspect(
	inspected *InspectedClusterState) error {
	var err error
	var log = inspector.log

	// Cluster state.
	// The function inspector.inspectedCluster(inspectedCluster) get the prestoclusters from the API Server(ETCD),
	// which holds the desired prostoclusters object.
	var inspectedCluster = new(prestooperatorv1alpha1.PrestoCluster)
	err = inspector.inspectCluster(inspectedCluster)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get the cluster resource")
			return err
		}
		log.Info("Inspected cluster", "cluster", "nil")
		inspectedCluster = nil
	} else {
		log.Info("Inspected cluster", "cluster", *inspectedCluster)
		inspected.cluster = inspectedCluster
	}

	// ConfigMap for Catalog config.
	var inspectedCatalogConfigMap = new(corev1.ConfigMap)
	err = inspector.inspectConfigMap(inspectedCatalogConfigMap, "-catalog-configmap")
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get configMap %s", "CatalogConfigConfigmap")
			return err
		}
		log.Info("Inspected configMap", "state", "nil")
		inspectedCatalogConfigMap = nil
	} else {
		log.Info("Inspected configMap", "state", *inspectedCatalogConfigMap)
		inspected.catalogConfigMap = inspectedCatalogConfigMap
	}
	// ConfigMap for Presto config.
	var inspectedPrestoConfigMap = new(corev1.ConfigMap)
	err = inspector.inspectConfigMap(inspectedPrestoConfigMap, "-presto-configmap")
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get configMap %s", "PrestoConfigMap")
			return err
		}
		log.Info("Inspected configMap", "state", "nil")
		inspectedPrestoConfigMap = nil
	} else {
		log.Info("Inspected configMap", "state", *inspectedPrestoConfigMap)
		inspected.prestoConfigMap = inspectedPrestoConfigMap
	}

	// Coordiator deployment.
	var inspectedCoordiatorDeployment = new(appsv1.Deployment)
	err = inspector.inspectCoordiatorDeployment(inspectedCoordiatorDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Coordiator deployment")
			return err
		}
		log.Info("Inspected Coordiator deployment", "state", "nil")
		inspectedCoordiatorDeployment = nil
	} else {
		log.Info("Inspected Coordiator deployment", "state", *inspectedCoordiatorDeployment)
		inspected.coordinatorDeployment = inspectedCoordiatorDeployment
	}

	// Coordiator service.
	var inspectedCoordiatorService = new(corev1.Service)
	err = inspector.inspectCoordinatorService(inspectedCoordiatorService)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Presto service")
			return err
		}
		log.Info("Inspected Presto service", "state", "nil")
		inspectedCoordiatorService = nil
	} else {
		log.Info("Inspected Presto service", "state", *inspectedCoordiatorService)
		inspected.coordinatorService = inspectedCoordiatorService
	}
	// Worker deployment.
	var inspectedWorkerDeployment = new(appsv1.Deployment)
	err = inspector.inspectWorkerDeployment(inspectedWorkerDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Worker deployment")
			return err
		}
		log.Info("Inspected Worker deployment", "state", "nil")
		inspectedWorkerDeployment = nil
	} else {
		log.Info("Inspected Worker deployment", "state", *inspectedWorkerDeployment)
		inspected.workerDeployment = inspectedWorkerDeployment
	}
	return nil
}

func (inspector *ClusterStateInspector) inspectCluster(
	cluster *prestooperatorv1alpha1.PrestoCluster) error {
	return inspector.k8sClient.Get(
		inspector.context, inspector.request.NamespacedName, cluster)
}

func (inspector *ClusterStateInspector) inspectConfigMap(
	inspectedConfigMap *corev1.ConfigMap, configmapName string) error {
	var clusterNamespace = inspector.request.Namespace
	var clusterName = inspector.request.Name
	return inspector.k8sClient.Get(
		inspector.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      clusterName + configmapName,
		},
		inspectedConfigMap)
}

func (inspector *ClusterStateInspector) inspectCoordiatorDeployment(
	inspectedDeployment *appsv1.Deployment) error {
	var clusterNamespace = inspector.request.Namespace
	var clusterName = inspector.request.Name
	var coordinatorDeploymentName = getCoordinatorDeploymentName(clusterName)
	return inspector.inspectDeployment(
		clusterNamespace, coordinatorDeploymentName, "coordinator", inspectedDeployment)
}

func (inspector *ClusterStateInspector) inspectWorkerDeployment(
	inspectedDeployment *appsv1.Deployment) error {
	var clusterNamespace = inspector.request.Namespace
	var clusterName = inspector.request.Name
	var tmDeploymentName = getWorkerDeploymentName(clusterName)
	return inspector.inspectDeployment(
		clusterNamespace, tmDeploymentName, "Worker", inspectedDeployment)
}

func (inspector *ClusterStateInspector) inspectDeployment(
	namespace string,
	name string,
	component string,
	inspectedDeployment *appsv1.Deployment) error {
	var log = inspector.log.WithValues("component", component)
	var err = inspector.k8sClient.Get(
		inspector.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		inspectedDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get deployment")
		} else {
			log.Info("Deployment not found")
		}
	}
	return err
}

func (inspector *ClusterStateInspector) inspectCoordinatorService(
	inspectedService *corev1.Service) error {
	var clusterNamespace = inspector.request.Namespace
	var clusterName = inspector.request.Name

	return inspector.k8sClient.Get(
		inspector.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getCoordinatorServiceName(clusterName),
		},
		inspectedService)
}
