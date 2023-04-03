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
	Use:   "download <name>",
	Short: "Download solution",
	Long: `This command allows the current tenant specified in the profile to download a solution bundle archive into the current directory or the directory specified in command argument.

Example: fsoc solution download --name=spacefleet`,
	Args:             cobra.MaximumNArgs(1),
	Run:              downloadSolution,
	TraverseChildren: true,
}

func getSolutionDownloadCmd() *cobra.Command {
	solutionDownloadCmd.Flags().String("name", "", "name of the solution to download (required)")
	_ = solutionDownloadCmd.Flags().MarkDeprecated("name", "The --name flag is deprecated, please use argument instead.")
	//_ = solutionDownloadCmd.MarkFlagRequired("name")
	return solutionDownloadCmd
}

func downloadSolution(cmd *cobra.Command, args []string) {
	solutionName, _ := cmd.Flags().GetString("name")
	if len(args) > 0 {
		solutionName = args[0]
	} else {
		if len(solutionName) == 0 {
			log.Fatalf("Solution name cannot be empty")
		}
	}
	var solutionNameWithZipExtension = getSolutionNameWithZip(solutionName)

	headers := map[string]string{
		"stage":            "STABLE",
		"tag":              "stable",
		"solutionFileName": solutionNameWithZipExtension,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(solutionName), &bufRes, &httpOptions); err != nil {
		log.Fatalf("Solution download command failed: %v", err)
	}

	message := fmt.Sprintf("Solution bundle %q downloaded successfully.\n", solutionName)
	output.PrintCmdStatus(cmd, message)
}

func getSolutionDownloadUrl(solutionName string) string {
	return fmt.Sprintf("solnmgmt/v1beta/solutions/%s", solutionName)
}

func getSolutionNameWithZip(solutionName string) string {
	return fmt.Sprintf("%s.zip", solutionName)
}
