// Copyright 2022 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package solution

import (
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionDownloadCmd = &cobra.Command{
	Use:   "download --name=SOLUTION [--stage=STABLE|TEST]",
	Short: "Download solution",
	Long: `This command allows the current tenant specified in the profile to download a solution bundle archive into the current directory or the directory specified in command argument.

    Command: fsoc solution download --name=<solutionName> [--stage=<STABLE|TEST>]

	Usage:
	fsoc solution download  --name=<solution-name> --stage=[STABLE|TEST] `,
	Args:             cobra.ExactArgs(0),
	Run:              downloadSolution,
	TraverseChildren: true,
}

func getSolutionDownloadCmd() *cobra.Command {
	solutionDownloadCmd.Flags().String("name", "", "name of the solution that needs to be downloaded")
	_ = solutionDownloadCmd.MarkFlagRequired("name")
	solutionDownloadCmd.Flags().String("stage", "STABLE", "The pipeline stage[STABLE or TEST] of solution that needs to be downloaded. Default value is STABLE")
	return solutionDownloadCmd
}

func downloadSolution(cmd *cobra.Command, args []string) {
	solutionName, _ := cmd.Flags().GetString("name")
	if solutionName == "" {
		log.Fatalf("solution-name cannot be empty, use --name=<solution-name>")
	}

	var stage string
	var message string

	stage, _ = cmd.Flags().GetString("stage")
	if stage != "STABLE" && stage != "TEST" {
		log.Fatalf("%s isn't a valid value for the --stage flag. Possible values are TEST or STABLE", stage)
	}

	var solutionNameWithZipExtension = getSolutionNameWithZip(solutionName)

	headers := map[string]string{
		"stage":            stage,
		"solutionFileName": solutionNameWithZipExtension,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(solutionName), &bufRes, &httpOptions); err != nil {
		log.Fatalf("Solution download command failed: %v", err.Error())
	}

	message = fmt.Sprintf("Solution bundle %s was successfully downloaded in current directory.\r\n", solutionName)
	output.PrintCmdStatus(message)
}

func getSolutionDownloadUrl(solutionName string) string {
	return fmt.Sprintf("solnmgmt/v1beta/solutions/%s", solutionName)
}

func getSolutionNameWithZip(solutionName string) string {
	return fmt.Sprintf("%s.zip", solutionName)
}
