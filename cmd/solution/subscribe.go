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

	"github.com/cisco-open/fsoc/config"
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
		config.SetActiveProfile(cmd, args, false)
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

func subscribeToSolution(cmd *cobra.Command, args []string) {
	manageSubscription(cmd, args, true)
}

func manageSubscription(cmd *cobra.Command, args []string, isSubscribed bool) {
	name := getSolutionNameFromArgs(cmd, args, "name")
	tag, _ := cmd.Flags().GetString("tag")

	var message string
	if isSubscribed {
		message = "Subscribing to solution"
	} else {
		message = "Unsubscribing from solution"
	}
	log.WithFields(
		log.Fields{"solution": name, "tag": tag},
	).Info(message)

	// locate solution object (temporary support for now-deprecated
	// pseudo-isolation)
	objectUrl := locateSolutionUrl(name, tag)

	// reject attempts to unsubscribe from a system solution
	if !isSubscribed {
		isSystemSolution, err := isSystemSolution(objectUrl)
		if err != nil {
			log.Fatalf("Failed to get solution status: %v", err)
		}
		if isSystemSolution {
			log.Fatalf("Cannot unsubscribe tenant from solution %s because it is a system solution", name)
		}
	}

	// update subscription status in solution object at the tenant layer
	var res any
	subscribe := subscriptionStruct{IsSubscribed: isSubscribed}
	err := api.JSONPatch(objectUrl, &subscribe, &res, &api.Options{Headers: getHeaders()})
	if err != nil {
		log.Fatalf("Solution command failed: %v", err)
	}

	// display status message
	tenant := config.GetCurrentContext().Tenant
	if isSubscribed {
		message = fmt.Sprintf("Tenant %s has successfully subscribed to solution %s\n", tenant, name)
	} else {
		message = fmt.Sprintf("Tenant %s has successfully unsubscribed from solution %s\n", tenant, name)
	}
	output.PrintCmdStatus(cmd, message)
}

func locateSolutionUrl(name string, tag string) string {
	// handle stable tag where solution ID == solution name
	if tag == "" || tag == "stable" {
		return getSolutionObjectUrl(name)
	}

	// first, try to find the solution using native isolation
	url := getSolutionObjectUrl(name + "." + tag)
	var data any
	err := api.JSONGet(url, &data, &api.Options{Headers: getHeaders(), ExpectedErrors: []int{404}})
	if err == nil {
		return url
	}

	// next, construct a pseudo-isolated solution's name
	// respecting different rules for dev and prod environments
	// (dev environments don't allow 'dev' tag)
	if config.GetCurrentContext().EnvType == "dev" {
		name = name + tag // no ".dev" is needed for dev environments
	} else if tag == "dev" {
		name = name + ".dev" // no pseudo-isolation, just set the ".dev" suffix
	} else {
		name = name + tag + ".dev" // pseudo-isolation and ".dev" suffix
	}

	return getSolutionObjectUrl(name)
}

func isSystemSolution(objUrl string) (bool, error) {
	var solData struct {
		Data SolutionDef `json:"data"`
	}

	err := api.JSONGet(objUrl, &solData, &api.Options{Headers: getHeaders()})
	if err != nil {
		return false, fmt.Errorf("failed to get solution info: %v", err)
	}

	return solData.Data.IsSystem, nil
}
