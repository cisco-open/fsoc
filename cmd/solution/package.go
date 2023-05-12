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
	Short: "Package solution folder into .zip file",
	Long:  `This command allows the user to turn their solution folder into a zip file from the command line directly`,
	Example: `  fsoc solution package --directory <path/to/solution/root/folder>
  fsoc solution package --directory <path/to/solution/root/folder> --solution-bundle <path/to/directory/where/zip/should/exist>`,
	Run: packageSolution,
}

func getSolutionPackageCmd() *cobra.Command {

	solutionPackageCmd.Flags().
		String("solution-bundle", "", "fully qualified path name to directory where you want the packaged solution to exist after creation.  If this isn't specified, the solution zip will be created and stored in a temp directory (the path to which will be specified in the output of the command)")

	solutionPackageCmd.Flags().
		StringP("directory", "d", "", "fully qualified path name of the solution folder to be zipped")

	return solutionPackageCmd
}

func packageSolution(cmd *cobra.Command, args []string) {
	solutionDirectoryPath, _ := cmd.Flags().GetString("directory")
	outputDirectoryPath, _ := cmd.Flags().GetString("solution-bundle")

	if solutionDirectoryPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err.Error())
		}
		solutionDirectoryPath = currentDir
	}
	if !isSolutionPackageRoot(solutionDirectoryPath) {
		log.Fatal("directory flag doesn't point to a solution root folder")
	}
	manifest, err := getSolutionManifest(solutionDirectoryPath)
	if err != nil {
		log.Fatalf("Failed to read solution manifest: %v", err)
	}
	var message string
	message = fmt.Sprintf("Generating solution %s - %s bundle archive \n", manifest.Name, manifest.SolutionVersion)
	log.WithFields(log.Fields{
		"solution-bundle": solutionDirectoryPath,
	}).Info(message)

	output.PrintCmdStatus(cmd, message)
	solutionArchive := generateZip(cmd, solutionDirectoryPath, outputDirectoryPath)
	solutionArchive.Close()

	message = fmt.Sprintf("Solution %s - %s bundle is ready. \n", manifest.Name, manifest.SolutionVersion)
	output.PrintCmdStatus(cmd, message)
}

// Helper functions for managing solution directory and zip bundle
func generateZip(cmd *cobra.Command, sltnPackagePath string, outputDirectoryPath string) *os.File {
	var archive *os.File
	var err error
	var archiveFileTemplate string
	solutionName := filepath.Base(sltnPackagePath)
	solutionNameWithZipSuffix := fmt.Sprintf("%s.zip", solutionName)

	if outputDirectoryPath != "" {
		if filepath.Base(outputDirectoryPath) != solutionNameWithZipSuffix {
			outputDirectoryPath = filepath.Join(filepath.Dir(outputDirectoryPath), solutionNameWithZipSuffix)
		}
		archive, err = os.Create(outputDirectoryPath)
	} else {
		archiveFileTemplate = fmt.Sprintf("%s*.zip", solutionName)
		archive, err = os.CreateTemp("", archiveFileTemplate)
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("Creating archive zip (%q)\n", archive.Name()))
	log.WithField("path", archive.Name()).Info("Creating solution bundle file")
	if err != nil {
		log.Fatalf("failed to create file: %s", archive.Name())
		panic(err)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	fsocWorkingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't read the current working directory: %v", err)
	}

	solutionRootFolder := filepath.Dir(sltnPackagePath)
	err = os.Chdir(solutionRootFolder)
	if err != nil {
		log.Fatalf("Couldn't switch working folder to solution package folder: %v", err)
	}

	defer func() {
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
	log.WithField("path", archive.Name()).Info("Created a temporary solution bundle file")

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
