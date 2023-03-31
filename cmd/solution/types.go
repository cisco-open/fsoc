// Copyright 2022 Cisco Systems, Inc.
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

package solution

type Manifest struct {
	ManifestVersion string         `json:"manifestVersion,omitempty"`
	Name            string         `json:"name,omitempty"`
	SolutionVersion string         `json:"solutionVersion,omitempty"`
	Dependencies    []string       `json:"dependencies"`
	Description     string         `json:"description,omitempty"`
	Contact         string         `json:"contact,omitempty"`
	HomePage        string         `json:"homepage,omitempty"`
	GitRepoUrl      string         `json:"gitRepoUrl,omitempty"`
	Readme          string         `json:"readme,omitempty"`
	Objects         []ComponentDef `json:"objects,omitempty"`
	Types           []string       `json:"types,omitempty"`
}

type ComponentDef struct {
	Type        string `json:"type,omitempty"`
	ObjectsFile string `json:"objectsFile,omitempty"`
	ObjectsDir  string `json:"objectsDir,omitempty"`
}

type ServiceDef struct {
	Name  string `json:"name,omitempty"`
	Image string `json:"image,omitempty"`
}

type IdGenerationDef struct {
	GenerateRandomId        bool   `json:"generateRandomId"`
	EnforceGlobalUniqueness bool   `json:"enforceGlobalUniqueness"`
	IdGenerationMechanism   string `json:"idGenerationMechanism,omitempty"`
}

type KnowledgeDef struct {
	Name             string                 `json:"name,omitempty"`
	AllowedLayers    []string               `json:"allowedLayers,omitempty"`
	IdGeneration     *IdGenerationDef       `json:"idGeneration,omitempty"`
	SecureProperties []string               `json:"secureProperties,omitempty"`
	JsonSchema       map[string]interface{} `json:"jsonSchema,omitempty"`
}

type SolutionDef struct {
	Dependencies []string `json:"dependencies,omitempty"`
	IsSubscribed bool     `json:"isSubscribed,omitempty"`
	IsSystem     bool     `json:"isSystem,omitempty"`
	Name         string   `json:"name,omitempty"`
}

type FmmTypeDef struct {
	Namespace   FmmNamespaceAssignTypeDef `json:"namespace"`
	Kind        string                    `json:"kind"`
	Name        string                    `json:"name"`
	DisplayName string                    `json:"displayName,omitempty"`
}

type FmmNamespaceAssignTypeDef struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type FmmLifecycleConfigTypeDef struct {
	PurgeTtlInMinutes     int64 `json:"purgeTtlInMinutes"`
	RetentionTtlInMinutes int64 `json:"retentionTtlInMinutes"`
}

type FmmAttributeTypeDef struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type FmmAttributeDefinitionsTypeDef struct {
	Required   []string                        `json:"required"`
	Optimized  []string                        `json:"optimized"`
	Attributes map[string]*FmmAttributeTypeDef `json:"attributes"`
}

type FmmAssociationTypesTypeDef struct {
	Aggregates_of []string `json:"common:aggregates_of,omitempty"`
	Consists_of   []string `json:"common:consists_of,omitempty"`
	Is_a          []string `json:"common:is_a,omitempty"`
	Has           []string `json:"common:has,omitempty"`
	Relates_to    []string `json:"common:relates_to,omitempty"`
	Uses          []string `json:"common:uses,omitempty"`
}

type FmmEntity struct {
	*FmmTypeDef
	AttributeDefinitions  *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions"`
	LifecyleConfiguration *FmmLifecycleConfigTypeDef      `json:"lifecycleConfiguration"`
	MetricTypes           []string                        `json:"metricTypes,omitempty"`
	EventTypes            []string                        `json:"eventTypes,omitempty"`
	AssociationTypes      *FmmAssociationTypesTypeDef     `json:"associationTypes,omitempty"`
}

type FmmEvent struct {
	*FmmTypeDef
	AttributeDefinitions *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions"`
}

type FmmResourceMapping struct {
	*FmmTypeDef
	EntityType            string               `json:"entityType"`
	ScopeFilter           string               `json:"scopeFilter"`
	Mappings              []FmmMapAndTransform `json:"mappings,omitempty"`
	AttributeNameMappings FmmNameMappings      `json:"attributeNameMappings,omitempty"`
}

type FmmMapAndTransform struct {
	To   string `json:"to"`
	From string `json:"from"`
}

type FmmNameMappings map[string]string

type FmmNamespace struct {
	Name string `json:"name"`
}

type FmmMetric struct {
	*FmmTypeDef
	Category               FmmMetricCategory               `json:"category"`
	ContentType            FmmMetricContentType            `json:"contentType"`
	AggregationTemporality string                          `json:"aggregationTemporality"`
	IsMonotonic            bool                            `json:"isMonotonic"`
	Type                   FmmMetricType                   `json:"type"`
	Unit                   string                          `json:"unit"`
	AttributeDefinitions   *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions"`
}

type FmmMetricCategory string

const (
	Category_Sum     FmmMetricCategory = "sum"
	Category_Average FmmMetricCategory = "average"
	Category_Rate    FmmMetricCategory = "rate"
)

type FmmMetricContentType string

const (
	ContentType_Sum          FmmMetricContentType = "sum"
	ContentType_Gauge        FmmMetricContentType = "gauge"
	ContentType_Distribution FmmMetricContentType = "distribution"
)

type FmmMetricType string

const (
	Type_Long   FmmMetricType = "long"
	Type_Double FmmMetricType = "double"
)

type FmmTemporality string

const (
	Temp_Delta FmmTemporality = "delta"
	Temp_False FmmTemporality = "unspecified"
)
