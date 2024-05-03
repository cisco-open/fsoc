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
)

const (
	InternalSupportMessage = "To get help, please, write us to #ask-margot internal space and provide us " +
		"tenant Id and workflow Id together with your problem description."
)

// Config defines the subsystem configuration under fsoc
type Config struct {
	ApiVersion *ApiVersion `mapstructure:"apiver,omitempty" fsoc-help:"API version to use for tenant provisioning. The default is \"v1beta\"."`
}

// To make it work provisioning should be registered as subsystem and
// provisioning.GlobalConfig needs to be passed as provisioning config
var GlobalConfig Config

func NewSubCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:              "provisioning",
		Short:            "Tenant provisioning and management",
		Long:             `Use to provision new tenant and troubleshoot provisioning workflow.`,
		Aliases:          []string{"prov", "tep"},
		Example:          `  fsoc provisioning`,
		TraverseChildren: true,
		Hidden:           true,
	}

	cmd.AddCommand(newCmdLookup())
	cmd.AddCommand(newCmdGetTenant())
	cmd.AddCommand(newCmdGetProgress())
	cmd.AddCommand(newCmdDevTenant())
	cmd.AddCommand(newCmdApplyLicense())
	return cmd
}
