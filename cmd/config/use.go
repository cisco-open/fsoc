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

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func newCmdConfigUse() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "use CONTEXT_NAME",
		Short: "Set the current context in an fsoc config file",
		Long:  `Set the current context in an fsoc config file`,
		Args:  cobra.MaximumNArgs(1),
		Run:   configUseContext,
	}

	return cmd
}

func configUseContext(cmd *cobra.Command, args []string) {
	var newContext string

	// determine which profile to use (supporting --profile for backward compatibility)
	if cmd.Flags().Changed("profile") {
		newContext, _ = cmd.Flags().GetString("profile")
		if len(args) > 0 {
			_ = cmd.Usage()
			log.Fatalf("The context can be specified either as an argument or as a flag but not as both")
		} else {
			log.Warn("using the --profile flag for this command is deprecated; please, use just the profile name as an argument")
		}
	}
	if len(args) > 0 {
		newContext = args[0]
	}
	if newContext == "" { // also handles empty string argument
		_ = cmd.Usage()
		log.Fatalf("Missing the context name argument")
	}

	// look up selected context
	contextExists := false
	cfg := getConfig()
	for _, c := range cfg.Contexts {
		if c.Name == newContext {
			contextExists = true
			break
		}
	}
	if !contextExists {
		log.Fatalf("no context exists with the name: %q", newContext)
	}

	// update config file
	updateConfigFile(map[string]interface{}{"current_context": newContext})
	output.PrintCmdStatus(cmd, fmt.Sprintf("Switched to context %q\n", newContext))
}
