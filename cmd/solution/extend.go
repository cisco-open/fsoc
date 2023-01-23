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
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

var solutionExtendCmd = &cobra.Command{
	Use:   "extend",
	Short: "Extends your solution package by adding new components",
	Long: `This command allows you to easily add new components to your solution package.

    Command: fsoc solution extend --add-knowledge=<knowldgetypename>

	Options:
    --add-service - Flag to add a new service component to the current solution package
	--add-knowledge - Flag to add a new knowledge type component to the current solution package  
    --add-meltworkflow - Flag to add a new melt workflow component to the current solution package
    --add-dash-ui - Flag to add a new user experience component to the current solution package
	--add-metric - Flag to add a new metric type component to the current solution package

	Usage:
	fsoc solution extend --add-knowledge=<knowledgeTypeName>`,

	Run:              addSolutionComponent,
	TraverseChildren: true,
}

func getSolutionExtendCmd() *cobra.Command {
	solutionExtendCmd.Flags().
		String("add-service", "", "Add as a new service component definition to this solution")
	solutionExtendCmd.Flags().
		String("add-knowledge", "", "Add as a new knowledge type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-entity", "", "Add as a new entity type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-metric", "", "Add as a new metric type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-resourceMapping", "", "Add as a new metric type definition to this solution")

	return solutionExtendCmd

}

func addSolutionComponent(cmd *cobra.Command, args []string) {

	// manifestFile, err := os.Open("manifest.json")
	// if err != nil {
	// 	output.PrintCmdStatus("Can't find the manifest.json, run this command from your solution package root folder")
	// 	return
	// }
	var err error

	manifestFile := openFile("manifest.json")
	defer manifestFile.Close()

	manifestBytes, _ := io.ReadAll(manifestFile)
	var manifest *Manifest

	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		log.Fatalf("Failed to parse solution manifest: %v", err)
	}

	if cmd.Flags().Changed("add-knowledge") {
		componentName, _ := cmd.Flags().GetString("add-knowledge")
		componentName = strings.ToLower(componentName)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %s knowledge component to %s's solution package folder structure... \n", componentName, manifest.Name))
		folderName := "knowledge"
		fileName := fmt.Sprintf("%s.json", componentName)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Creating the %s file\n", fileName))
		manifest.Types = append(manifest.Types, fmt.Sprintf("%s/%s", folderName, fileName))
		bytes, _ := json.Marshal(manifest)
		err := os.WriteFile("./manifest.json", bytes, 0644)
		if err != nil {
			log.Fatalf("Failed to update manifest.json file to reflect new knowledge type: %v", err)
		}

		knowledgeComp := getKnowledgeComponent(componentName)
		createComponentFile(knowledgeComp, folderName, fileName)
	}

	if cmd.Flags().Changed("add-service") {
		componentName, _ := cmd.Flags().GetString("add-service")
		componentName = strings.ToLower(componentName)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %s service component to %s's solution package folder structure... \n", componentName, manifest.Name))
		folderName := "services"
		fileName := fmt.Sprintf("%s.json", componentName)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Creating the %s file\n", fileName))

		appendDependency("zodiac", manifest)

		serviceComponentDef := &ComponentDef{
			Type:        "zodiac:function",
			ObjectsFile: fmt.Sprintf("services/%s", fileName),
		}

		manifest.Objects = append(manifest.Objects, *serviceComponentDef)
		bytes, _ := json.Marshal(manifest)
		err := os.WriteFile("./manifest.json", bytes, 0644)
		if err != nil {
			log.Fatalf("Failed to update manifest.json file to reflect new service component: %v", err)
		}

		serviceComp := getServiceComponent(componentName)
		createComponentFile(serviceComp, folderName, fileName)
	}

	if cmd.Flags().Changed("add-entity") {
		componentName, _ := cmd.Flags().GetString("add-entity")
		componentName = strings.ToLower(componentName)
		folderName := "model"

		addFmmEntity(cmd, manifest, folderName, componentName)
	}

	if cmd.Flags().Changed("add-resourceMapping") {
		entityName, _ := cmd.Flags().GetString("add-resourceMapping")
		entityName = strings.ToLower(entityName)
		folderName := "model"

		addFmmResourceMapping(cmd, manifest, folderName, entityName)
	}

	if cmd.Flags().Changed("add-metric") {
		componentName, _ := cmd.Flags().GetString("add-metric")
		componentName = strings.ToLower(componentName)
		folderName := "model"

		addFmmMetric(cmd, manifest, folderName, componentName)
	}
}

func addFmmMetric(cmd *cobra.Command, manifest *Manifest, folderName, componentName string) {
	var filePath string
	checkCreateSolutionNamespace(cmd, manifest, folderName)

	metricComponentDef := getComponentDef("fmm:metric", manifest)
	var metricsArray []*FmmMetric
	if metricComponentDef.Type == "" {
		metricComponentDef = &ComponentDef{
			Type:        "fmm:metric",
			ObjectsFile: "model/metrics.json",
		}
		manifest.Objects = append(manifest.Objects, *metricComponentDef)
		filePath = metricComponentDef.ObjectsFile
		createFile(filePath)
		output.PrintCmdStatus(cmd, "Added model/metrics.json file to your solution \n")
	} else {
		// filePath = metricComponentDef.ObjectsFile
		// metricsDefFile := openFile(filePath)
		// defer metricsDefFile.Close()

		// metricsBytes, _ := io.ReadAll(metricsDefFile)
		metricsBytes := readComponentDef(metricComponentDef)
		err := json.Unmarshal(metricsBytes, &metricsArray)
		if err != nil {
			log.Errorf("Can't generate an array of entity definition objects from the %s file, make sure your %s file is correct.", filePath, filePath)
			return
		}
	}

	newMetricComp := getMetricComponent(componentName, ContentType_Sum, Category_Sum, Type_Long, manifest.Name)
	metricsArray = append(metricsArray, newMetricComp)
	splitPath := strings.Split(metricComponentDef.ObjectsFile, "/")
	fileName := splitPath[len(splitPath)-1]
	createComponentFile(metricsArray, folderName, fileName)
	output.PrintCmdStatus(cmd, "Updating the solution manifest\n")
	createSolutionManifestFile(".", manifest)

}

func addFmmResourceMapping(cmd *cobra.Command, manifest *Manifest, folderName, entityName string) {
	var filePath string
	checkCreateSolutionNamespace(cmd, manifest, folderName)
	resourceMapComponentDef := getComponentDef("fmm:resourceMapping", manifest)
	var resourceMappingArray []*FmmResourceMapping
	if resourceMapComponentDef.Type == "" {
		resourceMapComponentDef = &ComponentDef{
			Type:        "fmm:resourceMapping",
			ObjectsFile: "model/resourceMappings.json",
		}
		manifest.Objects = append(manifest.Objects, *resourceMapComponentDef)
		filePath = resourceMapComponentDef.ObjectsFile
		createFile(filePath)
		output.PrintCmdStatus(cmd, "Added model/resourceMappings.json file to your solution \n")
	} else {
		componentDefBytes := readComponentDef(resourceMapComponentDef)
		err := json.Unmarshal(componentDefBytes, &resourceMappingArray)
		if err != nil {
			log.Errorf("Can't generate an array of resource mapping definition objects from the %s file, make sure your %s file is correct.", filePath, filePath)
			return
		}
	}

	resourceMapComp := getResourceMap(cmd, entityName, manifest)
	resourceMappingArray = append(resourceMappingArray, resourceMapComp)
	splitPath := strings.Split(resourceMapComponentDef.ObjectsFile, "/")
	fileName := splitPath[len(splitPath)-1]
	createComponentFile(resourceMappingArray, folderName, fileName)
	output.PrintCmdStatus(cmd, "Updating the solution manifest\n")
	createSolutionManifestFile(".", manifest)
}

func addFmmEntity(cmd *cobra.Command, manifest *Manifest, folderName, componentName string) {
	var filePath string
	checkCreateSolutionNamespace(cmd, manifest, folderName)

	entityComponentDef := getComponentDef("fmm:entity", manifest)
	var entitiesArray []*FmmEntity
	if entityComponentDef.Type == "" {
		entityComponentDef = &ComponentDef{
			Type:        "fmm:entity",
			ObjectsFile: "model/entities.json",
		}
		manifest.Objects = append(manifest.Objects, *entityComponentDef)
		filePath = entityComponentDef.ObjectsFile
		createFile(filePath)
		output.PrintCmdStatus(cmd, "Added model/entities.json file to your solution \n")
	} else {
		// filePath = entityComponentDef.ObjectsFile
		// entitiesDefFile := openFile(filePath)
		// defer entitiesDefFile.Close()

		// entitiesBytes, _ := io.ReadAll(entitiesDefFile)

		entitiesBytes := readComponentDef(entityComponentDef)

		err := json.Unmarshal(entitiesBytes, &entitiesArray)
		if err != nil {
			log.Errorf("Can't generate an array of entity definition objects from the %s file, make sure your %s file is correct.", filePath, filePath)
			return
		}
	}

	entityComp := getEntityComponent(componentName, manifest.Name)
	entitiesArray = append(entitiesArray, entityComp)
	splitPath := strings.Split(entityComponentDef.ObjectsFile, "/")
	fileName := splitPath[len(splitPath)-1]
	createComponentFile(entitiesArray, folderName, fileName)
	output.PrintCmdStatus(cmd, "Updating the solution manifest\n")
	createSolutionManifestFile(".", manifest)
}

func getResourceMap(cmd *cobra.Command, entityName string, manifest *Manifest) *FmmResourceMapping {
	hasEntity := false
	entityComponentDef := getComponentDef("fmm:entity", manifest)
	var entitiesArray []*FmmEntity
	var newResoureMapping *FmmResourceMapping
	entityBytes := readComponentDef(entityComponentDef)
	err := json.Unmarshal(entityBytes, &entitiesArray)
	if err != nil {
		log.Errorf("Can't generate an array of %s type definition objects from the %s file.", entityComponentDef.Type, entityComponentDef.ObjectsFile)
		return nil
	}
	for _, entity := range entitiesArray {
		if entity.Name == entityName {
			hasEntity = true
			namespace := entity.Namespace
			name := fmt.Sprintf("%s_%s_entity_mapping", manifest.Name, entityName)
			entityType := fmt.Sprintf("%s:%s", manifest.Name, entityName)
			scopeFilterFields := make([]string, 0)
			attributeMaps := make(FmmNameMappings, 0)
			displayName := fmt.Sprintf("Resource mapping configuration for the %s entity", entityType)
			fmmTypeDef := &FmmTypeDef{
				Namespace:   namespace,
				Kind:        "resourceMapping",
				Name:        name,
				DisplayName: displayName,
			}
			for _, requiredField := range entity.AttributeDefinitions.Required {
				scopeForField := fmt.Sprintf("%s.%s.%s", manifest.Name, entityName, requiredField)
				scopeFilterFields = append(scopeFilterFields, scopeForField)
				attributeMaps[requiredField] = scopeForField
			}

			scopeFilter := fmt.Sprintf("containsAll(resourceAttributes, %s)", getStringfiedArray(scopeFilterFields))
			newResoureMapping = &FmmResourceMapping{
				FmmTypeDef:            fmmTypeDef,
				EntityType:            entityType,
				ScopeFilter:           scopeFilter,
				AttributeNameMappings: attributeMaps,
			}
		}
	}

	if !hasEntity {
		message := fmt.Sprintf("A fmm:resourceMapping was not created! Could not find an fmm:entity named %s in this solution", entityName)
		output.PrintCmdStatus(cmd, message)
	}
	return newResoureMapping
}

func readComponentDef(componentDef *ComponentDef) []byte {
	filePath := componentDef.ObjectsFile
	componentDefFile := openFile(filePath)
	defer componentDefFile.Close()

	componentDefBytes, _ := io.ReadAll(componentDefFile)

	return componentDefBytes
}

func appendDependency(solutionName string, manifest *Manifest) {
	hasDependency := checkDependencyExists(solutionName, manifest)
	if !hasDependency {
		manifest.Dependencies = append(manifest.Dependencies, solutionName)
	}

}

func checkDependencyExists(solutionName string, manifest *Manifest) bool {
	hasDependency := false
	for _, deps := range manifest.Dependencies {
		if deps == solutionName {
			hasDependency = true
		}
	}
	return hasDependency
}

func getComponentDef(typeName string, manifest *Manifest) *ComponentDef {
	var componentDef ComponentDef
	for _, compDefs := range manifest.Objects {
		if compDefs.Type == typeName {
			componentDef = compDefs
		}
	}
	return &componentDef
}

func getNamespaceComponent(solutionName string) *FmmNamespace {
	namespaceDef := &FmmNamespace{
		Name: solutionName,
	}
	return namespaceDef
}

func getEntityComponent(entityName string, solutionName string) *FmmEntity {
	emptyStringArray := make([]string, 0)
	emptyAttributeArray := make(map[string]*FmmAttributeTypeDef, 1)
	// emptyAssociationTypes := &FmmAssociationTypesTypeDef{}

	emptyAttributeArray["name"] = &FmmAttributeTypeDef{
		Type:        "string",
		Description: fmt.Sprintf("The name of the %s", entityName),
	}

	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    solutionName,
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

func getMetricComponent(metricName string, contentType FmmMetricContentType, category FmmMetricCategory, metricType FmmMetricType, solutionName string) *FmmMetric {
	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    solutionName,
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

func checkCreateSolutionNamespace(cmd *cobra.Command, manifest *Manifest, folderName string) {
	namespaceComponentDef := getComponentDef("fmm:namespace", manifest)
	if namespaceComponentDef.Type == "" {
		output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %s namespace component to %s's solution package folder structure... \n", manifest.Name, manifest.Name))
		appendFolder(folderName)
		appendDependency("fmm", manifest)
		namespaceCompDef := &ComponentDef{
			Type:        "fmm:namespace",
			ObjectsFile: "model/namespace.json",
		}
		manifest.Objects = append(manifest.Objects, *namespaceCompDef)
		namespaceComp := getNamespaceComponent(manifest.Name)
		createComponentFile(namespaceComp, folderName, "namespace.json")
		createSolutionManifestFile(".", manifest)
		output.PrintCmdStatus(cmd, "Added model/namespace.json file to your solution \n")
	}
}

func getStringfiedArray(array []string) string {
	initialFormat := fmt.Sprintf("%q", array)
	tokenized := strings.Split(initialFormat, " ")
	prettyArrayString := strings.Replace(strings.Join(tokenized, ", "), "\"", "'", -1)
	return prettyArrayString
}
