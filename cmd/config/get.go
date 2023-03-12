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

func newCmdConfigGet() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "get",
		Short: "Displays the selected context",
		Long:  `Displays the selected context`,
		RunE:  configGetContext,
	}

	cmd.Flags().StringP("output", "o", "", "Output format (human*, json, yaml)")
	cmd.Flags().Bool("unmask", false, "Unmask secrets in output")

	return cmd
}

func configGetContext(cmd *cobra.Command, args []string) error {
	// fail and display help if any arguments were supplied
	if len(args) > 0 {
		return fmt.Errorf("Unexpected argument(s): %v", args)
	}

	// get current context and mask secret values
	ctx := GetCurrentContext()
	if ctx == nil {
		log.Fatalf("There is no current context, use `fsoc config set` to set up a context")
	}
	unmask, err := cmd.Flags().GetBool("unmask")
	if err != nil || !unmask {
		if ctx.Token != "" {
			ctx.Token = "(present)"
		}
		if ctx.RefreshToken != "" {
			ctx.RefreshToken = "(present)"
		}
	}

	// "upgrade" config schema if needed
	if ctx.SecretFile == "" && ctx.CsvFile != "" {
		ctx.SecretFile, ctx.CsvFile = ctx.CsvFile, ""
	}

	// Map fields for human display
	headers := []string{"Name"}
	values := []string{ctx.Name}

	appendIfPresent := func(header, value string) {
		if value != "" {
			headers = append(headers, header)
			values = append(values, value)
		}
	}
	appendIfPresent("Auth Method", ctx.AuthMethod)
	appendIfPresent("URL", ctx.URL)
	appendIfPresent("Tenant", ctx.Tenant)
	appendIfPresent("User ID", ctx.User)
	appendIfPresent("Token", ctx.Token)
	appendIfPresent("Refresh Token", ctx.RefreshToken)
	appendIfPresent("Secret File", ctx.SecretFile)

	output.PrintCmdOutputCustom(cmd, ctx, &output.Table{
		Headers: headers,
		Lines:   [][]string{values},
		Detail:  true,
	})
	return nil
}
