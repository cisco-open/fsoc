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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

var solutionPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package a solution into a zip file",
	Long:  `This command packages the solution directory into a zip file that's easy to push or archive`,
	Example: `  fsoc solution package --solution-bundle=../mysolution.zip
  fsoc solution package -d mysolution --solution-bundle=/somepath/mysolution-1234.zip`,
	Run: packageSolution,
}

func getSolutionPackageCmd() *cobra.Command {

	solutionPackageCmd.Flags().
		String("solution-bundle", "", "full path to solution zip fileto create (defaults to temp dir)")

	solutionPackageCmd.Flags().
		StringP("directory", "d", "", "full path to the solution root directory (defaults to current dir)")

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

	// load manifest
	manifest, err := getSolutionManifest(solutionDirectoryPath)
	if err != nil {
		log.Fatalf("Failed to read solution manifest: %v", err)
	}

	var message string
	message = fmt.Sprintf("Generating solution %s version %s bundle archive\n", manifest.Name, manifest.SolutionVersion)
	output.PrintCmdStatus(cmd, message)

	// create archive
	solutionArchive := generateZip(cmd, solutionDirectoryPath, outputFilePath)
	solutionArchive.Close()

	message = fmt.Sprintf("Solution %s version %s bundle is ready in %s\n", manifest.Name, manifest.SolutionVersion, solutionArchive.Name())
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
	output.PrintCmdStatus(cmd, fmt.Sprintf("Creating archive zip: %q\n", archive.Name()))
	log.WithField("path", archive.Name()).Info("Creating solution bundle file")
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
		log.Fatalf("Couldn't switch working folder to solution root's parent directory %q: %v", solutionParentPath, err)
	}
	defer func() {
		// restore original working directory
		err := os.Chdir(fsocWorkingDir)
		if err != nil {
			log.Fatalf("Couldn't switch working folder back to starting working folder: %v", err)
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
		log.Fatalf("Error traversing the folder: %v", err)
	}
	zipWriter.Close()
	log.WithField("path", archive.Name()).Info("Created a solution bundle file")

	return archive
}

func isAllowedPath(path string, info os.FileInfo) bool {
	fileInclude := []string{".json", ".md"}
	excludePath := []string{".git"}
	allow := false

	if info.IsDir() {
		for _, exclP := range excludePath {
			if strings.Contains(path, exclP) {
				return allow
			}
		}
		allow = true
	} else {
		ext := filepath.Ext(path)
		for _, inclExt := range fileInclude {
			if ext == inclExt {
				allow = true
			}
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
	manifestPath := fmt.Sprintf("%s/manifest.json", path)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		log.Errorf("The folder %s is not a solution package root folder", path)
		return false
	}
	manifestFile.Close()
	return true
}

func getSolutionManifest(path string) (*Manifest, error) {
	manifestPath := fmt.Sprintf("%s/manifest.json", path)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("%q is not a solution package root folder", path)
	}
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, err
	}

	var manifest *Manifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		return nil, err
	}

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
