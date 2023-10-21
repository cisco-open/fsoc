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
	"io"
	"math"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type configureFlags struct {
	Cluster              string
	Namespace            string
	WorkloadName         string
	optimizerId          string
	workloadId           string
	filePath             string
	create               bool
	start                bool
	overrideSoftBlockers bool
	overrideHardBlockers bool
	solutionName         string
}

var errOptimizerConfigNotFound = errors.New("optimizer config not found")
var errProfilerMissingData = errors.New("missing data in profiler report")
var errProfilerInvalidData = errors.New("invalid data found in profiler report")

func init() {
	// TODO move this logic to optimize root when implementing unit tests
	optimizeCmd.AddCommand(NewCmdConfigure())
}

func NewCmdConfigure() *cobra.Command {
	flags := configureFlags{}
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure (and/or create) an optimizer",
		Long: `
Configure (and/or create) an optimizer

This command allows one to create a new optimization (via --create flag) or configure existing optimizers.
It requires a means of identifying the workload (or existing optimizer) and will retrieve the latest profiler report
for the workload under optimization. It will populate default optimizer configuration values based on this report
and push the configuration to the knowledge store. You may optionally override these defaults with the --file parameter.
`,
		Example: `  fsoc optimize configure --workload-id uS2J001gM2+Tz8eXhpuROw
  fsoc optimize configure --workload-id k8s:deployment:uS2J001gM2+Tz8eXhpuROw
  fsoc optimize configure --cluster your-cluster --namespace your-namespace --workload-name your-workload
  fsoc optimize configure --optimizer-id namespace-name-00000000-0000-0000-0000-000000000000
`,
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

	configureCmd.Flags().StringVarP(&flags.filePath, "file", "f", "", "Override profiler report values with a local yaml/json file matching the json schema of the optimize:optimizer Orion type")

	configureCmd.Flags().BoolVarP(&flags.create, "create", "", false, "Create a new optimizer from report data and provided configuration file")
	configureCmd.Flags().BoolVarP(&flags.start, "start", "s", false, "Set the desired state of the specified or new optimizer to started")

	configureCmd.MarkFlagsMutuallyExclusive("optimizer-id", "create")

	configureCmd.Flags().BoolVar(&flags.overrideSoftBlockers, "override-soft-blockers", false, "override soft blockers for specified optimizer")
	configureCmd.Flags().BoolVar(&flags.overrideHardBlockers, "override-hard-blockers", false, "override hard blockers for specified optimizer")

	configureCmd.Flags().StringVarP(&flags.solutionName, "solution-name", "", "optimize", "Intended for developer usage, overrides the name of the solution defining the Orion types for reading/writing")
	if err := configureCmd.LocalFlags().MarkHidden("solution-name"); err != nil {
		log.Warnf("Failed to set solution-name flag hidden: %v", err)
	}
	if err := configureCmd.LocalFlags().MarkHidden("override-hard-blockers"); err != nil {
		log.Warnf("Failed to set override-hard-blockers flag hidden: %v", err)
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
				return fmt.Errorf("unable to get config for existing optimizer. getOptimizerConfig: %w", optimizerConfigError)
			}

			workloadId = optimizerConfig.Target.K8SDeployment.WorkloadID
			if !strings.HasPrefix(workloadId, "k8s:") {
				workloadId = fmt.Sprintf("k8s:deployment:%v", workloadId)
			}
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

			mainDataSet := resp.Main()
			if mainDataSet == nil {
				return errors.New("unable to configure optimizer; UQL main data set was nil for the given criteria")
			}
			if workloadIdsFound := len(mainDataSet.Data); workloadIdsFound != 1 {
				return fmt.Errorf("unable to configure optimizer; found %v workload IDs for the given criteria", workloadIdsFound)
			}
			var ok bool
			workloadId, ok = mainDataSet.Data[0][0].(string)
			if !ok {
				return fmt.Errorf("unable to convert workloadId query value %q to string", mainDataSet.Data[0][0])
			}

			profilerReport, err = getProfilerReport(workloadId)
			if err != nil {
				return fmt.Errorf("flags.Cluster getProfilerReport: %w", err)
			}
			optimizerConfig, optimizerConfigError = getOptimizerConfig("", workloadId, flags.solutionName)

		} else {
			return errors.New("no identifying information provided for workload/optimizer to be configured")
		}

		// collective checking of optimizerConfigError. Not found is OK if creating
		if optimizerConfigError != nil && !errors.Is(optimizerConfigError, errOptimizerConfigNotFound) {
			return optimizerConfigError
		}

		// validate --create=True/False with nonexisting/existing optimizer config
		if flags.create {
			if optimizerConfigError == nil || !errors.Is(optimizerConfigError, errOptimizerConfigNotFound) {
				return errors.New("found existing optimizer config on request to create optimizer for given workload")
			} // else no config found, we're in the expected state
		} else {
			if optimizerConfigError != nil {
				return optimizerConfigError // report optimizerConfigNotFoundError error if not creating
			}
		}

		// I'm not sure if theres a more idiomatic way to do this but it was the least boilerplatey
		// solution I could think of to check for missing data and prevent panics
		err = validateProfilerReportConfigData(&profilerReport, []string{
			"resource_metadata.namespace_name", "resource_metadata.workload_name", "k8s.deployment.uid",
			"resource_metadata.cluster_id", "resource_metadata.cluster_name", "report_contents.main_container_name",
			"report_contents.optimization_configuration.guardrails.cpu.max",
			"report_contents.optimization_configuration.guardrails.cpu.min",
			"report_contents.optimization_configuration.guardrails.cpu.pinned",
			"report_contents.optimization_configuration.guardrails.memory.max",
			"report_contents.optimization_configuration.guardrails.memory.min",
			"report_contents.optimization_configuration.guardrails.memory.pinned",
			"report_contents.optimization_configuration.slo.error_percent.target",
			"report_contents.optimization_configuration.slo.median_response_time.target",
		})
		if err != nil {
			return fmt.Errorf("validateProfilerReportConfigData: %w", err)
		}
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
		newOptimizerConfig.Target.K8SDeployment.DeploymentUID = profilerReport["k8s.deployment.uid"].(string)
		newOptimizerConfig.Target.K8SDeployment.ClusterName = profilerReport["resource_metadata.cluster_name"].(string)
		newOptimizerConfig.Target.K8SDeployment.ContainerName = profilerReport["report_contents.main_container_name"].(string)
		newOptimizerConfig.Target.K8SDeployment.NamespaceName = profilerReport["resource_metadata.namespace_name"].(string)
		newOptimizerConfig.Target.K8SDeployment.WorkloadID = strings.Split(workloadId, ":")[2]
		newOptimizerConfig.Target.K8SDeployment.WorkloadName = profilerReport["resource_metadata.workload_name"].(string)
		// Config
		newOptimizerConfig.Config.Guardrails.CPU.Max, err = strconv.ParseFloat(
			profilerReport["report_contents.optimization_configuration.guardrails.cpu.max"].(string), 64)
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.guardrails.cpu.max into float64: %w", err)
		}
		newOptimizerConfig.Config.Guardrails.CPU.Min, err = strconv.ParseFloat(
			profilerReport["report_contents.optimization_configuration.guardrails.cpu.min"].(string), 64)
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.guardrails.cpu.min into float64: %w", err)
		}
		newOptimizerConfig.Config.Guardrails.CPU.Pinned, err = strconv.ParseBool(
			profilerReport["report_contents.optimization_configuration.guardrails.cpu.pinned"].(string))
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.guardrails.cpu.pinned into boolean: %w", err)
		}
		newOptimizerConfig.Config.Guardrails.Mem.Max, err = strconv.ParseFloat(
			profilerReport["report_contents.optimization_configuration.guardrails.memory.max"].(string), 64)
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.guardrails.memory.max into float64: %w", err)
		}
		newOptimizerConfig.Config.Guardrails.Mem.Min, err = strconv.ParseFloat(
			profilerReport["report_contents.optimization_configuration.guardrails.memory.min"].(string), 64)
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.guardrails.memory.min into float64: %w", err)
		}
		newOptimizerConfig.Config.Guardrails.Mem.Pinned, err = strconv.ParseBool(
			profilerReport["report_contents.optimization_configuration.guardrails.memory.pinned"].(string))
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.guardrails.memory.pinned into boolean: %w", err)
		}
		// SLOs
		errorPercentTarget, err := strconv.ParseFloat(
			profilerReport["report_contents.optimization_configuration.slo.error_percent.target"].(string), 64)
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.slo.error_percent.target into float64: %w", err)
		}
		medianResponseTimeTarget, err := strconv.ParseFloat(
			profilerReport["report_contents.optimization_configuration.slo.median_response_time.target"].(string), 64)
		if err != nil {
			return fmt.Errorf(
				"unable to parse profiler report_contents.optimization_configuration.slo.median_response_time.target into float64: %w", err)
		}
		// Use rounding for parity with UI logic
		newOptimizerConfig.Config.Slo.ErrorPercent.Target = adaptivePrecisionRound(errorPercentTarget, 4)
		newOptimizerConfig.Config.Slo.MedianResponseTime.Target = adaptivePrecisionRound(medianResponseTimeTarget, 3)

		// Set ignored blockers
		prefix := "report_contents.optimization_blockers."
		rawBlockers := make(map[string]interface{})

		for key, value := range profilerReport {
			if strings.HasPrefix(key, prefix) {
				parts := strings.TrimPrefix(key, prefix)
				segments := strings.Split(parts, ".")
				setNestedMap(rawBlockers, segments, value)
			}
		}
		if len(rawBlockers) != 0 {
			var ignoredBlockers IgnoredBlockers
			ignoredBlockers.Timestamp = time.Now().UTC().String()
			ignoredBlockers.Principal = Principal{
				Type: config.GetCurrentContext().AuthMethod,
				Id:   config.GetCurrentContext().User,
			}

			blockers := assignToBlocker(rawBlockers)

			log.Warn("Optimizer has the following unresolved blockers:\n")

			// Format output
			headers := []string{"Blocker Name", "Description", "Impact", "Overridable"}
			var blockersOutput [][]string

			val := reflect.ValueOf(&blockers).Elem()
			typ := val.Type()
			for i := 0; i < val.NumField(); i++ {
				blockerField := val.Field(i)
				if blocker, ok := blockerField.Interface().(*Blocker); ok && blocker != nil {
					fieldName := typ.Field(i).Name
					blockersOutput = append(blockersOutput, []string{fieldName, blocker.Description, blocker.Impact, strconv.FormatBool(blocker.Overridable)})
				}
			}

			output.PrintCmdOutputCustom(cmd, blockers, &output.Table{
				Headers: headers,
				Lines:   blockersOutput,
				Detail:  true,
			})

			// Ascertain overrideability of blockers
			if flags.overrideSoftBlockers || flags.overrideHardBlockers {
				if flags.overrideHardBlockers {
					log.Warn("Caution: overriding hard blockers")
				} else if flags.overrideSoftBlockers {
					if checkHardBlockers(&blockers) {
						return fmt.Errorf("cannot soft override, hard blockers present; resolve before onboarding the optimizer")
					} else {
						log.Warn("overriding soft blockers")
					}
				}
			} else {
				return fmt.Errorf("resolve the listed blockers before onboarding the optimizer")
			}

			ignoredBlockers.Blockers = blockers
			newOptimizerConfig.IgnoredBlockers = ignoredBlockers
		}

		// Set suspensions to empty object
		newOptimizerConfig.Suspensions = make(map[string]Suspension)

		// config file overrides
		if flags.filePath != "" {
			configFile, err := os.Open(flags.filePath)
			if err != nil {
				return fmt.Errorf("os.Open(flags.filePath): %w", err)
			}
			defer configFile.Close()

			configBytes, _ := io.ReadAll(configFile)
			// NOTE unmarshalling on top of the existing config will overwrite it with only values explicitly set by the file
			err = yaml.Unmarshal(configBytes, &newOptimizerConfig)
			if err != nil {
				return fmt.Errorf("yaml.Unmarshal(configBytes, &configStruct): %w", err)
			}
		}

		// write new config to ORION
		headers := map[string]string{
			"layer-type": "TENANT",
			"layer-id":   config.GetCurrentContext().Tenant,
		}
		var res any

		if flags.create {
			urlStr := fmt.Sprintf("knowledge-store/v1/objects/%v:optimizer", flags.solutionName)
			if err = api.JSONPost(urlStr, newOptimizerConfig, &res, &api.Options{Headers: headers}); err != nil {
				return fmt.Errorf("failed to create knowledge object for optimizer configuration: %w", err)
			}
		} else {
			urlStr := fmt.Sprintf("knowledge-store/v1a/objects/%v:optimizer/%v", flags.solutionName, newOptimizerConfig.OptimizerID)
			if err = api.JSONPut(urlStr, newOptimizerConfig, &res, &api.Options{Headers: headers}); err != nil {
				return fmt.Errorf("failed to update knowledge object with new optimizer configuration: %w", err)
			}
		}

		output.PrintCmdStatus(cmd, fmt.Sprintf("Optimizer configured with ID %q\n", newOptimizerConfig.OptimizerID))
		return nil
	}
}

func assignToBlocker(rawBlockers map[string]interface{}) (blockers Blockers) {
	for key := range rawBlockers {
		blocker := &Blocker{}
		switch key {
		case "stateful":
			blocker.Description = "There are stateful pods in the workload"
			blocker.Impact = "Stateful workloads can’t be spun up and down frequently, which is required for our tuning instance to perform optimization"
			blockers.Stateful = blocker
		case "no_traffic":
			blocker.Description = "Your workload doesn’t have enough traffic to run optimization tests"
			blocker.Impact = "We can’t evaluate the impact of the changes we’re making on application performance and reliability"
			blockers.NoTraffic = blocker
		case "resources_not_specified":
			blocker.Description = "Your workloads don't have any existing thresholds for requests and limits on CPU and memory resources"
			blocker.Impact = "Without these thresholds, we aren't able to determine a baseline and usable range for optimization tests"
			blockers.ResourcesNotSpecified = blocker
		case "cpu_not_specified":
			blocker.Description = "Your workloads don't have any existing thresholds for requests and limits on CPU"
			blocker.Impact = "Without these thresholds, we aren't able to determine a baseline and usable range for optimization tests"
			blockers.CPUNotSpecified = blocker
		case "mem_not_specified":
			blocker.Description = "Your workloads don't have any existing thresholds for requests and limits on memory"
			blocker.Impact = "Without these thresholds, we aren't able to determine a baseline and usable range for optimization tests"
			blockers.MemNotSpecified = blocker
		case "cpu_resources_change":
			blocker.Description = "CPU requests and/or limits on the workload have changed during the last 7 days"
			blocker.Impact = "These values must be stable for 7 days prior to optimization so that we can analyze historical metrics from a stable resource"
			blockers.CPUResourcesChange = blocker
		case "mem_resources_change":
			blocker.Description = "Memory requests and/or limits on the workload have changed during the last 7 days"
			blocker.Impact = "These values must be stable for 7 days prior to optimization so that we can analyze historical metrics from a stable resource"
			blockers.MemoryResourcesChange = blocker
		case "k8s_metrics_deficient":
			blocker.Description = "Kubernetes metrics for CPU and/or memory were not recorded at least 90% of the time during the last 7 days"
			blocker.Impact = "Without adequate coverage (90%) over the last 7 days, we can't analyze the historical metrics to determine baseline performance"
			blockers.K8sMetricsDeficient = blocker
		case "apm_metrics_missing":
			blocker.Description = "There were no APM metrics for this workload during the last 7 days. The workload may not be instrumented with APM"
			blocker.Impact = "We aren't able to perform optimization for workloads that don't have APM metrics"
			blockers.APMMetricsMissing = blocker
		case "apm_metrics_deficient":
			blocker.Description = "APM metrics for latency, error percent, and/or calls per minute were not recorded at least 90% of the time during the last 7 days"
			blocker.Impact = "Without adequate coverage (90%) over the last 7 days, we can't analyze the historical metrics to determine baseline performance"
			blockers.APMMetricsDeficient = blocker
		case "multiple_apm":
			blocker.Description = "More than one container in this workload is reporting APM metrics"
			blocker.Impact = "If multiple containers have APM metrics, we don't know which one to optimize"
			blockers.MultipleAPM = blocker
		case "unequal_load_distribution":
			blocker.Description = "Individual pods in your workload aren't receiving equal shares of incoming traffic"
			blocker.Impact = "Since pods may be serving different load, this prevents us from benchmarking a pod with an optimized configuration against a baseline pod"
			blockers.UnequalLoadDistribution = blocker
		case "no_scaling":
			blocker.Description = "We didn't observe any horizontal scaling on this workload in the last 7 day."
			blocker.Impact = "To perform optimization, the workload must have autoscaled in the last 7 days and must use average CPU utilization as a metric for driving auto-scaling"
			blockers.NoScaling = blocker
		case "insufficient_relative_scaling":
			blocker.Description = "During the last 7 days, the workload did not scale above the observed minimum of replicas during this period at least 25% of the time"
			blocker.Impact = "To perform optimization, the workload must be actively scaling above its minimum scale at least 25% of the time"
			blockers.InsufficientRelativeScaling = blocker
		case "insufficient_fixed_scaling":
			blocker.Description = "During the last 7 days, the workload did not have a minimum horizontal scale of at least 3 for the entire period"
			blocker.Impact = "To perform optimization, the workload must have a minimum horizontal scale of 3 (during the last 7 days). This ensures that optimization tests receive at most one quarter of the total live load"
			blockers.InsufficientFixedScaling = blocker
		case "mtbf_high":
			blocker.Description = "The per-pod mean time between failure (MTBF) is less than 1 day"
			blocker.Impact = "This may indicate that your workload is unstable, which could impact optimization results"
			blockers.MTBFHigh = blocker
		case "error_rate_high":
			blocker.Description = "More than 5% of requests are resulting in errors"
			blocker.Impact = "This may indicate that there's a larger problem in the service, so we cannot perform optimization tests"
			blockers.ErrorRateHigh = blocker
		case "no_orchestration":
			blocker.Description = "We didn't detect the Orchestration Client on your workload, which is responsible for provisioning and deprovisioning the Servo Agent"
			blocker.Impact = "Optimization cannot be performed because the Servo Agent can't be provisioned on your cluster"
			blockers.NoOrchestrationAgent = blocker
		default:
			log.Warnf("Unknown blocker %q encountered", key)
		}
		data := rawBlockers[key].(map[string]interface{})
		if data["overridable"] == "true" {
			blocker.Overridable = true
		} else {
			blocker.Overridable = false
		}
	}
	return blockers
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

func adaptivePrecisionRound(value float64, maxPrecision int) float64 {
	if floored := math.Floor(value); floored == value {
		return floored
	}
	precision := math.Max(0, math.Min(float64(maxPrecision), math.Round(3-math.Log10(value))))
	return math.Round(value*math.Pow10(int(precision))) / math.Pow10(int(precision))
}

func getOptimizerConfig(optimizerId string, workloadId string, solutionName string) (OptimizerConfiguration, error) {
	var optimizerConfig OptimizerConfiguration
	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   config.GetCurrentContext().Tenant,
	}

	if optimizerId != "" {
		var response configJsonStoreItem

		urlStr := fmt.Sprintf("knowledge-store/v1/objects/%v:optimizer/%v", solutionName, optimizerId)
		err := api.JSONGet(urlStr, &response, &api.Options{Headers: headers})
		if err != nil {
			if problem, ok := err.(api.Problem); ok && problem.Status == 404 {
				return optimizerConfig, fmt.Errorf("%w: No matches found for the given optimizerId", errOptimizerConfigNotFound)
			}
			return optimizerConfig, fmt.Errorf("unable to fetch existing config by optimizer ID. api.JSONGet: %w", err)
		}
		optimizerConfig = response.Data
	} else if workloadId != "" {
		var configPage configJsonStorePage
		// NOTE orion objects only store the last portion of the workloadId. Only support k8sDeployment currently
		var queryStr string
		idSuffix := strings.Split(workloadId, ":")[2]
		if strings.HasPrefix(workloadId, "k8s:deployment:") {
			queryStr = url.QueryEscape(fmt.Sprintf("data.target.k8sDeployment.workloadId eq %q", idSuffix))
		} else {
			return optimizerConfig, fmt.Errorf("optimizer object does not support workloads type for given ID %q", workloadId)
		}
		urlStr := fmt.Sprintf("knowledge-store/v1/objects/%v:optimizer?filter=%v", solutionName, queryStr)

		err := api.JSONGet(urlStr, &configPage, &api.Options{Headers: headers})
		if err != nil {
			return optimizerConfig, fmt.Errorf("unable to fetch existing config by workload ID. api.JSONGet: %w", err)
		}
		if configPage.Total > 1 {
			return optimizerConfig, fmt.Errorf("found %v optimizer configurations for the given workloadID", configPage.Total)
		}
		if configPage.Total < 1 {
			return optimizerConfig, fmt.Errorf("%w: No matches found for the given workloadId", errOptimizerConfigNotFound)
		}

		optimizerConfig = configPage.Items[0].Data
	} else {
		return optimizerConfig, errors.New("must provide either workloadId or optimizerId")
	}

	return optimizerConfig, nil
}

var singleReportTemplate = template.Must(template.New("").Parse(`
SINCE -1w
FETCH events(k8sprofiler:report){attributes}
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

	mainDataSet := resp.Main()
	if mainDataSet == nil {
		return nil, errors.New("no events found, main data set was nil")
	}
	mainDataSetData := mainDataSet.Data
	if len(mainDataSetData) < 1 {
		return nil, errors.New("no events found, main data set had no rows")
	}
	if len(mainDataSetData[0]) < 1 {
		return nil, errors.New("no events found, main data first row had no columns")
	}
	eventDataSet, ok := mainDataSetData[0][0].(*uql.DataSet)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T for event data set", mainDataSetData[0][0])
	}
	if eventDataSet == nil {
		return nil, errors.New("no events found, event data set was nil")
	}
	if len(eventDataSet.Data) < 1 {
		return nil, errors.New("no events found, event data set had no rows")
	}
	if len(eventDataSet.Data[0]) < 1 {
		return nil, errors.New("no events found, event data first row had no columns")
	}
	eventComplexData, ok := eventDataSet.Data[0][0].(uql.ComplexData)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T for event data set", eventDataSet.Data[0][0])
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
			return fmt.Errorf("%w: key %q does not exist", errProfilerMissingData, key)
		}
		if _, ok := val.(string); !ok {
			return fmt.Errorf("%w: string assertion failed for key %q value %q", errProfilerInvalidData, key, val)
		}
	}
	return nil
}
