/*
Copyright 2019 Google LLC.

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
	"math"
	"regexp"

	prestooperatorv1alpha1 "github.com/kinderyj/presto-operator/api/v1alpha1"
	. "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	//v1beta1 "github.com/googlecloudplatform/flink-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Converter which converts the FlinkCluster spec to the desired
// underlying Kubernetes resource specs.

const (
	delayDeleteClusterMinutes int32 = 5
	flinkConfigMapPath              = "/opt/flink/conf"
	flinkConfigMapVolume            = "flink-config-volume"
	gcpServiceAccountVolume         = "gcp-service-account-volume"
	hadoopConfigVolume              = "hadoop-config-volume"
)

var flinkSysProps = map[string]struct{}{
	"jobmanager.rpc.address": {},
	"jobmanager.rpc.port":    {},
	"blob.server.port":       {},
	"query.server.port":      {},
	"rest.port":              {},
}

// DesiredClusterState holds desired state of a cluster.
type DesiredClusterState struct {
	CoordinatorDeployment *appsv1.Deployment
	WorkerDeployment      *appsv1.Deployment
	CoordinatorService    *corev1.Service
	PrestoConfigMap       *corev1.ConfigMap
	CatalogConfigMap      *corev1.ConfigMap
	Job                   *batchv1.Job
	//JmIngress        *extensionsv1beta1.Ingress
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
		CoordinatorService:    getDesiredJobManagerService(cluster),
		WorkerDeployment:      getDesiredWorkerDeployment(cluster),
		//JmIngress:        getDesiredJobManagerIngress(cluster),
	}
}

// Gets the desired JobManager deployment spec from the FlinkCluster spec.
func getDesiredCoordinatorDeployment(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "presto-coordinator",
		"controller": prestoCluster.Name,
	}
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
			Template: PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: PodSpec{
					Containers: []Container{
						{
							Name:            "presto-coordinator",
							Resources:       prestoCluster.Spec.CoordinatorConfig.Resources,
							Image:           prestoCluster.Spec.Image,
							ImagePullPolicy: PullAlways,
							Ports:           []ContainerPort{{ContainerPort: 8080}},
							VolumeMounts: []VolumeMount{
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
							},
						},
					},
					Volumes: []Volume{
						{
							Name: "config",
							VolumeSource: VolumeSource{
								ConfigMap: &ConfigMapVolumeSource{
									LocalObjectReference: LocalObjectReference{
										Name: getPrestoConfigMapName(prestoCluster.Name),
									},
								},
							},
						},
						{
							Name: "catalog",
							VolumeSource: VolumeSource{
								ConfigMap: &ConfigMapVolumeSource{
									LocalObjectReference: LocalObjectReference{
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

// Gets the desired JobManager service spec from a cluster spec.
func getDesiredJobManagerService(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) *corev1.Service {
	labels := map[string]string{
		"app":        "presto-coordinator-service",
		"controller": prestoCluster.Name,
	}
	selectorLabels := map[string]string{
		"app":        "presto-coordinator",
		"controller": prestoCluster.Name,
	}
	return &Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prestoCluster.Spec.Name,
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
		Spec: ServiceSpec{
			Selector: selectorLabels,
			Ports: []ServicePort{
				{
					Protocol:   ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

// Gets the desired JobManager ingress spec from a cluster spec.
// func getDesiredJobManagerIngress(
// 	flinkCluster *v1beta1.FlinkCluster) *extensionsv1beta1.Ingress {
// 	var jobManagerIngressSpec = flinkCluster.Spec.JobManager.Ingress
// 	if jobManagerIngressSpec == nil {
// 		return nil
// 	}

// 	if shouldCleanup(flinkCluster, "JobManagerIngress") {
// 		return nil
// 	}

// 	var clusterNamespace = flinkCluster.ObjectMeta.Namespace
// 	var clusterName = flinkCluster.ObjectMeta.Name
// 	var jobManagerServiceName = getJobManagerServiceName(clusterName)
// 	var jobManagerServiceUIPort = intstr.FromString("ui")
// 	var ingressName = getJobManagerIngressName(clusterName)
// 	var ingressAnnotations = jobManagerIngressSpec.Annotations
// 	var ingressHost string
// 	var ingressTLS []extensionsv1beta1.IngressTLS
// 	var labels = map[string]string{
// 		"cluster":   clusterName,
// 		"app":       "flink",
// 		"component": "jobmanager",
// 	}
// 	if jobManagerIngressSpec.HostFormat != nil {
// 		ingressHost = getJobManagerIngressHost(*jobManagerIngressSpec.HostFormat, clusterName)
// 	}
// 	if jobManagerIngressSpec.UseTLS != nil && *jobManagerIngressSpec.UseTLS == true {
// 		var secretName string
// 		var hosts []string
// 		if ingressHost != "" {
// 			hosts = []string{ingressHost}
// 		}
// 		if jobManagerIngressSpec.TLSSecretName != nil {
// 			secretName = *jobManagerIngressSpec.TLSSecretName
// 		}
// 		if hosts != nil || secretName != "" {
// 			ingressTLS = []extensionsv1beta1.IngressTLS{{
// 				Hosts:      hosts,
// 				SecretName: secretName,
// 			}}
// 		}
// 	}
// 	var jobManagerIngress = &extensionsv1beta1.Ingress{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Namespace: clusterNamespace,
// 			Name:      ingressName,
// 			OwnerReferences: []metav1.OwnerReference{
// 				toOwnerReference(flinkCluster)},
// 			Labels:      labels,
// 			Annotations: ingressAnnotations,
// 		},
// 		Spec: extensionsv1beta1.IngressSpec{
// 			TLS: ingressTLS,
// 			Rules: []extensionsv1beta1.IngressRule{{
// 				Host: ingressHost,
// 				IngressRuleValue: extensionsv1beta1.IngressRuleValue{
// 					HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
// 						Paths: []extensionsv1beta1.HTTPIngressPath{{
// 							Path: "/",
// 							Backend: extensionsv1beta1.IngressBackend{
// 								ServiceName: jobManagerServiceName,
// 								ServicePort: jobManagerServiceUIPort,
// 							},
// 						}},
// 					},
// 				},
// 			}},
// 		},
// 	}

// 	return jobManagerIngress
// }

// Gets the desired TaskManager deployment spec from a cluster spec.
func getDesiredWorkerDeployment(
	prestoCluster *prestooperatorv1alpha1.PrestoCluster) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "presto-worker",
		"controller": prestoCluster.Name,
	}
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
			Template: PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: PodSpec{
					Containers: []Container{
						{
							Name:            "presto-worker",
							Resources:       prestoCluster.Spec.WorkerConfig.Resources,
							Image:           prestoCluster.Spec.Image,
							ImagePullPolicy: PullAlways,
							Ports:           []ContainerPort{{ContainerPort: 8080}},
							VolumeMounts: []VolumeMount{
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
							},
							Lifecycle: &Lifecycle{
								PostStart: nil,
								PreStop: &Handler{
									Exec: &ExecAction{
										Command: []string{"/bin/sh", "-c", "curl https://gist.githubusercontent.com/oneonestar/ea75a608d58aa7e40cc952ad20e5a31a/raw/1a0a8591537b6005d4bc0b5ec2ff42db6b709664/presto_shutdown.sh | sh"},
										// TODO: Migrate to the following command after https://github.com/prestosql/presto/pull/1224 being merged
										// Command: []string{"/bin/sh", "/usr/lib/presto/bin/stop-presto"},
									},
								},
							},
						},
					},
					Volumes: []Volume{
						{
							Name: "config",
							VolumeSource: VolumeSource{
								ConfigMap: &ConfigMapVolumeSource{
									LocalObjectReference: LocalObjectReference{
										Name: getPrestoConfigMapName(prestoCluster.Name),
									},
								},
							},
						},
						{
							Name: "catalog",
							VolumeSource: VolumeSource{
								ConfigMap: &ConfigMapVolumeSource{
									LocalObjectReference: LocalObjectReference{
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
	var configMap = &v1.ConfigMap{
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
		"coordinator":                        "true",
		"node-scheduler.include-coordinator": "true",
		"discovery-server.enabled":           "true",
		"http-server.http.port":              prestocluster.Spec.CoordinatorConfig.HTTPServerPort,
		"query.max-memory":                   prestocluster.Spec.CoordinatorConfig.MaxMemory,
		"query.max-memory-per-node":          prestocluster.Spec.CoordinatorConfig.MaxMemoryPerNode,
		"query.max-total-memory-per-node":    prestocluster.Spec.CoordinatorConfig.TotalMemoryPerNode,
		"discovery.uri":                      prestocluster.Spec.CoordinatorConfig.DiscoveryURI,
		"scale-writers":                      prestocluster.Spec.CoordinatorConfig.ScaleWriters,
		"writer-min-size":                    prestocluster.Spec.CoordinatorConfig.WriterMinSize,
	}
	var propertiesWorker = map[string]string{
		"coordinator":                        "false",
		"node-scheduler.include-coordinator": "true",
		"http-server.http.port":              prestocluster.Spec.WorkerConfig.HTTPServerPort,
		"query.max-memory":                   prestocluster.Spec.WorkerConfig.MaxMemory,
		"query.max-memory-per-node":          prestocluster.Spec.WorkerConfig.MaxMemoryPerNode,
		"query.max-total-memory-per-node":    prestocluster.Spec.WorkerConfig.TotalMemoryPerNode,
		//"discovery.uri":                      presto.Spec.WorkerConfig.DiscoveryURI,
		"discovery.uri": "http://" + prestocluster.Name + ":" + prestocluster.Spec.CoordinatorConfig.HTTPServerPort,
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
	//fmt.Printf("jvmCoordinator is %s\n", jvmCoordinator)
	//fmt.Printf("jvmWorker is %s\n", jvmWorker)
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
	}
	var clusterName = prestocluster.Name
	return newConfigmap(prestocluster, getPrestoConfigMapName(clusterName), data)
}

// Gets the desired Presto configMap.
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
	var clusterName = prestocluster.Name
	return newConfigmap(prestocluster, getCatalogConfigMapName(clusterName), data)

}

// Converts the FlinkCluster as owner reference for its child resources.
func toOwnerReference(
	flinkCluster *prestooperatorv1alpha1.PrestoCluster) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         flinkCluster.APIVersion,
		Kind:               flinkCluster.Kind,
		Name:               flinkCluster.Name,
		UID:                flinkCluster.UID,
		Controller:         &[]bool{true}[0],
		BlockOwnerDeletion: &[]bool{false}[0],
	}
}

var jobManagerIngressHostRegex = regexp.MustCompile("{{\\s*[$]clusterName\\s*}}")

func getJobManagerIngressHost(ingressHostFormat string, clusterName string) string {
	// TODO: Validating webhook should verify hostFormat
	return jobManagerIngressHostRegex.ReplaceAllString(ingressHostFormat, clusterName)
}

// Converts memory value to the format of divisor and returns ceiling of the value.
func convertResourceMemoryToInt64(memory resource.Quantity, divisor resource.Quantity) int64 {
	return int64(math.Ceil(float64(memory.Value()) / float64(divisor.Value())))
}

// Calculate heap size in MB
func calHeapSize(memSize int64, offHeapMin int64, offHeapRatio int64) int64 {
	var heapSizeMB int64
	offHeapSize := int64(math.Ceil(float64(memSize*offHeapRatio) / 100))
	if offHeapSize < offHeapMin {
		offHeapSize = offHeapMin
	}
	heapSizeCalculated := memSize - offHeapSize
	if heapSizeCalculated > 0 {
		divisor := resource.MustParse("1M")
		heapSizeQuantity := resource.NewQuantity(heapSizeCalculated, resource.DecimalSI)
		heapSizeMB = convertResourceMemoryToInt64(*heapSizeQuantity, divisor)
	}
	return heapSizeMB
}
