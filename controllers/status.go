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

// Status which updates the status of a cluster based on the status of its
// components.

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	prestooperatorv1alpha1 "github.com/kinderyj/presto-on-k8s-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStatusUpdater updates the status of the PrestoCluster CR.
type ClusterStatusUpdater struct {
	k8sClient client.Client
	context   context.Context
	log       logr.Logger
	recorder  record.EventRecorder
	inspected InspectedClusterState
}

func (updater *ClusterStatusUpdater) updateClusterStatus(
	status prestooperatorv1alpha1.PrestoClusterStatus) error {
	var cluster = prestooperatorv1alpha1.PrestoCluster{}
	updater.inspected.cluster.DeepCopyInto(&cluster)
	cluster.Status = status
	return updater.k8sClient.Status().Update(updater.context, &cluster)
}

func (updater *ClusterStatusUpdater) currentDeployment(
	namespace string,
	name string,
	component string,
	currentWorkerDeployment *appsv1.Deployment) error {
	var log = updater.log.WithValues("component", component)
	var err = updater.k8sClient.Get(
		updater.context,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		currentWorkerDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get deployment")
		} else {
			log.Info("Deployment not found")
		}
	}
	return err
}

func (updater *ClusterStatusUpdater) deriveClusterStatus(
	name, namespace string) prestooperatorv1alpha1.PrestoClusterStatus {
	var status = prestooperatorv1alpha1.PrestoClusterStatus{}
	var currentDeployment = new(appsv1.Deployment)
	updater.currentDeployment(
		namespace,
		getWorkerDeploymentName(name),
		"Worker",
		currentDeployment)
	availableReplicas := currentDeployment.Status.AvailableReplicas
	status.AvailableWorkers = availableReplicas
	return status
}

func (updater *ClusterStatusUpdater) updatePrestoClusterStatus(name, namespace string) (ctrl.Result, error) {

	var newStatus = updater.deriveClusterStatus(name, namespace)
	var oldStatus = updater.inspected.cluster.Status
	if newStatus.AvailableWorkers != oldStatus.AvailableWorkers {
		err := updater.updateClusterStatus(newStatus)
		if err != nil {
			return ctrl.Result{}, err
		}
		updater.recorder.Event(
			updater.inspected.cluster,
			"Normal",
			"workers counts update",
			fmt.Sprintf(
				"%v workers counts changed: %v -> %v", name, oldStatus.AvailableWorkers, newStatus.AvailableWorkers))

	}
	if newStatus.AvailableWorkers != *updater.inspected.cluster.Spec.Workers {
		return ctrl.Result{RequeueAfter: 5 * time.Second, Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}
