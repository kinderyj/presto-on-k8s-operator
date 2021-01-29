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
	"sort"

	prestooperatorv1alpha1 "github.com/kinderyj/presto-on-k8s-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Converter which converts the PrestoCluster spec to the desired
// underlying Kubernetes resource specs.

// DesiredClusterState holds desired state of a cluster.
type DesiredClusterState struct {
	CoordinatorDeployment *appsv1.Deployment
	WorkerDeployment      *appsv1.Deployment
	CoordinatorService    *corev1.Service
	PrestoConfigMap       *corev1.ConfigMap
	CatalogConfigMap      *corev1.ConfigMap
	Job                   *batchv1.Job
	//CoordinatorIngress        *extensionsv1beta1.Ingress
}

// Gets the desired state of a cluster.
func getDesiredClusterState(
	cluster *prestooperatorv1alpha1.PrestoCluster) DesiredClusterState {
	// The cluster has been deleted, all resources should be cleaned up.
	if cluster == nil {
		return DesiredClusterState{}
	}
	return DesiredClusterState{
		PrestoConfigMap:       getDesiredPrestoConfigMap(cluster),
		CatalogConfigMap:      getDesiredCatalogConfigMap(cluster),
		CoordinatorDeployment: getDesiredCoordinatorDeployment(cluster),
		CoordinatorService:    getDesiredCoordinatorService(cluster),
		WorkerDeployment:      getDesiredWorkerDeployment(cluster),
	}
}

// Gets the desired Coordinator deployment spec from the PrestoCluster spec.
func getDesiredCoordinatorDeployment(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "presto-coordinator",
		"controller": prestoCluster.Name,
	}
	var volumeMounts = []corev1.VolumeMount{}
	coordinatorEtcConfig := prestoCluster.Spec.CoordinatorConfig.EtcConfig
	if coordinatorEtcConfig != nil {
		coordinatorEtcConfigSorted := make([]string, 0)
		for key := range coordinatorEtcConfig {
			coordinatorEtcConfigSorted = append(coordinatorEtcConfigSorted, key)
		}
		sort.Strings(coordinatorEtcConfigSorted)
		for _, xProperties := range coordinatorEtcConfigSorted {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "config",
				MountPath: "/usr/lib/presto/etc/" + xProperties,
				SubPath:   "coordinator." + xProperties,
			})
		}
	}
	catalogProperties := prestoCluster.Spec.CatalogConfig
	if catalogProperties != nil {
		catalogPropertiesSorted := make([]string, 0)
		for key := range catalogProperties {
			catalogPropertiesSorted = append(catalogPropertiesSorted, key)
		}
		sort.Strings(catalogPropertiesSorted)
		for _, xProperties := range catalogPropertiesSorted {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "catalog",
				MountPath: "/usr/lib/presto/etc/catalog/" + xProperties,
				SubPath:   xProperties,
			})
		}
	}
	coresite := prestoCluster.Spec.Coresite
	if coresite != nil {
		coresiteSorted := make([]string, 0)
		for key := range coresite {
			coresiteSorted = append(coresiteSorted, key)
		}
		sort.Strings(coresiteSorted)
		for _, xProperties := range coresiteSorted {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "catalog",
				MountPath: "/tmp/" + xProperties,
				SubPath:   xProperties,
			})
		}
	}
	var runAsUser int64 = 1000
	var coordinatorDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: prestoCluster.Namespace,
			Name:      prestoCluster.Name + "-coordinator",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(prestoCluster, schema.GroupVersionKind{
					Group:   prestooperatorv1alpha1.GroupVersion.Group,
					Version: prestooperatorv1alpha1.GroupVersion.Version,
					Kind:    "PrestoCluster",
				}),
			},
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 { i := int32(1); return &i }(),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &runAsUser,
					},
					Containers: []corev1.Container{
						{
							Name:            "presto-coordinator",
							Resources:       prestoCluster.Spec.CoordinatorConfig.Resources,
							Image:           prestoCluster.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Ports:           []corev1.ContainerPort{{ContainerPort: 8080}},
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: getPrestoConfigMapName(prestoCluster.Name),
									},
								},
							},
						},
						{
							Name: "catalog",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: getCatalogConfigMapName(prestoCluster.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return coordinatorDeployment
}

// Gets the desired Coordinator service spec from a cluster spec.
func getDesiredCoordinatorService(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) *corev1.Service {
	labels := map[string]string{
		"app":        "presto-coordinator-service",
		"controller": prestoCluster.Name,
	}
	selectorLabels := map[string]string{
		"app":        "presto-coordinator",
		"controller": prestoCluster.Name,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prestoCluster.Name,
			Namespace: prestoCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(prestoCluster, schema.GroupVersionKind{
					Group:   prestooperatorv1alpha1.GroupVersion.Group,
					Version: prestooperatorv1alpha1.GroupVersion.Version,
					Kind:    "PrestoCluster",
				}),
			},
			Labels: labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

// Gets the desired Worker deployment spec from a cluster spec.
func getDesiredWorkerDeployment(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "presto-worker",
		"controller": prestoCluster.Name,
	}
	var volumeMounts = []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/pre-stop.sh",
			SubPath:   "pre-stop.sh",
		},
	}
	workerEtcConfig := prestoCluster.Spec.WorkerConfig.EtcConfig
	if workerEtcConfig != nil {
		workerEtcConfigSorted := make([]string, 0)
		for key := range workerEtcConfig {
			workerEtcConfigSorted = append(workerEtcConfigSorted, key)
		}
		sort.Strings(workerEtcConfigSorted)
		for _, xProperties := range workerEtcConfigSorted {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "config",
				MountPath: "/usr/lib/presto/etc/" + xProperties,
				SubPath:   "worker." + xProperties,
				ReadOnly:  true,
			})
		}
	}
	catalogProperties := prestoCluster.Spec.CatalogConfig
	if catalogProperties != nil {
		catalogPropertiesSorted := make([]string, 0)
		for key := range catalogProperties {
			catalogPropertiesSorted = append(catalogPropertiesSorted, key)
		}
		sort.Strings(catalogPropertiesSorted)
		for _, xProperties := range catalogPropertiesSorted {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "catalog",
				MountPath: "/usr/lib/presto/etc/catalog/" + xProperties,
				SubPath:   xProperties,
			})
		}
	}
	coresite := prestoCluster.Spec.Coresite
	if coresite != nil {
		coresiteSorted := make([]string, 0)
		for key := range coresite {
			coresiteSorted = append(coresiteSorted, key)
		}
		sort.Strings(coresiteSorted)
		for _, xProperties := range coresiteSorted {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "catalog",
				MountPath: "/tmp/" + xProperties,
				SubPath:   xProperties,
			})
		}
	}
	var runAsUser int64 = 1000
	var workerDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: prestoCluster.Namespace,
			Name:      prestoCluster.Name + "-worker",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(prestoCluster, schema.GroupVersionKind{
					Group:   prestooperatorv1alpha1.GroupVersion.Group,
					Version: prestooperatorv1alpha1.GroupVersion.Version,
					Kind:    "PrestoCluster",
				}),
			},
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: prestoCluster.Spec.Workers,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &runAsUser,
					},
					Containers: []corev1.Container{
						{
							Name:            "presto-worker",
							Resources:       prestoCluster.Spec.WorkerConfig.Resources,
							Image:           prestoCluster.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Ports:           []corev1.ContainerPort{{ContainerPort: 8080}},
							VolumeMounts:    volumeMounts,
							Lifecycle: &corev1.Lifecycle{
								PostStart: nil,
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "/usr/lib/presto/etc/pre-stop.sh"},
										// TODO: Watch the graceful shutdown feature of presto worker.
										// If the Presto worker supports graceful shutdown by itself, the PreStop Hook
										// here should be removed. https://github.com/prestosql/presto/pull/1224
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: getPrestoConfigMapName(prestoCluster.Name),
									},
								},
							},
						},
						{
							Name: "catalog",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: getCatalogConfigMapName(prestoCluster.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return workerDeployment
}

func newConfigmap(presto *prestooperatorv1alpha1.PrestoCluster, configMapName string, data map[string]string) *v1.ConfigMap {
	var namespace = presto.Namespace
	var configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
			OwnerReferences: []metav1.OwnerReference{
				toOwnerReference(presto)},
		},
		Data: data,
	}
	return configMap
}

// Gets the desired Presto configMap.
func getDesiredPrestoConfigMap(
	prestoclusters *prestooperatorv1alpha1.PrestoCluster) *corev1.ConfigMap {
	var prestocluster = prestooperatorv1alpha1.PrestoCluster{}
	prestoclusters.DeepCopyInto(&prestocluster)
	var configmapCoordinatorEtcConfig = prestocluster.Spec.CoordinatorConfig.EtcConfig
	var configmapWorkerEtcConfig = prestocluster.Spec.WorkerConfig.EtcConfig
	var data = map[string]string{}
	for key, value := range configmapCoordinatorEtcConfig {
		data["coordinator."+key] = value
	}
	for key, value := range configmapWorkerEtcConfig {
		data["worker."+key] = value
	}
	data["pre-stop.sh"] = preStopScript
	var clusterName = prestocluster.Name
	return newConfigmap(&prestocluster, getPrestoConfigMapName(clusterName), data)
}

// Gets the desired Catalog configMap.
func getDesiredCatalogConfigMap(
	prestoclusters *prestooperatorv1alpha1.PrestoCluster) *corev1.ConfigMap {
	var prestocluster = prestooperatorv1alpha1.PrestoCluster{}
	prestoclusters.DeepCopyInto(&prestocluster)
	var configmapCatalogConfig = prestocluster.Spec.CatalogConfig
	var configmapCoresite = prestocluster.Spec.Coresite
	var data = map[string]string{}
	for key, value := range configmapCatalogConfig {
		data[key] = value
	}
	for key, value := range configmapCoresite {
		data[key] = value
	}
	var clusterName = prestocluster.Name
	return newConfigmap(&prestocluster, getCatalogConfigMapName(clusterName), data)

}

// Converts the PrestoCluster as owner reference for its child resources.
func toOwnerReference(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         prestoCluster.APIVersion,
		Kind:               prestoCluster.Kind,
		Name:               prestoCluster.Name,
		UID:                prestoCluster.UID,
		Controller:         &[]bool{true}[0],
		BlockOwnerDeletion: &[]bool{false}[0],
	}
}
