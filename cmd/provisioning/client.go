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
	"github.com/cisco-open/fsoc/platform/api"
)

func postTenantProvisioningRequest(cmd *cobra.Command, request TenantProvisioningRequest, response *TenantProvisioningResponse) error {
	return postAndPrint(cmd, getTenantsUrl(), request, response)
}

func postLicenseProvisioningRequest(cmd *cobra.Command, tenantId string, request LicenseProvisioningRequest, response *LicenseProvisioningResponse) error {
	return postAndPrint(cmd, getLicenseUrl(tenantId), request, response)
}

func getTenantDetails(tenantId string) (TenantResponse, error) {
	var res TenantResponse
	err := api.JSONGet(getTenantsUrl()+"/"+tenantId, &res, nil)
	if err == nil {
		log.Infof("Workflow %v status: %v", res.Id, res.State)
	}
	return res, err
}

func getWorkflow(tenantId string, workflowId string) (WorkflowResponse, error) {
	var res WorkflowResponse
	err := api.JSONGet(getWorkflowUrl(tenantId, workflowId), &res, nil)
	if err == nil {
		log.Infof("Workflow %v status: %v", res.Id, res.State)
	}
	return res, err
}

func postAndPrint(cmd *cobra.Command, url string, request any, response any) error {
	jsonView := jsonVewEnabled(cmd)
	if jsonView {
		output.PrintCmdOutput(cmd, request)
	}
	errResponse := api.JSONPost(url, request, response, nil)
	if errResponse == nil {
		log.Infof("Successfully get response from : %v.\n", url)
		if jsonView {
			output.PrintCmdOutput(cmd, response)
		}
		return nil
	} else {
		return fmt.Errorf("POST request %v was failed. %v", url, errResponse)
	}
}

func jsonVewEnabled(cmd *cobra.Command) bool {
	outputFormat, err := cmd.Flags().GetString("output")
	jsonView := err == nil && outputFormat == "json"
	return jsonView
}
