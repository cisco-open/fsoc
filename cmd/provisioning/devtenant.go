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
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func newCmdDevTenant() *cobra.Command {

	var devTenantCmd = &cobra.Command{
		Use:   "dev-tenant",
		Short: "Simplified tenant provisioning command.",
		Long:  `Create tenant with predefined default values and trial license. This command is intended to be used for testing purposes.`,
		Example: `
  fsoc provisioning dev-tenant MYTENANT --email=tenantAdmin@cisco.com --valid-to=2023-12-18 --tokens=3`,
		Args:             cobra.ExactArgs(1),
		Run:              provisionDevTenant,
		TraverseChildren: true,
	}
	devTenantCmd.Flags().String("email", "", "Tenant administrator email.")
	_ = devTenantCmd.MarkFlagRequired("email")
	devTenantCmd.Flags().String("valid-to", "", "Date of the end of new license validity. Format [yyyy-mm-dd]. (By default for 1 year.)")
	devTenantCmd.Flags().Uint32("tokens", 100, "Amount of tokens for ingestion in millions")
	return devTenantCmd
}

func provisionDevTenant(cmd *cobra.Command, args []string) {
	tenantName := args[0]
	email, _ := cmd.Flags().GetString("email")
	validTo, _ := cmd.Flags().GetString("valid-to")
	tokens, _ := cmd.Flags().GetUint32("tokens")

	tenantProvisioningRequest := buildTenantRequest(tenantName, email)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Sending tenant provisioning request for account name: %v.\n", tenantProvisioningRequest.Account.Name))

	var response TenantProvisioningResponse
	err := postTenantProvisioningRequest(cmd, tenantProvisioningRequest, &response)
	if err != nil {
		log.Fatalf("Tenant provisioning failed: %v \n", err)
		return
	}
	tenantId := response.TenantId
	workflowId := response.WorkflowId
	output.PrintCmdStatus(cmd, fmt.Sprintf("Tenant provisioning request sucessfully sent: tenantId=%v, workflowId=%v.\n",
		tenantId, workflowId))
	timeout := 8 * time.Minute
	workflow, progressErr := workflowProgressPolling(cmd, tenantId, workflowId, timeout)
	if progressErr != nil {
		log.Fatalf("Cannot check workflow status because of error: %v", err)
		return
	}

	if workflow.State == successState {
		output.PrintCmdStatus(cmd, fmt.Sprintf("Tenant has been succesfully provisioned: tenantId=%v, workflowId=%v.\n",
			tenantId, workflowId))
		provisionNewLicense(cmd, tenantId, validTo, tokens)
	} else {
		if isFinalState(workflow) {
			output.PrintCmdStatus(cmd, fmt.Sprintf("Tenant provisioning failed, workflow state is: %v.\n", workflow.StateDescription))
		} else {
			output.PrintCmdStatus(cmd, fmt.Sprintf("Tenant provisioning took too more than %v, "+
				"current workflow state is: %v. \nPlease use [ provisioning get-progress --tenantId=%v --workflowId=%v ] "+
				"command to verify further progress.\n", timeout, workflow.StateDescription, tenantId, workflowId))
		}
		output.PrintCmdStatus(cmd, InternalSupportMessage)
	}
}

func buildTenantRequest(tenantName string, email string) TenantProvisioningRequest {
	var request = TenantProvisioningRequest{}
	request.Name = tenantName
	request.Account = Account{
		Name: "FSOC Dev User Account",
	}
	request.Organization = Organization{
		Name: "FSOC Dev User Organization",
	}
	request.User = User{
		Email: email,
	}
	return request
}
