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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
	"github.com/spf13/cobra"
)

type configureFlags struct {
	Cluster      string
	Namespace    string
	WorkloadName string
	optimizerId  string
	workloadId   string
	filePath     string
	create       bool
	start        bool
	solutionName string
}

var optimizerConfigNotFoundError = errors.New("Optimizer config not found")
var profilerMissingDataError = errors.New("Missing data in profiler report")
var profilerInvalidDataError = errors.New("Invalid data found in profiler report")

func init() {
	// TODO move this logic to optimize root when implementing unit tests
	optimizeCmd.AddCommand(NewCmdConfigure())
}

func NewCmdConfigure() *cobra.Command {
	flags := configureFlags{}
	configureCmd := &cobra.Command{
		Use:              "configure",
		Short:            "TODO",
		Long:             `TODO`,
		Example:          "TODO",
		Args:             cobra.NoArgs,
		RunE:             configureOptimizer(&flags),
		TraverseChildren: true,
	}

	//NOTE only one optimizer may be configured at a time. Support for bulk config may be supported in a future update
	configureCmd.Flags().StringVarP(&flags.Cluster, "cluster", "c", "", "Configure optimization for a workload with this cluster name")
	configureCmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "Configure optimization for a workload with this kubernetes namespace")
	configureCmd.Flags().StringVarP(&flags.WorkloadName, "workload-name", "w", "", "Configure optimization for a workload with this name in its kubernetes manifest")
	configureCmd.MarkFlagsRequiredTogether("cluster", "namespace", "workload-name")

	configureCmd.Flags().StringVarP(&flags.optimizerId, "optimizer-id", "i", "", "Configure a specific optimizer by its ID")
	configureCmd.Flags().StringVarP(&flags.workloadId, "workload-id", "r", "", "Configure a specific optimizer given the ID of the workload it optimizes")
	configureCmd.MarkFlagsMutuallyExclusive("workload-id", "optimizer-id", "cluster")
	configureCmd.MarkFlagsMutuallyExclusive("workload-id", "optimizer-id", "namespace")
	configureCmd.MarkFlagsMutuallyExclusive("workload-id", "optimizer-id", "workload-name")

	configureCmd.Flags().BoolVarP(&flags.create, "create", "", false, "Create a new optimizer from report data and provided configuraiton file")
	configureCmd.Flags().BoolVarP(&flags.start, "start", "s", false, "Set the desired state of the specified or new optimizer to started")

	configureCmd.MarkFlagsMutuallyExclusive("optimizer-id", "create")

	configureCmd.Flags().StringVarP(&flags.solutionName, "solution-name", "", "optimize", "Intended for developer usage, overrides the name of the solution defining the Orion types for reading/writing")
	if err := configureCmd.LocalFlags().MarkHidden("solution-name"); err != nil {
		log.Warnf("Failed to set solution-name flag hidden: %v", err)
	}

	return configureCmd
}

func configureOptimizer(flags *configureFlags) func(cmd *cobra.Command, args []string) error {
	var workloadTemplate = template.Must(template.New("").Parse(`
SINCE -1w
FETCH id
FROM entities(k8s:deployment)[attributes("k8s.cluster.name") = "{{.Cluster}}" && attributes("k8s.namespace.name") = "{{.Namespace}}" && attributes("k8s.workload.name") = "{{.WorkloadName}}"]
`))

	return func(cmd *cobra.Command, args []string) error {
		var profilerReport map[string]any
		var optimizerConfig OptimizerConfiguration
		var optimizerConfigError, err error
		var workloadId string

		if flags.optimizerId != "" {
			optimizerConfig, optimizerConfigError = getOptimizerConfig(flags.optimizerId, "", flags.solutionName)
			if optimizerConfigError != nil {
				return fmt.Errorf("Unable to get config for existing optimizer. getOptimizerConfig: %w", optimizerConfigError)
			}

			workloadId = optimizerConfig.Target.K8SDeployment.WorkloadID
			profilerReport, err = getProfilerReport(workloadId)
			if err != nil {
				return fmt.Errorf("flags.optimizerId getProfilerReport: %w", err)
			}

		} else if flags.workloadId != "" {
			if !strings.HasPrefix(flags.workloadId, "k8s:") {
				flags.workloadId = fmt.Sprintf("k8s:deployment:%v", flags.workloadId)
			}
			workloadId = flags.workloadId
			profilerReport, err = getProfilerReport(workloadId)
			if err != nil {
				return fmt.Errorf("flags.workloadId getProfilerReport: %w", err)
			}

			optimizerConfig, optimizerConfigError = getOptimizerConfig("", workloadId, flags.solutionName)

		} else if flags.Cluster != "" { //note MarkFlagsRequiredTogether is checking namespace and workloadName
			var query string
			var buff bytes.Buffer
			if err := workloadTemplate.Execute(&buff, flags); err != nil {
				return fmt.Errorf("workloadTemplate.Execute: %w", err)
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

			if workloadIdsFound := len(resp.Main().Data); workloadIdsFound != 1 {
				return fmt.Errorf("Unable to configure optimizer. Found %v workload IDs for the given criteria.", workloadIdsFound)
			}
			var ok bool
			workloadId, ok = resp.Main().Data[0][0].(string)
			if !ok {
				return fmt.Errorf("Unable to convert workloadId query value %q to string", resp.Main().Data[0][0])
			}

			profilerReport, err = getProfilerReport(workloadId)
			if err != nil {
				return fmt.Errorf("flags.Cluster getProfilerReport: %w", err)
			}
			optimizerConfig, optimizerConfigError = getOptimizerConfig("", workloadId, flags.solutionName)

		} else {
			return errors.New("No identifying information provided for workload/optimizer to be configured")
		}

		// collective checking of optimizerConfigError. Not found is OK if creating
		if optimizerConfigError != nil && !errors.Is(optimizerConfigError, optimizerConfigNotFoundError) {
			return optimizerConfigError
		}

		// validate --create=True/False with nonexisting/existing optimizer config
		if flags.create {
			if optimizerConfigError == nil || !errors.Is(optimizerConfigError, optimizerConfigNotFoundError) {
				return errors.New("Found existing optimizer config on request to create optimizer for given workload")
			} // else no config found, we're in the expected state
		} else {
			if optimizerConfigError != nil {
				return optimizerConfigError // report optimizerConfigNotFoundError error if not creating
			}
		}

		// I'm not sure if theres a more idiomatic way to do this but it was the least boilerplatey
		// solution I could think of to check for missing data and prevent panics
		validateProfilerReportConfigData(&profilerReport, []string{
			"resource_metadata.namespace_name", "resource_metadata.workload_name", "k8s.deployment.uid",
			"resource_metadata.cluster_id", "resource_metadata.cluster_name", "report_contents.main_container_name",
			"report_support_data.cpu_requests", "report_support_data.memory_requests",
		})
		var newOptimizerConfig OptimizerConfiguration = OptimizerConfiguration{}
		newOptimizerConfig.OptimizerID = buildOptimizerId(
			profilerReport["resource_metadata.namespace_name"].(string),
			profilerReport["resource_metadata.workload_name"].(string),
			profilerReport["k8s.deployment.uid"].(string),
		)
		newOptimizerConfig.RestartTimestamp = time.Now().UTC().String()
		if flags.start {
			newOptimizerConfig.DesiredState = "started"
		} else {
			newOptimizerConfig.DesiredState = "stopped"
		}
		// Target
		newOptimizerConfig.Target.K8SDeployment.ClusterID = profilerReport["resource_metadata.cluster_id"].(string)
		newOptimizerConfig.Target.K8SDeployment.ClusterName = profilerReport["resource_metadata.cluster_name"].(string)
		newOptimizerConfig.Target.K8SDeployment.ContainerName = profilerReport["report_contents.main_container_name"].(string)
		newOptimizerConfig.Target.K8SDeployment.NamespaceName = profilerReport["resource_metadata.namespace_name"].(string)
		newOptimizerConfig.Target.K8SDeployment.WorkloadID = strings.Split(workloadId, ":")[2]
		newOptimizerConfig.Target.K8SDeployment.WorkloadName = profilerReport["resource_metadata.workload_name"].(string)
		// Config
		cpuRequest, err := strconv.ParseFloat(profilerReport["report_support_data.cpu_requests"].(string), 64)
		if err != nil {
			return fmt.Errorf("Unable to parse profiler report_support_data.cpu_requests into float64: %w", err)
		}
		newOptimizerConfig.Config.Guardrails.CPU.Max = cpuRequest * 1.5
		newOptimizerConfig.Config.Guardrails.CPU.Min = cpuRequest * 0.5
		newOptimizerConfig.Config.Guardrails.CPU.Pinned = false

		memRequest, err := strconv.ParseFloat(profilerReport["report_support_data.memory_requests"].(string), 64)
		if err != nil {
			return fmt.Errorf("Unable to parse profiler report_support_data.memory_requests into float64: %w", err)
		}
		// convert from bytes to GiB
		memRequest = memRequest / math.Pow(1024, 3)
		newOptimizerConfig.Config.Guardrails.Mem.Max = memRequest * 1.5
		newOptimizerConfig.Config.Guardrails.Mem.Min = memRequest * 0.5
		newOptimizerConfig.Config.Guardrails.Mem.Pinned = false
		// Set suspensions to empty object
		newOptimizerConfig.Suspensions = make(map[string]Suspension)

		// config file overrides
		// TODO test this path
		if flags.filePath != "" {
			configFile, err := os.Open(flags.filePath)
			if err != nil {
				return fmt.Errorf("os.Open(flags.filePath): %w", err)
			}
			defer configFile.Close()

			configBytes, _ := io.ReadAll(configFile)
			// NOTE unmarshalling on top of the existing config will overwrite it with only values explicitly set by the file
			err = json.Unmarshal(configBytes, &newOptimizerConfig)
			if err != nil {
				return fmt.Errorf("json.Unmarshal(configBytes, &configStruct): %w", err)
			}
		}

		// write new config to ORION
		headers := map[string]string{
			"layer-type": "TENANT",
			"layer-id":   config.GetCurrentContext().Tenant,
		}
		var res any

		if flags.create {
			urlStr := fmt.Sprintf("objstore/v1beta/objects/%v:optimizer", flags.solutionName)
			if err = api.JSONPost(urlStr, newOptimizerConfig, &res, &api.Options{Headers: headers}); err != nil {
				return fmt.Errorf("Failed to create knowledge object for optimizer configuration: %w", err)
			}
		} else {
			urlStr := fmt.Sprintf("objstore/v1beta/objects/%v:optimizer/%v", flags.solutionName, newOptimizerConfig.OptimizerID)
			if err = api.JSONPut(urlStr, newOptimizerConfig, &res, &api.Options{Headers: headers}); err != nil {
				return fmt.Errorf("Failed to update knowledge object with new optimizer configuration: %w", err)
			}
		}

		output.PrintCmdStatus(cmd, fmt.Sprintf("Optimizer configured with ID %q", newOptimizerConfig.OptimizerID))
		return nil
	}
}

func buildOptimizerId(namespace string, workloadName string, workloadUid string) string {
	// NOTE convert to runes before slicing to account for UTF-8 chars
	var nsPortion, wnPortion string
	nsRunes, wnRunes := []rune(namespace), []rune(workloadName)
	if len(nsRunes) > 10 {
		nsPortion = string(nsRunes[:10])
	} else {
		nsPortion = string(nsRunes)
	}
	if len(wnRunes) > 10 {
		wnPortion = string(wnRunes[:10])
	} else {
		wnPortion = string(wnRunes)
	}
	return fmt.Sprintf("%v-%v-%v", nsPortion, wnPortion, workloadUid)
}

type configJsonStoreItem struct {
	Data OptimizerConfiguration `json:"data"`
	JsonStoreItem
}

type configJsonStorePage struct {
	Items []configJsonStoreItem `json:"items"`
	Total int                   `json:"total"`
}

func optimizerConfigNotFoundErrorWrapper(extraDetail string) error {
	return fmt.Errorf("%w: %v", optimizerConfigNotFoundError, extraDetail)
}

func getOptimizerConfig(optimizerId string, workloadId string, solutionName string) (OptimizerConfiguration, error) {
	var optimizerConfig OptimizerConfiguration
	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   config.GetCurrentContext().Tenant,
	}

	if optimizerId != "" {
		var response configJsonStoreItem

		urlStr := fmt.Sprintf("objstore/v1beta/objects/%v:optimizer/%v", solutionName, optimizerId)
		err := api.JSONGet(urlStr, &response, &api.Options{Headers: headers})
		if err != nil {
			if problem, ok := err.(api.Problem); ok && problem.Status == 404 {
				return optimizerConfig, fmt.Errorf("%w (api.JSONGet: %w): No matches found for the given optimizerId", optimizerConfigNotFoundError, problem)
			}
			return optimizerConfig, fmt.Errorf("unable to fetch existing config by optimizer ID. api.JSONGet: %w", err)
		}
		optimizerConfig = response.Data
	} else if workloadId != "" {
		var configPage configJsonStorePage
		queryStr := url.QueryEscape(fmt.Sprintf("data.target.k8sDeployment.workloadId eq %q", workloadId))
		urlStr := fmt.Sprintf("objstore/v1beta/objects/%v:optimizer?filter=%v", solutionName, queryStr)

		err := api.JSONGet(urlStr, &configPage, &api.Options{Headers: headers})
		if err != nil {
			return optimizerConfig, fmt.Errorf("unable to fetch existing config by workload ID. api.JSONGet: %w", err)
		}
		if configPage.Total > 1 {
			return optimizerConfig, fmt.Errorf("Found %v optimizer configurations for the given workloadID", configPage.Total)
		}
		if configPage.Total < 1 {
			return optimizerConfig, fmt.Errorf("%w: No matches found for the given workloadId", optimizerConfigNotFoundError)
		}

		optimizerConfig = configPage.Items[0].Data
	} else {
		return optimizerConfig, errors.New("Must provide either workloadId or optimizerId")
	}

	return optimizerConfig, nil
}

var singleReportTemplate = template.Must(template.New("").Parse(`
SINCE -1w 
FETCH events(optimize:profile){attributes} 
FROM entities({{.}}) 
LIMITS events.count(1)
`))

func getProfilerReport(workloadId string) (map[string]any, error) {
	var query string
	var buff bytes.Buffer
	if err := singleReportTemplate.Execute(&buff, workloadId); err != nil {
		return nil, fmt.Errorf("singleReportTemplate.Execute: %w", err)
	}
	query = buff.String()

	resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
	if err != nil {
		return nil, fmt.Errorf("uql.ExecuteQuery: %w", err)
	}
	if resp.HasErrors() {
		log.Error("Execution of report query encountered errors. Returned data may not be complete!")
		for _, e := range resp.Errors() {
			log.Errorf("%s: %s", e.Title, e.Detail)
		}
	}

	mainDataSetData := resp.Main().Data
	if len(mainDataSetData) < 1 {
		return nil, errors.New("No events found, main data set had no rows")
	}
	if len(mainDataSetData[0]) < 1 {
		return nil, errors.New("No events found, main data first row had no columns")
	}
	eventDataSet, ok := mainDataSetData[0][0].(*uql.DataSet)
	if !ok {
		return nil, fmt.Errorf("Unexpected type %T for event data set", mainDataSetData[0][0])
	}
	if len(eventDataSet.Data) < 1 {
		return nil, errors.New("No events found, event data set had no rows")
	}
	if len(eventDataSet.Data[0]) < 1 {
		return nil, errors.New("No events found, event data first row had no columns")
	}
	eventComplexData, ok := eventDataSet.Data[0][0].(uql.ComplexData)
	if !ok {
		return nil, fmt.Errorf("Unexpected type %T for event data set", eventDataSet.Data[0][0])
	}
	result, err := sliceToMap(eventComplexData.Data)
	if err != nil {
		return nil, fmt.Errorf("sliceToMap: %w", err)
	}

	return result, nil
}

// validateProfilerReportConfigData takes in a list of string keys and validates they are all present
// in the map and their values are strings as expected
func validateProfilerReportConfigData(profilerReport *map[string]any, keys []string) error {
	for _, key := range keys {
		val, ok := (*profilerReport)[key]
		if !ok {
			return fmt.Errorf("%w: key %q does not exist", profilerMissingDataError, key)
		}
		if _, ok := val.(string); !ok {
			return fmt.Errorf("%w: string assertion failed for key %q value %q", profilerInvalidDataError, key, val)
		}
	}
	return nil
}
