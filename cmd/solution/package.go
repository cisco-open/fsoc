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

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

var solutionPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Build and package a solution",
	Long: `This command allows the current tenant specified in the profile to build and package a solution bundle to be deployed into the FSO Platform.

Usage:
	fsoc solution package --solution-package=<solution-package-root-path>`,
	Args:             cobra.ExactArgs(0),
	Run:              packageSolution,
	TraverseChildren: true,
}

func getSolutionPackageCmd() *cobra.Command {
	solutionPackageCmd.Flags().
		String("solution-package", "", "The fully qualified path name for the solution package .zip file")
	_ = solutionPackageCmd.MarkFlagRequired("solution-package")

	return solutionPackageCmd

}

func packageSolution(cmd *cobra.Command, args []string) {

	solutionPackagePath, _ := cmd.Flags().GetString("solution-package")
	if solutionPackagePath == "" {
		log.Fatal("solution-package cannot be empty, use --solution-package=<solution-package-file-path>")
	}
	if !isSolutionPackageRoot(solutionPackagePath) {
		log.Fatal("solution-package path doesn't point to a solution package root folder")
	}
	manifest, _ := getSolutionManifest(solutionPackagePath)
	var message string
	message = fmt.Sprintf("Generating solution %s - %s bundle archive \n", manifest.Name, manifest.SolutionVersion)
	log.WithFields(log.Fields{
		"solution-package": solutionPackagePath,
	}).Info(message)

	output.PrintCmdStatus(cmd, message)
	solutionArchive := generateZip(cmd, solutionPackagePath)
	solutionArchive.Close()

	message = fmt.Sprintf("Solution %s - %s bundle is ready. \n", manifest.Name, manifest.SolutionVersion)
	output.PrintCmdStatus(cmd, message)
}

func generateZip(cmd *cobra.Command, sltnPackagePath string) *os.File {
	// splitPath := strings.Split(sltnPackagePath, "/")
	// solutionName := splitPath[len(splitPath)-1]
	solutionName := filepath.Base(sltnPackagePath)
	archiveFileName := fmt.Sprintf("%s.zip", solutionName)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Creating %s archive... \n", archiveFileName))
	archive, err := os.Create(archiveFileName)
	if err != nil {
		panic(err)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	fsocWorkingDir, err := os.Getwd()
	if err != nil {
		log.Errorf("Couldn't read fsoc working directory: %v", err)
	}

	solutionRootFolder := filepath.Dir(sltnPackagePath)
	err = os.Chdir(solutionRootFolder)
	if err != nil {
		log.Errorf("Couldn't switch working folder to solution package folder: %v", err)
	}

	defer func() {
		err := os.Chdir(fsocWorkingDir)
		if err != nil {
			log.Errorf("Couldn't switch working folder back to fsoc working folder: %v", err)
		}
	}()

	err = filepath.Walk(solutionName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			addFileToZip(zipWriter, path, info)
			return nil
		})
	if err != nil {
		log.Errorf("Error traversing the folder: %v", err.Error())
	}
	zipWriter.Close()

	return archive
}

func addFileToZip(zipWriter *zip.Writer, fileName string, info os.FileInfo) {
	newFile, err := os.Open(fileName)
	if err != nil {
		log.Errorf("Couldn't open file %v", err.Error())
	}
	defer newFile.Close()

	if info.IsDir() {
		fileName = fileName + string(os.PathSeparator)
	}

	archWriter, err := zipWriter.Create(fileName)

	if err != nil {
		log.Errorf("Couldn't create archive writer for file - %v", err.Error())
	}

	if !info.IsDir() {
		if _, err := io.Copy(archWriter, newFile); err != nil {
			log.Errorf("Couldn't write file to architve - %v", err.Error())
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
		log.Errorf("The folder %s is not a solution package root folder", path)
		return nil, err
	}
	defer manifestFile.Close()

	manifestBytes, _ := io.ReadAll(manifestFile)
	var manifest *Manifest

	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		log.Errorf("Can't generate a manifest objects from the manifest.json, make sure your manifest.json file is correct. - %v", err.Error())
		return nil, err
	}

	return manifest, nil
}
