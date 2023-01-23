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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Deploy your solution",
	Long: `This command allows the current tenant specified in the profile to deploy a solution bundle archive into the FSO Platform.

Usage:
	fsoc solution push --solution-bundle=<solution-bundle-archive-path> --stage=[STABLE | TEST]`,
	Args:             cobra.ExactArgs(0),
	Run:              pushSolution,
	TraverseChildren: true,
}

func getSolutionPushCmd() *cobra.Command {
	solutionPushCmd.Flags().
		String("solution-bundle", "", "The fully qualified path name for the solution bundle .zip file")
	_ = solutionPushCmd.MarkFlagRequired("solution-package")
	solutionPushCmd.Flags().
		String("stage", "", "The pipeline stage this solution version should be deployed to [STABLE or TEST]")

	return solutionPushCmd

}

func pushSolution(cmd *cobra.Command, args []string) {
	solutionBundlePath, _ := cmd.Flags().GetString("solution-bundle")
	if solutionBundlePath == "" {
		log.Fatal("solution-bundle cannot be empty, use --solution-bundle=<solution-bundle-archive-path>")
	}
	// if !isSolutionPackageRoot(solutionBundlePath) {
	// 	log.Fatal("solution-bundle path doesn't point to a solution package root folder")
	// }
	// manifest, _ := getSolutionManifest(solutionBundlePath)

	// solutionArchive := generateZip(solutionBundlePath)
	// solutionArchivePath := filepath.Base(solutionArchive.Name())
	solutionArchivePath := solutionBundlePath

	var stage string
	var message string

	if cmd.Flags().Changed("stage") {
		stage, _ = cmd.Flags().GetString("stage")
		if stage != "STABLE" && stage != "TEST" {
			log.Fatalf("%s isn't a valid value for the --stage flag. Possible values are TEST or STABLE")
		}
		// message = fmt.Sprintf("Deploying solution %s - %s as %s version", manifest.Name, manifest.SolutionVersion, stage)
	} else {
		stage = "TEST"
		// message = fmt.Sprintf("Deploying solution %s - %s as TEST version", manifest.Name, manifest.SolutionVersion)
	}

	log.WithFields(log.Fields{
		"solution-package": solutionBundlePath,
		"stage":            stage,
	}).Info(message)

	file, err := os.Open(solutionArchivePath)
	if err != nil {
		log.Fatalf("Failed to open file %s - %v", solutionArchivePath, err.Error())
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormFile("file", solutionArchivePath)
	if err != nil {
		log.Fatalf("Failed to create form file - %v", err.Error())
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		log.Errorf("Failed to copy file %s into file writer - %v", solutionArchivePath, err.Error())
	}

	writer.Close()

	headers := map[string]string{
		"stage":        "STABLE",
		"operation":    "UPLOAD",
		"Content-Type": writer.FormDataContentType(),
	}

	var res any

	output.PrintCmdStatus(cmd, message)

	err = api.HTTPPost(getSolutionPushUrl(), body.Bytes(), &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Solution command failed: %v", err.Error())
	}
	// message = fmt.Sprintf("Solution %s - %s was successfully deployed.", manifest.Name, manifest.SolutionVersion)
	message = fmt.Sprintf("Solution bundle %s was successfully deployed.", solutionArchivePath)
	output.PrintCmdStatus(cmd, message)
}

func getSolutionPushUrl() string {
	return "solnmgmt/v1beta/solutions"
}
