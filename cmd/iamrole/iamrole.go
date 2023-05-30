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
	"net/url"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

// iamRoleCmd represents the role command group
var iamRoleCmd = &cobra.Command{
	Use:   "iam-role",
	Short: "Manage IAM roles",
	Long: `Manage roles, as part of identity and access management (IAM)

Roles are usually prefixed with the domain, e.g., "iam:observer". Standard FSO roles include "iam:observer", 
"iam:tenantAdmin" and "iam:configManager". Solutions often define their own roles that can be bound to principals
in order to access solution's functionality.

Aliases for this command group include "role", "roles" and "iam-roles".`,
	Aliases: []string{"role", "roles", "iam-role-roles"},
	Example: `
  fsoc iam-roles list
  fsoc roles list
  fsoc role principals <role>
  fsoc role permissions <role>
  `,
	//TODO: add 'create', 'update', 'delete'
	TraverseChildren: true,
}

// Package registration function for the iam-role-binding command root
func NewSubCmd() *cobra.Command {
	cmd := iamRoleCmd

	cmd.AddCommand(newCmdRoleList())
	cmd.AddCommand(newCmdRolePrincipals())
	cmd.AddCommand(newCmdRolePermissions())

	return cmd
}

func getIamRoleUrl(role string, subObj string) string {
	urlBase := "/iam/policy-admin/v1beta2/roles" // [<role>/[<subObj>]]

	elements := []string{}
	if role != "" {
		elements = append(elements, role)
		if subObj != "" {
			elements = append(elements, subObj)
		}
		fullUrl, err := url.JoinPath(urlBase, elements...)
		if err != nil {
			log.Fatalf("(likely bug) failred to append %v to %q: %w", elements, urlBase, err)
		}
		return fullUrl
	}
	return urlBase
}
