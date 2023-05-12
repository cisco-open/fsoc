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
	"strings"

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

var solutionValidateCmd = &cobra.Command{
	Use:   "validate",
	Args:  cobra.ExactArgs(0),
	Short: "Validate solution",
	Long:  `This command allows the current tenant specified in the profile to upload the solution in the current directory just to validate its contents.  The --stable flag provides a default value of 'stable' for the tag associated with the given solution bundle.  `,
	Example: `  fsoc solution validate
  fsoc solution validate --bump --tag preprod
  fsoc solution validate --tag dev
  fsoc solution validate --stable`,
	Run:              validateSolution,
	TraverseChildren: true,
}

func getSolutionValidateCmd() *cobra.Command {
	solutionValidateCmd.Flags().
		String("tag", "", "Tag to associate with provided solution.  Ensure tag used for validation & upload are same.")

	solutionValidateCmd.Flags().
		Bool("stable", false, "Automatically associate the 'stable' tag with solution bundle to be validate.  This should only be used for validating solutions uploaded with the 'stable' tag.")

	solutionValidateCmd.Flags().
		BoolP("bump", "b", false, "Increment the patch version before validation")

	solutionValidateCmd.Flags().
		StringP("directory", "d", "", "fully qualified path name for the solution bundle that you want to validate (assumes that the solution folder has not been zipped yet)")

	solutionValidateCmd.Flags().
		String("solution-bundle", "", "fully qualified path name for the solution bundle (assumes that the solution folder has already been zipped)")

	solutionValidateCmd.MarkFlagsMutuallyExclusive("directory", "solution-bundle")
	solutionValidateCmd.MarkFlagsMutuallyExclusive("solution-bundle", "bump")

	solutionValidateCmd.MarkFlagsMutuallyExclusive("tag", "stable")

	return solutionValidateCmd
}

func validateSolution(cmd *cobra.Command, args []string) {
	var manifestPath string
	var solutionArchivePath string
	solutionDirectoryRootPath, _ := cmd.Flags().GetString("directory")
	zippedSolutionPath, _ := cmd.Flags().GetString("solution-bundle")
	bumpFlag, _ := cmd.Flags().GetBool("bump")
	solutionTagFlag, _ := cmd.Flags().GetString("tag")
	pushWithStableTag, _ := cmd.Flags().GetBool("stable")
	var solutionBundleAlreadyZipped bool
	var message string
	var solutionName string
	var solutionVersion string

	if pushWithStableTag {
		solutionTagFlag = "stable"
	}

	if solutionDirectoryRootPath == "" && zippedSolutionPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err.Error())
		}
		manifestPath = currentDir
		if !isSolutionPackageRoot(manifestPath) {
			log.Fatalf("No solution manifest found in %q; please run this command in a folder with a solution or use the --solution-bundle flag", manifestPath)
		}

		manifest, err := getSolutionManifest(manifestPath)
		if err != nil {
			log.Fatalf("Failed to read the solution manifest in %q: %v", manifestPath, err)
		}
		if bumpFlag {
			bumpSolutionVersionInManifest(cmd, manifest, manifestPath)
		}
		solutionName = manifest.Name
		solutionVersion = manifest.SolutionVersion
		message = fmt.Sprintf("Validating solution with name %s and version %s and tag %s", solutionName, solutionVersion, solutionTagFlag)
	} else {
		if solutionDirectoryRootPath == "" {
			manifestPath = zippedSolutionPath
		} else {
			manifestPath = solutionDirectoryRootPath
			manifest, err := getSolutionManifest(manifestPath)
			if err != nil {
				log.Fatalf("Failed to read the solution manifest in %q: %v", manifestPath, err)
			}
			if bumpFlag {
				bumpSolutionVersionInManifest(cmd, manifest, manifestPath)
			}
		}
		solutionArchivePath = manifestPath
		message = fmt.Sprintf("Zipping and validating solution specified with path %s with tag %s", solutionArchivePath, solutionTagFlag)
	}

	solutionBundleAlreadyZipped = strings.HasSuffix(solutionArchivePath, ".zip")

	if !solutionBundleAlreadyZipped {
		solutionArchive := generateZip(cmd, manifestPath, "")
		solutionArchivePath = solutionArchive.Name()
	} else {
		message = fmt.Sprintf("Validating already zipped solution with tag %s", solutionTagFlag)
	}

	log.Infof(message)
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
		writer.Close()
		log.Fatalf("Failed to copy file %q into file writer: %v", solutionArchivePath, err)
	}

	writer.Close()

	headers := map[string]string{
		"tag":          solutionTagFlag,
		"operation":    "VALIDATE",
		"Content-Type": writer.FormDataContentType(),
	}

	var res Result

	err = api.HTTPPost(getSolutionValidateUrl(), body.Bytes(), &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Solution validate request failed: %v", err)
	}

	if res.Valid {
		message = fmt.Sprintf("Solution bundle with path %s and tag %s was successfully validated.\n", solutionArchivePath, solutionTagFlag)
	} else {
		message = getSolutionValidationErrorsString(res.Errors.Total, res.Errors)
	}
	output.PrintCmdStatus(cmd, message)
	if !res.Valid {
		log.Fatalf("%d error(s) found while validating the solution", res.Errors.Total)
	}
}

func getSolutionValidationErrorsString(total int, errors Errors) string {
	var message = fmt.Sprintf("\n%d errors detected while validating solution package\n", total)
	for _, err := range errors.Items {
		message += fmt.Sprintf("- Error Content: %+v\n", err)
	}
	message += "\n"

	return message
}
