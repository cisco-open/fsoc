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

package iamrolebinding

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

// iamRbCmd represents the role binding command group
var iamRbListCmd = &cobra.Command{
	Use:   "list <principal>",
	Short: "List role bindings for a principal",
	Long: `List roles bound to a given principal.
	
Detail and json/yaml output include role permissions; the table view contains only role names.

To manage roles, as well as see permissions and principals for a given role, use the "iam-role" commands.

This command requires a principal with tenant administrator access.`,
	Example: `
  fsoc iam-role-bindings list john@example.com
  fsoc rb list john@example.com -o json
  fsoc rb list john@example.com -o detail`,
	Args: cobra.ExactArgs(1),
	Run:  listRoles,
	Annotations: map[string]string{
		output.TableFieldsAnnotation:  "id:.id, name:.data.displayName, description:.data.description",
		output.DetailFieldsAnnotation: "id:.id, name:.data.displayName, description:.data.description, permissions:(reduce .data.permissions[].id as $o ([]; . + [$o])), scopes:.data.scopes",
	},
}

// Package registration function for the iam-role-binding command root
func newCmdRbList() *cobra.Command {
	return iamRbListCmd
}

func listRoles(cmd *cobra.Command, args []string) {
	// get data
	var out any
	requestParams := PrincipalParameter{ID: args[0]}
	if err := api.JSONPost(getIamRoleBindingsUrl(), requestParams, &out, nil); err != nil {
		log.Fatal(err.Error())
	}

	// display with formatting
	output.PrintCmdOutput(cmd, out)
}
