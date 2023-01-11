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
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmdkit"
	"github.com/cisco-open/fsoc/output"
)

var solutionStatusCmd = &cobra.Command{
	Use:   "status [flags]",
	Short: "Get the installation/upload status of a solution",
	Long: `This command provides the ability to see the current installation and upload status of a solution.
	
	Usage:
	fsoc solution status --name <solution-name> --solution-version <optional-solution-version> --status-type [upload | install | all]`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return getSolutionStatus(cmd, args)
	},
	Args:             cobra.ExactArgs(0),
	TraverseChildren: true,
}

func getSolutionStatusCmd() *cobra.Command {
	solutionStatusCmd.Flags().
		String("name", "", "The name of the solution for which you would like to retrieve the upload status")
	_ = solutionStatusCmd.MarkFlagRequired("name")
	solutionStatusCmd.Flags().
		String("solution-version", "", "The version of the solution for which you would like to retrieve the upload status")
	solutionStatusCmd.Flags().
		String("status-type", "", "The status type that you want to see.  This can be one of [upload, install, all] and will default to all if not specified")

	return solutionStatusCmd
}

func getSolutionStatus(cmd *cobra.Command, args []string) error {
	var err error
	var filterQuery string
	cfg := config.GetCurrentContext()

	layerType := "TENANT"
	solutionName, err := cmd.Flags().GetString("name")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "name", err)
	}

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   cfg.Tenant,
	}
	solutionVersion, _ := cmd.Flags().GetString("solution-version")
	statusTypeToFetch, _ := cmd.Flags().GetString("status-type")

	if solutionVersion != "" {
		filterQuery = fmt.Sprintf(`data.solutionName eq "%s" and data.solutionVersion eq "%s"`, solutionName, solutionVersion)
	} else {
		filterQuery = fmt.Sprintf(`data.solutionName eq "%s"`, solutionName)
	}

	query := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(filterQuery))

	if statusTypeToFetch == "upload" {
		output.PrintCmdStatus("Solution Upload Status: \n\n")
		cmdkit.FetchAndPrint(cmd, fmt.Sprintf(getSolutionReleaseUrl(), query), &cmdkit.FetchAndPrintOptions{Headers: headers})
		output.PrintCmdStatus("\n\n")
	} else if statusTypeToFetch == "install" {
		output.PrintCmdStatus("Solution Installation Status: \n\n")
		cmdkit.FetchAndPrint(cmd, fmt.Sprintf(getSolutionInstallUrl(), query), &cmdkit.FetchAndPrintOptions{Headers: headers})
		output.PrintCmdStatus("\n\n")
	} else {
		output.PrintCmdStatus("Solution Upload Status: \n\n")
		cmdkit.FetchAndPrint(cmd, fmt.Sprintf(getSolutionReleaseUrl(), query), &cmdkit.FetchAndPrintOptions{Headers: headers})
		output.PrintCmdStatus("\n\n")
		output.PrintCmdStatus("Solution Installation Status: \n\n")
		cmdkit.FetchAndPrint(cmd, fmt.Sprintf(getSolutionInstallUrl(), query), &cmdkit.FetchAndPrintOptions{Headers: headers})
		output.PrintCmdStatus("\n\n")
	}

	return nil
}

func getSolutionReleaseUrl() string {
	return "objstore/v1beta/objects/extensibility:solutionRelease%s"
}

func getSolutionInstallUrl() string {
	return "objstore/v1beta/objects/extensibility:solutionInstall%s"
}
