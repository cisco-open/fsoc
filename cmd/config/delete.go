// Copyright 2023 Cisco Systems, Inc.
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
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func newCmdConfigDelete() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "delete CONTEXT_NAME",
		Short: "Delete a context from the fsoc config file",
		Long:  `Delete a context from the fsoc config file`,
		Args:  cobra.ExactArgs(1),
		Run:   configDeleteContext,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return ListContexts(toComplete), cobra.ShellCompDirectiveDefault
		},
	}

	return cmd
}

func configDeleteContext(cmd *cobra.Command, args []string) {
	profile := args[0]

	// Make sure profile exists
	ctx := getContext(profile)
	if ctx == nil {
		log.Fatalf("Could not find profile %q", profile)
	}

	cfg := getConfig()
	profileIdx := -1
	for idx, c := range cfg.Contexts {
		if profile == c.Name {
			profileIdx = idx
			break
		}
	}

	if profileIdx == -1 {
		log.Fatalf("(possible bug) Could not find profile %q", profile)
	}

	// Delete context from config
	newContexts := append(cfg.Contexts[:profileIdx], cfg.Contexts[profileIdx+1:]...)
	update := map[string]interface{}{"contexts": newContexts}
	if cfg.CurrentContext == profile {
		var newCurrentContext string
		if len(newContexts) > 0 {
			newCurrentContext = newContexts[0].Name
		} else {
			newCurrentContext = DefaultContext
		}
		update["current_context"] = newCurrentContext
		log.Infof("Setting current profile to %q", newCurrentContext)
	}

	updateConfigFile(update)
	log.Infof("Deleted prfofile %q", profile)
}
