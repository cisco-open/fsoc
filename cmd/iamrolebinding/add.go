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
)

// iamRbCmd represents the role binding command group
var iamRbAddCmd = &cobra.Command{
	Use:   "add <principal> [<role>]+",
	Short: "Add roles to a principal",
	Long:  `Add one or more roles to a principal`,
	Example: `
  fsoc rb add john@example.com iam:observer spacefleet:crewMember
  fsoc rb add srv_1ZGdlbcm8NajPxY4o43SNv optimize:optimizationManager`,
	Args: cobra.MinimumNArgs(2),
	Run:  addRoles,
}

// Package registration function for the iam-role-binding command root
func newCmdRbAdd() *cobra.Command {
	return iamRbAddCmd
}

func addRoles(cmd *cobra.Command, args []string) {
	if err := patchRoles(args[0], args[1:], true); err != nil {
		log.Fatal(err.Error())
	}

	output.PrintCmdStatus(cmd, "Roles added successfully.\n")
}
