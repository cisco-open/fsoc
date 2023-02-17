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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Deploy your solution",
	Long: `This command allows the current tenant specified in the profile to deploy a solution to the FSO Platform.

Examples:
  fsoc solution push
  fsoc solution push --solution-bundle=mysolution.zip
  
The first command deploys a solution from the current directory. The second command
deploys a solution from an existing archive file.`,
	Args:             cobra.ExactArgs(0),
	Run:              pushSolution,
	TraverseChildren: true,
}

func getSolutionPushCmd() *cobra.Command {
	solutionPushCmd.Flags().
		String("solution-bundle", "", "fully qualified path name for the solution bundle .zip file")

	return solutionPushCmd

}

func pushSolution(cmd *cobra.Command, args []string) {
	manifestPath := ""
	solutionBundlePath, _ := cmd.Flags().GetString("solution-bundle")
	var solutionArchivePath string
	if solutionBundlePath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal("Please run this command in a folder with a solution or use the --solution-bundle flag")
		}
		manifestPath = currentDir
		if !isSolutionPackageRoot(manifestPath) {
			log.Fatal("solution-bundle / current dir path doesn't point to a solution package root folder")
		}

		_, _ = getSolutionManifest(manifestPath)

		solutionArchive := generateZipNoCmd(manifestPath)
		solutionArchivePath = filepath.Base(solutionArchive.Name())

	} else {
		manifestPath = solutionBundlePath
		solutionArchivePath = manifestPath
	}

	//message := fmt.Sprintf("Deploying solution %s - %s", manifest.Name, manifest.SolutionVersion)
	message := "Deploying solution"

	log.WithFields(log.Fields{
		"solution-package": solutionBundlePath,
	}).Info(message)

	file, err := os.Open(solutionArchivePath)
	if err != nil {
		log.Fatalf("Failed to open file %q: %v", solutionArchivePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormFile("file", solutionArchivePath)
	if err != nil {
		log.Fatalf("Failed to create form file: %v", err)
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		log.Fatalf("Failed to copy file %q into file writer: %v", solutionArchivePath, err)
	}

	writer.Close()

	headers := map[string]string{
		"stage":        "STABLE",
		"operation":    "UPLOAD",
		"Content-Type": writer.FormDataContentType(),
	}

	var res any

	output.PrintCmdStatus(cmd, fmt.Sprintf("%v\n", message))

	err = api.HTTPPost(getSolutionPushUrl(), body.Bytes(), &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Solution command failed: %v", err)
	}
	// message = fmt.Sprintf("Solution %s - %s was successfully deployed.", manifest.Name, manifest.SolutionVersion)
	message = fmt.Sprintf("Solution bundle %q was successfully deployed.\n", solutionArchivePath)
	output.PrintCmdStatus(cmd, message)
}

func getSolutionPushUrl() string {
	return "solnmgmt/v1beta/solutions"
}

func generateZipNoCmd(sltnPackagePath string) *os.File {
	// splitPath := strings.Split(sltnPackagePath, "/")
	// solutionName := splitPath[len(splitPath)-1]
	solutionName := filepath.Base(sltnPackagePath)
	archiveFileName := fmt.Sprintf("%s.zip", solutionName)
	archive, err := os.Create(archiveFileName)
	if err != nil {
		log.Fatalf("Failed to create a bundle archive %q: %v", archiveFileName, err)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	fsocWorkingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't read the working directory: %v", err)
	}

	solutionRootFolder := filepath.Dir(sltnPackagePath)
	err = os.Chdir(solutionRootFolder)
	if err != nil {
		log.Fatalf("Couldn't switch working folder to solution package folder: %v", err)
	}

	defer func() {
		err := os.Chdir(fsocWorkingDir)
		if err != nil {
			log.Fatalf("Couldn't switch working folder back to the original one: %v", err)
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
		log.Fatalf("Error traversing the solution folder: %v", err)
	}
	zipWriter.Close()

	return archive
}
