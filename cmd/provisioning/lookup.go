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
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmdkit"
)

func newCmdLookup() *cobra.Command {

	var lookupCmd = &cobra.Command{
		Use:   "lookup",
		Short: "Lookup for a tenant Id by vanity URL",
		Long: `Check whether tenant exist and return tenant Id if it does.
Tenant lookup doesn't require valid authentication (auth=none) but any configured auth type/tenant will also work.`,
		Example: `  fsoc provisioning lookup MYTENANT.observe.appdynamics.com
  fsoc tep lookup MYTENANT.observe.appdynamics.com`,
		Args:             cobra.ExactArgs(1),
		Run:              lookup,
		TraverseChildren: true,
	}
	return lookupCmd
}

func lookup(cmd *cobra.Command, args []string) {
	vanityUrl := args[0]
	cmdkit.FetchAndPrint(cmd, getTenantLookupUrl(vanityUrl), nil)
}
