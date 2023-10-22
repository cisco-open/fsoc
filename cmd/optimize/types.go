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
	IgnoredBlockers  IgnoredBlockers       `json:"ignoredBlockers"`
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

type IgnoredBlockers struct {
	Principal Principal `json:"principal,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
	Blockers  Blockers  `json:"blockers,omitempty"`
}

type Principal struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type Blockers struct {
	Stateful                    *Blocker `json:"stateful,omitempty"`
	NoTraffic                   *Blocker `json:"noTraffic,omitempty"`
	ResourcesNotSpecified       *Blocker `json:"resourcesNotSpecified,omitempty"`
	CPUNotSpecified             *Blocker `json:"cpuNotSpecified,omitempty"`
	MemNotSpecified             *Blocker `json:"memNotSpecified,omitempty"`
	CPUResourcesChange          *Blocker `json:"cpuResourcesChange,omitempty"`
	MemoryResourcesChange       *Blocker `json:"memResourcesChange,omitempty"`
	K8sMetricsDeficient         *Blocker `json:"k8sMetricsDeficient,omitempty"`
	APMMetricsMissing           *Blocker `json:"apmMetricsMissing,omitempty"`
	APMMetricsDeficient         *Blocker `json:"apmMetricsDeficient,omitempty"`
	MultipleAPM                 *Blocker `json:"multipleAPM,omitempty"`
	UnequalLoadDistribution     *Blocker `json:"unequalLoadDistribution,omitempty"`
	NoScaling                   *Blocker `json:"noScaling,omitempty"`
	InsufficientRelativeScaling *Blocker `json:"insufficientRelativeScaling,omitempty"`
	InsufficientFixedScaling    *Blocker `json:"insufficientFixedScaling,omitempty"`
	MTBFHigh                    *Blocker `json:"mtbfHigh,omitempty"`
	ErrorRateHigh               *Blocker `json:"errorRateHigh,omitempty"`
	NoOrchestrationAgent        *Blocker `json:"noOrchestrationAgent,omitempty"`
}

type Blocker struct {
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Overridable bool   `json:"-"` // do not write to json for orion
}

type Suspension struct {
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
}
type K8SDeployment struct {
	ClusterID     string `json:"clusterId"`
	DeploymentUID string `json:"deploymentUid"`
	ClusterName   string `json:"clusterName"`
	ContainerName string `json:"containerName"`
	NamespaceName string `json:"namespaceName"`
	WorkloadID    string `json:"workloadId"`
	WorkloadName  string `json:"workloadName"`
}
type Target struct {
	K8SDeployment K8SDeployment `json:"k8sDeployment"`
}

type configJsonStoreItem struct {
	Data OptimizerConfiguration `json:"data"`
	JsonStoreItem
}

type configJsonStorePage struct {
	Items []configJsonStoreItem `json:"items"`
	Total int                   `json:"total"`
}

type OptimizerStatus struct {
	AgentState        string                 `json:"agentState"`
	OptimizationState string                 `json:"optimizationState"`
	Optimizer         OptimizerConfiguration `json:"optimizer"`
	OptimizerID       string                 `json:"optimizerId"`
	OptimizerState    string                 `json:"optimizerState"`
	ServoUID          string                 `json:"servoUid"`
	Suspended         bool                   `json:"suspended"`
	TuningState       string                 `json:"tuningState"`
}

type statusJsonStoreItem struct {
	Data OptimizerStatus `json:"data"`
	JsonStoreItem
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
