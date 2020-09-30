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
	"errors"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/validation"
)

// Gets Presto etc Configmap name
func getPrestoConfigMapName(clusterName string) string {
	return clusterName + "-presto-configmap"
}

// Gets Presto catalog Configmap name
func getCatalogConfigMapName(clusterName string) string {
	return clusterName + "-catalog-configmap"
}

// Gets CoordinatorManager deployment name
func getCoordinatorDeploymentName(clusterName string) string {
	return clusterName + "-coordinator"
}

// Gets Coordinator service name
func getCoordinatorServiceName(clusterName string) string {
	return clusterName
}

// Gets Worker deployment name
func getWorkerDeploymentName(clusterName string) string {
	return clusterName + "-worker"
}

// Gets properties
func getProperties(properties map[string]string) string {
	var keys = make([]string, len(properties))
	i := 0
	for k := range properties {
		keys[i] = k
		i = i + 1
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, properties[key]))
	}
	return builder.String()
}

// Gets properties
func getPropertiesFields(properties string) string {
	var keys = strings.Fields(properties)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s\n", key))
	}
	return builder.String()
}

// Gets properties
func getPropertiesStrings(properties string) string {
	var keys = strings.Split(properties, "\n")
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s\n", key))
	}
	return builder.String()
}

func getPodsPrefix(controllerName string) string {
	// use the dash (if the name isn't too long) to make the pod name a bit prettier
	prefix := fmt.Sprintf("%s-", controllerName)
	if len(validation.NameIsDNSSubdomain(prefix, true)) != 0 {
		prefix = controllerName
	}
	return prefix
}

func splitDynamicArgs(argsKeyValues string) (argsKey, argsValue string, err error) {
	splited := strings.Split(argsKeyValues, "=")
	if len(splited) != 2 {
		var err = errors.New("danamic args key values format error")
		return "", "", err
	}
	argsKey = splited[0]
	argsValue = splited[1]
	return
}

func splitDynamicConfigs(argsKeyValues string) (configKey, configValue string, err error) {
	splited := strings.Split(argsKeyValues, ":")
	if len(splited) != 2 {
		var err = errors.New("danamic configs format error")
		return "", "", err
	}
	configKey = splited[0]
	configValue = splited[1]
	return
}

func splitDynamicConfigsArgs(argsKeyValues string) []string {
	splited := strings.Split(argsKeyValues, ";")
	return splited
}
