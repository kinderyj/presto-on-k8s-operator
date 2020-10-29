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
	"fmt"

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
	var volumeMounts = []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/config.properties",
			SubPath:   "config.properties.coordinator",
		},
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/jvm.config",
			SubPath:   "jvm.config.coordinator",
		},
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/node.properties",
			SubPath:   "node.properties.coordinator",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/hive.properties",
			SubPath:   "hive.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/jmx.properties",
			SubPath:   "jmx.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/memory.properties",
			SubPath:   "memory.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/tpcds.properties",
			SubPath:   "tpcds.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/tpch.properties",
			SubPath:   "tpch.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/tmp/core-site.xml",
			SubPath:   "core-site.xml",
		},
	}
	dynamicConfigs := prestoCluster.Spec.CatalogConfig.DynamicConfigs
	var configNameChecker = map[string]string{}
	if len(dynamicConfigs) > 0 {
		for _, value := range dynamicConfigs {
			configName, _, err := splitDynamicConfigs(value)
			if err != nil {
				fmt.Printf("Failed to get configname from splitDynamicConfigs for value: %s, err: %v\n", value, err)
				continue
			}
			if _, exist := configNameChecker[configName]; exist {
				continue
			}
			configNameChecker[configName] = ""
			var danamicConfig = corev1.VolumeMount{
				Name:      "catalog",
				MountPath: "/usr/lib/presto/etc/catalog/" + configName,
				SubPath:   configName,
			}
			volumeMounts = append(volumeMounts, danamicConfig)
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
			MountPath: "/usr/lib/presto/etc/config.properties",
			SubPath:   "config.properties.worker",
		},
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/jvm.config",
			SubPath:   "jvm.config.worker",
		},
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/node.properties",
			SubPath:   "node.properties.worker",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/hive.properties",
			SubPath:   "hive.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/jmx.properties",
			SubPath:   "jmx.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/memory.properties",
			SubPath:   "memory.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/tpcds.properties",
			SubPath:   "tpcds.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/usr/lib/presto/etc/catalog/tpch.properties",
			SubPath:   "tpch.properties",
		},
		{
			Name:      "catalog",
			MountPath: "/tmp/core-site.xml",
			SubPath:   "core-site.xml",
		},
		{
			Name:      "config",
			MountPath: "/usr/lib/presto/etc/pre-stop.sh",
			SubPath:   "pre-stop.sh",
		},
	}
	dynamicConfigs := prestoCluster.Spec.CatalogConfig.DynamicConfigs
	var configNameChecker = map[string]string{}
	if len(dynamicConfigs) > 0 {
		for _, value := range dynamicConfigs {
			configName, _, err := splitDynamicConfigs(value)
			if err != nil {
				fmt.Printf("Failed to get configname from splitDynamicConfigs for value: %s, err: %v\n", value, err)
				continue
			}
			if _, exist := configNameChecker[configName]; exist {
				continue
			}
			configNameChecker[configName] = ""
			var danamicConfig = corev1.VolumeMount{
				Name:      "catalog",
				MountPath: "/usr/lib/presto/etc/catalog/" + configName,
				SubPath:   configName,
			}
			volumeMounts = append(volumeMounts, danamicConfig)
		}

	}
	var runAsUser int64 = 1000
	var coordinatorDeployment = &appsv1.Deployment{
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
	return coordinatorDeployment
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
	prestocluster *prestooperatorv1alpha1.PrestoCluster) *corev1.ConfigMap {
	var propertiesCoordinator = map[string]string{
		"coordinator": "true",
		//"node-scheduler.include-coordinator": prestocluster.Spec.CoordinatorConfig.NodeScheduler,
		"discovery-server.enabled":        "true",
		"http-server.http.port":           prestocluster.Spec.CoordinatorConfig.HTTPServerPort,
		"query.max-memory":                prestocluster.Spec.CoordinatorConfig.MaxMemory,
		"query.max-memory-per-node":       prestocluster.Spec.CoordinatorConfig.MaxMemoryPerNode,
		"query.max-total-memory-per-node": prestocluster.Spec.CoordinatorConfig.TotalMemoryPerNode,
		"discovery.uri":                   prestocluster.Spec.CoordinatorConfig.DiscoveryURI,
		"scale-writers":                   prestocluster.Spec.CoordinatorConfig.ScaleWriters,
		"writer-min-size":                 prestocluster.Spec.CoordinatorConfig.WriterMinSize,
		"spill-enabled":                   prestocluster.Spec.CoordinatorConfig.SpillEnabled,
		"spiller-spill-path":              prestocluster.Spec.CoordinatorConfig.SpillerSpillPath,
	}
	if nodeScheduler := prestocluster.Spec.CoordinatorConfig.NodeScheduler; nodeScheduler != "" {
		propertiesCoordinator["node-scheduler.include-coordinator"] = nodeScheduler
	}
	var dynamicArgsCoordinator = prestocluster.Spec.CoordinatorConfig.DynamicArgs
	for _, argsKeyValue := range dynamicArgsCoordinator {
		key, value, err := splitDynamicArgs(argsKeyValue)
		if err != nil {
			fmt.Printf("split DanamicArgs error for argsKeyValue %s, err: %v", argsKeyValue, err)
			continue
		}
		propertiesCoordinator[key] = value
	}
	var propertiesWorker = map[string]string{
		"coordinator": "false",
		//"node-scheduler.include-coordinator": prestocluster.Spec.WorkerConfig.NodeScheduler,
		"http-server.http.port":           prestocluster.Spec.WorkerConfig.HTTPServerPort,
		"query.max-memory":                prestocluster.Spec.WorkerConfig.MaxMemory,
		"query.max-memory-per-node":       prestocluster.Spec.WorkerConfig.MaxMemoryPerNode,
		"query.max-total-memory-per-node": prestocluster.Spec.WorkerConfig.TotalMemoryPerNode,
		"discovery.uri":                   "http://" + prestocluster.Name + ":" + prestocluster.Spec.CoordinatorConfig.HTTPServerPort,
	}
	if nodeScheduler := prestocluster.Spec.WorkerConfig.NodeScheduler; nodeScheduler != "" {
		propertiesWorker["node-scheduler.include-coordinator"] = nodeScheduler
	}
	var dynamicArgsWorker = prestocluster.Spec.WorkerConfig.DynamicArgs
	for _, argsKeyValue := range dynamicArgsWorker {
		key, value, err := splitDynamicArgs(argsKeyValue)
		if err != nil {
			fmt.Printf("split DanamicArgs error for argsKeyValue %s, err: %v", argsKeyValue, err)
			continue
		}
		propertiesWorker[key] = value
	}
	var jvmArgs = `-server -Xmx%s -XX:+UseG1GC -XX:G1HeapRegionSize=%s -XX:+UseGCOverheadLimit -XX:+ExplicitGCInvokesConcurrent -XX:+HeapDumpOnOutOfMemoryError -XX:+ExitOnOutOfMemoryError -Djdk.attach.allowAttachSelf=%s`
	var jvmCoordinator = fmt.Sprintf(jvmArgs,
		prestocluster.Spec.CoordinatorConfig.Xmx,
		prestocluster.Spec.CoordinatorConfig.G1HeapRegionSize,
		prestocluster.Spec.CoordinatorConfig.AllowAttachSelf)
	var jvmWorker = fmt.Sprintf(jvmArgs,
		prestocluster.Spec.WorkerConfig.Xmx,
		prestocluster.Spec.WorkerConfig.G1HeapRegionSize,
		prestocluster.Spec.WorkerConfig.AllowAttachSelf)
	var nodeCoordinator = map[string]string{
		"node.environment": prestocluster.Spec.CoordinatorConfig.Environment,
		"node.data-dir":    prestocluster.Spec.CoordinatorConfig.DataDir,
	}
	var nodeWorker = map[string]string{
		"node.environment": prestocluster.Spec.WorkerConfig.Environment,
		"node.data-dir":    prestocluster.Spec.WorkerConfig.DataDir,
	}
	var data = map[string]string{
		"config.properties.coordinator": getProperties(propertiesCoordinator),
		"config.properties.worker":      getProperties(propertiesWorker),
		"jvm.config.coordinator":        getPropertiesFields(jvmCoordinator),
		"jvm.config.worker":             getPropertiesFields(jvmWorker),
		"node.properties.coordinator":   getProperties(nodeCoordinator),
		"node.properties.worker":        getProperties(nodeWorker),
		"pre-stop.sh":                   preStopScript,
	}
	var clusterName = prestocluster.Name
	return newConfigmap(prestocluster, getPrestoConfigMapName(clusterName), data)
}

// Gets the desired Catalog configMap.
func getDesiredCatalogConfigMap(
	prestocluster *prestooperatorv1alpha1.PrestoCluster) *corev1.ConfigMap {
	var propertiesJmx = map[string]string{
		"connector.name": "jmx",
	}
	var propertiesMemory = map[string]string{
		"connector.name": "memory",
	}
	var propertiesTpcds = map[string]string{
		"connector.name": "tpcds",
	}
	var propertiesTpch = map[string]string{
		"connector.name":       "tpch",
		"tpch.splits-per-node": "4",
	}
	var propertiesHive = map[string]string{
		"connector.name":                        "hive-hadoop2",
		"hive.metastore.uri":                    "thrift://" + prestocluster.Spec.CatalogConfig.HiveMetastoreIP + ":" + prestocluster.Spec.CatalogConfig.HiveMetastorePort,
		"hive.config.resources":                 "/tmp/core-site.xml",
		"hive.non-managed-table-writes-enabled": "true",
		"hive.allow-add-column":                 "true",
		"hive.allow-drop-column":                "true",
		"hive.allow-drop-table":                 "true",
		"hive.allow-rename-table":               "true",
		"hive.allow-rename-column":              "true",
	}
	var coresite = fmt.Sprintf(coreSite,
		prestocluster.Spec.CatalogConfig.FsDefaultFS,
		prestocluster.Spec.CatalogConfig.CosnSecretID,
		prestocluster.Spec.CatalogConfig.CosnSecretKey,
		prestocluster.Spec.CatalogConfig.CosnRegion,
	)

	var data = map[string]string{
		"jmx.properties":    getProperties(propertiesJmx),
		"memory.properties": getProperties(propertiesMemory),
		"tpcds.properties":  getProperties(propertiesTpcds),
		"tpch.properties":   getProperties(propertiesTpch),
		"hive.properties":   getProperties(propertiesHive),
		"core-site.xml":     getPropertiesStrings(coresite),
	}
	// dynimic config name
	dynamicConfigs := prestocluster.Spec.CatalogConfig.DynamicConfigs
	for _, dynamicConfig := range dynamicConfigs {
		configName, configValue, err := splitDynamicConfigs(dynamicConfig)
		if err != nil {
			fmt.Printf("Failed to convert dynamicConfig for %s, err: %v", dynamicConfig, err)
			continue
		}
		danamicConfigsArgs := splitDynamicConfigsArgs(configValue)
		if len(danamicConfigsArgs) == 0 {
			continue
		}
		var dynamicArgs = map[string]string{}
		for _, danamicConfigsArg := range danamicConfigsArgs {
			argsKey, argsValue, err := splitDynamicArgs(danamicConfigsArg)
			if err != nil {
				fmt.Printf("Failed to convert args for dynamicConfig %s, err: %v", dynamicConfig, err)
				continue
			}
			dynamicArgs[argsKey] = argsValue
		}
		if value, exist := data[configName]; exist {
			data[configName] = getProperties(dynamicArgs) + value
		} else {
			data[configName] = getProperties(dynamicArgs)
		}
	}

	var clusterName = prestocluster.Name
	return newConfigmap(prestocluster, getCatalogConfigMapName(clusterName), data)

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
