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

func newCmdLookup() *cobra.Command {

	var lookupCmd = &cobra.Command{
		Use:              "lookup-tenant",
		Short:            "Lookup for a tenant by vanity URL",
		Long:             `Check whether tenant exist and return tenant Id if it does.`,
		Example:          `  provisioning lookup-tenant --vanityUrl=fsoc-test.saas.appd-test.com`,
		Args:             cobra.ExactArgs(0),
		Run:              lookup,
		TraverseChildren: true,
	}

	vanityUrlFlag := "vanityUrl"
	lookupCmd.Flags().String(vanityUrlFlag, "", "Vanity URL without a scheme. Provide only domain part.")
	_ = lookupCmd.MarkFlagRequired(vanityUrlFlag)

	return lookupCmd
}

func lookup(cmd *cobra.Command, args []string) {
	vanityUrl, _ := cmd.Flags().GetString("vanityUrl")

	log.WithFields(log.Fields{"command": cmd.Name(), "vanityUrl": vanityUrl}).Info("Provisioning group command")

	response := callBackend(vanityUrl)

	output.PrintCmdOutput(cmd, response)
}

func callBackend(vanityUrl string) any {
	var response any
	err := api.JSONGet(fmt.Sprintf("/provisioning/v1beta/tenants/lookup/vanityUrl/%s", vanityUrl), &response, nil)
	if err != nil {
		log.Fatalf("Tenant lookup failed with %v", err.Error())
	}
	return response
}
