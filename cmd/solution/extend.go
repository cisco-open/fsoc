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

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionExtendCmd = &cobra.Command{
	Use:              "extend",
	Args:             cobra.ExactArgs(0),
	Short:            "Extends your solution package by adding new components",
	Long:             `This command allows you to easily add new components to your solution package.`,
	Example:          `  fsoc solution extend --add-knowledge=<knowldgetypename>`,
	Run:              extendSolution,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
	TraverseChildren: true,
}

// Planned options:
// --add-meltworkflow - Flag to add a new melt workflow component to the current solution package

func getSolutionExtendCmd() *cobra.Command {
	solutionExtendCmd.Flags().
		String("add-service", "", "Add a new service component definition to this solution")
	solutionExtendCmd.Flags().
		String("add-knowledge", "", "Add a new knowledge type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-entity", "", "Add a new entity type definition to this solution")
	solutionExtendCmd.Flags().
		String("add-associationDeclarations", "", "Add all associationDeclaration type definitions for a given entity within this solution")
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
	solutionExtendCmd.Flags().
		Bool("add-ecpHome", false, "Add a template extension definition to build the ecpHome experience for this solution")

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

	if cmd.Flags().Changed("add-associationDeclarations") {
		componentName, _ := cmd.Flags().GetString("add-associationDeclarations")
		componentName = strings.ToLower(componentName)
		folderName := "objects/model/association-declarations"
		addNewComponent(cmd, manifest, folderName, componentName, "fmm:associationDeclaration")
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

	if cmd.Flags().Changed("add-ecpHome") {
		folderName := "objects/dashui/templatePropsExtensions"

		addNewComponent(cmd, manifest, folderName, "ecpHome", "dashui:ecpHome")
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

	if strings.Contains(componentType, "fmm") {
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
	case "fmm:associationDeclaration":
		{
			entityName := strings.ToLower(componentName)
			entity := &newComponent{
				Filename:   entityName + "-associationDeclarations.json",
				Type:       componentType,
				Definition: getAssociationDeclarations(entityName, manifest),
			}

			newComponents = append(newComponents, entity)

		}
	case "fmm:metric":
		{
			metric := &newComponent{
				Filename:   componentName + ".json",
				Type:       componentType,
				Definition: getMetricComponent(componentName, ContentType_Gauge, Type_Long, manifest.Name),
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
			entity := findEntity(entityName, manifest)
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

			templateName = "dashui:name"
			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				ecpName := &newComponent{
					Filename:   "ecpName.json",
					Type:       "dashui:template",
					Definition: getEcpName(entity),
				}

				newComponents = append(newComponents, ecpName)
			}
		}
	case "dashui:ecpDetails":
		{
			entityName := strings.ToLower(componentName)
			entity := findEntity(entityName, manifest)

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
				Definition: getDashuiDetailsList(entity, manifest),
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

			templateName = "dashui:name"
			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				ecpName := &newComponent{
					Filename:   "ecpName.json",
					Type:       "dashui:template",
					Definition: getEcpName(entity),
				}

				newComponents = append(newComponents, ecpName)
			}

		}
	case "dashui:ecpHome":
		{
			ecpHome := &newComponent{
				Filename:   fmt.Sprintf("%s.json", componentName),
				Type:       "dashui:templatePropsExtension",
				Definition: getEcpHome(manifest),
			}

			newComponents = append(newComponents, ecpHome)

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
	componentDefs := manifest.GetComponentDefs(componentType)
	if len(componentDefs) > 0 {
		for _, componentDef := range componentDefs {
			if componentDef.ObjectsDir == folderName {
				return
			}
		}
	}
	solutionDep := strings.Split(componentType, ":")[0]
	manifest.AppendDependency(solutionDep)

	extComponentDef := &ComponentDef{
		Type:       componentType,
		ObjectsDir: folderName,
	}

	manifest.Objects = append(manifest.Objects, *extComponentDef)
	createSolutionManifestFile(".", manifest)
	statusMsg := fmt.Sprintf("Added new %s definition to the solution manifest \n", componentType)
	output.PrintCmdStatus(cmd, statusMsg)
}
