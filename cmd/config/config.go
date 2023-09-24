// Copyright 2022 Cisco Systems, Inc.
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

// Package config provides access to fsoc configuration, both to obtain the current
// configuration and to incrementally or fully modify the configuration.
// The fsoc configuration has two dimension: a config file and a context within the config file.
// Each config file contains one or more contexts plus a setting indicating which of them is the current one.
package config

import (
	"github.com/spf13/cobra"

	cfg "github.com/cisco-open/fsoc/config"
)

// Package registration function for the config root command
func NewSubCmd() *cobra.Command {
	// cmd represents the config sub command root
	var cmd = &cobra.Command{
		Use:   "config SUBCOMMAND [options]",
		Short: "Configure fsoc",
		Long:  `View and modify fsoc config files and contexts`,
		Example: `  fsoc config list
  fsoc config set auth=oauth url=https://mytenant.observe.appdynamics.com
  fsoc config set auth=service-principal secret-file=my-svc-principal.json --profile ci
  fsoc config get -o yaml
  fsoc config use ci
  fsoc config delete ci`,
		TraverseChildren: true,
	}

	cmd.AddCommand(newCmdConfigGet())
	cmd.AddCommand(newCmdConfigSet())
	cmd.AddCommand(newCmdConfigUse())
	cmd.AddCommand(newCmdConfigList())
	cmd.AddCommand(newCmdConfigDelete())
	cmd.AddCommand(newCmdConfigShowFields())

	return cmd
}

// GetAuthMethodsStringList returns the list of authentication methods as strings (for join, etc.)
func GetAuthMethodsStringList() []string {
	return []string{
		cfg.AuthMethodNone,
		cfg.AuthMethodOAuth,
		cfg.AuthMethodServicePrincipal,
		cfg.AuthMethodAgentPrincipal,
		cfg.AuthMethodJWT,
		cfg.AuthMethodLocal,
	}
}

func validArgsAutocomplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return cfg.ListContexts(toComplete), cobra.ShellCompDirectiveDefault
}
