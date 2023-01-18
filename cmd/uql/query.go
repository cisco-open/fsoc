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
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query QUERY",
	Short: "Perform a raw UQL query",
	Long: `Perform a raw UQL query over the tenant's MELT data. It will return the full result obtained
from UQE.`,
	Example:          `  fsoc uql query "FETCH id, type, attributes FROM entities(k8s:workload)"`,
	Args:             cobra.ExactArgs(1),
	Run:              rawQuery,
	TraverseChildren: true,
}

func init() {
	uqlCmd.AddCommand(queryCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// reportCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	//reportCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type queryStruct struct {
	Query string `json:"query"`
}

func rawQuery(cmd *cobra.Command, args []string) {
	log.WithFields(log.Fields{"command": cmd.Name(), "query": args[0]}).Info("UQL command")

	queryStr := args[0]

	resp, err := sendQuery(queryStr)
	if err != nil {
		log.Fatal(err.Error())
	}

	// display output
	output.PrintCmdOutput(cmd, resp) // no human format for raw uql
}

func sendQuery(queryStr string) (any, error) {
	// create a JSON query payload
	query := queryStruct{Query: queryStr}

	// call UQE to perform query
	//@@ TODO: support for specifying the API version in a feature flag for uql
	var resp any
	err := api.JSONPost("/monitoring/v1/query/execute", &query, &resp, nil)
	if err != nil {
		return nil, fmt.Errorf("Query failed: %v", err)
	}
	return resp, nil
}
