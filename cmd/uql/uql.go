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

	"github.com/apex/log"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

// uqlCmd represents the uql command
var uqlCmd = &cobra.Command{
	Use:   "uql",
	Short: "Perform UQL query",
	Long: `Perform UQL query of MELT data for a tenant. By default (without a subcommand), parsed data
results from UQE will be returned. Raw queries are supported with full JSON output via 
the query subcommand. Utility queries and other output formats can be supported in the future.`,
	Example: `# Get parsed results
  fsoc uql "FETCH id, type, attributes FROM entities(k8s:workload)"

# Get raw UQL results
  fsoc uql query "FETCH id, type, attributes FROM entities(k8s:workload)"`,
	Args:             cobra.ExactArgs(1),
	Run:              uqlQuery,
	TraverseChildren: true,
}

func init() {
	uqlCmd.PersistentFlags().StringP("output", "o", "json", "Output format (human*, json, yaml)")

}

func NewSubCmd() *cobra.Command {
	return uqlCmd
}

func uqlQuery(cmd *cobra.Command, args []string) {
	log.WithFields(log.Fields{"command": cmd.Name(), "args": args[0]}).Info("UQL command")

	queryStr := args[0]
	data, err := GatherUql(queryStr)
	if err != nil {
		log.Fatal(err.Error())
	}

	output.PrintCmdOutput(cmd, data) // TODO if possible: create a human-readable output (table)
}

func GatherUql(queryStr string) (UQEData, error) {
	// Call UQE and parse to only return result data
	resp, err := sendQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("Query failed: %v", err)
	}
	results := []DataResponse{}
	var data []UQEData

	// convert from map to struct
	err = mapstructure.Decode(resp, &results)
	if err != nil {
		return nil, fmt.Errorf("UQL parsing failed: %v (hint: use a raw query to see the query response)", err)
	}
	// only collect data
	for _, r := range results {
		if r.Type == "data" {
			data = append(data, r.Data)
		}
	}
	// convert to [][]interface{}
	var flattenedRes UQEData
	for _, res := range data {
		resArr := flattenUQEData(res)
		if resArr != nil {
			flattenedRes = append(flattenedRes, resArr)
		}
	}
	return flattenedRes, nil
}
