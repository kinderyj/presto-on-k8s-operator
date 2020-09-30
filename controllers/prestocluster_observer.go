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
	prestooperatorv1alpha1 "github.com/kinderyj/presto-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStateObserver gets the observed state of the cluster.
type ClusterStateObserver struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
}

// ObservedClusterState holds observed state of a cluster.
type ObservedClusterState struct {
	cluster               *prestooperatorv1alpha1.PrestoCluster
	prestoConfigMap       *corev1.ConfigMap
	catalogConfigMap      *corev1.ConfigMap
	coordinatorDeployment *appsv1.Deployment
	coordinatorService    *corev1.Service
	workerDeployment      *appsv1.Deployment
}

// Observes the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (observer *ClusterStateObserver) observe(
	observed *ObservedClusterState) error {
	var err error
	var log = observer.log

	// Cluster state.
	// The function observer.observeCluster(observedCluster) get the prestoclusters from the API Server(ETCD),
	// which holds the desired prostoclusters object.
	var observedCluster = new(prestooperatorv1alpha1.PrestoCluster)
	err = observer.observeCluster(observedCluster)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get the cluster resource")
			return err
		}
		log.Info("Observed cluster", "cluster", "nil")
		observedCluster = nil
	} else {
		log.Info("Observed cluster", "cluster", *observedCluster)
		observed.cluster = observedCluster
	}

	// ConfigMap for Catalog config.
	var observedCatalogConfigMap = new(corev1.ConfigMap)
	err = observer.observeConfigMap(observedCatalogConfigMap, "-catalog-configmap")
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get configMap %s", "CatalogConfigConfigmap")
			return err
		}
		log.Info("Observed configMap", "state", "nil")
		observedCatalogConfigMap = nil
	} else {
		log.Info("Observed configMap", "state", *observedCatalogConfigMap)
		observed.catalogConfigMap = observedCatalogConfigMap
	}
	// ConfigMap for Presto config.
	var observedPrestoConfigMap = new(corev1.ConfigMap)
	err = observer.observeConfigMap(observedPrestoConfigMap, "-presto-configmap")
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get configMap %s", "PrestoConfigMap")
			return err
		}
		log.Info("Observed configMap", "state", "nil")
		observedPrestoConfigMap = nil
	} else {
		log.Info("Observed configMap", "state", *observedPrestoConfigMap)
		observed.prestoConfigMap = observedPrestoConfigMap
	}

	// Coordiator deployment.
	var observedCoordiatorDeployment = new(appsv1.Deployment)
	err = observer.observeCoordiatorDeployment(observedCoordiatorDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Coordiator deployment")
			return err
		}
		log.Info("Observed Coordiator deployment", "state", "nil")
		observedCoordiatorDeployment = nil
	} else {
		log.Info("Observed Coordiator deployment", "state", *observedCoordiatorDeployment)
		observed.coordinatorDeployment = observedCoordiatorDeployment
	}

	// Coordiator service.
	var observedCoordiatorService = new(corev1.Service)
	err = observer.observeCoordinatorService(observedCoordiatorService)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Presto service")
			return err
		}
		log.Info("Observed Presto service", "state", "nil")
		observedCoordiatorService = nil
	} else {
		log.Info("Observed Presto service", "state", *observedCoordiatorService)
		observed.coordinatorService = observedCoordiatorService
	}
	// Worker deployment.
	var observedWorkerDeployment = new(appsv1.Deployment)
	err = observer.observeWorkerDeployment(observedWorkerDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Worker deployment")
			return err
		}
		log.Info("Observed Worker deployment", "state", "nil")
		observedWorkerDeployment = nil
	} else {
		log.Info("Observed Worker deployment", "state", *observedWorkerDeployment)
		observed.workerDeployment = observedWorkerDeployment
	}
	return nil
}

func (observer *ClusterStateObserver) observeCluster(
	cluster *prestooperatorv1alpha1.PrestoCluster) error {
	return observer.k8sClient.Get(
		observer.context, observer.request.NamespacedName, cluster)
}

func (observer *ClusterStateObserver) observeConfigMap(
	observedConfigMap *corev1.ConfigMap, configmapName string) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name
	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      clusterName + configmapName,
		},
		observedConfigMap)
}

func (observer *ClusterStateObserver) observeCoordiatorDeployment(
	observedDeployment *appsv1.Deployment) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name
	var coordinatorDeploymentName = getCoordinatorDeploymentName(clusterName)
	return observer.observeDeployment(
		clusterNamespace, coordinatorDeploymentName, "coordinator", observedDeployment)
}

func (observer *ClusterStateObserver) observeWorkerDeployment(
	observedDeployment *appsv1.Deployment) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name
	var tmDeploymentName = getWorkerDeploymentName(clusterName)
	return observer.observeDeployment(
		clusterNamespace, tmDeploymentName, "Worker", observedDeployment)
}

func (observer *ClusterStateObserver) observeDeployment(
	namespace string,
	name string,
	component string,
	observedDeployment *appsv1.Deployment) error {
	var log = observer.log.WithValues("component", component)
	var err = observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		observedDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get deployment")
		} else {
			log.Info("Deployment not found")
		}
	}
	return err
}

func (observer *ClusterStateObserver) observeCoordinatorService(
	observedService *corev1.Service) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name

	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getCoordinatorServiceName(clusterName),
		},
		observedService)
}
