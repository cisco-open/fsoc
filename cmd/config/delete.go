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
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	cfg "github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

func newCmdConfigDelete() *cobra.Command {

	var cmd = &cobra.Command{
		Use:               "delete CONTEXT_NAME",
		Short:             "Delete a context from the fsoc config file",
		Long:              `Delete a context from the fsoc config file`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: validArgsAutocomplete,
		Run:               configDeleteContext,
	}

	return cmd
}

func configDeleteContext(cmd *cobra.Command, args []string) {
	profile := args[0]
	if err := cfg.DeleteContext(profile); err != nil {
		log.Fatalf("%v", err)
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Deleted profile %q\n", profile))
}
