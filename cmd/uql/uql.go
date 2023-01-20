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

package uql

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	fsoc "github.com/cisco-open/fsoc/output"
)

var outputFlag string
var rawFlag bool

const (
	availableFormats string = "auto, table, json, yaml"
)

// uqlCmd represents the uql command
var uqlCmd = &cobra.Command{
	Use:   "uql",
	Short: "Perform UQL query",
	Long: `Perform UQL query of MELT data for a tenant.
Parsed response data are displayed in a table by default.
Available output formats: ` + availableFormats + `.
If the "raw" flag is provided, the actual response from the backend API is displayed instead.`,
	Example: `# Get parsed results
  fsoc uql "FETCH id, type, attributes FROM entities(k8s:workload)"`,
	Args:             cobra.ExactArgs(1),
	RunE:             uqlQuery,
	TraverseChildren: true,
}

type format int

const (
	table format = iota
	auto
	raw
	detail
	jsonFormat
	yamlFormat
)

func init() {
	uqlCmd.Flags().StringVarP(&outputFlag, "output", "o", "table", "overridden")
	uqlCmd.Flags().BoolVar(&rawFlag, "raw", false, "Display actual response from the backend. Cannot be used together with the output flag.")
	uqlCmd.MarkFlagsMutuallyExclusive("output", "raw")
	uqlCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		changeFlagUsage(cmd.Parent())
		cmd.Parent().HelpFunc()(cmd, args)
	})
	uqlCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		changeFlagUsage(cmd.Parent())
		return cmd.Parent().UsageFunc()(cmd)
	})
}

func NewSubCmd() *cobra.Command {
	return uqlCmd
}

func uqlQuery(cmd *cobra.Command, args []string) error {
	log.WithFields(log.Fields{"command": cmd.Name(), "args": args[0]}).Info("Performing UQL query")

	output, err := outputFormat(outputFlag, rawFlag)
	if err != nil {
		return err
	}
	queryStr := args[0]
	response, err := runQuery(queryStr)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = printResponse(cmd, response, output)
	if err != nil {
		return err
	}
	return nil
}

func outputFormat(output string, useRaw bool) (format, error) {
	if useRaw {
		return raw, nil
	}
	switch strings.ToLower(output) {
	case "table", "auto":
		return table, nil
	case "json":
		return jsonFormat, nil
	case "yaml":
		return yamlFormat, nil

	default:
		return -1, fmt.Errorf(
			"unsupported output format %s for sub-command uql. This sub-command supports only following formats: [auto, table, json]",
			output,
		)
	}
}

func runQuery(query string) (*Response, error) {
	log.Info("fetch data")

	resp, err := ExecuteQuery(&Query{Str: query}, ApiVersion1)
	if err != nil {
		return nil, err
	}

	if resp.HasErrors() {
		return nil, Errors(resp.Errors())
	}

	return resp, nil
}

func printResponse(cmd *cobra.Command, response *Response, output format) error {
	switch output {
	case table:
		t := makeFlatTable(response)
		cmd.Println(t.Render())
	case jsonFormat:
		json, err := transformForJsonOutput(response)
		if err != nil {
			return err
		}
		return fsoc.PrintJson(cmd, json)
	case yamlFormat:
		json, err := transformForJsonOutput(response)
		if err != nil {
			return err
		}
		return fsoc.PrintYaml(cmd, json)
	case raw:
		fsoc.PrintCmdOutput(cmd, string(*response.raw))
	}
	return nil
}

func changeFlagUsage(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "output" {
			flag.Usage = fmt.Sprintf("output format (%s)", availableFormats)
		}
	})
}
