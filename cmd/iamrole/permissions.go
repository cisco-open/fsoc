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

// iamRolePermissionsCmd represents the role permissions command
var iamRolePermissionsCmd = &cobra.Command{
	Use:   "permissions <role>",
	Short: "List permissions that a role provides",
	Long: `List all permissions that a given role provides.

This command requires a principal with tenant administrator access.

The json/yaml output include the actions and resources for each permissions; the table and detail views include the permission names.`,
	Example: `
  fsoc iam-role permissions iam:observer
  fsoc role permissions spacefleet:commandingOfficer
  fsoc role permissions john@example.com -o json
  fsoc role permissions john@example.com -o detail`,
	Args: cobra.ExactArgs(1),
	Run:  listPermissions,
	Annotations: map[string]string{
		output.TableFieldsAnnotation: "id:.id, name:.data.displayName, description:.data.description",
	},
}

// Package registration function for the iam-role-binding command root
func newCmdRolePermissions() *cobra.Command {
	return iamRolePermissionsCmd
}

func listPermissions(cmd *cobra.Command, args []string) {
	cmdkit.FetchAndPrint(cmd, getIamRoleUrl(args[0], "permissions"), &cmdkit.FetchAndPrintOptions{IsCollection: true})
}
