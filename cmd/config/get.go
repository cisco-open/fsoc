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
	"slices"
	"strings"
	"unicode"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	cfg "github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

func newCmdConfigGet() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "get",
		Short: "Displays the selected context",
		Long:  `Displays the selected context`,
		Args:  cobra.NoArgs,
		Run:   configGetContext,
	}

	cmd.Flags().Bool("unmask", false, "Unmask secrets in output")

	return cmd
}

func configGetContext(cmd *cobra.Command, args []string) {
	// get current context and mask secret values
	ctx := cfg.GetCurrentContext()
	if ctx == nil {
		log.Fatalf("There is no current context, use `fsoc config set` to set up a context")
	}

	outputContext(cmd, ctx, "")
}

func outputContext(cmd *cobra.Command, context *cfg.Context, useIndicator string) {
	ctx := *context // shallow copy, to allow modifying top-level fields

	// mask sensitive values (unless --unmask flag)
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
			headers = append(headers, fmt.Sprintf("%14s", header)) // the widest head is 13 chars
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
	appendIfPresent("Environment", humanizeEnvType(ctx.EnvType))
	appendIfPresent("Local Auth", ctx.LocalAuthOptions.String())

	if ctx.SubsystemConfigs != nil && len(ctx.SubsystemConfigs) > 0 {
		// get sorted list of subsystems
		subsystems := maps.Keys(ctx.SubsystemConfigs)
		slices.Sort(subsystems)
		num := len(subsystems)

		// determine the max width of subsystem name (assuming ascii characters)
		width := 0
		for _, name := range subsystems {
			if l := len(name); l > width {
				width = l
			}
		}

		// output
		appendIfPresent("Subsystems", " ") // Add as a header
		for i, name := range subsystems {
			// choose graph character
			graph := '\u251c' // ├ (tree with branch)
			if i == num-1 {
				graph = '\u2514' // └ (tree corner) for the last element
			}

			// produce single line config for the subsystem
			values := formatSubsystemConfig(ctx.SubsystemConfigs[name])
			appendIfPresent(fmt.Sprintf("\t%c %*s", graph, width, name), values) // tab indents unlike spaces
		}
	}

	output.PrintCmdOutputCustom(cmd, ctx, &output.Table{
		Headers: headers,
		Lines:   [][]string{values},
		Detail:  true,
	})
}

func humanizeEnvType(val string) string {
	switch val {
	case "":
		return ""
	case "dev":
		return "Development"
	case "prod":
		return "Production"
	default:
		return fmt.Sprintf("Unrecognized (%q)", val)
	}
}

func formatSubsystemConfig(config map[string]any) string {
	if len(config) == 0 {
		return "(empty)" // shouldn't happen but provide for it if it does
	}

	params := []string{}
	for name, value := range config {
		params = append(params, fmt.Sprintf("%v=%v", name, subsystemValue(value)))
	}

	// TODO: ellide if too long
	return strings.Join(params, " ")
}

func subsystemValue(v any) string {
	// format to string using Go default value formatting
	val := fmt.Sprintf("%v", v)

	// ensure it's printable and quote it if it contain spaces
	if strings.ContainsFunc(val, func(r rune) bool {
		return !unicode.IsPrint(r) // all Go-printable runes, incl. ASCII space
	}) {
		val = "(unprintable)"
	}
	if strings.ContainsFunc(val, func(r rune) bool {
		return unicode.IsSpace(r)
	}) {
		val = fmt.Sprintf("%q", val) // quote the already stringified value
	}

	return val
}
