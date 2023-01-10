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
)

var outputFlag string

// uqlCmd represents the uql command
var uqlCmd = &cobra.Command{
	Use:   "uql",
	Short: "Perform UQL query",
	Long: `Perform UQL query of MELT data for a tenant.
Parsed response data are displayed in a table.
Other output formats can be supported in the future.`,
	Example: `# Get parsed results
  fsoc uql "FETCH id, type, attributes FROM entities(k8s:workload)"`,
	Args:             cobra.ExactArgs(1),
	RunE:             uqlQuery,
	TraverseChildren: true,
}

type format int

const (
	table format = iota
)

func init() {
	uqlCmd.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table)")

}

func NewSubCmd() *cobra.Command {
	return uqlCmd
}

func uqlQuery(cmd *cobra.Command, args []string) error {
	log.WithFields(log.Fields{"command": cmd.Name(), "args": args[0]}).Info("Performing UQL query")

	output, err := outputFormat(outputFlag)
	if err != nil {
		return err
	}
	queryStr := args[0]
	response, err := runQuery(queryStr)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = printResponse(response, output)
	if err != nil {
		return err
	}
	return nil
}

func outputFormat(output string) (format, error) {
	switch strings.ToLower(output) {
	case "table":
		return table, nil
	default:
		return -1, fmt.Errorf(
			"unsupported output format %s for sub-command uql. This sub-command supports only following formats: [table]",
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

func printResponse(response *Response, output format) error {
	switch output {
	case table:
		t := MakeFlatTable(response)
		fmt.Println(t.Render())
	}
	return nil
}
