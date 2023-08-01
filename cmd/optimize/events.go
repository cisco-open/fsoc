package optimize

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/output"
)

var defaultEvents = []string{
	"optimization_baselined",
	"optimization_started",
	"optimization_ended",
	"stage_started",
	"stage_ended",
	"experiment_started",
	"experiment_ended",
	"experiment_deployment_started",
	"experiment_deployment_completed",
	"experiment_measurement_started",
	"experiment_measurement_completed",
	"experiment_described",
	"recommendation_identified",
	"recommendation_verified",
	"recommendation_invalidated",
}

var progressEvents = []string{
	"optimization_progress",
	"stage_progress",
	"experiment_progress",
}

func init() {
	// TODO move this logic to optimize root when implementing unit tests
	optimizeCmd.AddCommand(NewCmdEvents())
	optimizeCmd.AddCommand(NewCmdRecommendations())
}

type eventsFlags struct {
	clusterId    string
	namespace    string
	workloadName string
	optimizerId  string
	since        string
	until        string
	count        int
	solutionName string
}

type eventsCmdFlags struct {
	eventsFlags
	includeProgress bool
	events          []string
}

type eventsRow struct {
	Timestamp       time.Time
	EventAttributes map[string]any
}

func NewCmdEvents() *cobra.Command {
	var flags eventsCmdFlags
	command := &cobra.Command{
		Use:   "events",
		Short: "Retrieve event logs for a given optimization/workload. Useful for monitoring and debug",
		Example: `  fsoc optimize events
  fsoc optimize events --since -7d until 2023-07-31
  fsoc optimize events --events="experiment_deployment_started,experiment_deployment_completed"
  fsoc optimize events --optimizer-id namespace-name-00000000-0000-0000-0000-000000000000 --count 5
  fsoc optimize events --namespace some-namespace --cluster-id 00000000-0000-0000-0000-000000000000
  fsoc optimize events --workload-name some-workload`,
		RunE:             listEvents(&flags),
		TraverseChildren: true,
		Annotations: map[string]string{
			output.TableFieldsAnnotation:  "OptimizerId: .EventAttributes[\"optimize.optimization.optimizer_id\"], EventType: .EventAttributes[\"appd.event.type\"], Timestamp: .Timestamp",
			output.DetailFieldsAnnotation: "OptimizerId: .EventAttributes[\"optimize.optimization.optimizer_id\"], EventType: .EventAttributes[\"appd.event.type\"], Timestamp: .Timestamp, Attributes: .EventAttributes",
		},
	}

	command.Flags().StringVarP(&flags.clusterId, "cluster-id", "c", "", "Retrieve events constrained to a specific cluster by its ID")
	command.Flags().StringVarP(&flags.namespace, "namespace", "n", "", "Retrieve events constrained to a specific namespace by its name")
	command.Flags().StringVarP(&flags.workloadName, "workload-name", "w", "", "Retrieve events constrained to a specific workload by its name")
	command.Flags().StringVarP(&flags.optimizerId, "optimizer-id", "i", "", "Retrieve events for a specific optimizer by its ID")
	command.MarkFlagsMutuallyExclusive("optimizer-id", "cluster-id")
	command.MarkFlagsMutuallyExclusive("optimizer-id", "namespace")
	command.MarkFlagsMutuallyExclusive("optimizer-id", "workload-name")

	command.Flags().BoolVarP(&flags.includeProgress, "include-progress", "p", false, "Include progress events in query and output")
	command.Flags().StringSliceVarP(&flags.events, "events", "e", defaultEvents, "Customize the types of events to be retrieved")
	command.MarkFlagsMutuallyExclusive("include-progress", "events")

	command.Flags().StringVarP(&flags.since, "since", "s", "", "Retrieve events contained in the time interval starting at a relative or exact time. (default: -1h)")
	command.Flags().StringVarP(&flags.until, "until", "u", "", "Retrieve events contained in the time interval ending at a relative or exact time. (default: now)")

	command.Flags().IntVarP(&flags.count, "count", "", -1, "Limit the number of events retrieved to the specified count")

	command.Flags().StringVarP(&flags.solutionName, "solution-name", "", "optimize", "Intended for developer usage, overrides the name of the solution defining the FMM types for reading")
	if err := command.LocalFlags().MarkHidden("solution-name"); err != nil {
		log.Warnf("Failed to set events solution-name flag hidden: %v", err)
	}

	return command
}

type eventsTemplateValues struct {
	Since  string
	Until  string
	Events string
	Filter string
	Limits string
}

var eventsTemplate = template.Must(template.New("").Parse(`
{{ with .Since }}SINCE {{ . }}
{{ end -}}
{{ with .Until }}UNTIL {{ . }}
{{ end -}}
FETCH events(
		{{ .Events }}
	)
	{{ with .Filter }}[{{ . }}]
	{{ end -}}
	{attributes, timestamp}
{{ with .Limits }}LIMITS events.count({{ . }})
{{ end -}}
`))

func listEvents(flags *eventsCmdFlags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// setup query
		tempVals := eventsTemplateValues{
			Since: flags.since,
			Until: flags.until,
		}

		if flags.includeProgress {
			flags.events = append(flags.events, progressEvents...)
		}
		fullyQualifiedEvents := make([]string, 0, len(flags.events))
		for _, value := range flags.events {
			fullyQualifiedEvents = append(fullyQualifiedEvents, fmt.Sprintf("%v:%v", flags.solutionName, value))
		}
		tempVals.Events = strings.Join(fullyQualifiedEvents, ",\n		")

		filterList := make([]string, 0, 2)
		if flags.clusterId != "" {
			filterList = append(filterList, fmt.Sprintf("attributes(k8s.cluster.id) = %q", flags.clusterId))
		}
		if flags.optimizerId != "" {
			filterList = append(filterList, fmt.Sprintf("attributes(optimize.optimization.optimizer_id) = %q", flags.optimizerId))
		} else if flags.namespace != "" || flags.workloadName != "" {
			optimizerIds, err := listOptimizations(&flags.eventsFlags)
			if err != nil {
				return fmt.Errorf("listOptimizations: %w", err)
			}
			if len(optimizerIds) < 1 {
				output.PrintCmdStatus(cmd, "No optimization entities found matching the given criteria\n")
				return nil
			}
			optIdStr := strings.Join(optimizerIds, "\", \"")
			filterList = append(filterList, fmt.Sprintf("attributes(optimize.optimization.optimizer_id) IN [\"%v\"]", optIdStr))
		}
		tempVals.Filter = strings.Join(filterList, " && ")

		if flags.count != -1 {
			if flags.count > 1000 {
				return errors.New("Counts higher than 1000 are not supported")
			}
			tempVals.Limits = strconv.Itoa(flags.count)
		}

		var buff bytes.Buffer
		if err := eventsTemplate.Execute(&buff, tempVals); err != nil {
			return fmt.Errorf("eventsTemplate.Execute: %w", err)
		}
		query := buff.String()

		// execute query, process results
		resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
		if err != nil {
			return fmt.Errorf("uql.ExecuteQuery: %w", err)
		}
		if resp.HasErrors() {
			log.Error("Execution of events query encountered errors. Returned data may not be complete!")
			for _, e := range resp.Errors() {
				log.Errorf("%s: %s", e.Title, e.Detail)
			}
		}

		main_data_set := resp.Main()
		if main_data_set == nil || len(main_data_set.Data) < 1 {
			output.PrintCmdStatus(cmd, "No event results found for given input\n")
			return nil
		}
		if len(main_data_set.Data[0]) < 1 {
			return fmt.Errorf("Main dataset %v first row has no columns", main_data_set.Name)
		}

		data_set, ok := main_data_set.Data[0][0].(*uql.DataSet)
		if !ok {
			return fmt.Errorf("Main dataset %v first row first column (type %T) could not be converted to *uql.DataSet", main_data_set.Name, main_data_set.Data[0][0])
		}
		eventRows, err := extractEventsData(data_set)
		if err != nil {
			return fmt.Errorf("extractEventsData: %w", err)
		}

		// handle pagination
		_, next_ok := data_set.Links["next"]
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
				log.Errorf("Continuation of events query (page %v) encountered errors. Returned data may not be complete!", page)
				for _, e := range resp.Errors() {
					log.Errorf("%s: %s", e.Title, e.Detail)
				}
			}
			main_data_set := resp.Main()
			if main_data_set == nil {
				log.Errorf("Continuation of events query (page %v) has nil main data. Returned data may not be complete!", page)
				break
			}
			if len(main_data_set.Data) < 1 {
				return fmt.Errorf("Page %v main dataset %v has no rows", page, main_data_set.Name)
			}
			if len(main_data_set.Data[0]) < 1 {
				return fmt.Errorf("Page %v main dataset %v first row has no columns", page, main_data_set.Name)
			}
			data_set, ok = main_data_set.Data[0][0].(*uql.DataSet)
			if !ok {
				return fmt.Errorf("Page %v main dataset %v first row first column (type %T) could not be converted to *uql.DataSet", page, main_data_set.Name, main_data_set.Data[0][0])
			}

			newRows, err := extractEventsData(data_set)
			if err != nil {
				return fmt.Errorf("page %v extractEventsData: %w", page, err)
			}
			eventRows = append(eventRows, newRows...)
			_, next_ok = data_set.Links["next"]
		}

		output.PrintCmdOutput(cmd, struct {
			Items []eventsRow `json:"items"`
			Total int         `json:"total"`
		}{Items: eventRows, Total: len(eventRows)})

		return nil
	}
}

type recommendationsCmdFlags struct {
	eventsFlags
	includeInvalidated bool
}

func NewCmdRecommendations() *cobra.Command {
	var flags recommendationsCmdFlags
	command := &cobra.Command{
		Use:              "recommendations",
		Short:            "Retrieve resulting recommendations for a given optimization/workload",
		Example:          `TODO`,
		RunE:             listRecommendations(&flags),
		TraverseChildren: true,
		Annotations: map[string]string{
			output.TableFieldsAnnotation:  "OptimizerId: .EventAttributes[\"optimize.optimization.optimizer_id\"], State: .EventAttributes[\"optimize.recommendation.state\"], CPUcores: .EventAttributes[\"optimize.recommendation.settings.cpu\"], MemoryGiB: .EventAttributes[\"optimize.recommendation.settings.memory\"], Timestamp: .Timestamp",
			output.DetailFieldsAnnotation: "OptimizerId: .EventAttributes[\"optimize.optimization.optimizer_id\"], State: .EventAttributes[\"optimize.recommendation.state\"], CPUcores: .EventAttributes[\"optimize.recommendation.settings.cpu\"], MemoryGiB: .EventAttributes[\"optimize.recommendation.settings.memory\"], Timestamp: .Timestamp, Attributes: .EventAttributes",
		},
	}

	command.Flags().StringVarP(&flags.clusterId, "cluster-id", "c", "", "Retrieve recommendations constrained to a specific cluster by its ID")
	command.Flags().StringVarP(&flags.namespace, "namespace", "n", "", "Retrieve recommendations constrained to a specific namespace by its name")
	command.Flags().StringVarP(&flags.workloadName, "workload-name", "w", "", "Retrieve recommendations constrained to a specific workload by its name")
	command.Flags().StringVarP(&flags.optimizerId, "optimizer-id", "i", "", "Retrieve recommendations for a specific optimizer by its ID")
	command.MarkFlagsMutuallyExclusive("optimizer-id", "cluster-id")
	command.MarkFlagsMutuallyExclusive("optimizer-id", "namespace")
	command.MarkFlagsMutuallyExclusive("optimizer-id", "workload-name")

	command.Flags().BoolVarP(&flags.includeInvalidated, "include-invalidated", "", false, "Include recommendations that have not been verified")

	command.Flags().StringVarP(&flags.since, "since", "s", "-52w", "Retrieve recommendations contained in the time interval starting at a relative or exact time.")
	command.Flags().StringVarP(&flags.until, "until", "u", "", "Retrieve recommendations contained in the time interval ending at a relative or exact time. (default: now)")

	command.Flags().IntVarP(&flags.count, "count", "", 1, "Limit the number of recommendations retrieved to the specified count")

	command.Flags().StringVarP(&flags.solutionName, "solution-name", "", "optimize", "Intended for developer usage, overrides the name of the solution defining the FMM types for reading")
	if err := command.LocalFlags().MarkHidden("solution-name"); err != nil {
		log.Warnf("Failed to set recommendations solution-name flag hidden: %v", err)
	}

	return command
}

type recommendationsTemplateValues struct {
	Since              string
	Until              string
	IncludeInvalidated bool
	Filter             string
	Limits             string
	SolutionName       string
}

var recommendationsTemplate = template.Must(template.New("").Parse(`
{{ with .Since }}SINCE {{ . }}
{{ end -}}
{{ with .Until }}UNTIL {{ . }}
{{ end -}}
FETCH events(
		{{- if .IncludeInvalidated }}
		{{ .SolutionName }}:recommendation_identified,
		{{ .SolutionName }}:recommendation_invalidated,
		{{- end }}
		{{ .SolutionName }}:recommendation_verified
	)
	{{ with .Filter }}[{{ . }}]
	{{ end -}}
	{attributes, timestamp}
{{ with .Limits }}LIMITS events.count({{ . }})
{{ end -}}
`))

func listRecommendations(flags *recommendationsCmdFlags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// setup query
		tempVals := recommendationsTemplateValues{
			Since:              flags.since,
			Until:              flags.until,
			IncludeInvalidated: flags.includeInvalidated,
			SolutionName:       flags.solutionName,
		}

		filterList := make([]string, 0, 2)
		if flags.clusterId != "" {
			filterList = append(filterList, fmt.Sprintf("attributes(k8s.cluster.id) = %q", flags.clusterId))
		}
		if flags.optimizerId != "" {
			filterList = append(filterList, fmt.Sprintf("attributes(optimize.optimization.optimizer_id) = %q", flags.optimizerId))
		} else if flags.namespace != "" || flags.workloadName != "" {
			optimizerIds, err := listOptimizations(&flags.eventsFlags)
			if err != nil {
				return fmt.Errorf("listOptimizations: %w", err)
			}
			if len(optimizerIds) < 1 {
				output.PrintCmdStatus(cmd, "No optimization entities found matching the given criteria\n")
				return nil
			}
			optIdStr := strings.Join(optimizerIds, "\", \"")
			filterList = append(filterList, fmt.Sprintf("attributes(optimize.optimization.optimizer_id) IN [\"%v\"]", optIdStr))
		}
		tempVals.Filter = strings.Join(filterList, " && ")

		if flags.count != -1 {
			if flags.count > 1000 {
				return errors.New("Counts higher than 1000 are not supported")
			}
			tempVals.Limits = strconv.Itoa(flags.count)
		}

		var buff bytes.Buffer
		if err := recommendationsTemplate.Execute(&buff, tempVals); err != nil {
			return fmt.Errorf("recommendationsTemplate.Execute: %w", err)
		}
		query := buff.String()

		// execute query, process results
		resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
		if err != nil {
			return fmt.Errorf("uql.ExecuteQuery: %w", err)
		}
		if resp.HasErrors() {
			log.Error("Execution of recommendations query encountered errors. Returned data may not be complete!")
			for _, e := range resp.Errors() {
				log.Errorf("%s: %s", e.Title, e.Detail)
			}
		}

		main_data_set := resp.Main()
		if main_data_set == nil || len(main_data_set.Data) < 1 {
			output.PrintCmdStatus(cmd, "No recommendation results found for given input\n")
			return nil
		}
		if len(main_data_set.Data[0]) < 1 {
			return fmt.Errorf("Main dataset %v first row has no columns", main_data_set.Name)
		}

		data_set, ok := main_data_set.Data[0][0].(*uql.DataSet)
		if !ok {
			return fmt.Errorf("Main dataset %v first row first column (type %T) could not be converted to *uql.DataSet", main_data_set.Name, main_data_set.Data[0][0])
		}
		recommendationRows, err := extractEventsData(data_set)
		if err != nil {
			return fmt.Errorf("extractEventsData: %w", err)
		}

		// handle pagination
		_, next_ok := data_set.Links["next"]
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
				log.Errorf("Continuation of recommendations query (page %v) encountered errors. Returned data may not be complete!", page)
				for _, e := range resp.Errors() {
					log.Errorf("%s: %s", e.Title, e.Detail)
				}
			}
			main_data_set := resp.Main()
			if main_data_set == nil {
				log.Errorf("Continuation of recommendations query (page %v) has nil main data. Returned data may not be complete!", page)
				break
			}
			if len(main_data_set.Data) < 1 {
				return fmt.Errorf("Page %v main dataset %v has no rows", page, main_data_set.Name)
			}
			if len(main_data_set.Data[0]) < 1 {
				return fmt.Errorf("Page %v main dataset %v first row has no columns", page, main_data_set.Name)
			}
			data_set, ok = main_data_set.Data[0][0].(*uql.DataSet)
			if !ok {
				return fmt.Errorf("Page %v main dataset %v first row first column (type %T) could not be converted to *uql.DataSet", page, main_data_set.Name, main_data_set.Data[0][0])
			}

			newRows, err := extractEventsData(data_set)
			if err != nil {
				return fmt.Errorf("page %v extractEventsData: %w", page, err)
			}
			recommendationRows = append(recommendationRows, newRows...)
			_, next_ok = data_set.Links["next"]
		}

		output.PrintCmdOutput(cmd, struct {
			Items []eventsRow `json:"items"`
			Total int         `json:"total"`
		}{Items: recommendationRows, Total: len(recommendationRows)})

		return nil
	}
}

func extractEventsData(dataset *uql.DataSet) ([]eventsRow, error) {
	if dataset == nil {
		return []eventsRow{}, nil
	}
	resp_data := &dataset.Data
	results := make([]eventsRow, 0, len(*resp_data))

	for _, row := range *resp_data {
		attributes := row[0].(uql.ComplexData)
		attributesMap, _ := sliceToMap(attributes.Data)
		timestamp := row[1].(time.Time)
		results = append(results, eventsRow{Timestamp: timestamp, EventAttributes: attributesMap})
	}

	return results, nil
}

type optimizationTemplateValues struct {
	Since        string
	Until        string
	SolutionName string
	Filter       string
}

var optimizationTemplate = template.Must(template.New("").Parse(`
{{ with .Since }}SINCE {{ . }}
{{ end -}}
{{ with .Until }}UNTIL {{ . }}
{{ end -}}
FETCH attributes(optimize.optimization.optimizer_id)
FROM entities({{ .SolutionName }}:optimization)[{{ .Filter }}]
`))

// listOptimizations takes applicable filter criteria from the eventsFlags and returns a list of applicable optimizer IDs
// from the FMM entity optimize:optimization
func listOptimizations(flags *eventsFlags) ([]string, error) {
	tempVals := optimizationTemplateValues{
		Since:        flags.since,
		Until:        flags.until,
		SolutionName: flags.solutionName,
	}

	filterList := make([]string, 0, 3)
	if flags.namespace != "" {
		filterList = append(filterList, fmt.Sprintf("attributes(\"k8s.namespace.name\") = %q", flags.namespace))
	}
	if flags.workloadName != "" {
		filterList = append(filterList, fmt.Sprintf("attributes(\"k8s.workload.name\") = %q", flags.workloadName))
	}
	if len(filterList) < 1 {
		return []string{}, errors.New("Sanity check failed, optimizations query must at least filter on namespace or workload name, otherwise this query can be skipped")
	}
	if flags.clusterId != "" {
		filterList = append(filterList, fmt.Sprintf("attributes(\"k8s.cluster.id\") = %q", flags.clusterId))
	}
	tempVals.Filter = strings.Join(filterList, " && ")

	var buff bytes.Buffer
	if err := optimizationTemplate.Execute(&buff, tempVals); err != nil {
		return []string{}, fmt.Errorf("optimizationTemplate.Execute: %w", err)
	}
	query := buff.String()

	resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
	if err != nil {
		return []string{}, fmt.Errorf("uql.ExecuteQuery: %w", err)
	}
	if resp.HasErrors() {
		log.Error("Execution of optimization query encountered errors. Returned data may not be complete!")
		for _, e := range resp.Errors() {
			log.Errorf("%s: %s", e.Title, e.Detail)
		}
	}

	mainDataSet := resp.Main()
	if mainDataSet == nil {
		return []string{}, nil
	}
	results := make([]string, 0, len(mainDataSet.Data))
	for index, row := range mainDataSet.Data {
		if len(row) < 1 {
			return results, fmt.Errorf("optimization data row %v has no columns", index)
		}
		idStr, ok := row[0].(string)
		if !ok {
			return results, fmt.Errorf("optimization data row %v value %v (type %T) could not be converted to string", index, row[0], row[0])
		}
		results = append(results, idStr)
	}

	_, next_ok := mainDataSet.Links["next"]
	for page := 2; next_ok; page++ {
		resp, err = uql.ContinueQuery(mainDataSet, "next")
		if err != nil {
			return results, fmt.Errorf("page %v uql.ContinueQuery: %w", page, err)
		}

		if resp.HasErrors() {
			log.Errorf("Continuation of optimization query (page %v) encountered errors. Returned data may not be complete!", page)
			for _, e := range resp.Errors() {
				log.Errorf("%s: %s", e.Title, e.Detail)
			}
		}
		mainDataSet = resp.Main()
		if mainDataSet == nil {
			log.Errorf("Continuation of optimization query (page %v) has nil main data. Returned data may not be complete!", page)
			break
		}

		for index, row := range mainDataSet.Data {
			if len(row) < 1 {
				return results, fmt.Errorf("page %v optimization data row %v has no columns", page, index)
			}
			idStr, ok := row[0].(string)
			if !ok {
				return results, fmt.Errorf("page %v optimization data row %v value %v (type %T) could not be converted to string", page, index, row[0], row[0])
			}
			results = append(results, idStr)
		}

		_, next_ok = mainDataSet.Links["next"]
	}

	return results, nil
}
