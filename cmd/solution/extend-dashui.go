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
	ocpSingle := NewDashuiOcpSingle(getNamingAttribute(entity))

	ocpSingle.Elements = []DashuiWidget{
		{InstanceOf: fmt.Sprintf("%sDetailsList", entity.GetTypeName())},
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

func getEcpHome(manifest *Manifest) *DashuiTemplatePropsExtension {
	namespaceName := manifest.GetNamespaceName()
	id := fmt.Sprintf("%s:%sEcpHomeExtension", namespaceName, namespaceName)
	name := "dashui:ecpHome"
	view := "default"
	target := "*"

	ecpHomeSectionName := fmt.Sprintf("%sCoreSection", namespaceName)
	ecpHomeSectionTitle := fmt.Sprintf("%s - %s", namespaceName, manifest.SolutionVersion)
	ecpHome := &EcpHome{
		Sections: []*DashuiEcpHomeSection{
			{
				Index: 6,
				Name:  ecpHomeSectionName,
				Title: ecpHomeSectionTitle,
			},
		},
	}

	entityRefs := make([]string, 0)
	fmmEntities := manifest.GetFmmEntities()

	ecpHomeEntities := make([]*DashuiEcpHomeEntity, 0)
	for i, entity := range fmmEntities {
		ref := entity.GetTypeName()
		entityRefs = append(entityRefs, ref)
		ecpHomeEntities = append(ecpHomeEntities, &DashuiEcpHomeEntity{
			Index:           i,
			Section:         ecpHomeSectionName,
			EntityAttribute: "id",
			TargetType:      ref,
		})
	}
	ecpHome.Entities = ecpHomeEntities

	ecpHomeTemplateExtension := NewDashuiTemplatePropsExtension(id, name, target, view, entityRefs)

	ecpHomeTemplateExtension.Props = ecpHome

	return ecpHomeTemplateExtension
}

func getEcpName(entity *FmmEntity) *DashuiTemplate {
	namePath := []string{fmt.Sprintf("attributes(%s)", getNamingAttribute(entity)), "id"}

	dashuiNameTemplate := &DashuiTemplate{
		Kind:   "template",
		Target: entity.GetTypeName(),
		Name:   "dashui:name",
		View:   "default",
		Element: &DashuiLabel{
			InstanceOf: "string",
			Path:       namePath,
		},
	}

	return dashuiNameTemplate
}

func getRelationshipMap(entity *FmmEntity) *DashuiTemplate {

	ecpLeftBar := &DashuiWidget{
		InstanceOf: "leftbar",
	}

	ecpRelationshipMap := &DashuiTemplate{
		Kind:   "template",
		Target: fmt.Sprintf("%s:%s", entity.Namespace.Name, entity.Name),
		Name:   "dashui:ecpRelationshipMap",
		View:   "default",
	}

	ecpRelationshipMap.Element = ecpLeftBar

	elements := make([]EcpRelationshipMapEntry, 0)

	nameAttribute := getNamingAttribute(entity)

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

	namePath := []string{fmt.Sprintf("attributes(%s)", getNamingAttribute(entity)), "id"}
	focusedEntityNameWidget := &DashuiFocusedEntity{
		Mode: "SINGLE",
		DashuiWidget: &DashuiWidget{
			InstanceOf: "focusedEntities",
			Element: &DashuiLabel{
				InstanceOf: map[string]interface{}{"name": "nameWidget"},
				Path:       namePath,
			},
		},
	}

	focusedEntityEntityInspector := &DashuiFocusedEntity{
		Mode: "SINGLE",
		DashuiWidget: &DashuiWidget{
			InstanceOf: "focusedEntities",
			Element: &DashuiWidget{
				InstanceOf: map[string]interface{}{
					"name": fmt.Sprintf("%sInspectorWidget", entity.GetTypeName()),
				},
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

	instanceOf := map[string]interface{}{
		"name": "alerting",
	}

	alertingWidget := &DashuiWidget{
		InstanceOf: instanceOf,
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

	instanceOf := map[string]interface{}{"name": "health"}

	healthColumn := &DashuiGridColumn{
		Label: "Health",
		Flex:  0,
		Width: 80,
		Cell: &DashuiGridCell{
			Default: &DashuiWidget{
				InstanceOf: instanceOf,
			},
		},
	}

	columns = append(columns, healthColumn)

	namingAttribute := getNamingAttribute(entity)

	namingColumn := &DashuiGridColumn{
		Label: getColumnLabel(namingAttribute),
		Flex:  0,
		Width: 80,
		Cell: &DashuiGridCell{
			Default: NewDashuiTooltip(namingAttribute, true),
		},
	}

	columns = append(columns, namingColumn)

	for attribute := range entity.AttributeDefinitions.Attributes {
		if attribute == namingAttribute {
			continue
		}

		attrColumn := &DashuiGridColumn{
			Label: getColumnLabel(attribute),
			Flex:  0,
			Width: 80,
			Cell: &DashuiGridCell{
				Default: NewDashuiTooltip(attribute, false),
			},
		}
		columns = append(columns, attrColumn)
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

func getDashuiDetailsList(entity *FmmEntity, manifest *Manifest) *DashuiTemplate {

	htmlWidget := NewDashuiHtmlWidget()

	htmlWidget.Style = map[string]interface{}{
		"display":       "flex",
		"flexDirection": "column",
		"gap":           12,
	}

	elements := make([]interface{}, 0)

	logsWidget := NewDashuiLogsWidget()

	elements = append(elements, logsWidget)

	fmmMetrics := manifest.GetFmmMetrics()

	for _, metricRef := range entity.MetricTypes {
		cardTitle := ""

		for _, metric := range fmmMetrics {
			metricTypeName := fmt.Sprintf("%s:%s", metric.Namespace.Name, metric.Name)
			if metricTypeName == metricRef {
				cardTitle = metric.DisplayName
				break
			}
		}
		if cardTitle == "" {
			cardTitle = metricRef
		}

		chart := NewDashuiCartesian()
		chart.Children = []*DashuiCartesianSeries{
			NewDashuiCartesianSeries(metricRef, metricRef, "fsoc-melt", "LINE"),
		}

		cardContainer := &DashuiWidget{
			InstanceOf: "card",
			Props: map[string]interface{}{
				"title": cardTitle,
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

func NewDashuiHtmlWidget() *DashuiHtmlWidget {
	return &DashuiHtmlWidget{
		DashuiWidget: &DashuiWidget{
			InstanceOf: "html",
		},
	}

}

func NewDashuiLogsWidget() *DashuiLogsWidget {
	return &DashuiLogsWidget{
		InstanceOf: map[string]interface{}{
			"name": "logsWidget",
		},
		Source: "derived_metric",
	}
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

	inspectorWidget.Elements = propertiesWidget

	ecpInspectorListElement := &DashuiWidget{
		InstanceOf: "elements",
		Elements:   inspectorWidget,
	}
	ecpInspectorWidget := &DashuiTemplate{
		Kind:    "template",
		Target:  entity.GetTypeName(),
		Name:    fmt.Sprintf("%sInspectorWidget", entity.GetTypeName()),
		View:    "default",
		Element: ecpInspectorListElement,
	}

	return ecpInspectorWidget
}

func getNamingAttribute(entity *FmmEntity) string {
	var nameAttribute string
	_, exists := entity.AttributeDefinitions.Attributes["name"]
	if exists {
		return "name"
	}

	nameAttribute = fmt.Sprintf("%s.%s.name", entity.Namespace.Name, entity.Name)
	_, exists = entity.AttributeDefinitions.Attributes[nameAttribute]
	if exists {
		return nameAttribute
	}

	nameAttribute = entity.AttributeDefinitions.Required[0]
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
		DashuiWidget: &DashuiWidget{
			InstanceOf: "tooltip",
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
	} else {
		toolTipObj.Trigger = &DashuiLabel{
			InstanceOf: "string",
			Path:       []string{fmt.Sprintf("attributes(%s)", attributeName)},
		}
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
	rowSets := map[string]interface{}{
		"default": map[string]interface{}{
			"keyPath": "id",
		},
	}

	grid.RowSets = rowSets

	return grid
}

func NewEcpInspectorWidget(title string) *EcpInspectorWidget {
	inspectorWidget := &EcpInspectorWidget{
		DashuiWidget: &DashuiWidget{
			InstanceOf: map[string]interface{}{"name": "inspectorWidget"},
		},
		Title: title,
	}
	return inspectorWidget
}

func NewDashuiProperties() *DashuiProperties {
	propertiesWidget := &DashuiProperties{
		InstanceOf: map[string]interface{}{"name": "properties"},
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

func NewDashuiTemplatePropsExtension(id string, name string, target string, view string, requiredEntityTypes []string) *DashuiTemplatePropsExtension {
	templatePropsExtension := &DashuiTemplatePropsExtension{
		Kind:                "templatePropsExtension",
		Id:                  id,
		Name:                name,
		View:                view,
		Target:              target,
		RequiredEntityTypes: requiredEntityTypes,
	}

	return templatePropsExtension
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

func getColumnLabel(attributeName string) string {
	attrSplit := strings.Split(attributeName, ".")
	var label string
	if len(attrSplit) > 0 {
		label = attrSplit[len(attrSplit)-1]
	} else {
		label = attributeName
	}

	return label
}
