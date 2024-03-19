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
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionExtendCmd = &cobra.Command{
	Use:              "extend [flags]",
	Args:             cobra.NoArgs,
	Short:            "Extends your solution by adding new components",
	Long:             `This command allows you to easily add new components to your solution.`,
	Example:          `  fsoc solution extend --add-knowledge=dataCollectorConfiguration --add-service=ingestor`,
	Run:              extendSolution,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""}, // this command does not require a valid context
	TraverseChildren: true,
}

// Planned options:
// --add-meltworkflow - Flag to add a new melt workflow component to the current solution package

func getSolutionExtendCmd() *cobra.Command {
	// component(s) to be added to the solution
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

	// file format override flags (mutually exclusive)
	solutionExtendCmd.Flags().
		Bool("json", false, "Use JSON format for the component file, even if the manifest is in YAML.")
	solutionExtendCmd.Flags().
		Bool("yaml", false, "Use YAML format for the component file, even if the manifest is in JSON.")
	solutionExtendCmd.MarkFlagsMutuallyExclusive("json", "yaml")

	return solutionExtendCmd

}

func extendSolution(cmd *cobra.Command, args []string) {
	manifest, err := GetManifest(".")
	if err != nil {
		log.Fatalf("Failed to read manifest file: %v", err)
	}

	if cmd.Flags().Changed("add-knowledge") {
		componentName, _ := cmd.Flags().GetString("add-knowledge")
		componentName = strings.ToLower(componentName)
		if strings.Contains(componentName, ":") {
			log.Fatalf(`":" is a disallowed character. Note that solution name is not required for the add-knowledge flag`)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Adding %s knowledge type.\n", componentName))
		addNewKnowledgeComponent(cmd, manifest, getKnowledgeComponent(componentName))
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
	var namespaceName string
	if strings.Contains(componentType, "fmm") {
		checkCreateSolutionNamespace(cmd, manifest, "objects/model/namespaces")
		namespaceName = manifest.GetNamespaceName()
	}

	switch componentType {
	case "zodiac:function":
		{
			component := &newComponent{
				Filename:   componentFileName(cmd, manifest, componentName),
				Type:       componentType,
				Definition: getServiceComponent(componentName),
			}

			newComponents = append(newComponents, component)
		}
	case "fmm:entity":
		{
			entity := &newComponent{
				Filename:   componentFileName(cmd, manifest, componentName),
				Type:       componentType,
				Definition: getEntityComponent(componentName, namespaceName),
			}

			newComponents = append(newComponents, entity)
		}
	case "fmm:resourceMapping":
		{
			entityName, _ := cmd.Flags().GetString("add-resourceMapping")
			entityName = strings.ToLower(entityName)
			entity := &newComponent{
				Filename:   componentFileName(cmd, manifest, componentName+"-resourceMapping"),
				Type:       componentType,
				Definition: getResourceMap(nil, entityName, manifest),
			}

			newComponents = append(newComponents, entity)

		}
	case "fmm:associationDeclaration":
		{
			entityName := strings.ToLower(componentName)
			entity := &newComponent{
				Filename:   componentFileName(cmd, manifest, entityName+"-associationDeclarations"),
				Type:       componentType,
				Definition: getAssociationDeclarations(entityName, manifest),
			}

			newComponents = append(newComponents, entity)

		}
	case "fmm:metric":
		{
			metric := &newComponent{
				Filename:   componentFileName(cmd, manifest, componentName),
				Type:       componentType,
				Definition: getMetricComponent(componentName, ContentType_Gauge, Type_Long, namespaceName),
			}

			newComponents = append(newComponents, metric)
		}
	case "fmm:event":
		{
			event := &newComponent{
				Filename:   componentFileName(cmd, manifest, componentName),
				Type:       componentType,
				Definition: getEventComponent(componentName, namespaceName),
			}

			newComponents = append(newComponents, event)
		}
	case "dashui:ecpList":
		{
			entityName := strings.ToLower(componentName)
			entity := findEntity(entityName, manifest)
			dashuiTemplates := manifest.GetDashuiTemplates()

			ecpList := &newComponent{
				Filename:   componentFileName(cmd, manifest, "ecpList"),
				Type:       "dashui:template",
				Definition: getEcpList(entity),
			}

			newComponents = append(newComponents, ecpList)

			entityGridTable := &newComponent{
				Filename:   componentFileName(cmd, manifest, entity.Name+"GridTable"),
				Type:       "dashui:template",
				Definition: getDashuiGridTable(entity),
			}

			newComponents = append(newComponents, entityGridTable)

			ecpRelationshipMap := &newComponent{
				Filename:   componentFileName(cmd, manifest, "ecpRelationshipMap"),
				Type:       "dashui:template",
				Definition: getRelationshipMap(entity),
			}

			newComponents = append(newComponents, ecpRelationshipMap)

			ecpListInspector := &newComponent{
				Filename:   componentFileName(cmd, manifest, "ecpListInspector"),
				Type:       "dashui:template",
				Definition: getEcpListInspector(entity),
			}

			newComponents = append(newComponents, ecpListInspector)

			templateName := fmt.Sprintf("%sInspectorWidget", entity.GetTypeName())

			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				entityInspectorWidget := &newComponent{
					Filename:   componentFileName(cmd, manifest, entity.Name+"InspectorWidget"),
					Type:       "dashui:template",
					Definition: getEcpInspectorWidget(entity),
				}

				newComponents = append(newComponents, entityInspectorWidget)
			}

			templateName = "dashui:name"
			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				ecpName := &newComponent{
					Filename:   componentFileName(cmd, manifest, "ecpName"),
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
				Filename:   componentFileName(cmd, manifest, "ecpDetails"),
				Type:       "dashui:template",
				Definition: getEcpDetails(entity),
			}

			newComponents = append(newComponents, ecpDetails)

			ecpDetailsList := &newComponent{
				Filename:   componentFileName(cmd, manifest, entity.Name+"DetailsList.json"),
				Type:       "dashui:template",
				Definition: getDashuiDetailsList(entity, manifest),
			}

			newComponents = append(newComponents, ecpDetailsList)

			ecpDetailsInspector := &newComponent{
				Filename:   componentFileName(cmd, manifest, "ecpDetailsInspector"),
				Type:       "dashui:template",
				Definition: getEcpDetailsInspector(entity),
			}

			newComponents = append(newComponents, ecpDetailsInspector)

			templateName := fmt.Sprintf("%sInspectorWidget", entity.GetTypeName())

			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				entityInspectorWidget := &newComponent{
					Filename:   componentFileName(cmd, manifest, entity.Name+"InspectorWidget"),
					Type:       "dashui:template",
					Definition: getEcpInspectorWidget(entity),
				}

				newComponents = append(newComponents, entityInspectorWidget)
			}

			templateName = "dashui:name"
			if !hasDashuiTemplate(entity, dashuiTemplates, templateName) {
				ecpName := &newComponent{
					Filename:   componentFileName(cmd, manifest, "ecpName"),
					Type:       "dashui:template",
					Definition: getEcpName(entity),
				}

				newComponents = append(newComponents, ecpName)
			}

		}
	case "dashui:ecpHome":
		{
			ecpHome := &newComponent{
				Filename:   componentFileName(cmd, manifest, componentName),
				Type:       "dashui:templatePropsExtension",
				Definition: getEcpHome(manifest),
			}

			newComponents = append(newComponents, ecpHome)

		}
	}

	for _, newObject := range newComponents {
		checkStructTags(reflect.TypeOf(newObject.Definition))

		addCompDefToManifest(cmd, manifest, newObject.Type, folderName)
		createComponentFile(newObject.Definition, folderName, newObject.Filename)
		objFilePath := filepath.Join(folderName, newObject.Filename)
		statusMsg := fmt.Sprintf("Added file %s to your solution\n", objFilePath)
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

	knowledgeComponent := &KnowledgeDef{
		Name:                  name,
		AllowedLayers:         []string{"TENANT"},
		IdentifyingProperties: []string{"/name"},
		SecureProperties:      []string{"$.secret"},
		JsonSchema:            jsonSchema,
	}

	return knowledgeComponent
}

func getStringfiedArray(array []string) string {
	initialFormat := fmt.Sprintf("%q", array)
	tokenized := strings.Split(initialFormat, " ")
	prettyArrayString := strings.Replace(strings.Join(tokenized, ", "), "\"", "'", -1)
	return prettyArrayString
}

func GetManifest(path string) (*Manifest, error) {
	return getSolutionManifest(path)
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

func addNewKnowledgeComponent(cmd *cobra.Command, manifest *Manifest, obj *KnowledgeDef) {
	// finalize file format
	fileFormat := manifest.ManifestFormat
	if isJSON, _ := cmd.Flags().GetBool("json"); isJSON {
		fileFormat = FileFormatJSON
	} else if isYAML, _ := cmd.Flags().GetBool("yaml"); isYAML {
		fileFormat = FileFormatYAML
	}

	// construct file path for the type file
	folderName := "types"
	fileName := fmt.Sprintf("%s.%s", obj.Name, fileFormat)
	filePath := filepath.Join(folderName, fileName)

	// fail if the file already exists, prevent overwriting existing type
	if _, err := os.Stat(filePath); err == nil {
		log.Fatalf("Type file %s already exists in the solution. Please use a different type name.", filePath)
	}

	// add the file if not already in the list
	found := false
	for _, t := range manifest.Types {
		if t == obj.Name {
			found = true
		}
	}
	if !found {
		manifest.Types = append(manifest.Types, filePath)

	}

	// add type to manifest & create type file
	createComponentFile(obj, folderName, fileName)
	createSolutionManifestFile(".", manifest)

	statusMsg := fmt.Sprintf("Added knowledge type %s to your solution in %s\n", obj.Name, fileName)
	output.PrintCmdStatus(cmd, statusMsg)
}

// componentFileName returns a file name for a component with a file extension reflecting the format.
// The manfest and the optional cobra command are provided as means to determine the file format.
// If the command is provided and a format is specified with a flag, that takes precedence
// over the manifest format; otherwise the component is created in the same format as the manifest.
// If neither command nor manifest is provided, the default format is JSON.
// This function cannot fail.
func componentFileName(cmd *cobra.Command, manifest *Manifest, componentName string) string {
	// first, assume default format is JSON
	format := FileFormatJSON

	// then, match manifest's format if the manifest is provided
	if manifest != nil {
		format = manifest.ManifestFormat
	}

	// and then, use command's format flag, if specified
	if cmd != nil {
		if isJSON, _ := cmd.Flags().GetBool("json"); isJSON {
			format = FileFormatJSON
		} else if isYAML, _ := cmd.Flags().GetBool("yaml"); isYAML {
			format = FileFormatYAML
		}
	}

	// compose the file name
	return fmt.Sprintf("%s.%s", componentName, format)
}

// checkStructTags checks if the struct tags for json and yaml are identical,
// and logs a warning if they are not. This helps to prevent issues with
// serialization and deserialization of the component definition in JSON vs. YAML format,
// as we want to ensure that field names are consistent across both formats. There doesn't
// seem to be a good way to avoid the duplication, so this check help make sure that
// we don't accidentally introduce inconsistencies in the field names across the formats.
// Note that this function checks the structure definition, not the actual data; failures
// here indicate a bug in the code, not a problem with the data.
func checkStructTags(t reflect.Type) {
	switch t.Kind() {
	case reflect.Ptr:
		checkStructTags(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			// assert the two tags are the same (or neither exists)
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			yamlTag := field.Tag.Get("yaml")
			if jsonTag != yamlTag {
				log.Warnf("(possible bug) Field %q in struct %q has different json and yaml tags: %q vs. %q", field.Name, t.Name(), jsonTag, yamlTag)
			}

			// nest into struct and *struct fields (incl. anonymous structs)
			if field.Type.Kind() == reflect.Struct {
				checkStructTags(field.Type)
			} else if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
				checkStructTags(field.Type.Elem())
			}
		}
	default:
		log.Warnf("(possible bug) Expected a struct or struct pointer but got %q for type %q", t.Kind(), t.Name())
	}
}
