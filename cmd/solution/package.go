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
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package a solution into a zip file",
	Long: `This command packages the solution directory into a zip file that's easy to push or archive.
	
The input is a solution directory, defaulting to the current working directory.
The output is either a directory path (in which a fsoc will create the zip file) or path to the zip flie to create.

Note that when using native solution isolation, there is no need to define a tag, as the package is not tag-specific.

If fsoc-based solution pseudo-isolation is used, then use the --tag, --stable or --env-file flags. 
Pseudo-isolation is automatically enabled if ${} substitution is present in the solution name in the
manifest file. There are several ways to specify the tags, based on convenience and use cases. 
The following priority is available:
1. --tag=xyz or --stable: use this tag, ignoring env file or env vars
2. A tag is defined in the FSOC_SOLUTION_TAG environment variable (ignores env file)
3. An explicitly provided --env-file path
4. Implicitly looking into env.json file in the solution directory (usually not version controlled)
`,
	Example: `  fsoc solution package --solution-bundle=../mysolution.zip
  fsoc solution package -d mysolution --solution-bundle=/somepath/mysolution-1234.zip`,
	Run:         packageSolution,
	Annotations: map[string]string{config.AnnotationForConfigBypass: ""},
}

func getSolutionPackageCmd() *cobra.Command {

	solutionPackageCmd.Flags().
		String("solution-bundle", "", "Path to output directory or file to place solution zip into (defaults to temp dir)")

	solutionPackageCmd.Flags().
		StringP("directory", "d", "", "Path to the solution root directory (defaults to current dir)")

	solutionPackageCmd.Flags().
		String("tag", "", "Isolation tag to use if using fsoc isolation; if specified, takes precedence over env vars and .tag file")
	solutionPackageCmd.Flags().
		Bool("stable", false, "Mark the solution as production-ready.  This is equivalent to supplying --tag=stable")
	solutionPackageCmd.Flags().
		String("env-file", "", "Path to the env vars json file with isolation tag and, optionally, dependency tags")
	solutionPackageCmd.Flags().
		Bool("no-isolate", false, "Disable fsoc-supported solution isolation")
	solutionPackageCmd.MarkFlagsMutuallyExclusive("tag", "stable", "env-file", "no-isolate")

	return solutionPackageCmd
}

func packageSolution(cmd *cobra.Command, args []string) {
	outputFilePath, _ := cmd.Flags().GetString("solution-bundle")
	solutionDirectoryPath, _ := cmd.Flags().GetString("directory")

	// finalize solution path
	if solutionDirectoryPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err.Error())
		}
		solutionDirectoryPath = currentDir
	}
	if !isSolutionPackageRoot(solutionDirectoryPath) {
		log.Fatal("Could not find solution manifest") //nb: isSolutionPackageRoot prints clear message
	}

	// isolate if needed
	solutionDirectoryPath, tag, err := embeddedConditionalIsolate(cmd, solutionDirectoryPath)
	if err != nil {
		log.Fatalf("Failed to isolate solution with tag: %v", err)
	}

	// load manifest
	manifest, err := getSolutionManifest(solutionDirectoryPath)
	if err != nil {
		log.Fatalf("Failed to read solution manifest: %v", err)
	}

	var message string
	message = fmt.Sprintf("Packaging solution %s version %s with tag %s\n", manifest.Name, manifest.SolutionVersion, tag)
	output.PrintCmdStatus(cmd, message)

	// create archive
	solutionArchive := generateZip(cmd, solutionDirectoryPath, outputFilePath)
	solutionArchive.Close()

	message = fmt.Sprintf("Solution %s version %s is ready in %s\n", manifest.Name, manifest.SolutionVersion, solutionArchive.Name())
	output.PrintCmdStatus(cmd, message)
}

// --- Helper functions for managing solution directory and zip bundle

// generateZip creates a solution bundle (zip file) from a given solutionPath directory.
// If outputPath is specified, the zip will be placed in that directory (if an existing directory) or filename (otherwise);
// if outputPath is empty, the zip file will be placed in the temp directory.
// If solutionPath is not specified, the current directory is assumed (it must contain the solution
// manifest in its final form).
func generateZip(cmd *cobra.Command, solutionPath string, outputPath string) *os.File {
	var archive *os.File
	var err error
	var archiveFileTemplate string
	solutionName := filepath.Base(solutionPath)
	solutionNameWithZipSuffix := fmt.Sprintf("%s.zip", solutionName)

	// create zip file
	if outputPath != "" {
		// absolutize path
		outputPath = absolutizePath(outputPath)

		// if outputPath is an existing directory, place zip there; otherwise, treat as file path
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(outputPath)
		if err == nil && fileInfo.IsDir() {
			outputPath = filepath.Join(filepath.Dir(outputPath), solutionNameWithZipSuffix)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("Failed to access target path %q: %v", outputPath, err)
		} // else treat as file path, possibly overwriting existing file
		archive, err = os.Create(outputPath)
	} else {
		archiveFileTemplate = fmt.Sprintf("%s*.zip", solutionName)
		archive, err = os.CreateTemp("", archiveFileTemplate)
		outputPath = archive.Name()
	}
	if err != nil {
		log.Fatalf("failed to create file %s: %v", outputPath, err)
		panic(err)
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Creating solution zip: %q\n", archive.Name()))
	log.WithField("path", archive.Name()).Info("Creating solution file")
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	// determine the solution directory's parent folder to start archiving from
	solutionPath = absolutizePath(solutionPath)
	solutionParentPath := filepath.Dir(solutionPath)

	// switch cwd to the solution directory for archiving
	fsocWorkingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't get the current working directory: %v", err)
	}
	err = os.Chdir(solutionParentPath)
	if err != nil {
		log.Fatalf("Couldn't switch working directory to solution root's parent directory %q: %v", solutionParentPath, err)
	}
	defer func() {
		// restore original working directory
		err := os.Chdir(fsocWorkingDir)
		if err != nil {
			log.Fatalf("Couldn't switch working directory back to starting working directory: %v", err)
		}
	}()

	err = filepath.Walk(solutionName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if isAllowedPath(path, info) {
				addFileToZip(zipWriter, path, info)
			}
			return nil
		})
	if err != nil {
		log.Fatalf("Error traversing the directory: %v", err)
	}
	zipWriter.Close()
	log.WithField("path", archive.Name()).Info("Created a solution with path")

	return archive
}

func isAllowedPath(path string, info os.FileInfo) bool {
	// blacklist files by adding them here.
	excludeFiles := []string{".DS_Store", TagFileName} // .tag files should not be included in the zip
	// blacklist paths by adding them here.
	excludePaths := []string{".git"}
	allow := true

	if info.IsDir() {
		// check for blacklisted dirs
		for _, exclP := range excludePaths {
			if strings.Contains(path, exclP) {
				allow = false
			}
		}
	} else {
		// check for blacklisted files
		if slices.Contains(excludeFiles, filepath.Base(path)) {
			allow = false
		}
	}

	return allow
}

func addFileToZip(zipWriter *zip.Writer, fileName string, info os.FileInfo) {
	newFile, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Couldn't open file %q: %v", fileName, err)
	}
	defer newFile.Close()

	if info.IsDir() {
		fileName = fileName + string(os.PathSeparator)
	}

	fileName = filepath.ToSlash(fileName)

	archWriter, err := zipWriter.Create(fileName)

	if err != nil {
		log.Fatalf("Couldn't create archive writer for file: %v", err)
	}

	if !info.IsDir() {
		if _, err := io.Copy(archWriter, newFile); err != nil {
			log.Fatalf("Couldn't write file to architve: %v", err)
		}
	}
}

func isSolutionPackageRoot(path string) bool {
	_, err := getSolutionManifest(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Errorf("The directory %s is not a solution root directory", path)
		} else {
			log.Errorf("Failed to read solution manifest: %v", err)
		}
		return false
	}
	return true
}

func getSolutionManifest(path string) (*Manifest, error) {
	checkStructTags(reflect.TypeOf(Manifest{})) // ensure struct tags are correct

	// Determine manifest name, in JSON or YAML format
	var manifestPath string
	manifestPathJson := filepath.Join(path, "manifest.json")
	manifestPathYaml := filepath.Join(path, "manifest.yaml")
	_, err := os.Stat(manifestPathJson)
	jsonExists := err == nil
	_, err = os.Stat(manifestPathYaml)
	yamlExists := err == nil
	switch {
	case jsonExists && yamlExists:
		return nil, fmt.Errorf("found both JSON and YAML manifests; only one can exist")
	case jsonExists:
		manifestPath = manifestPathJson
	case yamlExists:
		manifestPath = manifestPathYaml
	default:
		return nil, fmt.Errorf("%q is not a solution root directory", path)
	}

	// Read manifest
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("%q is not a solution root directory", path)
	}
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, err
	}

	// Unmarshal manifest
	var manifest *Manifest
	err = yaml.Unmarshal(manifestBytes, &manifest) // json is a subset of yaml
	if err != nil {
		return nil, err
	}

	// Store manifest type
	if jsonExists {
		manifest.ManifestFormat = FileFormatJSON
	} else {
		manifest.ManifestFormat = FileFormatYAML
	}

	// Log manifest summary
	log.WithFields(log.Fields{
		"manifest_path":    absolutizePath(manifestPath),
		"manifest_version": manifest.ManifestVersion,
		"manifest_format":  manifest.ManifestFormat,
		"solution_name":    manifest.Name,
		"solution_version": manifest.SolutionVersion,
		"solution_type":    manifest.SolutionType,
	}).Info("Read solution manifest")

	return manifest, nil
}

// absolutizePath takes a path in any form (absolute, relative or home-dir-relative)
// and converts it to an absolute path (which is also cleaned up/canonicalized).
// Note that this works both for files and directories, including just "~"
func absolutizePath(inputPath string) string {
	path := inputPath // keep original value for error messages

	// replace ~ with home directory, if needed
	if strings.HasPrefix(path, "~") {
		dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get current directory to use for %q: %v", inputPath, err)
		}
		path = dirname + path[1:] // can't use Join because source may be just "~"
	}

	// convert to absolute path
	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Failed to get absolute path for %q: %v", inputPath, err)
	}

	// clean path
	path = filepath.Clean(path)

	return path
}
