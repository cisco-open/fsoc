// Copyright 2023 Cisco Systems, Inc.
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

package iamrole

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

// iamRolePrincipalsCmd represents the role principals list command
var iamRolePrincipalsCmd = &cobra.Command{
	Use:   "principals <role>",
	Short: "List principals bound to a role",
	Long: `List all principals that are bound to a particular role.

Role bindings can be managed with the "iam-role-binding" commands.

This command requires a principal with tenant administrator access.`,
	Example: `
  fsoc iam-role principals spacefleet:commandingOfficer
  fsoc role principals iam:agent -o json`,
	Args: cobra.ExactArgs(1),
	Run:  listPrincipals,
	Annotations: map[string]string{
		output.TableFieldsAnnotation: "id:.id, type:.type",
	},
}

// Package registration function for the iam-role-binding command root
func newCmdRolePrincipals() *cobra.Command {
	return iamRolePrincipalsCmd
}

type principalEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type principalsResponse struct {
	Total      uint             `json:"total"`
	Principals []principalEntry `json:"principals"`
}

type principalsCollection struct {
	Total uint             `json:"total"`
	Items []principalEntry `json:"items"`
}

func listPrincipals(cmd *cobra.Command, args []string) {
	// note: the API is not compliant with collections/pagination, so collect as a single request
	var out principalsResponse
	err := api.JSONGet(getIamRoleUrl(args[0], "principals"), &out, nil)
	if err != nil {
		log.Fatal(err.Error())
	}

	// reflow into a collection structure
	data := principalsCollection{Total: out.Total, Items: out.Principals}
	output.PrintCmdOutput(cmd, data)
}
