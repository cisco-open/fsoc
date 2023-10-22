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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type servoLogsFlags struct {
	optimizerId  string
	since        string
	until        string
	count        int
	solutionName string
}

func init() {
	// TODO move this logic to optimize root when implementing unit tests
	optimizeCmd.AddCommand(NewCmdServoLogs())
}

func NewCmdServoLogs() *cobra.Command {
	flags := servoLogsFlags{}
	servoLogsCmd := &cobra.Command{
		Use:              "servo-logs",
		Short:            "Retrieve the logs of the Servo agent currently running for the given optimization",
		Example:          "  fsoc optimize servo-logs -i namespace-name-00000000-0000-0000-0000-000000000000",
		Args:             cobra.NoArgs,
		RunE:             getServoLogs(&flags),
		TraverseChildren: true,
	}

	servoLogsCmd.Flags().StringVarP(&flags.optimizerId, "optimizer-id", "i", "", "ID of Optimizer for which to retrieve servo logs.")
	if err := servoLogsCmd.MarkFlagRequired("optimizer-id"); err != nil {
		log.Warnf("Failed to set servo-logs flag optimizer-id required: %v", err)
	}

	servoLogsCmd.Flags().StringVarP(&flags.since, "since", "s", "", "Retrieve logs contained in the time interval starting at a relative or exact time. (default: -1h)")
	servoLogsCmd.Flags().StringVarP(&flags.until, "until", "u", "", "Retrieve logs contained in the time interval ending at a relative or exact time. (default: now)")
	servoLogsCmd.Flags().IntVarP(&flags.count, "count", "c", -1, "Limit the number of log lines retrieved to the specified count")

	servoLogsCmd.Flags().StringVarP(&flags.solutionName, "solution-name", "", "optimize", "Intended for developer usage, overrides the name of the solution defining the Knowledge Store types for reading")
	if err := servoLogsCmd.LocalFlags().MarkHidden("solution-name"); err != nil {
		log.Warnf("Failed to set servo-logs solution-name flag hidden: %v", err)
	}

	return servoLogsCmd
}

type servoLogsTemplateValues struct {
	Since     string
	Until     string
	ClusterId string
	ServoId   string
	Limits    string
}

var servoLogsTemplate = template.Must(template.New("").Parse(`
{{ with .Since }}SINCE {{ . }}
{{ end -}}
{{ with .Until }}UNTIL {{ . }}
{{ end -}}
FETCH events(logs:generic_record)[
    attributes(k8s.cluster.id) = "{{ .ClusterId }}"
    && attributes(k8s.deployment.name) = "servox-{{ .ServoId }}"
    && attributes(k8s.container.name) = "servo"
]{raw}
{{ with .Limits }}LIMITS events.count({{ . }})
{{ end -}}
ORDER events.asc()
`))

func getServoLogs(flags *servoLogsFlags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// get optimizer status
		headers := getOrionTenantHeaders()
		var response statusJsonStoreItem
		urlStr := fmt.Sprintf("knowledge-store/v1/objects/%v:status/%v", flags.solutionName, flags.optimizerId)

		err := api.JSONGet(urlStr, &response, &api.Options{Headers: headers})
		if err != nil {
			return fmt.Errorf("JSONGet: Unable to fetch %v:status by optimizer ID. api.JSONGet: %w", flags.solutionName, err)
		}
		optimizerStatus := response.Data

		// setup query
		tempVals := servoLogsTemplateValues{
			ClusterId: optimizerStatus.Optimizer.Target.K8SDeployment.ClusterID,
			ServoId:   optimizerStatus.ServoUID,
			Since:     flags.since,
			Until:     flags.until,
		}

		if flags.count != -1 {
			if flags.count > 1000 {
				return errors.New("counts higher than 1000 are not supported")
			}
			tempVals.Limits = strconv.Itoa(flags.count)
		}

		var buff bytes.Buffer
		if err := servoLogsTemplate.Execute(&buff, tempVals); err != nil {
			return fmt.Errorf("servoLogsTemplate.Execute: %w", err)
		}
		query := buff.String()

		// execute query, process results
		resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
		if err != nil {
			return fmt.Errorf("uql.ExecuteQuery: %w", err)
		}
		if resp.HasErrors() {
			log.Error("Execution of servo-logs query encountered errors. Returned data may not be complete!")
			for _, e := range resp.Errors() {
				log.Errorf("%s: %s", e.Title, e.Detail)
			}
		}

		main_data_set := resp.Main()
		if main_data_set == nil || len(main_data_set.Data) < 1 {
			output.PrintCmdStatus(cmd, "No servo logs results found for given input\n")
			return nil
		}
		if len(main_data_set.Data[0]) < 1 {
			return fmt.Errorf("main dataset %v first row has no columns", main_data_set.Name)
		}

		data_set, ok := main_data_set.Data[0][0].(*uql.DataSet)
		if !ok {
			return fmt.Errorf("main dataset %v first row first column (type %T) could not be converted to *uql.DataSet", main_data_set.Name, main_data_set.Data[0][0])
		}
		logRows, err := extractLogsData(data_set)
		if err != nil {
			return fmt.Errorf("extractLogsData: %w", err)
		}

		// handle pagination
		next_ok := false
		if data_set != nil {
			_, next_ok = data_set.Links["next"]
		}
		if flags.count != -1 {
			// skip pagination if limits provided. Otherwise, we return the full result list (chunked into count per response)
			// instead of constraining to count
			next_ok = false
		}
		for page := 2; next_ok; page++ {
			resp, err = uql.ContinueQuery(data_set, "next")
			if err != nil {
				return fmt.Errorf("page %v uql.ContinueQuery: %w", page, err)
			}
			if resp.HasErrors() {
				log.Errorf("Continuation of servo logs query (page %v) encountered errors. Returned data may not be complete!", page)
				for _, e := range resp.Errors() {
					log.Errorf("%s: %s", e.Title, e.Detail)
				}
			}
			main_data_set := resp.Main()
			if main_data_set == nil {
				log.Errorf("Continuation of servo logs query (page %v) has nil main data. Returned data may not be complete!", page)
				break
			}
			if len(main_data_set.Data) < 1 {
				return fmt.Errorf("page %v main dataset %v has no rows", page, main_data_set.Name)
			}
			if len(main_data_set.Data[0]) < 1 {
				return fmt.Errorf("page %v main dataset %v first row has no columns", page, main_data_set.Name)
			}
			data_set, ok = main_data_set.Data[0][0].(*uql.DataSet)
			if !ok {
				return fmt.Errorf("page %v main dataset %v first row first column (type %T) could not be converted to *uql.DataSet", page, main_data_set.Name, main_data_set.Data[0][0])
			}

			newRows, err := extractLogsData(data_set)
			if err != nil {
				return fmt.Errorf("page %v extractLogsData: %w", page, err)
			}
			logRows = append(logRows, newRows...)

			next_ok = false
			if data_set == nil {
				log.Warnf("Page %v dataset was nil, returned results may no be complete!", page)
			} else {
				_, next_ok = data_set.Links["next"]
			}
		}

		output.PrintCmdStatus(cmd, strings.Join(logRows, "\n"))
		return nil
	}
}

func extractLogsData(dataset *uql.DataSet) ([]string, error) {
	if dataset == nil {
		return []string{}, nil
	}
	resp_data := &dataset.Data
	result := make([]string, 0, len(*resp_data))

	for index, row := range *resp_data {
		if len(row) < 1 {
			return result, fmt.Errorf("servo log row %v has no columns", index)
		}
		logStr, ok := row[0].(string)
		if !ok {
			return result, fmt.Errorf("servo log row %v value %v (type %T) could not be converted to string", index, row[0], row[0])
		}
		result = append(result, logStr)
	}
	return result, nil
}
