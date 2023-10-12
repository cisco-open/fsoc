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

package config

import (
	"slices"

	"github.com/apex/log"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"

	cfg "github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

const tableFieldSpec = `` +
	`use:.use, ` +
	`name:.name, ` +
	`auth_method:(if .auth_method == "" then "-" else .auth_method end), ` +
	`url:(if .url == "" then "-" else .url end), ` +
	`env_type:(if .env_type == null then "" else .env_type end)`

func newCmdConfigList() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "list",
		Short: "Displays all contexts in an fsoc config file",
		Long:  `Displays all contexts in an fsoc config file`,
		Args:  cobra.NoArgs,
		RunE:  configListContexts,
		Annotations: map[string]string{
			output.TableFieldsAnnotation: tableFieldSpec,
		},
	}

	return cmd
}

func configListContexts(cmd *cobra.Command, args []string) error {
	// determine if detailed human format which requires special handling
	outputFormat, err := cmd.Flags().GetString("output")
	detailView := err == nil && outputFormat == "detail"

	// get names of all profiles (sorted) and which ones are active/default
	profiles := cfg.ListAllContexts()
	activeProfile := cfg.GetCurrentProfileName()  // whether it is the default or not
	defaultProfile := cfg.GetDefaultContextName() // the "current" set in the config file
	slices.Sort(profiles)

	contextList := []map[string]any{}
	for _, name := range profiles {
		context, err := cfg.GetContext(name)
		if err != nil {
			log.Warnf("(bug?) can't find listed context %q: %v; skipping", name, err)
			continue
		}

		// determine how the profile is used (active, default or neither)
		use := ""
		if name == activeProfile {
			use = "Current"
		} else if name == defaultProfile {
			use = "Default"
		}

		// display detailed view using the "get" command displayer
		if detailView {
			outputContext(cmd, context, use) // adds extra line
			continue
		}

		var cMap map[string]any
		err = mapstructure.Decode(context, &cMap)
		if err != nil {
			log.Warnf("(bug?) failed to marshal context %q to mapstructure: %v; skipping", name, err)
			continue
		}

		// add the use indicator and append the entry
		cMap["use"] = use
		contextList = append(contextList, cMap)
	}

	// output data (for all output formats except "detail" which is already displayed)
	if !detailView {
		output.PrintCmdOutput(cmd, struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		}{
			contextList,
			len(contextList),
		})
	}

	return nil
}
