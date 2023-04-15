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

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionUnsubscribeCmd = &cobra.Command{
	Use:              "unsubscribe <solution-name>",
	Args:             cobra.MaximumNArgs(1),
	Short:            "Unsubscribe from a solution",
	Long:             `This command allows the current tenant specified in the profile to unsubscribe from a solution.`,
	Example:          `  fsoc solution unsubscribe spacefleet`,
	Run:              unsubscribeFromSolution,
	TraverseChildren: true,
}

func getUnsubscribeSolutionCmd() *cobra.Command {
	solutionUnsubscribeCmd.Flags().
		String("name", "", "The name of the solution the tenant is unsubscribing from")
	_ = solutionUnsubscribeCmd.Flags().MarkDeprecated("name", "please use argument instead.")

	return solutionUnsubscribeCmd

}

func unsubscribeFromSolution(cmd *cobra.Command, args []string) {
	solutionName := getSolutionNameFromArgs(cmd, args, "name")

	isSystemSolution, err := isSystemSolution(solutionName)
	if err != nil {
		log.Fatalf("Failed to get solution status: %v", err)
	}
	if isSystemSolution {
		log.Fatalf("Cannot unsubscribe tenant from solution %s because it is a system solution", solutionName)
	} else {
		manageSubscription(cmd, args, false)
	}
}

func isSystemSolution(solutionName string) (bool, error) {
	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	var solData struct {
		Data SolutionDef `json:"data"`
	}

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	getSolutionUrl := fmt.Sprintf(getSolutionListUrl()+"/%s", solutionName)
	err := api.JSONGet(getSolutionUrl, &solData, &api.Options{Headers: headers})
	if err != nil {
		return false, fmt.Errorf("Failed to get solution info: %v", err)
	}

	return solData.Data.IsSystem, nil
}
