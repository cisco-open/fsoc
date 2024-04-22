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

type reportFlags struct {
	commonFlags
	workloadId string
	eligible   bool
}

// TODO clarify blocker structure and pre-format
type reportRow struct {
	WorkloadId         string
	WorkloadAttributes map[string]any
	ProfileAttributes  map[string]any
	ProfileTimestamp   time.Time
}

type templateValues struct {
	WorkloadId      string
	WorkloadFilters string
}

var tempVals templateValues

var reportTemplate = template.Must(template.New("").Parse(`
SINCE -1w
FETCH id, attributes, events(k8sprofiler:report){attributes, timestamp}
FROM entities(k8s:deployment{{with .WorkloadId}}:{{.}}{{end}}){{with .WorkloadFilters}}[{{.}}]{{end}}
LIMITS events.count(1)
`))

func NewCmdReport() *cobra.Command {
	reportFlags := &reportFlags{}
	var reportCmd = &cobra.Command{
		Use:   "report",
		Short: "List workloads and optimization eligibility",
		Long: `
	List workloads and optimization eligibility
	
	If no flags are provided, all deployment workloads will be listed
	You can optionally filter workloads to by cluster, namespace and/or name
	You may specify also particular workloadId to fetch details for a single workload (recommended with -o detail or -o yaml)
	`,
		Example:          `fsoc optimize report --namespace kube-system`,
		Args:             cobra.NoArgs,
		RunE:             listReports(reportFlags),
		TraverseChildren: true,
		Annotations: map[string]string{
			output.TableFieldsAnnotation:  "WorkloadId: .WorkloadId, Name: .WorkloadAttributes[\"k8s.workload.name\"], Eligible: .ProfileAttributes[\"report_contents.optimizable\"], LastProfiled: .ProfileTimestamp",
			output.DetailFieldsAnnotation: "WorkloadId: .WorkloadId, Cluster: .WorkloadAttributes[\"k8s.cluster.name\"], Namespace: .WorkloadAttributes[\"k8s.namespace.name\"], Name: .WorkloadAttributes[\"k8s.workload.name\"], Eligible: .ProfileAttributes[\"report_contents.optimizable\"], Blockers: (.ProfileAttributes // {}) | with_entries(select(.key | startswith(\"report_contents.optimization_blockers\"))), LastProfiled: .ProfileTimestamp",
		},
	}
	reportCmd.Flags().StringVarP(&reportFlags.Cluster, "cluster", "c", "", "Filter reports by kubernetes cluster name")
	reportCmd.Flags().StringVarP(&reportFlags.Namespace, "namespace", "n", "", "Filter reports by kubernetes namespace")
	reportCmd.Flags().StringVarP(&reportFlags.WorkloadName, "workload-name", "w", "", "Filter reports by name of kubernetes workload")

	reportCmd.Flags().StringVarP(&reportFlags.workloadId, "workload-id", "i", "", "Retrieve a specific report by its workload's ID (best used with -o detail)")
	reportCmd.MarkFlagsMutuallyExclusive("workload-id", "cluster")
	reportCmd.MarkFlagsMutuallyExclusive("workload-id", "namespace")
	reportCmd.MarkFlagsMutuallyExclusive("workload-id", "workload-name")

	reportCmd.Flags().BoolVarP(&reportFlags.eligible, "eligible", "e", false, "Only list reports for eligbile workloads")

	registerReportCompletion(reportCmd, profilerReportFlagCluster)
	registerReportCompletion(reportCmd, profilerReportFlagNamespace)
	registerReportCompletion(reportCmd, profilerReportFlagWorkloadName)
	return reportCmd
}

func listReports(flags *reportFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		filtersList := make([]string, 0, 3)
		if flags.Cluster != "" {
			filtersList = append(filtersList, fmt.Sprintf("attributes(\"k8s.cluster.name\") = %q", flags.Cluster))
		}
		if flags.Namespace != "" {
			filtersList = append(filtersList, fmt.Sprintf("attributes(\"k8s.namespace.name\") = %q", flags.Namespace))
		}
		if flags.WorkloadName != "" {
			filtersList = append(filtersList, fmt.Sprintf("attributes(\"k8s.workload.name\") = %q", flags.WorkloadName))
		}
		tempVals.WorkloadFilters = strings.Join(filtersList, " && ")
		tempVals.WorkloadId, _ = strings.CutPrefix(flags.workloadId, "k8s:deployment:")

		var query string
		var buff bytes.Buffer
		if err := reportTemplate.Execute(&buff, tempVals); err != nil {
			return fmt.Errorf("reportTemplate.Execute: %w", err)
		}
		query = buff.String()

		resp, err := uql.ClientV1.ExecuteQuery(&uql.Query{Str: query})
		if err != nil {
			return fmt.Errorf("uql.ClientV1.ExecuteQuery: %w", err)
		}

		if resp.HasErrors() {
			log.Error("Execution of report query encountered errors. Returned data may not be complete!")
			for _, e := range resp.Errors() {
				log.Errorf("%s: %s", e.Title, e.Detail)
			}
		}

		mainDataSet := resp.Main()
		if mainDataSet == nil {
			output.PrintCmdStatus(cmd, "No results found for given input\n")
			return nil
		}

		reportRows, err := extractReportData(mainDataSet, flags.eligible)
		if err != nil {
			return fmt.Errorf("extractReportData: %w", err)
		}

		_, next_ok := mainDataSet.Links["next"]
		for page := 2; next_ok; page++ {
			resp, err = uql.ClientV1.ContinueQuery(mainDataSet, "next")
			if err != nil {
				return fmt.Errorf("page %v uql.ClientV1.ContinueQuery: %w", page, err)
			}

			if resp.HasErrors() {
				log.Errorf("Continuation of report query (page %v) encountered errors. Returned data may not be complete!", page)
				for _, e := range resp.Errors() {
					log.Errorf("%s: %s", e.Title, e.Detail)
				}
			}

			mainDataSet = resp.Main()
			if mainDataSet == nil {
				log.Errorf("Continuation of report query (page %v) has nil main data. Returned data may not be complete!", page)
				break
			}

			newRows, err := extractReportData(mainDataSet, flags.eligible)
			if err != nil {
				return fmt.Errorf("page %v extractReportData: %w", page, err)
			}

			reportRows = append(reportRows, newRows...)
			_, next_ok = mainDataSet.Links["next"]
		}

		if len(reportRows) < 1 {
			output.PrintCmdStatus(cmd, "No results found for given input\n")
			return nil
		}

		output.PrintCmdOutput(cmd, struct {
			Items []reportRow `json:"items"`
			Total int         `json:"total"`
		}{Items: reportRows, Total: len(reportRows)})

		return nil
	}
}

func extractReportData(dataset *uql.DataSet, eligible bool) ([]reportRow, error) {
	resp_data := &dataset.Data
	var err error
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
		log.WithField("workloadId", workloadId).Info("Processing workload report")
		reportRow := reportRow{WorkloadId: workloadId}

		workloadAttributeDataset, ok := row[1].(*uql.DataSet)
		if !ok {
			return results, fmt.Errorf("workload entity attributes uql.DataSet type assertion failed (main dataset row %v): %+v", index, row)
		}
		if workloadAttributeDataset == nil {
			log.Warnf("(main dataset row %v) got nil data for entity attributes on %v", index, workloadId)
		} else {
			reportRow.WorkloadAttributes, err = sliceToMap(workloadAttributeDataset.Data)
			if err != nil {
				return results, fmt.Errorf("(main dataset row %v) sliceToMap(workloadAttributeDataset.Data): %w", index, err)
			}
		}

		profileAttributesDataSet, ok := row[2].(*uql.DataSet)
		if !ok {
			return results, fmt.Errorf("profile event attributes uql.DataSet type assertion failed (main dataset row %v): %+v", index, row)
		}
		if profileAttributesDataSet != nil && len(profileAttributesDataSet.Data) > 0 {
			// uql LIMITS events.count(1) means we're only interested in the first (and only) row of returned events
			firstRow := profileAttributesDataSet.Data[0]
			if len(firstRow) < 2 {
				log.Warnf("k8sprofiler:report dataset had incomplete row at index %s: %+v", index, firstRow)
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
			if eligible && reportRow.ProfileAttributes["report_contents.optimizable"] != "true" {
				continue
			}
			reportRow.ProfileTimestamp, ok = firstRow[1].(time.Time)
			if !ok {
				log.Warnf("Returned data is not complete. Type assertion failed for profile event timestamp (main dataset row %v): %+v", index, firstRow)
			}
		} else if eligible {
			continue // filter out workloads with no eligible event returned
		}

		results = append(results, reportRow)
	}
	return results, nil
}
