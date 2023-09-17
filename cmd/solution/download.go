// Copyright 2023 Cisco Systems, Inc.
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

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionDownloadCmd = &cobra.Command{
	Use:              "download <solution-name>",
	Args:             cobra.MaximumNArgs(1),
	Short:            "Download solution",
	Long:             `This downloads the indicated solution into the current directory. Also see the "fork" command.`,
	Example:          `  fsoc solution download spacefleet`,
	Run:              downloadSolution,
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		config.SetActiveProfile(cmd, args, false)
		return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
	},
}

func getSolutionDownloadCmd() *cobra.Command {
	solutionDownloadCmd.Flags().String("name", "", "name of the solution to download (required)")
	_ = solutionDownloadCmd.Flags().MarkDeprecated("name", "please use argument instead.")

	solutionDownloadCmd.Flags().String("tag", "stable", "tag related to the solution to download")
	return solutionDownloadCmd
}

func downloadSolution(cmd *cobra.Command, args []string) {
	solutionName := getSolutionNameFromArgs(cmd, args, "name")
	solutionNameWithZipExtension := getSolutionNameWithZip(solutionName)
	solutionTagFlag, _ := cmd.Flags().GetString("tag")

	headers := map[string]string{
		"stage":            "STABLE",
		"tag":              solutionTagFlag,
		"solutionFileName": solutionNameWithZipExtension,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(solutionName), &bufRes, &httpOptions); err != nil {
		log.Fatalf("Solution download command failed: %v", err)
	}

	message := fmt.Sprintf("Solution %q with tag %s downloaded successfully.\n", solutionName, solutionTagFlag)
	output.PrintCmdStatus(cmd, message)
}

func getSolutionDownloadUrl(solutionName string) string {
	return fmt.Sprintf("solution-manager/v1/solutions/%s", solutionName)
}

func getSolutionNameWithZip(solutionName string) string {
	return fmt.Sprintf("%s.zip", solutionName)
}
