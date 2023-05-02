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

import "time"

type OptimizerConfiguration struct {
	Config           Config                `json:"config"`
	DesiredState     string                `json:"desiredState"`
	OptimizerID      string                `json:"optimizerId"`
	RestartTimestamp string                `json:"restartTimestamp"`
	Suspensions      map[string]Suspension `json:"suspensions"`
	Target           Target                `json:"target"`
}
type CPU struct {
	Max    float64 `json:"max"`
	Min    float64 `json:"min"`
	Pinned bool    `json:"pinned"`
}
type Mem struct {
	Max    float64 `json:"max"`
	Min    float64 `json:"min"`
	Pinned bool    `json:"pinned"`
}
type Guardrails struct {
	CPU CPU `json:"cpu"`
	Mem Mem `json:"mem"`
}
type ErrorPercent struct {
	Target float64 `json:"target"`
}
type MedianResponseTime struct {
	Target float64 `json:"target"`
}
type Slo struct {
	ErrorPercent       ErrorPercent       `json:"errorPercent"`
	MedianResponseTime MedianResponseTime `json:"medianResponseTime"`
}
type Config struct {
	Guardrails Guardrails `json:"guardrails"`
	Slo        Slo        `json:"slo"`
}
type Suspension struct {
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
}
type K8SDeployment struct {
	ClusterID     string `json:"clusterId"`
	ClusterName   string `json:"clusterName"`
	ContainerName string `json:"containerName"`
	NamespaceName string `json:"namespaceName"`
	WorkloadID    string `json:"workloadId"`
	WorkloadName  string `json:"workloadName"`
}
type Target struct {
	K8SDeployment K8SDeployment `json:"k8sDeployment"`
}

// TODO move to Orion package?
type JsonStoreItem struct {
	CreatedAt time.Time `json:"createdAt"`
	// Data           Data      `json:"data"`  NOTE: leave out data so that its type can be specified by embedding this type into another
	ID             string    `json:"id"`
	LayerID        string    `json:"layerId"`
	LayerType      string    `json:"layerType"`
	ObjectMimeType string    `json:"objectMimeType"`
	ObjectType     string    `json:"objectType"`
	ObjectVersion  int       `json:"objectVersion"`
	Patch          any       `json:"patch"`
	TargetObjectID any       `json:"targetObjectId"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
