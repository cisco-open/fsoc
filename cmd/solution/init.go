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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionInitCmd = &cobra.Command{
	Use:   "init <solution-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Create a new solution",
	Long: `This command creates a skeleton of a solution in the current directory.

Solution names must start with a lowercase letter and contain only lowercase letters and digits.

It creates a subdirectory named <solution-name> in the current directory and
a solution manifest. Once the solution is created, the "solution extend" command
can be used to add types and objects to it.`,
	Example: `  fsoc solution init mycomponent
  fsoc solution init mymodule --solution-type=module --yaml`,
	Run:              createNewSolution,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""}, // this command does not require a valid context
	TraverseChildren: true,
}

func getInitSolutionCmd() *cobra.Command {
	solutionInitCmd.Flags().
		String("solution-type", "component", "The type of the solution you are creating (should be one of component, module, or application).")
	solutionInitCmd.Flags().
		Bool("yaml", false, "Use YAML format instead of JSON for the solution manifest and objects.")

	return solutionInitCmd
}

func createNewSolution(cmd *cobra.Command, args []string) {
	solutionName := strings.ToLower(args[0])
	solutionType, _ := cmd.Flags().GetString("solution-type") // checked when creating manifest

	// check solution name for validity / safety for creating a directory (incl. empty name)
	match, err := regexp.Match(`^[a-z][a-z0-9]*$`, []byte(solutionName))
	if err != nil {
		log.Fatalf("Failed to validate solution name %q: %v", solutionName, err)
	}
	if !match {
		log.Fatalf("Invalid solution name %q: must start with a lowercase letter and contain only lowercase letters and digits", solutionName)
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("Preparing the solution directory structure for %q... \n", solutionName))
	if err := os.Mkdir(solutionName, os.ModePerm); err != nil {
		log.Fatalf("Failed to create a new directory %q: %v", solutionName, err)
	}

	manifest := createInitialSolutionManifest(solutionName, WithSolutionType(solutionType))
	if useYaml, _ := cmd.Flags().GetBool("yaml"); useYaml {
		manifest.ManifestFormat = FileFormatYAML
	}
	createSolutionManifestFile(solutionName, manifest)

	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution %q created successfully.\n", solutionName))
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
	checkStructTags(reflect.TypeOf(manifest)) // ensure json/yaml struct tags are correct

	// create the manifest file, overwriting prior manifest
	filepath := filepath.Join(folderName, fmt.Sprintf("manifest.%s", manifest.ManifestFormat))
	manifestFile, err := os.Create(filepath) // create new or truncate existing
	if err != nil {
		return fmt.Errorf("failed to create manifest file %q: %w", filepath, err)
	}
	defer manifestFile.Close()

	// write the manifest into the file, in manifest's selected format
	switch manifest.ManifestFormat {
	case FileFormatJSON:
		err = output.WriteJson(manifest, manifestFile)
	case FileFormatYAML:
		err = output.WriteYaml(manifest, manifestFile)
	default:
		err = fmt.Errorf("(bug) unknown manifest format %q", manifest.ManifestFormat)
	}
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

func createComponentFile(compDef any, folderName string, fileName string) {
	// create directory if it doesn't exist
	if _, err := os.Stat(folderName); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(folderName, os.ModePerm); err != nil {
			log.Fatalf("Failed to create solution component directory %q: %v", folderName, err)
		}
	}

	// determine file format
	var format FileFormat
	ext, _ := strings.CutPrefix(filepath.Ext(fileName), ".") // extension without the leading dot
	switch ext {
	case FileFormatJSON.String():
		format = FileFormatJSON
	case FileFormatYAML.String():
		format = FileFormatYAML
	}

	// create the component file
	filepath := filepath.Join(folderName, fileName)
	svcFile, err := os.Create(filepath)
	if err != nil {
		log.Fatalf("Failed to create solution component file %q: %v", filepath, err)
	}
	defer svcFile.Close()

	// write the component definition into the file
	switch format {
	case FileFormatYAML:
		err = output.WriteYaml(compDef, svcFile)
	case FileFormatJSON:
		// nb: don't use output.WriteJson in order to be able to control HTML escaping
		enc := json.NewEncoder(svcFile)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", output.JsonIndent)
		err = enc.Encode(compDef)
	}
	if err != nil {
		log.Fatalf("Failed to write the solution component into file %q: %v", filepath, err)
	}
}

func openFile(filePath string) *os.File {
	svcFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Can't open the file named %q: %v", filePath, err)
	}
	return svcFile
}
