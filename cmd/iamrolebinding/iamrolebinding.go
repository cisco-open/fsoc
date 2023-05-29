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
	"encoding/json"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/platform/api"
)

// iamRbCmd represents the role binding command group
var iamRbCmd = &cobra.Command{
	Use:   "iam-role-binding",
	Short: "Manage IAM role bindings",
	Long: `Manage role bindings for a principal, as part of identity and access management (IAM)

Principals can be user principals, service principals or agent principals. For user principals, use the email
address of the user; for service and agent principal, use the principal's ID (client ID).

Roles are usually prefixed with the domain, e.g., "iam:observer". Standard FSO roles include "iam:observer", 
"iam:tenantAdmin" and "iam:configManager". Solutions often define their own roles that can be bound to principals
in order to access solution's functionality.

Aliases for this command group include "rb", "role-binding", "role-bindings" and "iam-role-bindings".`,
	Aliases: []string{"rb", "role-binding", "role-bindings", "iam-role-bindings"},
	Example: `
  fsoc iam-role-bindings list john@example.com
  fsoc rb list john@example.com
  fsoc rb add jill@example.com iam:configManager optimize:optimizationManager
  fsoc rb remove jay@example.com iam:observer iam:tenantAdmin
  `,
	TraverseChildren: true,
}

// Package registration function for the iam-role-binding command root
func NewSubCmd() *cobra.Command {
	cmd := iamRbCmd

	cmd.AddCommand(newCmdRbList())
	cmd.AddCommand(newCmdRbAdd())
	cmd.AddCommand(newCmdRbRemove())

	return cmd
}

func getIamRoleBindingsUrl() string {
	return "iam/policy-admin/v1beta2/principals/roles"
}

func patchRoles(principal string, roles []string, is_add bool) error {
	// choose value to use for roles in the request
	var roleValue any
	if is_add {
		roleValue = true
	} else {
		roleValue = nil
	}

	// prepare request parameter
	requestParams := ManageParameter{Principal: PrincipalParameter{ID: principal}, Roles: map[string]any{}}
	for _, role := range roles {
		requestParams.Roles[role] = roleValue
	}

	// request operation, the API requires the application/json type (bug?)
	// JSONPatch converts to JSON with application/json-patch+json; to override this
	// we must pre-marshal to JSON here
	body, err := json.Marshal(requestParams) // must marshal here in order to be able to supply content-type
	if err != nil {
		log.Fatalf("(bug) failed to marshal to json: %v", err)
	}
	options := api.Options{Headers: map[string]string{"Content-Type": "application/json"}}
	if err := api.JSONPatch(getIamRoleBindingsUrl(), body, nil, &options); err != nil {
		return err
	}

	return nil
}
