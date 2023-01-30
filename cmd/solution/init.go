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
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Creates a new solution package",
	Long: `This command list all the solutions that are deployed in the current tenant specified in the profile.

    Command: fsoc solution init --name=<solutionName> [--include-service] [--include-knowledge]

    Parameters:
    solutionName - Name of the solution

	Options:
    include-service - Flag to include sample service component
	include-metric - Flag to include sample metric type component
    include-knowledge - Flag to include sample knowledge type component  
    include-meltworkflow - Flag to include sample melt workflow
    include-dash-ui - Flag to include sample dash-ui template

	Usage:
	fsoc solution init --name=<solutionName> [--include-service] [--include-knowldege]`,

	Run:              generateSolutionPackage,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
	TraverseChildren: true,
}

func getInitSolutionCmd() *cobra.Command {
	solutionInitCmd.Flags().
		String("name", "", "The name of the new solution")
	solutionInitCmd.Flags().
		Bool("include-service", true, "Add a service component definition to this solution")
	solutionInitCmd.Flags().
		Bool("include-knowledge", true, "Add a knowledge type definition to this solution")

	return solutionInitCmd

}

func generateSolutionPackage(cmd *cobra.Command, args []string) {

	solutionName, _ := cmd.Flags().GetString("name")
	solutionName = strings.ToLower(solutionName)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Preparing the %s solution package folder structure... \n", solutionName))

	if err := os.Mkdir(solutionName, os.ModePerm); err != nil {
		log.Errorf("Solution init failed - %v", err.Error())
	}

	manifest := createInitialSolutionManifest(solutionName)

	if cmd.Flags().Changed("include-service") {
		output.PrintCmdStatus(cmd, "Adding the service-component.json \n")
		folderName := solutionName + "/services"
		fileName := "service-component.json"

		manifest.Dependencies = append(manifest.Dependencies, "zodiac")

		serviceComponentDef := &ComponentDef{
			Type:        "zodiac:function",
			ObjectsFile: "services/service-component.json",
		}

		manifest.Objects = append(manifest.Objects, *serviceComponentDef)
		serviceComp := getServiceComponent("sampleservice")

		createComponentFile(serviceComp, folderName, fileName)
	}

	if cmd.Flags().Changed("include-knowledge") {
		output.PrintCmdStatus(cmd, "Adding the knowledge-component.json \n")
		folderName := solutionName + "/knowledge"
		fileName := "knowledge-component.json"
		manifest.Types = append(manifest.Types, fmt.Sprintf("knowledge/%s", fileName))

		knowledgeComp := createKnowledgeComponent(manifest)
		createComponentFile(knowledgeComp, folderName, fileName)
	}

	output.PrintCmdStatus(cmd, "Adding the manifest.json \n")
	createSolutionManifestFile(solutionName, manifest)

}

func createInitialSolutionManifest(solutionName string) *Manifest {

	emptyDeps := make([]string, 0)
	manifest := &Manifest{
		ManifestVersion: "1.0.0",
		SolutionVersion: "1.0.0",
		Dependencies:    emptyDeps,
		Description:     "description of your solution",
		GitRepoUrl:      "the url for the git repo holding your solution",
		Contact:         "the email for this solution's point of contact",
		HomePage:        "the url for this solution's homepage",
		Readme:          "the url for this solution's readme file",
	}
	manifest.Name = solutionName

	return manifest

}

func createSolutionManifestFile(folderName string, manifest *Manifest) {
	filepath := fmt.Sprintf("%s/manifest.json", folderName)
	manifestFile, err := os.Create(filepath)

	if err != nil {
		log.Errorf("Failed to create manifest.json %v", err.Error())
	}

	manifestJson, _ := json.Marshal(manifest)

	_, _ = manifestFile.WriteString(string(manifestJson))
	manifestFile.Close()
}

func createKnowledgeComponent(manifest *Manifest) *KnowledgeDef {
	jsonSchema := map[string]interface{}{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"title":                "Data Collector Configuration",
		"description":          "Sample Knowledge type representing the configuration information required for a data collector service component to access an external service",
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"cloudCollectorTargetURL": map[string]interface{}{
				"type":        "string",
				"description": "The URL that the data collector will connect to using the api key",
			},
			"cloudCollectorTargetApiKey": map[string]interface{}{
				"type":        "string",
				"description": "The apiKey used to secure the REST endpoint and added to calls by the poller function",
			},
		},
		"required": []string{"cloudCollectorTargetURL", "cloudCollectorTargetApiKey"},
	}
	idGen := &IdGenerationDef{
		EnforceGlobalUniqueness: true,
		GenerateRandomId:        true,
		IdGenerationMechanism:   "{{layer.id}}",
	}

	knowledgeComponent := &KnowledgeDef{
		Name:             "dataCollectorConfiguration",
		AllowedLayers:    []string{"TENANT"},
		IdGeneration:     idGen,
		SecureProperties: []string{"$.collectorTargetApiKey"},
		JsonSchema:       jsonSchema,
	}

	return knowledgeComponent
}

func createComponentFile(compDef any, folderName string, fileName string) {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		if err := os.Mkdir(folderName, os.ModePerm); err != nil {
			log.Errorf("Create solution component file failed - %v", err.Error())
		}
	}

	filepath := fmt.Sprintf("%s/%s", folderName, fileName)

	svcFile, err := os.Create(filepath)
	if err != nil {
		log.Errorf("Create solution component file failed - %v", err.Error())
	}
	defer svcFile.Close()

	svcJson, _ := json.Marshal(compDef)

	_, _ = svcFile.WriteString(string(svcJson))
	svcFile.Close()
}

func appendFolder(folderName string) {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		if err := os.Mkdir(folderName, os.ModePerm); err != nil {
			log.Errorf("Error adding folder named %s - %v", folderName, err.Error())
		}
	}
}

func openFile(filePath string) *os.File {
	svcFile, err := os.Open(filePath)
	if err != nil {
		log.Errorf("Can't open the file named %s \n", filePath)
		return nil
	}
	return svcFile
}

func createFile(filePath string) {
	var svcFile *os.File
	var err error
	if svcFile, err = os.Create(filePath); err != nil {
		log.Errorf("Can't create the file named %s - %v", filePath, err.Error())
	}
	svcFile.Close()
}
