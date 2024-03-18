// Copyright 2024 Cisco Systems, Inc.
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
	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionDescribeCmd = &cobra.Command{
	Use:     "describe <solution-name>",
	Args:    cobra.MaximumNArgs(1),
	Short:   "Describe solution",
	Long:    `Obtain metadata about a solution`,
	Example: `  fsoc solution describe spacefleet`,
	Run:     solutionDescribe,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		config.SetActiveProfile(cmd, args, false)
		return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
	},
}

func getSolutionDescribeCmd() *cobra.Command {
	solutionDescribeCmd.Flags().
		String("solution", "", "The name of the solution to describe")
	_ = solutionDescribeCmd.Flags().MarkDeprecated("solution", "please use argument instead.")

	return solutionDescribeCmd
}

func solutionDescribe(cmd *cobra.Command, args []string) {
	solution := getSolutionNameFromArgs(cmd, args, "solution")

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	log.WithField("solution", solution).Info("Getting solution details")
	var res Solution
	err := api.JSONGet(getSolutionObjectUrl(solution), &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Cannot get solution details: %v", err)
	}
	output.PrintCmdOutput(cmd, res)
}
