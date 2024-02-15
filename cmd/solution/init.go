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
	"os"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionInitCmd = &cobra.Command{
	Use:   "init <solution-name>",
	Args:  cobra.MaximumNArgs(1),
	Short: "Create a new solution",
	Long: `This command creates a skeleton of a solution in the current directory.

It creates a subdirectory named <solution-name> in the current directory and populates
it with a solution manifest and objects for it. Once the solution is created,
the "solution extend" command can be used to add objects to it.`,
	Example:          `  fsoc solution init mysolution`,
	Run:              generateSolutionPackage,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
	TraverseChildren: true,
}

func getInitSolutionCmd() *cobra.Command {
	solutionInitCmd.Flags().
		String("name", "", "The name of the new solution (required)")
	_ = solutionInitCmd.Flags().MarkDeprecated("name", "please use argument instead.")

	solutionInitCmd.Flags().
		Bool("include-service", true, "Add a service component definition to this solution")
	_ = solutionInitCmd.Flags().MarkDeprecated("include-service", `please use the "solution extend" command instead.`)
	solutionInitCmd.Flags().
		Bool("include-knowledge", true, "Add a knowledge type definition to this solution")
	_ = solutionInitCmd.Flags().MarkDeprecated("include-knowledge", `please use the "solution extend" command instead.`)

	solutionInitCmd.Flags().
		String("solution-type", "component", "The type of the solution you are creating (should be one of component, module, or application).  Default value is component.")

	return solutionInitCmd
}

func generateSolutionPackage(cmd *cobra.Command, args []string) {
	solutionName := getSolutionNameFromArgs(cmd, args, "name")
	solutionName = strings.ToLower(solutionName)
	solutionType, err := cmd.Flags().GetString("solution-type")

	if err != nil {
		log.Fatalf(err.Error())
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("Preparing the solution directory structure for %q... \n", solutionName))

	if err := os.Mkdir(solutionName, os.ModePerm); err != nil {
		log.Fatal(err.Error())
	}

	manifest := createInitialSolutionManifest(solutionName, WithSolutionType(solutionType))

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

// --- Solution Manifest Helpers

type solutionManifestOptions struct {
	manifestVersion string
	solutionVersion string
	solutionType    string
}

type SolutionManifestOption func(*solutionManifestOptions)

func WithManifestVersion(version string) SolutionManifestOption {
	return func(o *solutionManifestOptions) {
		o.manifestVersion = version
	}
}

func WithSolutionVersion(version string) SolutionManifestOption {
	return func(o *solutionManifestOptions) {
		o.solutionVersion = version
	}
}

func WithSolutionType(solutionType string) SolutionManifestOption {
	return func(o *solutionManifestOptions) {
		o.solutionType = solutionType
	}
}

var knownSolutionTypes = []string{"component", "module", "application"}
var knownManifestVersions = []string{"1.0.0", "1.1.0"}

func createInitialSolutionManifest(solutionName string, options ...SolutionManifestOption) *Manifest {

	opts := solutionManifestOptions{
		manifestVersion: "1.1.0",
		solutionVersion: "1.0.0",
		solutionType:    "component",
	}
	for _, o := range options {
		o(&opts)
	}

	// soft-validate options
	if !slices.Contains(knownSolutionTypes, opts.solutionType) {
		log.Warnf("Unknown solution type %q (expected one of %q); proceeding anyway", opts.solutionType, knownSolutionTypes)
	}
	if !slices.Contains(knownManifestVersions, opts.manifestVersion) {
		log.Warnf("Unknown manifest version %q (expected one of %q); proceeding anyway", opts.manifestVersion, knownManifestVersions)
	}

	emptyDeps := make([]string, 0)
	manifest := &Manifest{
		ManifestVersion: opts.manifestVersion,
		Name:            solutionName,
		SolutionType:    opts.solutionType,
		SolutionVersion: opts.solutionVersion,
		Dependencies:    emptyDeps,
		Description:     "description of your solution",
		GitRepoUrl:      "the url for the git repo holding your solution",
		Contact:         "the email for this solution's point of contact",
		HomePage:        "the url for this solution's homepage",
		Readme:          "the url for this solution's readme file",
	}

	return manifest
}

func writeSolutionManifest(folderName string, manifest *Manifest) error {
	filepath := fmt.Sprintf("%s/manifest.json", folderName)
	manifestFile, err := os.Create(filepath) // create new or truncate existing
	if err != nil {
		return fmt.Errorf("failed to create manifest file %q: %w", filepath, err)
	}
	defer manifestFile.Close()

	err = output.WriteJson(manifest, manifestFile)
	if err != nil {
		return fmt.Errorf("failed to write the manifest into file %q: %w", filepath, err)
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
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name for the data collector configuration that will be used to generate the ID for this knowledge object",
			},
		},
		"required": []string{"cloudCollectorTargetURL", "cloudCollectorTargetApiKey", "name"},
	}

	knowledgeComponent := &KnowledgeDef{
		Name:                  "dataCollectorConfiguration",
		AllowedLayers:         []string{"TENANT"},
		IdentifyingProperties: []string{"/name"},
		SecureProperties:      []string{"$.collectorTargetApiKey"},
		JsonSchema:            jsonSchema,
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

	enc := json.NewEncoder(svcFile)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", output.JsonIndent)
	err = enc.Encode(compDef)

	if err != nil {
		log.Fatalf("Failed to write the solution component into file %q: %v", folderName+"/"+fileName, err)
	}

	// if err = output.WriteJson(compDef, svcFile); err != nil {
	// 	log.Fatalf("Failed to write the solution component into file %q: %v", folderName+"/"+fileName, err)
	// }
}

func openFile(filePath string) *os.File {
	svcFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Can't open the file named %q: %v", filePath, err)
	}
	return svcFile
}
