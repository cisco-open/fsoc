// Copyright 2024 Cisco Systems, Inc.
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
	Namespace   *FmmNamespaceAssignTypeDef `json:"namespace" yaml:"namespace"`
	Kind        string                     `json:"kind" yaml:"kind"`
	Name        string                     `json:"name" yaml:"name"`
	DisplayName string                     `json:"displayName,omitempty" yaml:"displayName,omitempty"`
}

type FmmNamespaceAssignTypeDef struct {
	Name    string `json:"name" yaml:"name"`
	Version int    `json:"version" yaml:"version"`
}

type FmmLifecycleConfigTypeDef struct {
	PurgeTtlInMinutes     int64 `json:"purgeTtlInMinutes" yaml:"purgeTtlInMinutes"`
	RetentionTtlInMinutes int64 `json:"retentionTtlInMinutes" yaml:"retentionTtlInMinutes"`
}

type FmmAttributeTypeDef struct {
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type FmmRequiredAttributeDefinitionsTypeDef struct {
	Required                        []string `json:"required" yaml:"required"`
	*FmmAttributeDefinitionsTypeDef `json:",inline" yaml:",inline"`
}

type FmmAttributeDefinitionsTypeDef struct {
	Optimized  []string                        `json:"optimized" yaml:"optimized"`
	Attributes map[string]*FmmAttributeTypeDef `json:"attributes" yaml:"attributes"`
}

type FmmAssociationTypesTypeDef struct {
	Aggregates_of []string `json:"common:aggregates_of,omitempty" yaml:"common:aggregates_of,omitempty"`
	Consists_of   []string `json:"common:consists_of,omitempty" yaml:"common:consists_of,omitempty"`
	Is_a          []string `json:"common:is_a,omitempty" yaml:"common:is_a,omitempty"`
	Has           []string `json:"common:has,omitempty" yaml:"common:has,omitempty"`
	Relates_to    []string `json:"common:relates_to,omitempty" yaml:"common:relates_to,omitempty"`
	Uses          []string `json:"common:uses,omitempty" yaml:"common:uses,omitempty"`
}

type FmmEntity struct {
	*FmmTypeDef           `json:",inline" yaml:",inline"`
	AttributeDefinitions  *FmmRequiredAttributeDefinitionsTypeDef `json:"attributeDefinitions,omitempty" yaml:"attributeDefinitions,omitempty"`
	LifecyleConfiguration *FmmLifecycleConfigTypeDef              `json:"lifecycleConfiguration" yaml:"lifecycleConfiguration"`
	MetricTypes           []string                                `json:"metricTypes,omitempty" yaml:"metricTypes,omitempty"`
	EventTypes            []string                                `json:"eventTypes,omitempty" yaml:"eventTypes,omitempty"`
	AssociationTypes      *FmmAssociationTypesTypeDef             `json:"associationTypes,omitempty" yaml:"associationTypes,omitempty"`
}

func (entity *FmmEntity) GetTypeName() string {
	return fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name)
}

type FmmEvent struct {
	*FmmTypeDef          `json:",inline" yaml:",inline"`
	AttributeDefinitions *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions" yaml:"attributeDefinitions"` // required always, do not omitempty
}

type FmmResourceMapping struct {
	*FmmTypeDef           `json:",inline" yaml:",inline"`
	EntityType            string               `json:"entityType" yaml:"entityType"`
	ScopeFilter           string               `json:"scopeFilter" yaml:"scopeFilter"`
	Mappings              []FmmMapAndTransform `json:"mappings,omitempty" yaml:"mappings,omitempty"`
	AttributeNameMappings FmmNameMappings      `json:"attributeNameMappings,omitempty" yaml:"attributeNameMappings,omitempty"`
}

type FmmAssociationDeclaration struct {
	*FmmTypeDef     `json:",inline" yaml:",inline"`
	ScopeFilter     string `json:"scopeFilter" yaml:"scopeFilter"`
	FromType        string `json:"fromType" yaml:"fromType"`
	ToType          string `json:"toType" yaml:"toType"`
	AssociationType string `json:"associationType" yaml:"associationType"`
}

type FmmMapAndTransform struct {
	To   string `json:"to" yaml:"to"`
	From string `json:"from" yaml:"from"`
}

type FmmNameMappings map[string]string

type FmmNamespace struct {
	Name string `json:"name" yaml:"name"`
}

type FmmMetric struct {
	*FmmTypeDef            `json:",inline" yaml:",inline"`
	Category               FmmMetricCategory               `json:"category" yaml:"category"`
	ContentType            FmmMetricContentType            `json:"contentType" yaml:"contentType"`
	AggregationTemporality string                          `json:"aggregationTemporality" yaml:"aggregationTemporality"`
	IsMonotonic            bool                            `json:"isMonotonic" yaml:"isMonotonic"`
	Type                   FmmMetricType                   `json:"type" yaml:"type"`
	Unit                   string                          `json:"unit" yaml:"unit"`
	AttributeDefinitions   *FmmAttributeDefinitionsTypeDef `json:"attributeDefinitions,omitempty" yaml:"attributeDefinitions,omitempty"`
	IngestGranularities    []int                           `json:"ingestGranularities,omitempty" yaml:"ingestGranularities,omitempty"`
}

type FmmMetricCategory string

const (
	Category_Sum       FmmMetricCategory = "sum"
	Category_Average   FmmMetricCategory = "average"
	Category_Rate      FmmMetricCategory = "rate"
	Category_Current   FmmMetricCategory = "current"
	Category_Histogram FmmMetricCategory = "histogram"
)

type FmmMetricContentType string

const (
	ContentType_Sum          FmmMetricContentType = "sum"
	ContentType_Gauge        FmmMetricContentType = "gauge"
	ContentType_Distribution FmmMetricContentType = "distribution"
	ContentType_Histogram    FmmMetricContentType = "histogram"
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
