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
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new solution",
	Long: `This command creates a skeleton of a solution in the current directory.

Example:

   fsoc solution init --name=testSolution --include-service --include-knowledge

Creates a subdirectory named "testSolution" in the current directory and populates
it with a solution manifest and objects for it. The optional --include-... flags
define what objects are added to the solution. Once the solution is created,
the "solution extend" command can be used to add more objects.`,
	Run:              generateSolutionPackage,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
	TraverseChildren: true,
}

// Planned options:
//    include-service - Flag to include sample service component
//    include-metric - Flag to include sample metric type component
//    include-knowledge - Flag to include sample knowledge type component
//    include-meltworkflow - Flag to include sample melt workflow
//    include-dash-ui - Flag to include sample dash-ui template

func getInitSolutionCmd() *cobra.Command {
	solutionInitCmd.Flags().
		String("name", "", "The name of the new solution (required)")
	_ = solutionInitCmd.MarkFlagRequired("name")

	solutionInitCmd.Flags().
		Bool("include-service", true, "Add a service component definition to this solution")
	solutionInitCmd.Flags().
		Bool("include-knowledge", true, "Add a knowledge type definition to this solution")

	return solutionInitCmd
}

func generateSolutionPackage(cmd *cobra.Command, args []string) {

	solutionName, _ := cmd.Flags().GetString("name")
	solutionName = strings.ToLower(solutionName)

	if len(solutionName) == 0 {
		log.Fatal("A non-empty flag \"--name\" is required.")
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("Preparing the %s solution package folder structure... \n", solutionName))

	if err := os.Mkdir(solutionName, os.ModePerm); err != nil {
		log.Fatalf("Solution init failed - %v", err)
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

func writeSolutionManifest(folderName string, manifest *Manifest) error {
	filepath := fmt.Sprintf("%s/manifest.json", folderName)
	manifestFile, err := os.Create(filepath) // create new or truncate existing
	if err != nil {
		return fmt.Errorf("Failed to create manifest file %q: %w", filepath, err)
	}
	defer manifestFile.Close()

	err = output.WriteJson(manifest, manifestFile)
	if err != nil {
		return fmt.Errorf("Failed to write the manifest into file %q: %w", filepath, err)
	}

	// the file is closed before returning (see defer above)
	return nil
}

// createSolutionManifestFile is a "must succeed" version of writeSolutionManifests
func createSolutionManifestFile(folderName string, manifest *Manifest) {
	if err := writeSolutionManifest(folderName, manifest); err != nil {
		log.Fatalf(err.Error())
	}
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
		if err := os.MkdirAll(folderName, os.ModePerm); err != nil {
			log.Fatalf("Failed to create solution component directory %q: %v", folderName, err)
		}
	}

	filepath := fmt.Sprintf("%s/%s", folderName, fileName)

	svcFile, err := os.Create(filepath)
	if err != nil {
		log.Fatalf("Failed to create solution component file %q: %v", folderName+"/"+fileName, err)
	}
	defer svcFile.Close()

	if err = output.WriteJson(compDef, svcFile); err != nil {
		log.Fatalf("Failed to write the solution component into file %q: %v", folderName+"/"+fileName, err)
	}
}

func appendFolder(folderName string) {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		if err := os.Mkdir(folderName, os.ModePerm); err != nil {
			log.Fatalf("Error adding folder named %q: %v", folderName, err)
		}
	}
}

func openFile(filePath string) *os.File {
	svcFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Can't open the file named %q: %v", filePath, err)
	}
	return svcFile
}

func createFile(filePath string) {
	var svcFile *os.File
	var err error
	if svcFile, err = os.Create(filePath); err != nil {
		log.Fatalf("Can't create the file named %q: %v", filePath, err)
	}
	svcFile.Close()
}
