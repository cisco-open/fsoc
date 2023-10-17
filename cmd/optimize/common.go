// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package optimize

import (
	"fmt"
	"reflect"

	"github.com/cisco-open/fsoc/cmd/config"
)

// sliceToMap converts a list of lists (slice [][2]any) to a dictionary for table output jq support
// eg.
//
//	[
//		["k8s.cluster.name", "ignite-test"],
//		["k8s.namespace.name", "kube-system"],
//		["k8s.workload.kind", "Deployment"],
//		["k8s.workload.name", "coredns"]
//	]
//
// to
//
//	k8s.cluster.name: ignite-test
//	k8s.namespace.name: kube-system
//	k8s.workload.kind: Deployment
//	k8s.workload.name: coredns
func sliceToMap(slice [][]any) (map[string]any, error) {
	results := make(map[string]any)
	for index, subslice := range slice {
		if len(subslice) < 2 {
			return results, fmt.Errorf("subslice (at index %v) too short to construct key value pair: %+v", index, subslice)
		}
		key, ok := subslice[0].(string)
		if !ok {
			return results, fmt.Errorf("string type assertion failed on first subslice item (at index %v): %+v", index, subslice)
		}
		results[key] = subslice[1]
	}
	return results, nil
}

func setNestedMap(baseMap map[string]interface{}, keys []string, value interface{}) {
	if len(keys) == 1 {
		baseMap[keys[0]] = value
		return
	}

	if _, ok := baseMap[keys[0]]; !ok {
		baseMap[keys[0]] = make(map[string]interface{})
	}
	setNestedMap(baseMap[keys[0]].(map[string]interface{}), keys[1:], value)
}

func getOrionTenantHeaders() map[string]string {
	return map[string]string{
		"layer-type": "TENANT",
		"layer-id":   config.GetCurrentContext().Tenant,
	}
}

func checkHardBlockers(b *Blockers) bool {
	val := reflect.ValueOf(b).Elem()

	for i := 0; i < val.NumField(); i++ {
		blockerField := val.Field(i)
		if blocker, ok := blockerField.Interface().(*Blocker); ok && blocker != nil && !blocker.Overridable {
			return true
		}
	}
	return false
}
