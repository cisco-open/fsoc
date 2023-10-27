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
	"net/url"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmdkit"
	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionListCmd = &cobra.Command{
	Use:   "list [--subscribed | --unsubscribed]",
	Args:  cobra.ExactArgs(0),
	Short: "List all solutions available in this tenant",
	Long:  `This command list all the solutions that are deployed in the current tenant specified in the profile.`,
	Example: `  fsoc solution list
  fsoc solution list -o json`,
	Run:              getSolutionList,
	TraverseChildren: true,
	Annotations: map[string]string{
		output.TableFieldsAnnotation:  "name:.data.name, tag:.data.tag, isSystem:.data.isSystem, isSubscribed:.data.isSubscribed, dependencies:.data.dependencies",
		output.DetailFieldsAnnotation: "name:.data.name, tag:.data.tag, isSystem:.data.isSystem, isSubscribed:.data.isSubscribed, dependencies:.data.dependencies, installDate:.createdAt, updateDate:.updatedAt",
	},
}

func getSolutionListCmd() *cobra.Command {
	solutionListCmd.Flags().
		Bool("subscribed", false, "Use this to only see solutions that you are subscribed to")
	solutionListCmd.Flags().
		Bool("unsubscribed", false, "Use this to only see solutions that you are unsubscribed to")

	solutionListCmd.MarkFlagsMutuallyExclusive("subscribed", "unsubscribed")

	return solutionListCmd

}

func getSolutionList(cmd *cobra.Command, args []string) {
	log.Info("Fetching the list of solutions...")
	// get subscribe and unsubscribe flags
	subscribed := cmd.Flags().Lookup("subscribed").Changed
	unsubscribed := cmd.Flags().Lookup("unsubscribed").Changed

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	// get data and display
	solutionBaseURL := getSolutionObjectUrl("")
	var filters []string
	if subscribed {
		filters = []string{"filter=" + url.QueryEscape("data.isSubscribed eq true")}
	} else if unsubscribed {
		filters = []string{"filter=" + url.QueryEscape("data.isSubscribed ne true")}
	}
	cmdkit.FetchAndPrint(cmd, solutionBaseURL, &cmdkit.FetchAndPrintOptions{Headers: headers, IsCollection: true, Filters: filters})
}

func getSolutionNames(prefix string) (names []string) {
	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   config.GetCurrentContext().Tenant,
	}
	httpOptions := &api.Options{Headers: headers}

	var result api.CollectionResult[Solution]
	err := api.JSONGetCollection[Solution](getSolutionObjectUrl(""), &result, httpOptions)
	if err != nil {
		return names
	}

	for _, s := range result.Items {
		if strings.HasPrefix(s.ID, prefix) {
			names = append(names, s.ID)
		}
	}
	return names
}
