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
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type subscriptionStruct struct {
	IsSubscribed bool `json:"isSubscribed"`
}

var solutionSubscribeCmd = &cobra.Command{
	Use:              "subscribe <solution-name>",
	Args:             cobra.MaximumNArgs(1),
	Short:            "Subscribe to a solution",
	Long:             `This command allows the current tenant specified in the profile to subscribe to a solution.`,
	Example:          `	fsoc solution subscribe spacefleet`,
	Run:              subscribeToSolution,
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
	},
}

func getSubscribeSolutionCmd() *cobra.Command {
	solutionSubscribeCmd.Flags().
		String("name", "", "The name of the solution the tenant is subscribing to")
	_ = solutionSubscribeCmd.Flags().MarkDeprecated("name", "please use argument instead.")
	solutionSubscribeCmd.Flags().
		String("tag", "", "The tag related to the solution to subscribe to. This will default to the stable version of the solution if not specified")

	return solutionSubscribeCmd

}

func manageSubscription(cmd *cobra.Command, args []string, isSubscribed bool) {
	solutionName := getSolutionNameFromArgs(cmd, args, "name")

	var message string
	if isSubscribed {
		message = "Subscribing to solution"
	} else {
		message = "Unsubscribing from solution"
	}
	log.WithField("solution", solutionName).Info(message)

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	subscribe := subscriptionStruct{IsSubscribed: isSubscribed}

	var res any
	err := api.JSONPatch(getSolutionSubscribeUrl()+"/"+solutionName, &subscribe, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Solution command failed: %v", err)
	}

	if isSubscribed {
		message = fmt.Sprintf("Tenant %s has successfully subscribed to solution %s\n", layerID, solutionName)
	} else {
		message = fmt.Sprintf("Tenant %s has successfully unsubscribed from solution %s\n", layerID, solutionName)
	}

	output.PrintCmdStatus(cmd, message)
}

func subscribeToSolution(cmd *cobra.Command, args []string) {
	manageSubscription(cmd, args, true)
}

func getSolutionSubscribeUrl() string {
	return "objstore/v1beta/objects/extensibility:solution"
}
