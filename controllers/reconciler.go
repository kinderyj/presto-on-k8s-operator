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
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterReconciler takes actions to drive the inspected state towards the
// desired state.
type ClusterReconciler struct {
	k8sClient client.Client
	context   context.Context
	log       logr.Logger
	inspected InspectedClusterState
	desired   DesiredClusterState
	recorder  record.EventRecorder
}

var requeueResult = ctrl.Result{RequeueAfter: 10 * time.Second, Requeue: true}

// Compares the desired state and the inspected state, if there is a difference,
// takes actions to drive the inspected state towards the desired state.
func (reconciler *ClusterReconciler) reconcile() (ctrl.Result, error) {
	var err error

	// Child resources of the cluster CR will be automatically reclaimed by K8S.
	if reconciler.inspected.cluster == nil {
		reconciler.log.Info("The cluster has been deleted, no action to take")
		return ctrl.Result{}, nil
	}

	err = reconciler.reconcilePrestoConfigMap()
	if err != nil {
		return ctrl.Result{}, err
	}
	err = reconciler.reconcileCatalogConfigMap()
	if err != nil {
		return ctrl.Result{}, err
	}
	err = reconciler.reconcileCoordinatorDeployment()
	if err != nil {
		return ctrl.Result{}, err
	}
	err = reconciler.reconcileCoordinatorService()
	if err != nil {
		return ctrl.Result{}, err
	}
	err = reconciler.reconcileWorkerDeployment()
	if err != nil {
		return ctrl.Result{}, err
	}
	//return result, nil
	return ctrl.Result{}, err
}

func (reconciler *ClusterReconciler) reconcileCoordinatorDeployment() error {
	return reconciler.reconcileDeployment(
		"Coordinator",
		reconciler.desired.CoordinatorDeployment,
		reconciler.inspected.coordinatorDeployment)
}

func (reconciler *ClusterReconciler) reconcileWorkerDeployment() error {
	return reconciler.reconcileDeployment(
		"Worker",
		reconciler.desired.WorkerDeployment,
		reconciler.inspected.workerDeployment)
}

func (reconciler *ClusterReconciler) reconcileDeployment(
	component string,
	desiredDeployment *appsv1.Deployment,
	inspectedDeployment *appsv1.Deployment) error {
	var log = reconciler.log.WithValues("component", component)

	if desiredDeployment != nil && inspectedDeployment == nil {
		return reconciler.createDeployment(desiredDeployment, component)
	}

	if desiredDeployment != nil && inspectedDeployment != nil {
		if *desiredDeployment.Spec.Replicas != *inspectedDeployment.Spec.Replicas {
			log.Info("Replicas changed",
				"old replicas", *inspectedDeployment.Spec.Replicas,
				"new replicas", *desiredDeployment.Spec.Replicas)
			return reconciler.updateDeployment(desiredDeployment, component)
		}
		log.Info("Deployment already exists, no action")
	}

	if desiredDeployment == nil && inspectedDeployment != nil {
		return reconciler.deleteDeployment(inspectedDeployment, component)
	}

	return nil
}

func (reconciler *ClusterReconciler) createDeployment(
	deployment *appsv1.Deployment, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating deployment", "deployment", *deployment)
	var err = k8sClient.Create(context, deployment)
	if err != nil {
		log.Error(err, "Failed to create deployment")
	} else {
		log.Info("Deployment created")
	}
	return err
}

func (reconciler *ClusterReconciler) updateDeployment(
	deployment *appsv1.Deployment, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Updating deployment", "deployment", deployment)
	var err = k8sClient.Update(context, deployment)
	if err != nil {
		log.Error(err, "Failed to update deployment")
	} else {
		log.Info("Deployment updated")
	}
	return err
}

func (reconciler *ClusterReconciler) deleteDeployment(
	deployment *appsv1.Deployment, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting deployment", "deployment", deployment)
	var err = k8sClient.Delete(context, deployment)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete deployment")
	} else {
		log.Info("Deployment deleted")
	}
	return err
}

func (reconciler *ClusterReconciler) reconcileCoordinatorService() error {
	var desiredCoordinatorService = reconciler.desired.CoordinatorService
	var inspectedCoordinatorService = reconciler.inspected.coordinatorService

	if desiredCoordinatorService != nil && inspectedCoordinatorService == nil {
		return reconciler.createService(desiredCoordinatorService, "CoordinatorService")
	}

	if desiredCoordinatorService != nil && inspectedCoordinatorService != nil {
		reconciler.log.Info("CoordinatorService service already exists, no action")
		return nil
		// TODO: compare and update if needed.
	}

	if desiredCoordinatorService == nil && inspectedCoordinatorService != nil {
		return reconciler.deleteService(inspectedCoordinatorService, "CoordinatorService")
	}

	return nil
}

func (reconciler *ClusterReconciler) createService(
	service *corev1.Service, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating service", "resource", *service)
	var err = k8sClient.Create(context, service)
	if err != nil {
		log.Info("Failed to create service", "error", err)
	} else {
		log.Info("Service created")
	}
	return err
}

func (reconciler *ClusterReconciler) deleteService(
	service *corev1.Service, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting service", "service", service)
	var err = k8sClient.Delete(context, service)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete service")
	} else {
		log.Info("service deleted")
	}
	return err
}

func (reconciler *ClusterReconciler) createIngress(
	ingress *extensionsv1beta1.Ingress, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating ingress", "resource", *ingress)
	var err = k8sClient.Create(context, ingress)
	if err != nil {
		log.Info("Failed to create ingress", "error", err)
	} else {
		log.Info("Ingress created")
	}
	return err
}

func (reconciler *ClusterReconciler) deleteIngress(
	ingress *extensionsv1beta1.Ingress, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting ingress", "ingress", ingress)
	var err = k8sClient.Delete(context, ingress)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete ingress")
	} else {
		log.Info("Ingress deleted")
	}
	return err
}

func (reconciler *ClusterReconciler) reconcilePrestoConfigMap() error {
	var desiredConfigMap = reconciler.desired.PrestoConfigMap
	var inspectedConfigMap = reconciler.inspected.prestoConfigMap

	if desiredConfigMap != nil && inspectedConfigMap == nil {
		return reconciler.createConfigMap(desiredConfigMap, "ConfigMap")
	}

	if desiredConfigMap != nil && inspectedConfigMap != nil {
		reconciler.log.Info("ConfigMap already exists, no action")
		return nil
		// TODO: compare and update if needed.
	}

	if desiredConfigMap == nil && inspectedConfigMap != nil {
		return reconciler.deleteConfigMap(inspectedConfigMap, "ConfigMap")
	}
	return nil
}

func (reconciler *ClusterReconciler) reconcileCatalogConfigMap() error {
	var desiredConfigMap = reconciler.desired.CatalogConfigMap
	var inspectedConfigMap = reconciler.inspected.catalogConfigMap

	if desiredConfigMap != nil && inspectedConfigMap == nil {
		return reconciler.createConfigMap(desiredConfigMap, "ConfigMap")
	}

	if desiredConfigMap != nil && inspectedConfigMap != nil {
		reconciler.log.Info("ConfigMap already exists, no action")
		return nil
		// TODO: compare and update if needed.
	}

	if desiredConfigMap == nil && inspectedConfigMap != nil {
		return reconciler.deleteConfigMap(inspectedConfigMap, "ConfigMap")
	}

	return nil
}

func (reconciler *ClusterReconciler) createConfigMap(
	cm *corev1.ConfigMap, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating configMap", "configMap", *cm)
	var err = k8sClient.Create(context, cm)
	if err != nil {
		log.Info("Failed to create configMap", "error", err)
	} else {
		log.Info("ConfigMap created")
	}
	return err
}

func (reconciler *ClusterReconciler) deleteConfigMap(
	cm *corev1.ConfigMap, component string) error {
	var context = reconciler.context
	var log = reconciler.log.WithValues("component", component)
	var k8sClient = reconciler.k8sClient

	log.Info("Deleting configMap", "configMap", cm)
	var err = k8sClient.Delete(context, cm)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete configMap")
	} else {
		log.Info("ConfigMap deleted")
	}
	return err
}
