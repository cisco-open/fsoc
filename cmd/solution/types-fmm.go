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

import (
	"fmt"
)

type FmmTypeDef struct {
	Namespace   *FmmNamespaceAssignTypeDef `json:"namespace"`
	Kind        string                     `json:"kind"`
	Name        string                     `json:"name"`
	DisplayName string                     `json:"displayName,omitempty"`
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
	AttributeDefinitions  *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions,omitempty"`
	LifecyleConfiguration *FmmLifecycleConfigTypeDef      `json:"lifecycleConfiguration"`
	MetricTypes           []string                        `json:"metricTypes,omitempty"`
	EventTypes            []string                        `json:"eventTypes,omitempty"`
	AssociationTypes      *FmmAssociationTypesTypeDef     `json:"associationTypes,omitempty"`
}

func (entity *FmmEntity) GetTypeName() string {
	return fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name)
}

type FmmEvent struct {
	*FmmTypeDef
	AttributeDefinitions *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions"` // required always, do not omitempty
}

type FmmResourceMapping struct {
	*FmmTypeDef
	EntityType            string               `json:"entityType"`
	ScopeFilter           string               `json:"scopeFilter"`
	Mappings              []FmmMapAndTransform `json:"mappings,omitempty"`
	AttributeNameMappings FmmNameMappings      `json:"attributeNameMappings,omitempty"`
}

type FmmAssociationDeclaration struct {
	*FmmTypeDef
	ScopeFilter     string `json:"scopeFilter"`
	FromType        string `json:"fromType"`
	ToType          string `json:"toType"`
	AssociationType string `json:"associationType"`
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
	AttributeDefinitions   *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions,omitempty"`
	IngestGranularities    []int                           `json:"ingestGranularities,omitempty"`
}

type FmmMetricCategory string

const (
	Category_Sum     FmmMetricCategory = "sum"
	Category_Average FmmMetricCategory = "average"
	Category_Rate    FmmMetricCategory = "rate"
	Category_Current FmmMetricCategory = "current"
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
