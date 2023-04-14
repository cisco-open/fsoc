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
	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmdkit"
	"github.com/cisco-open/fsoc/output"
)

var solutionListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.ExactArgs(0),
	Short: "List all solutions available in this tenant",
	Long: `This command list all the solutions that are deployed in the current tenant specified in the profile.

Usage:
	fsoc solution list`,
	Run:              getSolutionList,
	TraverseChildren: true,
	Annotations: map[string]string{
		output.TableFieldsAnnotation:  "name:.data.name, isSystem:.data.isSystem, isSubscribed:.data.isSubscribed, dependencies:.data.dependencies",
		output.DetailFieldsAnnotation: "name:.data.name, isSystem:.data.isSystem, isSubscribed:.data.isSubscribed, dependencies:.data.dependencies, installDate:.createdAt, updateDate:.updatedAt",
	},
}

func getSolutionList(cmd *cobra.Command, args []string) {
	log.Info("Fetching the list of solutions...")

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	// get data and display
	cmdkit.FetchAndPrint(cmd, getSolutionListUrl(), &cmdkit.FetchAndPrintOptions{Headers: headers, IsCollection: true})
}

func getSolutionListUrl() string {
	return "objstore/v1beta/objects/extensibility:solution"
}
