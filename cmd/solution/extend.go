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
		output.PrintCmdStatus("Can't generate a manifest objects from the manifest.json, make sure your manifest.json file is correct.")
		return
	}

	if cmd.Flags().Changed("add-knowledge") {
		componentName, _ := cmd.Flags().GetString("add-knowledge")
		output.PrintCmdStatus(fmt.Sprintf("Adding %s knowledge component to %s's solution package folder structure... \n", componentName, manifest.Name))
		folderName := "knowledge"
		fileName := fmt.Sprintf("%s.json", componentName)
		output.PrintCmdStatus(fmt.Sprintf("Creating the %s file\n", fileName))
		manifest.Types = append(manifest.Types, fmt.Sprintf("%s/%s", folderName, fileName))

		knowledgeComp := getKnowledgeComponent(componentName)
		createComponentFile(knowledgeComp, folderName, fileName)
	}

	if cmd.Flags().Changed("add-service") {
		componentName, _ := cmd.Flags().GetString("add-service")
		output.PrintCmdStatus(fmt.Sprintf("Adding %s service component to %s's solution package folder structure... \n", componentName, manifest.Name))
		folderName := "services"
		fileName := fmt.Sprintf("%s.json", componentName)
		output.PrintCmdStatus(fmt.Sprintf("Creating the %s file\n", fileName))

		appendDependency("zodiac", manifest)

		serviceComponentDef := &ComponentDef{
			Type:        "zodiac:function",
			ObjectsFile: fmt.Sprintf("services/%s", fileName),
		}

		manifest.Objects = append(manifest.Objects, *serviceComponentDef)

		serviceComp := getServiceComponent(componentName)
		createComponentFile(serviceComp, folderName, fileName)
	}

	if cmd.Flags().Changed("add-entity") {
		folderName := "model"
		var filePath string
		// var entitiesDefFile *os.File

		namespaceComponentDef := getComponentDef("fmm:namespace", manifest)
		if namespaceComponentDef.Type == "" {
			output.PrintCmdStatus(fmt.Sprintf("Adding %s namespace component to %s's solution package folder structure... \n", manifest.Name, manifest.Name))
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
			output.PrintCmdStatus("Added model/namespace.json file to your solution \n")
		}

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
			output.PrintCmdStatus("Added model/entities.json file to your solution \n")
		} else {
			filePath = entityComponentDef.ObjectsFile
			entitiesDefFile := openFile(filePath)
			defer entitiesDefFile.Close()

			entitiesBytes, _ := io.ReadAll(entitiesDefFile)

			err = json.Unmarshal(entitiesBytes, &entitiesArray)
			if err != nil {
				log.Errorf("Can't generate an array of entity definition objects from the %s file, make sure your %s file is correct.", filePath, filePath)
				return
			}
		}

		componentName, _ := cmd.Flags().GetString("add-entity")
		entityComp := getEntityComponent(componentName, manifest.Name)
		entitiesArray = append(entitiesArray, entityComp)
		splitPath := strings.Split(entityComponentDef.ObjectsFile, "/")
		fileName := splitPath[len(splitPath)-1]
		createComponentFile(entitiesArray, folderName, fileName)
		output.PrintCmdStatus("Updating the manifest.json \n")
		createSolutionManifestFile(".", manifest)
	}

	if cmd.Flags().Changed("add-metric") {
		folderName := "model"
		var filePath string
		// var entitiesDefFile *os.File

		namespaceComponentDef := getComponentDef("fmm:namespace", manifest)
		if namespaceComponentDef.Type == "" {
			output.PrintCmdStatus(fmt.Sprintf("Adding %s namespace component to %s's solution package folder structure... \n", manifest.Name, manifest.Name))
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
			output.PrintCmdStatus("Added model/namespace.json file to your solution \n")
		}

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
			output.PrintCmdStatus("Added model/metrics.json file to your solution \n")
		} else {
			filePath = metricComponentDef.ObjectsFile
			metricsDefFile := openFile(filePath)
			defer metricsDefFile.Close()

			metricsBytes, _ := io.ReadAll(metricsDefFile)

			err = json.Unmarshal(metricsBytes, &metricsArray)
			if err != nil {
				log.Errorf("Can't generate an array of entity definition objects from the %s file, make sure your %s file is correct.", filePath, filePath)
				return
			}
		}

		componentName, _ := cmd.Flags().GetString("add-metric")
		newMetricComp := getMetricComponent(componentName, ContentType_Sum, Category_Sum, Type_Long, manifest.Name)
		metricsArray = append(metricsArray, newMetricComp)
		splitPath := strings.Split(metricComponentDef.ObjectsFile, "/")
		fileName := splitPath[len(splitPath)-1]
		createComponentFile(metricsArray, folderName, fileName)
		output.PrintCmdStatus("Updating the manifest.json \n")
		createSolutionManifestFile(".", manifest)
	}
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
	emptyAssociationTypes := &FmmAssociationTypesTypeDef{}

	emptyAttributeArray["name"] = &FmmAttributeTypeDef{
		Type:        "string",
		Description: fmt.Sprintf("The name of the %s", entityName),
	}

	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    solutionName,
		Version: "1.0",
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
		FmmTypeDef:           fmmTypeDef,
		MetricTypes:          emptyStringArray,
		EventTypes:           emptyStringArray,
		AttributeDefinitions: attributesDefinition,
		AssociationTypes:     emptyAssociationTypes,
	}

	return entityComponentDef
}

func getMetricComponent(metricName string, contentType FmmMetricContentType, category FmmMetricCategory, metricType FmmMetricType, solutionName string) *FmmMetric {
	namespaceAssign := &FmmNamespaceAssignTypeDef{
		Name:    solutionName,
		Version: "1.0",
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
