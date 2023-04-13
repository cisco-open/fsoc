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

package optimize

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/output"
)

// TODO clarify blocker structure and pre-format
type reportRow struct {
	WorkloadId         string
	WorkloadAttributes map[string]any
	ProfileAttributes  map[string]any
	ProfileTimestamp   time.Time
}

type templateValues struct {
	WorkloadId      string
	Eligible        bool
	WorkloadFilters string
}

var (
	tempVals     templateValues
	cluster      string
	namespace    string
	workloadName string
)

var reportTemplate = template.Must(template.New("").Parse(`
SINCE -1w 
FETCH id, attributes, events(optimize:profile){{if .Eligible}}[attributes("report_contents.optimizable") = "true"]{{end}}{attributes} 
FROM entities(k8s:deployment{{with .WorkloadId}}:{{.}}{{end}}){{with .WorkloadFilters}}[{{.}}]{{end}} 
LIMITS events.count(1)
`))

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "List workloads and optimization eligibility",
	Long: `
List workloads and optimization eligibility
	
If no flags are provided, all deployment workloads will be listed
You can optionally filter worklaods to by cluster, namespace and/or name
You may specify also particular workloadId to fetch details for a single workload (recommended with -o detail or -o yaml)
`,
	Example:          `fsoc optimize report --namespace kube-system`,
	Args:             cobra.NoArgs,
	RunE:             listReports,
	TraverseChildren: true,
	Annotations: map[string]string{
		output.TableFieldsAnnotation:  "WorkloadId: .WorkloadId, Name: .WorkloadAttributes[\"k8s.workload.name\"], Eligible: .ProfileAttributes[\"report_contents.optimizable\"], LastProfiled: .ProfileTimestamp",
		output.DetailFieldsAnnotation: "WorkloadId: .WorkloadId, Cluster: .WorkloadAttributes[\"k8s.cluster.name\"], Namespace: .WorkloadAttributes[\"k8s.namespace.name\"], Name: .WorkloadAttributes[\"k8s.workload.name\"], Eligible: .ProfileAttributes[\"report_contents.optimizable\"], Blockers: .ProfileAttributes | with_entries(select(.key | startswith(\"report_contents.optimization_blockers\"))), LastProfiled: .ProfileTimestamp",
	},
}

func init() {
	optimizeCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVarP(&cluster, "cluster", "c", "", "Filter reports by kubernetes cluster name")
	reportCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Filter reports by kubernetes namespace")
	reportCmd.Flags().StringVarP(&workloadName, "workload-name", "w", "", "Filter reports by name of kubernetes workload")

	reportCmd.Flags().StringVarP(&tempVals.WorkloadId, "workload-id", "i", "", "Retrieve a specific report by its workload's ID (best used with -o detail)")
	reportCmd.MarkFlagsMutuallyExclusive("workload-id", "cluster")
	reportCmd.MarkFlagsMutuallyExclusive("workload-id", "namespace")
	reportCmd.MarkFlagsMutuallyExclusive("workload-id", "workload-name")

	reportCmd.Flags().BoolVarP(&tempVals.Eligible, "eligible", "e", false, "Only list reports for eligbile workloads")
}

func listReports(cmd *cobra.Command, args []string) error {
	filtersList := make([]string, 0, 3)
	if cluster != "" {
		filtersList = append(filtersList, fmt.Sprintf("attributes(\"k8s.cluster.name\") = %q", cluster))
	}
	if namespace != "" {
		filtersList = append(filtersList, fmt.Sprintf("attributes(\"k8s.namespace.name\") = %q", namespace))
	}
	if workloadName != "" {
		filtersList = append(filtersList, fmt.Sprintf("attributes(\"k8s.workload.name\") = %q", workloadName))
	}
	tempVals.WorkloadFilters = strings.Join(filtersList, " && ")

	var query string
	var buff bytes.Buffer
	if err := reportTemplate.Execute(&buff, tempVals); err != nil {
		return fmt.Errorf("reportTemplate.Execute: %w", err)
	}
	query = buff.String()

	resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
	if err != nil {
		return fmt.Errorf("uql.ExecuteQuery: %w", err)
	}

	if resp.HasErrors() {
		log.Error("Execution of report query encountered errors. Returned data may not be complete!")
		for _, e := range resp.Errors() {
			log.Errorf("%s: %s", e.Title, e.Detail)
		}
	}

	reportRows, err := extractReportData(resp)
	if err != nil {
		return fmt.Errorf("extractReportData: %w", err)
	}

	output.PrintCmdOutput(cmd, struct {
		Items []reportRow `json:"items"`
		Total int         `json:"total"`
	}{Items: reportRows, Total: len(reportRows)})

	return nil
}

func extractReportData(response *uql.Response) ([]reportRow, error) {
	resp_data := &response.Main().Data
	results := make([]reportRow, 0, len(*resp_data))
	for index, row := range *resp_data {
		if len(row) < 3 {
			log.Warnf("Returned data is not complete. Main dataset had incomplete row at index %v: %+v", index, row)
			continue
		}
		workloadId, ok := row[0].(string)
		if !ok {
			return results, fmt.Errorf("entity id string type assertion failed on main dataset row %v: %+v", index, row)
		}
		reportRow := reportRow{WorkloadId: workloadId}

		workloadAttributeDataset, ok := row[1].(*uql.DataSet)
		if !ok {
			return results, fmt.Errorf("workload entity attributes uql.DataSet type assertion failed (main dataset row %v): %+v", index, row)
		}
		var err error
		reportRow.WorkloadAttributes, err = sliceToMap(workloadAttributeDataset.Data)
		if err != nil {
			return results, fmt.Errorf("(main dataset row %v) sliceToMap(workloadAttributeDataset.Data): %w", index, err)
		}

		profileAttributesDataSet, ok := row[2].(*uql.DataSet)
		if !ok {
			return results, fmt.Errorf("profile event attributes uql.DataSet type assertion failed (main dataset row %v): %+v", index, row)
		}
		if len(profileAttributesDataSet.Data) > 0 {
			// uql LIMITS events.count(1) means we're only interested in the first (and only) row of returned events
			firstRow := profileAttributesDataSet.Data[0]
			if len(firstRow) < 2 {
				log.Warnf("optimize:profile dataset had incomplete row at index %s: %+v", index, firstRow)
				continue
			}
			firstRowComplexData, ok := firstRow[0].(uql.ComplexData)
			if !ok {
				return results, fmt.Errorf("uql.ComplexData type assertion failed on profile event attributes (main dataset row %v): %+v", index, firstRow)
			}
			reportRow.ProfileAttributes, err = sliceToMap(firstRowComplexData.Data)
			if err != nil {
				return results, fmt.Errorf("row %v sliceToMap(firstRowComplexData.Data): %w", index, err)
			}
			reportRow.ProfileTimestamp, ok = firstRow[1].(time.Time)
			if !ok {
				log.Warnf("Returned data is not complete. Type assertion failed for profile event timestamp (main dataset row %v): %+v", index, firstRow)
			}
		} else if tempVals.Eligible {
			continue // filter out workloads with no eligible event returned
		}

		results = append(results, reportRow)
	}
	return results, nil
}

// sliceToMap converts a list of lists (slice [][2]any) to a dictionary for table output jq support
// eg.
//
//	[
//		["k8s.cluster.name", "ignite-test"],
//		["k8s.namespace.name", "kube-system"],
//		["k8s.workload.kind", "Deployment"],
//		["k8s.workload.name", "coredns"]
//	]
//
// to
//
//	k8s.cluster.name: ignite-test
//	k8s.namespace.name: kube-system
//	k8s.workload.kind: Deployment
//	k8s.workload.name: coredns
func sliceToMap(slice [][]any) (map[string]any, error) {
	results := make(map[string]any)
	for index, subslice := range slice {
		if len(subslice) < 2 {
			return results, fmt.Errorf("subslice (at index %v) too short to construct key value pair: %+v", index, subslice)
		}
		key, ok := subslice[0].(string)
		if !ok {
			return results, fmt.Errorf("string type assertion failed on first subslice item (at index %v): %+v", index, subslice)
		}
		results[key] = subslice[1]
	}
	return results, nil
}
