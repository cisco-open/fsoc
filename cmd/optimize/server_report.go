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

package optimize

import (
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/cmdkit"
)

// reportCmd represents the report command
var serverReportCmd = &cobra.Command{
	Use:   "serverreport",
	Short: "Obtain workload efficiency profile",
	Long: `Obtain a workload report from the efficiency and risk profiler. The profiler will determine
opportunities, recommendations, cautions, and blockers for optimizing the workload. By default, called
with workload name, but can be called via id (full or partial) by setting --type=id. Can only generate
reports with deployment type workloads.`,
	Example: `  fsoc optimize report "frontend"
  fsoc optimize report "H4uJtAA+MciSmzMDFeAueA" --type=id
  fsoc optimize report "k8s:deployment:H4uJtAA+MciSmzMDFeAueA" --type=id`,
	Args:             cobra.ExactArgs(1),
	RunE:             workloadReport,
	TraverseChildren: true,
}

func init() {
	optimizeCmd.AddCommand(serverReportCmd)
	serverReportCmd.Flags().StringP("type", "t", "name", "Specifier for workload (name or id)")
}

func workloadReport(cmd *cobra.Command, args []string) error {

	log.WithFields(log.Fields{"targetWorkload": args[0]}).Info("requesting workload report")

	byType, err := cmd.Flags().GetString("type")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "by", err)
	}

	targetWorkload := args[0]

	// fetch workload ID if calling via name
	var workloadId *string
	if byType == "id" {
		if strings.Contains(targetWorkload, "k8s:deployment") {
			formattedWorkload := formatAppDIdentifier(targetWorkload)
			workloadId = &formattedWorkload
		} else {
			workloadId = &targetWorkload
		}
	} else if byType == "name" {
		workloadId, err = getWorkloadId(targetWorkload)
		if err != nil {
			log.Fatalf("error retrieving workload ID: %v", err)
		}
	}
	encodedWorkloadId := base32.StdEncoding.EncodeToString([]byte(*workloadId))

	// fetch data and display
	cmdkit.FetchAndPrint(cmd, "/ignite/v1beta/reports/workloads/"+encodedWorkloadId, nil)
	return nil
}

func getWorkloadId(workloadName string) (*string, error) {

	// UQL query to retrieve ID via workload.name attribute
	queryStr := fmt.Sprintf("SINCE -7d FETCH id FROM entities(k8s:workload)[isActive = true][attributes(k8s.workload.name) = '%s']", workloadName)

	response, err := uql.ClientV1.ExecuteQuery(&uql.Query{Str: queryStr})
	if err != nil {
		return nil, err
	}
	mainDataSet := response.Main()
	if mainDataSet == nil {
		return nil, fmt.Errorf("nil main data set when querying for workloads with name %q", workloadName)
	}
	workloadIds := columnValues(mainDataSet, 0)

	// Check if either none or multiple workload IDs found
	if len(workloadIds) < 1 {
		return nil, fmt.Errorf("no workloads with name %q found", workloadName)
	} else if len(workloadIds) > 1 {
		log.Warnf("Multiple workloads with name \"%v\" found: %v", workloadName, strings.Join(workloadIds[:], ", "))
		log.Warn("Consider specifying report with 'optimize report \"workloadID\" --type=id' instead")
		log.Warnf("Retrieving report for first workload ID in list: %v", workloadIds[0])
	}

	workloadId := formatAppDIdentifier(workloadIds[0])
	return &workloadId, nil
}

func columnValues(dataset *uql.DataSet, colIdx int) (res []string) {
	for _, row := range dataset.Values() {
		val, ok := row[colIdx].(string)
		if !ok {
			continue
		}
		res = append(res, val)
	}
	return res
}

func formatAppDIdentifier(s string) (res string) {
	split := strings.Split(s, ":")
	identifier := split[len(split)-1]
	return identifier
}
