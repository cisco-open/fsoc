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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

var solutionExtendCmd = &cobra.Command{
	Use:              "extend",
	Args:             cobra.ExactArgs(0),
	Short:            "Extends your solution package by adding new components",
	Long:             `This command allows you to easily add new components to your solution package.`,
	Example:          `  fsoc solution extend --add-knowledge=<knowldgetypename>`,
	Run:              extendSolution,
	TraverseChildren: true,
}

// Planned options:
// --add-meltworkflow - Flag to add a new melt workflow component to the current solution package
// --add-dash-ui - Flag to add a new user experience component to the current solution package

func getSolutionExtendCmd() *cobra.Command {
	solutionExtendCmd.Flags().
		String("add-service", "", "Add a new service component definition to this solution")
	solutionExtendCmd.Flags().
		String("add-knowledge", "", "Add a new knowledge type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-entity", "", "Add a new entity type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-metric", "", "Add a new metric type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-resourceMapping", "", "Add a new resource mapping type definition for a given entity within this solution")
	solutionExtendCmd.Flags().
		String("add-event", "", "Add a new event type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-ecpList", "", "Add all template definitions to build a list experience for a given entity within this solution")
	solutionExtendCmd.Flags().
		String("add-ecpDetails", "", "Add all template definition to build the details experience for a given entity within this solution")

	return solutionExtendCmd

}

func extendSolution(cmd *cobra.Command, args []string) {
	manifest := GetManifest()

	if cmd.Flags().Changed("add-knowledge") {
		componentName, _ := cmd.Flags().GetString("add-knowledge")
		componentName = strings.ToLower(componentName)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %s knowledge component to %s's solution package folder structure... \n", componentName, manifest.Name))
		folderName := "types"
		fileName := fmt.Sprintf("%s.json", componentName)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Creating the %s file\n", fileName))
		manifest.Types = append(manifest.Types, fmt.Sprintf("%s/%s", folderName, fileName))
		f, err := os.OpenFile("./manifest.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Can't open manifest file: %v", err)
		}
		defer f.Close()
		err = output.WriteJson(manifest, f)
		if err != nil {
			log.Fatalf("Failed to update manifest.json file to reflect new knowledge type: %v", err)
		}

		knowledgeComp := getKnowledgeComponent(componentName)
		createComponentFile(knowledgeComp, folderName, fileName)
	}

	if cmd.Flags().Changed("add-service") {
		componentName, _ := cmd.Flags().GetString("add-service")
		componentName = strings.ToLower(componentName)
		folderName := "objects/services"
		addNewComponent(cmd, manifest, folderName, componentName, "zodiac:function")
	}

	if cmd.Flags().Changed("add-entity") {
		componentName, _ := cmd.Flags().GetString("add-entity")
		componentName = strings.ToLower(componentName)
		folderName := "objects/model/entities"
		addNewComponent(cmd, manifest, folderName, componentName, "fmm:entity")
	}

	if cmd.Flags().Changed("add-resourceMapping") {
		componentName, _ := cmd.Flags().GetString("add-resourceMapping")
		componentName = strings.ToLower(componentName)
		folderName := "objects/model/resource-mappings"
		addNewComponent(cmd, manifest, folderName, componentName, "fmm:resourceMapping")
	}

	if cmd.Flags().Changed("add-metric") {
		componentName, _ := cmd.Flags().GetString("add-metric")
		componentName = strings.ToLower(componentName)
		folderName := "objects/model/metrics"
		addNewComponent(cmd, manifest, folderName, componentName, "fmm:metric")
	}

	if cmd.Flags().Changed("add-event") {
		componentName, _ := cmd.Flags().GetString("add-event")
		componentName = strings.ToLower(componentName)
		folderName := "objects/model/events"

		addNewComponent(cmd, manifest, folderName, componentName, "fmm:event")
	}

	if cmd.Flags().Changed("add-ecpList") {
		componentName, _ := cmd.Flags().GetString("add-ecpList")
		entityName := strings.ToLower(componentName)
		folderName := fmt.Sprintf("objects/dashui/templates/%s", entityName)

		addNewComponent(cmd, manifest, folderName, entityName, "dashui:ecpList")
	}

	if cmd.Flags().Changed("add-ecpDetails") {
		componentName, _ := cmd.Flags().GetString("add-ecpDetails")
		entityName := strings.ToLower(componentName)
		folderName := fmt.Sprintf("objects/dashui/templates/%s", entityName)

		addNewComponent(cmd, manifest, folderName, entityName, "dashui:ecpDetails")
	}

}

func addNewComponent(cmd *cobra.Command, manifest *Manifest, folderName, componentName, componentType string) {
	type newComponent struct {
		Type       string
		Definition interface{}
		Filename   string
	}

	hasDashuiTemplate := func(entity *FmmEntity, dashuiTemplates []*DashuiTemplate, templateName string) bool {

		hasInspectorWidget := false
		for _, e := range dashuiTemplates {
			if e.Target == entity.GetTypeName() && e.Name == templateName {
				hasInspectorWidget = true
				break
			}
		}

		return hasInspectorWidget
	}

	var newComponents []*newComponent

	if strings.Index(componentType, "fmm") >= 0 {
		checkCreateSolutionNamespace(cmd, manifest, "objects/model/namespaces")
	}

	switch componentType {
	case "zodiac:function":
		{
			component := &newComponent{
				Type:       componentType,
				Definition: getServiceComponent(componentName),
			}

			newComponents = append(newComponents, component)
		}
	case "fmm:entity":
		{
			entity := &newComponent{
				Filename:   componentName + ".json",
				Type:       componentType,
				Definition: getEntityComponent(componentName, manifest.Name),
			}

			newComponents = append(newComponents, entity)
		}
	case "fmm:resourceMapping":
		{
			entityName, _ := cmd.Flags().GetString("add-resourceMapping")
			entityName = strings.ToLower(entityName)
			entity := &newComponent{
				Filename:   componentName + "-resourceMapping.json",
				Type:       componentType,
				Definition: getResourceMap(nil, entityName, manifest),
			}

			newComponents = append(newComponents, entity)

		}
	case "fmm:metric":
		{
			metric := &newComponent{
				Filename:   componentName + ".json",
				Type:       componentType,
				Definition: getMetricComponent(componentName, ContentType_Sum, Category_Sum, Type_Long, manifest.Name),
			}

			newComponents = append(newComponents, metric)
		}
	case "fmm:event":
		{
			event := &newComponent{
				Filename:   componentName + ".json",
				Type:       componentType,
				Definition: getEventComponent(componentName, manifest.Name),
			}

			newComponents = append(newComponents, event)
		}
	case "dashui:ecpList":
		{
			entityName := strings.ToLower(componentName)
			fmmEntities := manifest.GetFmmEntities()
			var entity *FmmEntity
			for _, e := range fmmEntities {
				if e.Name == entityName {
					entity = e
					break
				}
			}
			if entity == nil {
				log.Fatalf("Couldn't find an entity type named %s", entityName)
			}

			dashuiTemplates := manifest.GetDashuiTemplates()

			ecpList := &newComponent{
				Filename:   "ecpList.json",
				Type:       "dashui:template",
				Definition: getEcpList(entity),
			}

			newComponents = append(newComponents, ecpList)

			entityGridTable := &newComponent{
				Filename:   fmt.Sprintf("%sGridTable.json", entity.Name),
				Type:       "dashui:template",
				Definition: getDashuiGridTable(entity),
			}

			newComponents = append(newComponents, entityGridTable)

			ecpRelationshipMap := &newComponent{
				Filename:   "ecpRelationshipMap.json",
				Type:       "dashui:template",
				Definition: getRelationshipMap(entity),
			}

			newComponents = append(newComponents, ecpRelationshipMap)

			ecpListInspector := &newComponent{
				Filename:   "ecpListInspector.json",
				Type:       "dashui:template",
				Definition: getEcpListInspector(entity),
			}

			newComponents = append(newComponents, ecpListInspector)

			templateName := fmt.Sprintf("%sInspectorWidget", entity.GetTypeName())

			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				entityInspectorWidget := &newComponent{
					Filename:   fmt.Sprintf("%sInspectorWidget.json", entity.Name),
					Type:       "dashui:template",
					Definition: getEcpInspectorWidget(entity),
				}

				newComponents = append(newComponents, entityInspectorWidget)
			}

		}
	case "dashui:ecpDetails":
		{
			entityName := strings.ToLower(componentName)
			fmmEntities := manifest.GetFmmEntities()
			var entity *FmmEntity
			for _, e := range fmmEntities {
				if e.Name == entityName {
					entity = e
					break
				}
			}
			if entity == nil {
				log.Fatalf("Couldn't find an entity type named %s", entityName)
			}

			dashuiTemplates := manifest.GetDashuiTemplates()

			ecpDetails := &newComponent{
				Filename:   "ecpDetails.json",
				Type:       "dashui:template",
				Definition: getEcpDetails(entity),
			}

			newComponents = append(newComponents, ecpDetails)

			ecpDetailsList := &newComponent{
				Filename:   fmt.Sprintf("%sDetailsList.json", entity.Name),
				Type:       "dashui:template",
				Definition: getDashuiDetailsList(entity),
			}

			newComponents = append(newComponents, ecpDetailsList)

			ecpDetailsInspector := &newComponent{
				Filename:   "ecpDetailsInspector.json",
				Type:       "dashui:template",
				Definition: getEcpDetailsInspector(entity),
			}

			newComponents = append(newComponents, ecpDetailsInspector)

			templateName := fmt.Sprintf("%sInspectorWidget", entity.GetTypeName())

			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				entityInspectorWidget := &newComponent{
					Filename:   fmt.Sprintf("%sInspectorWidget.json", entity.Name),
					Type:       "dashui:template",
					Definition: getEcpInspectorWidget(entity),
				}

				newComponents = append(newComponents, entityInspectorWidget)
			}

		}
	}

	for _, newObject := range newComponents {
		addCompDefToManifest(cmd, manifest, newObject.Type, folderName)
		createComponentFile(newObject.Definition, folderName, newObject.Filename)
		objFilePath := fmt.Sprintf("%s/%s", folderName, newObject.Filename)
		statusMsg := fmt.Sprintf("Added %s file to your solution \n", objFilePath)
		output.PrintCmdStatus(cmd, statusMsg)
	}
}

func getResourceMap(cmd *cobra.Command, entityName string, manifest *Manifest) *FmmResourceMapping {
	entities := manifest.GetFmmEntities()
	var newResoureMapping *FmmResourceMapping
	var entity *FmmEntity
	for _, e := range entities {
		if e.Name == entityName {
			entity = e
			break
		}
	}
	if entity == nil {
		log.Fatalf("Couldn't find an entity type named %s", entityName)
	}

	namespace := entity.Namespace
	name := fmt.Sprintf("%s_%s_entity_mapping", manifest.Name, entityName)
	entityType := fmt.Sprintf("%s:%s", manifest.Name, entityName)
	scopeFilterFields := make([]string, 0)
	attributeMaps := make(FmmNameMappings, 0)
	displayName := fmt.Sprintf("Resource mapping configuration for the %q entity", entityType)
	fmmTypeDef := &FmmTypeDef{
		Namespace:   namespace,
		Kind:        "resourceMapping",
		Name:        name,
		DisplayName: displayName,
	}

	for _, requiredField := range entity.AttributeDefinitions.Required {
		scopeForField := fmt.Sprintf("%s.%s.%s", manifest.Name, entityName, requiredField)
		scopeFilterFields = append(scopeFilterFields, scopeForField)
	}

	for k, _ := range entity.AttributeDefinitions.Attributes {
		scopeForField := fmt.Sprintf("%s.%s.%s", manifest.Name, entityName, k)
		attributeMaps[k] = scopeForField
	}

	scopeFilter := fmt.Sprintf("containsAll(resourceAttributes, %s)", getStringfiedArray(scopeFilterFields))
	newResoureMapping = &FmmResourceMapping{
		FmmTypeDef:            fmmTypeDef,
		EntityType:            entityType,
		ScopeFilter:           scopeFilter,
		AttributeNameMappings: attributeMaps,
	}

	return newResoureMapping
}

func getNamespaceComponent(solutionName string) *FmmNamespace {
	namespaceDef := &FmmNamespace{
		Name: solutionName,
	}
	return namespaceDef
}

func getEntityComponent(entityName string, namespaceName string) *FmmEntity {
	emptyStringArray := make([]string, 0)
	emptyAttributeArray := make(map[string]*FmmAttributeTypeDef, 1)
	// emptyAssociationTypes := &FmmAssociationTypesTypeDef{}

	emptyAttributeArray["name"] = &FmmAttributeTypeDef{
		Type:        "string",
		Description: fmt.Sprintf("The name of the %s", entityName),
	}

	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    namespaceName,
		Version: 1,
	}

	lifecycleConfig := &FmmLifecycleConfigTypeDef{
		PurgeTtlInMinutes:     4200,
		RetentionTtlInMinutes: 1440,
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   *namespaceAssign,
		Kind:        "entity",
		Name:        entityName,
		DisplayName: entityName,
	}

	requiredArray := append(emptyStringArray, "name")
	attributesDefinition := &FmmAttributeDefinitionsTypeDef{
		Required:   requiredArray,
		Optimized:  emptyStringArray,
		Attributes: emptyAttributeArray,
	}

	entityComponentDef := &FmmEntity{
		FmmTypeDef:            fmmTypeDef,
		LifecyleConfiguration: lifecycleConfig,
		AttributeDefinitions:  attributesDefinition,
	}

	return entityComponentDef
}

func getEventComponent(eventName string, namespaceName string) *FmmEvent {
	emptyStringArray := make([]string, 0)
	emptyAttributeArray := make(map[string]*FmmAttributeTypeDef, 1)

	emptyAttributeArray["name"] = &FmmAttributeTypeDef{
		Type:        "string",
		Description: fmt.Sprintf("The name of the %s", eventName),
	}

	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    namespaceName,
		Version: 1,
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   *namespaceAssign,
		Kind:        "event",
		Name:        eventName,
		DisplayName: eventName,
	}

	requiredArray := append(emptyStringArray, "name")
	attributesDefinition := &FmmAttributeDefinitionsTypeDef{
		Required:   requiredArray,
		Optimized:  emptyStringArray,
		Attributes: emptyAttributeArray,
	}

	eventComponentDef := &FmmEvent{
		FmmTypeDef:           fmmTypeDef,
		AttributeDefinitions: attributesDefinition,
	}

	return eventComponentDef
}

func getMetricComponent(metricName string, contentType FmmMetricContentType, category FmmMetricCategory, metricType FmmMetricType, namespaceName string) *FmmMetric {
	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    namespaceName,
		Version: 1,
	}

	fmmTypeDef := &FmmTypeDef{
		Namespace:   *namespaceAssign,
		Kind:        "metric",
		Name:        metricName,
		DisplayName: metricName,
	}

	metricComponentDef := &FmmMetric{
		FmmTypeDef:             fmmTypeDef,
		Category:               category,
		ContentType:            contentType,
		AggregationTemporality: "delta",
		IsMonotonic:            false,
		Type:                   metricType,
		Unit:                   "{Count}",
	}

	return metricComponentDef
}
func getServiceComponent(serviceName string) *ServiceDef {
	serviceComponentDef := &ServiceDef{
		Name:  serviceName,
		Image: "dockerRegistryURL",
	}

	return serviceComponentDef
}

func getKnowledgeComponent(name string) *KnowledgeDef {
	jsonSchema := map[string]interface{}{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"title":                fmt.Sprintf("%s knowledge type", name),
		"description":          "",
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "this is a sample attribute",
			},
			"secret": map[string]interface{}{
				"type":        "string",
				"description": "this is a sample secret attribute",
			},
		},
		"required": []string{"name"},
	}
	idGen := &IdGenerationDef{
		EnforceGlobalUniqueness: true,
		GenerateRandomId:        true,
		IdGenerationMechanism:   "{{layer.id}}",
	}

	knowledgeComponent := &KnowledgeDef{
		Name:             name,
		AllowedLayers:    []string{"TENANT"},
		IdGeneration:     idGen,
		SecureProperties: []string{"$.secret"},
		JsonSchema:       jsonSchema,
	}

	return knowledgeComponent
}

func readComponentDef(componentDef *ComponentDef) []byte {
	filePath := componentDef.ObjectsFile
	componentDefFile := openFile(filePath)
	defer componentDefFile.Close()

	componentDefBytes, _ := io.ReadAll(componentDefFile)

	return componentDefBytes
}

func checkCreateSolutionNamespace(cmd *cobra.Command, manifest *Manifest, folderName string) {
	componentType := "fmm:namespace"
	namespaceName := manifest.Name
	fileName := namespaceName + ".json"
	objFilePath := fmt.Sprintf("%s/%s", folderName, fileName)

	componentDef := manifest.GetComponentDef(componentType)

	if componentDef.Type == "" {
		addCompDefToManifest(cmd, manifest, componentType, folderName)
	}

	if _, err := os.Stat(objFilePath); os.IsNotExist(err) {
		namespaceComp := getNamespaceComponent(namespaceName)
		createComponentFile(namespaceComp, folderName, fileName)
		statusMsg := fmt.Sprintf("Added %s file to your solution \n", objFilePath)
		output.PrintCmdStatus(cmd, statusMsg)
	}

}

func getStringfiedArray(array []string) string {
	initialFormat := fmt.Sprintf("%q", array)
	tokenized := strings.Split(initialFormat, " ")
	prettyArrayString := strings.Replace(strings.Join(tokenized, ", "), "\"", "'", -1)
	return prettyArrayString
}

func GetManifest() *Manifest {
	manifestFile := openFile("manifest.json")
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		log.Fatalf("Failed to read solution manifest: %v", err)
	}

	var manifest *Manifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		log.Fatalf("Failed to parse solution manifest: %v", err)
	}
	return manifest
}

func addCompDefToManifest(cmd *cobra.Command, manifest *Manifest, componentType string, folderName string) {
	componentDef := manifest.GetComponentDef(componentType)
	if componentDef.Type == "" {
		solutionDep := strings.Split(componentType, ":")[0]
		manifest.AppendDependency(solutionDep)

		componentDef := &ComponentDef{
			Type:       componentType,
			ObjectsDir: folderName,
		}

		manifest.Objects = append(manifest.Objects, *componentDef)
		createSolutionManifestFile(".", manifest)
		statusMsg := fmt.Sprintf("Added new %s definition to the solution manifest \n", componentType)
		output.PrintCmdStatus(cmd, statusMsg)
	}
}
