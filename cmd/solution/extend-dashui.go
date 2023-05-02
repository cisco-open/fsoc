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

package solution

import (
	"fmt"
	"strings"
)

func getEcpList(entity *FmmEntity) *DashuiTemplate {
	ecpList := &DashuiTemplate{
		Kind:   "template",
		View:   "default",
		Name:   "dashui:ecpList",
		Target: entity.GetTypeName(),
		Element: &DashuiWidget{
			InstanceOf: "ocpList",
			Elements: &DashuiWidget{
				InstanceOf: "card",
				Props: map[string]interface{}{
					"style": map[string]interface{}{
						"width":   "100%",
						"height":  "calc(100% - 298px)",
						"padding": 0,
					},
				},
				Elements: []DashuiWidget{
					{
						InstanceOf: fmt.Sprintf("%sGridTable", entity.GetTypeName()),
					},
				},
			},
		},
	}

	return ecpList
}

func getEcpDetails(entity *FmmEntity) *DashuiTemplate {
	ocpSingle := NewDashuiOcpSingle(getNameAttribute(entity))

	ocpSingle.Elements = []DashuiWidget{
		{InstanceOf: fmt.Sprintf("%sDetailList", entity.GetTypeName())},
	}

	ecpDetails := &DashuiTemplate{
		Kind:    "template",
		View:    "default",
		Name:    "dashui:ecpDetails",
		Target:  entity.GetTypeName(),
		Element: ocpSingle,
	}
	return ecpDetails
}

func getEcpName(entity *FmmEntity) *DashuiTemplate {
	namePath := []string{fmt.Sprintf("attributes(%s)", getNameAttribute(entity)), "id"}

	dashuiNameTemplate := &DashuiTemplate{
		Kind:   "template",
		Target: fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name),
		Name:   "dashui:name",
		View:   "default",
		Element: &DashuiLabel{
			InstanceOf: "nameWidget",
			Path:       namePath,
		},
	}

	return dashuiNameTemplate
}

func getRelationshipMap(entity *FmmEntity) *DashuiTemplate {

	ecpLeftBar := &DashuiWidget{
		InstanceOf: "leftBar",
	}

	ecpRelationshipMap := &DashuiTemplate{
		Kind:   "template",
		Target: fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name),
		Name:   "dashui:ecpRelationshipMap",
		View:   "default",
	}

	ecpRelationshipMap.Element = ecpLeftBar

	elements := make([]EcpRelationshipMapEntry, 0)

	nameAttribute := getNameAttribute(entity)

	elements = append(elements, EcpRelationshipMapEntry{
		Key:             entity.Name,
		Path:            ".",
		EntityAttribute: nameAttribute,
		IconName:        "AgentType.Appd",
	})

	if entity.AssociationTypes != nil {
		if entity.AssociationTypes.Consists_of != nil {
			for _, assoc := range entity.AssociationTypes.Consists_of {
				ascEntity := strings.Split(assoc, ":")[1]
				elements = append(elements, EcpRelationshipMapEntry{
					Key:             ascEntity,
					Path:            fmt.Sprintf("out(common:consists_of).to(%s)", assoc),
					EntityAttribute: "id",
					IconName:        "AgentType.Appd",
				})
			}
		}
		if entity.AssociationTypes.Aggregates_of != nil {

			for _, assoc := range entity.AssociationTypes.Aggregates_of {
				ascEntity := strings.Split(assoc, ":")[1]
				elements = append(elements, EcpRelationshipMapEntry{
					Key:             ascEntity,
					Path:            fmt.Sprintf("out(common:consists_of).to(%s)", assoc),
					EntityAttribute: "id",
					IconName:        "AgentType.Appd",
				})
			}
		}

		if entity.AssociationTypes.Has != nil {
			for _, assoc := range entity.AssociationTypes.Has {
				ascEntity := strings.Split(assoc, ":")[1]
				elements = append(elements, EcpRelationshipMapEntry{
					Key:             ascEntity,
					Path:            fmt.Sprintf("out(common:consists_of).to(%s)", assoc),
					EntityAttribute: "id",
					IconName:        "AgentType.Appd",
				})
			}
		}
	}

	ecpLeftBar.Elements = elements

	return ecpRelationshipMap
}

func getEcpListInspector(entity *FmmEntity) *DashuiTemplate {
	ecpListInspector := &DashuiTemplate{
		Kind:   "template",
		Target: fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name),
		Name:   "dashui:ecpListInspector",
		View:   "default",
	}

	namePath := []string{fmt.Sprintf("attributes(%s)", getNameAttribute(entity)), "id"}
	focusedEntityNameWidget := &DashuiFocusedEntity{
		Mode: "SINGLE",
		DashuiWidget: &DashuiWidget{
			InstanceOf: "focusedEntities",
			Element: &DashuiLabel{
				InstanceOf: "nameWidget",
				Path:       namePath,
			},
		},
	}

	focusedEntityEntityInspector := &DashuiFocusedEntity{
		Mode: "SINGLE",
		DashuiWidget: &DashuiWidget{
			InstanceOf: "focusedEntities",
			Element: &DashuiWidget{
				InstanceOf: fmt.Sprintf("%s:%sInspectorWidget", entity.Namespace.Name, entity.Name),
			},
		},
	}

	elements := []*DashuiFocusedEntity{focusedEntityNameWidget, focusedEntityEntityInspector}

	elementsWidget := &DashuiWidget{
		InstanceOf: "elements",
		Elements:   elements,
	}

	ecpListInspector.Element = elementsWidget
	return ecpListInspector
}

func getEcpDetailsInspector(entity *FmmEntity) *DashuiTemplate {
	ecpDetailsInspector := &DashuiTemplate{
		Kind:   "template",
		Target: entity.GetTypeName(),
		Name:   "dashui:ecpDetailsInspector",
		View:   "default",
	}

	alertingWidget := &DashuiWidget{
		InstanceOf: "alerting",
	}

	inspectorWidget := &DashuiWidget{
		InstanceOf: fmt.Sprintf("%sInspectorWidget", entity.GetTypeName()),
	}

	elements := []*DashuiWidget{alertingWidget, inspectorWidget}

	elementsWidget := &DashuiWidget{
		InstanceOf: "elements",
		Elements:   elements,
	}

	ecpDetailsInspector.Element = elementsWidget
	return ecpDetailsInspector
}

func getDashuiGridTable(entity *FmmEntity) *DashuiTemplate {
	grid := NewDashuiGrid()
	grid.Mode = "server"
	columns := make([]*DashuiGridColumn, 0)

	healthColumn := &DashuiGridColumn{
		Label: "Health",
		Flex:  0,
		Width: 80,
		Cell: &DashuiGridCell{
			Default: &DashuiWidget{
				InstanceOf: "health",
			},
		},
	}
	columns = append(columns, healthColumn)

	attrCount := 0
	for attribute := range entity.AttributeDefinitions.Attributes {
		attrSplit := strings.Split(attribute, ".")
		var label string
		if len(attrSplit) > 0 {
			label = attrSplit[len(attrSplit)-1]
		} else {
			label = attribute
		}

		attrColumn := &DashuiGridColumn{
			Label: label,
			Flex:  0,
			Width: 80,
			Cell: &DashuiGridCell{
				Default: NewDashuiTooltip(attribute, attrCount == 0),
			},
		}
		columns = append(columns, attrColumn)
		attrCount++
	}

	grid.Columns = columns
	grid.OnRowSingleClick = &DashuiEvent{
		Type:       "common.focusEntity",
		Expression: "{ \"id\": $params.key }",
	}

	grid.OnRowDoubleClick = &DashuiEvent{
		Type:       "navigate.entity.detail",
		Expression: "{ \"id\": $params.key }",
	}

	gridTable := &DashuiTemplate{
		Kind:    "template",
		Target:  entity.GetTypeName(),
		Name:    fmt.Sprintf("%sGridTable", entity.GetTypeName()),
		View:    "default",
		Element: grid,
	}

	return gridTable
}

func getDashuiDetailsList(entity *FmmEntity) *DashuiTemplate {

	htmlWidget := &DashuiWidget{
		InstanceOf: "html",
		Props: map[string]interface{}{
			"style": map[string]interface{}{
				"display":       "flex",
				"flexDirection": "column",
				"gap":           12,
			},
		},
	}

	elements := make([]*DashuiWidget, 0)

	for _, metric := range entity.MetricTypes {
		chart := NewDashuiCartesian()
		chart.Children = []*DashuiCartesianSeries{
			NewDashuiCartesianSeries(metric, metric, "fsoc-melt", "LINE"),
		}

		cardContainer := &DashuiWidget{
			InstanceOf: "card",
			Props: map[string]interface{}{
				"title": metric,
			},
			Elements: []*DashuiCartesian{
				chart,
			},
		}

		elements = append(elements, cardContainer)
	}

	htmlWidget.Elements = elements

	detailsList := &DashuiTemplate{
		Kind:    "template",
		Target:  entity.GetTypeName(),
		Name:    fmt.Sprintf("%sDetailsList", entity.GetTypeName()),
		View:    "default",
		Element: htmlWidget,
	}

	return detailsList
}

func getEcpInspectorWidget(entity *FmmEntity) *DashuiTemplate {
	inspectorWidget := NewEcpInspectorWidget("Properties")

	propertiesWidget := NewDashuiProperties()

	elements := make([]*DashuiProperty, 0)

	for attribute := range entity.AttributeDefinitions.Attributes {
		property := &DashuiProperty{
			Label: &DashuiString{
				InstanceOf: "text",
				Content:    attribute,
			},
			Value: &DashuiLabel{
				InstanceOf: "string",
				Path:       fmt.Sprintf("attributes(%s)", attribute),
			},
		}
		elements = append(elements, property)
	}

	propertiesWidget.Elements = elements

	inspectorWidget.Element = propertiesWidget

	ecpInspectorWidget := &DashuiTemplate{
		Kind:    "template",
		Target:  entity.GetTypeName(),
		Name:    fmt.Sprintf("%sInspectorWidget", entity.GetTypeName()),
		View:    "default",
		Element: inspectorWidget,
	}

	return ecpInspectorWidget
}

func getNameAttribute(entity *FmmEntity) string {
	var nameAttribute string
	_, exists := entity.AttributeDefinitions.Attributes["name"]

	if exists {
		nameAttribute = "name"
	} else {
		nameAttribute = fmt.Sprintf("%s.%s.name", entity.Namespace.Name, entity.Name)
	}
	return nameAttribute
}

func NewDashuiClickable() *DashuiClickable {
	clickable := &DashuiClickable{
		DashuiWidget: &DashuiWidget{
			InstanceOf: "clickable",
		},
	}
	return clickable
}

func NewDashuiTooltip(attributeName string, isClickable bool) *DashuiTooltip {
	toolTipObj := &DashuiTooltip{
		DashuiLabel: &DashuiLabel{
			InstanceOf: "tooltip",
			Path:       fmt.Sprintf("attributes(%s)", attributeName),
		},
		Truncate: true,
	}

	if isClickable {
		clickable := NewDashuiClickable()
		clickable.Trigger = &DashuiLabel{
			InstanceOf: "string",
			Path:       []string{fmt.Sprintf("attributes(%s)", attributeName), "id"},
		}
		clickable.OnClick = &DashuiEvent{
			Type:       "navigate.entity.detail",
			Paths:      []string{"id"},
			Expression: "$ ~> |$|{\"id\": $data[0]}|",
		}

		toolTipObj.Trigger = clickable
	}

	return toolTipObj
}

func NewClickableDashuiGridCell(attribute string) *DashuiGridCell {
	clickableCell := &DashuiGridCell{
		Default: NewDashuiTooltip(attribute, true),
	}

	return clickableCell
}

func NewDashuiGridCell(attribute string) *DashuiGridCell {
	clickableCell := &DashuiGridCell{
		Default: NewDashuiTooltip(attribute, false),
	}
	return clickableCell
}

func NewDashuiGrid() *DashuiGrid {
	grid := &DashuiGrid{
		DashuiWidget: &DashuiWidget{
			InstanceOf: "grid",
		},
	}
	return grid
}

func NewEcpInspectorWidget(title string) *EcpInspectorWidget {
	inspectorWidget := &EcpInspectorWidget{
		DashuiWidget: &DashuiWidget{
			InstanceOf: "inspectorWidget",
		},
		Title: title,
	}
	return inspectorWidget
}

func NewDashuiProperties() *DashuiProperties {
	propertiesWidget := &DashuiProperties{
		InstanceOf: "properties",
	}
	return propertiesWidget
}

func NewDashuiOcpSingle(nameAttribute string) *DashuiOcpSingle {
	ocpSingle := &DashuiOcpSingle{
		DashuiWidget: &DashuiWidget{
			InstanceOf: "ocpSingle",
		},
		NameAttribute: nameAttribute,
	}

	return ocpSingle
}

func NewDashuiCartesian() *DashuiCartesian {
	cartesian := &DashuiCartesian{
		DashuiWidget: &DashuiWidget{
			InstanceOf: "cartesian",
			Props: map[string]interface{}{
				"style": map[string]interface{}{
					"height": 250,
				},
			},
		},
	}
	return cartesian
}

func NewDashuiCartesianSeries(seriesName string, metricName string, metricSource string, seriesType string) *DashuiCartesianSeries {
	cartesianSeries := &DashuiCartesianSeries{
		Props: map[string]interface{}{
			"name": seriesName,
		},
		Metric: &DashuiCartesianMetric{
			Name:   metricName,
			Source: metricSource,
			Y: &DashuiCartesianAxis{
				Field: "value",
			},
		},
		Type: seriesType,
	}
	return cartesianSeries
}
