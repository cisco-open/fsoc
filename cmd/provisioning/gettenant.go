// Copyright 2024 Cisco Systems, Inc.
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

package provisioning

import (
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func newCmdGetTenant() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "get-tenant",
		Short: "Get tenant with related workflows.",
		Example: `
  fsoc provisioning get-tenant --tenantId=6d3cd2c0-a2b6-49b9-a41b-f2966eec87ec`,
		Aliases:          []string{},
		Args:             cobra.ExactArgs(0),
		Run:              getTenant,
		TraverseChildren: true,
	}
	cmd.Flags().String("tenantId", "", "Tenant Id.")
	_ = cmd.MarkFlagRequired("tenantId")
	return cmd
}

func getTenant(cmd *cobra.Command, _ []string) {
	tenantId, _ := cmd.Flags().GetString("tenantId")
	tenant, err := getTenantDetails(tenantId)
	if err == nil {
		output.PrintCmdStatus(cmd, "Tenant found:\n")
		output.PrintCmdOutput(cmd, tenant)
	} else {
		log.Fatal(fmt.Sprintf("Failed to get tenant details because of error. %v", err))
	}
}
