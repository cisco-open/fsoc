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
		Use:   "use --profile CONTEXT_NAME",
		Short: "Set the current context in an fsoc config file",
		Long:  `Set the current context in an fsoc config file`,
		Args:  cobra.ExactArgs(0),
		Run:   configUseContext,
	}

	return cmd
}

func configUseContext(cmd *cobra.Command, args []string) {
	newContext := GetCurrentProfileName()
	contextExists := false

	cfg := getConfig()
	for _, c := range cfg.Contexts {
		if c.Name == newContext {
			contextExists = true
			break
		}
	}

	if !contextExists {
		log.Fatalf("no context exists with the name: \"%s\"", newContext)
	}

	updateConfigFile(map[string]interface{}{"current_context": newContext})
	output.PrintCmdStatus(cmd, fmt.Sprintf("Switched to context \"%s\"\n", newContext))
}
