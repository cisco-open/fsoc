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
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmdkit"
	"github.com/cisco-open/fsoc/output"
)

// iamRoleListCmd defines the list roles command
var iamRoleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles",
	Long: `List available roles.

Roles can be assigned to principals using the "iam-role-binding" commands.

This command requires a principal with tenant administrator access.

Detail and json/yaml output include role permissions; the table view contains only role names.`,
	Example: `
  fsoc iam-role list
  fsoc iam-roles list 
  fsoc roles list -o json
  fsoc role list -o detail`,
	Args: cobra.NoArgs,
	Run:  listRoles,
	Annotations: map[string]string{
		output.TableFieldsAnnotation:  "id:.id, name:.data.displayName, description:.data.description",
		output.DetailFieldsAnnotation: "id:.id, name:.data.displayName, description:.data.description, permissions:(reduce .data.permissions[].id as $o ([]; . + [$o])), scopes:.data.scopes",
	},
}

// Package registration function for the iam-role-binding command root
func newCmdRoleList() *cobra.Command {
	return iamRoleListCmd
}

func listRoles(cmd *cobra.Command, args []string) {
	cmdkit.FetchAndPrint(cmd, getIamRoleUrl("", ""), &cmdkit.FetchAndPrintOptions{IsCollection: true})
}
