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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	prestooperatorv1alpha1 "github.com/kinderyj/presto-operator/api/v1alpha1"
)

// PrestoClusterReconciler reconciles a PrestoCluster object
type PrestoClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=prestooperator.k8s.io,resources=prestoclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=prestooperator.k8s.io,resources=prestoclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses/status,verbs=get

// Reconcile implements the Reconciler interface in the controller-runime.
func (r *PrestoClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("prestocluster", req.NamespacedName)
	var log = r.Log.WithValues(
		"cluster", req.NamespacedName)
	var handler = PrestoClusterHandler{
		k8sClient: r.Client,
		request:   req,
		context:   context.Background(),
		log:       log,
		recorder:  r.Recorder,
		observed:  ObservedClusterState{},
	}
	return handler.reconcile(req)
}

// PrestoClusterHandler holds the context and state for a reconcile request.
type PrestoClusterHandler struct {
	k8sClient client.Client
	request   ctrl.Request
	context   context.Context
	log       logr.Logger
	recorder  record.EventRecorder
	observed  ObservedClusterState
	desired   DesiredClusterState
}

func (handler *PrestoClusterHandler) reconcile(
	request ctrl.Request) (ctrl.Result, error) {
	var k8sClient = handler.k8sClient
	var log = handler.log
	var context = handler.context
	var observed = &handler.observed
	var desired = &handler.desired
	var err error
	log.Info("==========================================\n")
	log.Info("Reconcile 1: Inspect the current state.\n")
	var observer = ClusterStateObserver{
		k8sClient: k8sClient,
		request:   request,
		context:   context,
		log:       log,
	}
	err = observer.observe(observed)
	if err != nil {
		log.Error(err, "Failed to inspect the current state")
		return ctrl.Result{}, err
	}
	log.Info("==========================================\n")
	log.Info("Reconcile 2: Get the desired prestocluster object.\n")
	*desired = getDesiredClusterState(observed.cluster)
	if desired.PrestoConfigMap != nil {
		log.Info("Desired state", "PrestoConfigMap", *desired.PrestoConfigMap)
	} else {
		log.Info("Desired state", "PrestoConfigMap", "nil")
	}
	if desired.CatalogConfigMap != nil {
		log.Info("Desired state", "CatalogConfigMap", *desired.CatalogConfigMap)
	} else {
		log.Info("Desired state", "CatalogConfigMap", "nil")
	}
	if desired.CoordinatorDeployment != nil {
		log.Info("Desired state", "Coordinator deployment", *desired.CoordinatorDeployment)
	} else {
		log.Info("Desired state", "Coordinator deployment", "nil")
	}
	if desired.CoordinatorService != nil {
		log.Info("Desired state", "Coordinator service", *desired.CoordinatorService)
	} else {
		log.Info("Desired state", "Coordinator service", "nil")
	}
	if desired.WorkerDeployment != nil {
		log.Info("Desired state", "Worker deployment", *desired.WorkerDeployment)
	} else {
		log.Info("Desired state", "Worker deployment", "nil")
	}
	log.Info("==========================================\n")
	log.Info("Reconcile 3: start to reconcile.\n")
	var reconciler = ClusterReconciler{
		k8sClient: handler.k8sClient,
		context:   handler.context,
		log:       handler.log,
		observed:  handler.observed,
		desired:   handler.desired,
		recorder:  handler.recorder,
	}
	result, err := reconciler.reconcile()
	if err != nil {
		log.Error(err, "Failed to reconcile")
	}
	if result.RequeueAfter > 0 {
		log.Info("Requeue reconcile request", "after", result.RequeueAfter)
	}
	// Finally, we update the status block of the PrestoCluster resource to reflect the
	// current state of the world
	log.Info("==========================================\n")
	log.Info("Reconcile 4: start to update Presto cluster Status.\n")
	var statusUpdater = ClusterStatusUpdater{
		k8sClient: handler.k8sClient,
		context:   handler.context,
		log:       handler.log,
		recorder:  handler.recorder,
		observed:  handler.observed,
	}
	if statusUpdater.observed.cluster == nil {
		statusUpdater.log.Info("The cluster has been deleted, no status to update")
		return ctrl.Result{}, nil
	}
	result, err = statusUpdater.updatePrestoClusterStatus(request.Name, request.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	//return ctrl.Result{}, err
	return result, err
}

// SetupWithManager defines the type of Object being *reconciled*, and configures the
// ControllerManagedBy to respond to create / delete /update events by *reconciling the object*.
func (r *PrestoClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&prestooperatorv1alpha1.PrestoCluster{}).
		Complete(r)
}
