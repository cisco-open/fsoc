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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
)

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

func (entity *FmmEntity) GetTypeName() string {
	return fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name)
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

type DashuiTemplate struct {
	Kind    string      `json:"kind"`
	Name    string      `json:"name"`
	Target  string      `json:"target"`
	View    string      `json:"view"`
	Element interface{} `json:"element"`
}

type DashuiWidget struct {
	InstanceOf string      `json:"instanceOf"`
	Props      interface{} `json:"props,omitempty"`
	Elements   interface{} `json:"elements,omitempty"`
	Element    interface{} `json:"element,omitempty"`
}

type DashuiFocusedEntity struct {
	*DashuiWidget
	Mode string `json:"mode"`
}

type DashuiString struct {
	InstanceOf string `json:"instanceOf"`
	Content    string `json:"content"`
}

type DashuiLabel struct {
	InstanceOf string      `json:"instanceOf"`
	Path       interface{} `json:"path"`
}

type DashuiProperties struct {
	InstanceOf string            `json:"instanceOf"`
	Elements   []*DashuiProperty `json:"elements"`
}
type DashuiProperty struct {
	Label *DashuiString `json:"label"`
	Value *DashuiLabel  `json:"value"`
}

type DashuiGrid struct {
	*DashuiWidget
	RowSets          interface{}         `json:"rowSets"`
	Style            interface{}         `json:"style,omitempty"`
	Mode             string              `json:"mode"`
	Columns          []*DashuiGridColumn `json:"columns"`
	OnRowSingleClick *DashuiEvent        `json:"onRowSingleClick,omitempty"`
	OnRowDoubleClick *DashuiEvent        `json:"onRowDoubleClick,omitempty"`
}

type DashuiGridColumn struct {
	Label string          `json:"label"`
	Flex  int             `json:"flex"`
	Width int             `json:"width"`
	Cell  *DashuiGridCell `json:"cell"`
}

type DashuiGridCell struct {
	Default interface{} `json:"default,omitempty"`
}

type DashuiTooltip struct {
	*DashuiLabel
	Truncate bool        `json:"truncate,omitempty"`
	Trigger  interface{} `json:"trigger,omitempty"`
}

type DashuiClickable struct {
	*DashuiWidget
	OnClick *DashuiEvent `json:"onclick,omitempty"`
	Trigger *DashuiLabel `json:"trigger,omitempty"`
}

type DashuiEvent struct {
	Type       string   `json:"type"`
	Paths      []string `json:"paths,omitempty"`
	Expression string   `json:"expression"`
}

type EcpLeftBar struct {
	*DashuiWidget
	Label string `json:"label"`
}

type EcpRelationshipMapEntry struct {
	Key             string `json:"key"`
	Path            string `json:"path"`
	EntityAttribute string `json:"entityAttribute"`
	IconName        string `json:"iconName"`
}

type EcpInspectorWidget struct {
	*DashuiWidget
	Title string `json:"title"`
}

type DashuiOcpSingle struct {
	*DashuiWidget
	NameAttribute string `json:"nameAttribute"`
}

type DashuiCartesian struct {
	*DashuiWidget
	Children []*DashuiCartesianSeries `json:"children"`
}

type DashuiCartesianSeries struct {
	Props  interface{}            `json:"props"`
	Metric *DashuiCartesianMetric `json:"metric"`
	Type   string                 `json:"type"`
}

type DashuiCartesianMetric struct {
	Name   string               `json:"name"`
	Source string               `json:"source"`
	Y      *DashuiCartesianAxis `json:"y"`
}

type DashuiCartesianAxis struct {
	Field string `json:"type"`
}

type Solution struct {
	ID             string `json:"id"`
	LayerID        string `json:"layerId"`
	LayerType      string `json:"layerType"`
	ObjectMimeType string `json:"objectMimeType"`
	TargetObjectId string `json:"targetObjectId"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	DisplayName    string `json:"displayName"`
}

type SolutionList struct {
	Items []Solution `json:"items"`
}

func (manifest *Manifest) GetFmmEntities() []*FmmEntity {
	fmmEntities := make([]*FmmEntity, 0)
	entityComponentDefs := manifest.GetComponentDefs("fmm:entity")
	for _, compDef := range entityComponentDefs {
		if compDef.ObjectsFile != "" {
			filePath := compDef.ObjectsFile
			fmmEntities = append(fmmEntities, getFmmEntitiesFromFile(filePath)...)
		}
		if compDef.ObjectsDir != "" {
			filePath := compDef.ObjectsDir
			err := filepath.Walk(filePath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.Contains(path, ".json") {
						fmmEntities = append(fmmEntities, getFmmEntitiesFromFile(path)...)
					}
					return nil
				})
			if err != nil {
				log.Fatalf("Error traversing the folder: %v", err)
			}
		}

	}
	return fmmEntities
}

func (manifest *Manifest) GetFmmMetrics() []*FmmMetric {
	fmmMetrics := make([]*FmmMetric, 0)
	entityComponentDefs := manifest.GetComponentDefs("fmm:metric")
	for _, compDef := range entityComponentDefs {
		if compDef.ObjectsFile != "" {
			filePath := compDef.ObjectsFile
			fmmMetrics = append(fmmMetrics, getFmmMetricsFromFile(filePath)...)
		}
		if compDef.ObjectsDir != "" {
			filePath := compDef.ObjectsDir
			err := filepath.Walk(filePath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.Contains(path, ".json") {
						fmmMetrics = append(fmmMetrics, getFmmMetricsFromFile(path)...)
					}
					return nil
				})
			if err != nil {
				log.Fatalf("Error traversing the folder: %v", err)
			}
		}

	}
	return fmmMetrics
}

func (manifest *Manifest) GetFmmEvents() []*FmmEvent {
	fmmEvents := make([]*FmmEvent, 0)
	entityComponentDefs := manifest.GetComponentDefs("fmm:event")
	for _, compDef := range entityComponentDefs {
		if compDef.ObjectsFile != "" {
			filePath := compDef.ObjectsFile
			fmmEvents = append(fmmEvents, getFmmEventsFromFile(filePath)...)
		}
		if compDef.ObjectsDir != "" {
			filePath := compDef.ObjectsDir
			err := filepath.Walk(filePath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.Contains(path, ".json") {
						fmmEvents = append(fmmEvents, getFmmEventsFromFile(path)...)
					}
					return nil
				})
			if err != nil {
				log.Fatalf("Error traversing the folder: %v", err)
			}
		}

	}
	return fmmEvents
}

func (manifest *Manifest) GetDashuiTemplates() []*DashuiTemplate {
	dashuiTemplates := make([]*DashuiTemplate, 0)
	objectDefs := manifest.GetComponentDefs("dashui:template")
	for _, objDef := range objectDefs {
		if objDef.ObjectsFile != "" {
			filePath := objDef.ObjectsFile
			dashuiTemplates = append(dashuiTemplates, getDashuiTemplatesFromFile(filePath)...)
		}
		if objDef.ObjectsDir != "" {
			filePath := objDef.ObjectsDir
			err := filepath.Walk(filePath,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.Contains(path, ".json") {
						dashuiTemplates = append(dashuiTemplates, getDashuiTemplatesFromFile(path)...)
					}
					return nil
				})
			if err != nil {
				log.Fatalf("Error traversing the folder: %v", err)
			}
		}

	}
	return dashuiTemplates
}

func (manifest *Manifest) CheckDependencyExists(solutionName string) bool {
	hasDependency := false
	for _, deps := range manifest.Dependencies {
		if deps == solutionName {
			hasDependency = true
		}
	}
	return hasDependency
}

func (manifest *Manifest) AppendDependency(solutionName string) {
	hasDependency := manifest.CheckDependencyExists(getSolutionNameWithZip(solutionName))
	if !hasDependency {
		manifest.Dependencies = append(manifest.Dependencies, solutionName)
	}

}

func (manifest *Manifest) GetComponentDef(typeName string) *ComponentDef {
	var componentDef ComponentDef
	for _, compDefs := range manifest.Objects {
		if compDefs.Type == typeName {
			componentDef = compDefs
		}
	}
	return &componentDef
}

func (manifest *Manifest) GetComponentDefs(typeName string) []ComponentDef {
	var componentDefs []ComponentDef
	for _, compDefs := range manifest.Objects {
		if compDefs.Type == typeName {
			componentDefs = append(componentDefs, compDefs)
		}
	}
	return componentDefs
}

func getFmmEntitiesFromFile(filePath string) []*FmmEntity {
	fmmEntities := make([]*FmmEntity, 0)
	entityDefFile := openFile(filePath)
	defer entityDefFile.Close()
	entityDefBytes, _ := io.ReadAll(entityDefFile)
	entityDefContent := string(entityDefBytes)

	if strings.Index(entityDefContent, "[") == 0 {
		entitiesArray := make([]*FmmEntity, 0)
		err := json.Unmarshal(entityDefBytes, &entitiesArray)
		if err != nil {
			log.Fatalf("Can't parse an array of entity definition objects from the %q file:\n %v", filePath, err)
		}
		fmmEntities = append(fmmEntities, entitiesArray...)
	} else {
		var entity *FmmEntity
		err := json.Unmarshal(entityDefBytes, &entity)
		if err != nil {
			log.Fatalf("Can't parse an entity definition objects from the %q file:\n %v", filePath, err)
		}
		fmmEntities = append(fmmEntities, entity)
	}
	return fmmEntities
}

func getFmmMetricsFromFile(filePath string) []*FmmMetric {
	fmmMetrics := make([]*FmmMetric, 0)
	metricDefFile := openFile(filePath)
	defer metricDefFile.Close()
	metricDefBytes, _ := io.ReadAll(metricDefFile)
	metricDefContent := string(metricDefBytes)

	if strings.Index(metricDefContent, "[") == 0 {
		metricsArray := make([]*FmmMetric, 0)
		err := json.Unmarshal(metricDefBytes, &metricsArray)
		if err != nil {
			log.Fatalf("Can't parse an array of metric definition objects from the %q file:\n %v", filePath, err)
		}
		fmmMetrics = append(fmmMetrics, metricsArray...)
	} else {
		var metric *FmmMetric
		err := json.Unmarshal(metricDefBytes, &metric)
		if err != nil {
			log.Fatalf("Can't parse a metric definition objects from the %q file:\n %v ", filePath, err)
		}
		fmmMetrics = append(fmmMetrics, metric)
	}
	return fmmMetrics
}

func getFmmEventsFromFile(filePath string) []*FmmEvent {
	fmmEvents := make([]*FmmEvent, 0)
	eventsDefFile := openFile(filePath)
	defer eventsDefFile.Close()
	eventDefBytes, _ := io.ReadAll(eventsDefFile)
	eventDefContent := string(eventDefBytes)

	if strings.Index(eventDefContent, "[") == 0 {
		eventsArray := make([]*FmmEvent, 0)
		err := json.Unmarshal(eventDefBytes, &eventsArray)
		if err != nil {
			log.Fatalf("Can't parse an array of event definition objects from the %q file:\n %v", filePath, err)
		}
		fmmEvents = append(fmmEvents, eventsArray...)
	} else {
		var event *FmmEvent
		err := json.Unmarshal(eventDefBytes, &event)
		if err != nil {
			log.Fatalf("Can't parse a event` definition objects from the %q file:\n %v ", filePath, err)
		}
		fmmEvents = append(fmmEvents, event)
	}
	return fmmEvents
}

func getDashuiTemplatesFromFile(filePath string) []*DashuiTemplate {
	dashuiTemplates := make([]*DashuiTemplate, 0)
	objDefFile := openFile(filePath)
	defer objDefFile.Close()
	objDefBytes, _ := io.ReadAll(objDefFile)
	objDefContent := string(objDefBytes)

	if strings.Index(objDefContent, "[") == 0 {
		objectsArray := make([]*DashuiTemplate, 0)
		err := json.Unmarshal(objDefBytes, &objectsArray)
		if err != nil {
			log.Fatalf("Can't parse an array of event definition objects from the %q file:\n %v", filePath, err)
		}
		dashuiTemplates = append(dashuiTemplates, objectsArray...)
	} else {
		var event *DashuiTemplate
		err := json.Unmarshal(objDefBytes, &event)
		if err != nil {
			log.Fatalf("Can't parse a event` definition objects from the %q file:\n %v ", filePath, err)
		}
		dashuiTemplates = append(dashuiTemplates, event)
	}
	return dashuiTemplates
}
