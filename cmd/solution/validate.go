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

type ErrorItem struct {
	Error  string `json:"error"`
	Source string `json:"source"`
}

type Errors struct {
	Items []ErrorItem `json:"items"`
	Total int         `json:"total"`
}

type Result struct {
	Errors Errors `json:"errors"`
	Valid  bool   `json:"valid"`
}

func getSolutionValidateUrl() string {
	return "solnmgmt/v1beta/solutions"
}

func getSolutionValidateCmd() *cobra.Command {
	solutionValidateCmd.Flags().
		String("solution-bundle", "", "The fully qualified path name for the solution bundle .zip file that you want to validate")
	_ = solutionValidateCmd.MarkFlagRequired("solution-package")

	return solutionValidateCmd
}

var solutionValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate your solution package",
	Long: `This command allows the current tenant specified in the profile to upload the specified solution bundle for the purpose of validating its contents

Usage:
	fsoc solution validate --solution-bundle=<solution-bundle-archive-path>`,
	Args:             cobra.ExactArgs(0),
	Run:              validateSolution,
	TraverseChildren: true,
}

func validateSolution(cmd *cobra.Command, args []string) {
	solutionBundlePath, _ := cmd.Flags().GetString("solution-bundle")
	if solutionBundlePath == "" {
		log.Fatalf("solution-bundle cannot be empty, use --solution-bundle=<solution-bundle-archive-path>")
	}

	solutionArchivePath := solutionBundlePath

	var message string

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
		writer.Close()
		log.Fatalf("Failed to copy file %s into file writer - %v", solutionArchivePath, err.Error())
	}

	writer.Close()

	headers := map[string]string{
		"stage":        "STABLE",
		"operation":    "VALIDATE",
		"Content-Type": writer.FormDataContentType(),
	}

	var res Result

	err = api.HTTPPost(getSolutionValidateUrl(), body.Bytes(), &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Solution validate command failed: %v", err.Error())
	}

	if res.Valid {
		message = fmt.Sprintf("Solution bundle %s validated successfully\n", solutionArchivePath)
	} else {
		message = getSolutionValidationErrorsString(res.Errors.Total, res.Errors)
	}
	output.PrintCmdStatus(cmd, message)
}

func getSolutionValidationErrorsString(total int, errors Errors) string {
	var message = fmt.Sprintf("\n%d errors detected while validating solution package\n", total)
	for _, err := range errors.Items {
		message += fmt.Sprintf("Error Content: %+v \n", err)
	}
	message += "\n"

	return message
}
