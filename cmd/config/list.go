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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func newCmdConfigList() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "list",
		Short: "Displays all contexts in an fsoc config file",
		Long:  `Displays all contexts in an fsoc config file`,
		RunE:  configListContexts,
	}

	return cmd
}

var headers = []string{
	"Use", "Name", "Auth Method", "URL", "User",
}

func configListContexts(cmd *cobra.Command, args []string) error {
	// fail and display help if any arguments were supplied
	if len(args) > 0 {
		return fmt.Errorf("Unexpected argument(s): %v", args)
	}

	activeProfile := GetCurrentProfileName() // whether it is current or not

	// read all contexts from the config file
	cfg := getConfig()

	var contexts [][]string
	for _, c := range cfg.Contexts {
		current := ""
		if c.Name == activeProfile {
			current = "Current"
		} else if c.Name == cfg.CurrentContext {
			current = "Default"
		}
		// credentials := c.SecretFile
		// if credentials == "" && c.CsvFile != "" {
		// 	credentials = c.CsvFile
		// }
		contexts = append(contexts, []string{current, c.Name, c.AuthMethod, c.URL, c.User})
	}
	filter := output.CreateFilter("", []int{})
	output.PrintCmdOutputCustom(cmd, cfg, &output.Table{
		Headers: headers,
		Lines:   contexts,
		Detail:  false}, filter)

	return nil
}
