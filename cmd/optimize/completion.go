package optimize

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/uql"
	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/platform/api"
)

func registerReportCompletion(command *cobra.Command, flag profilerReportFlag) {
	_ = command.RegisterFlagCompletionFunc(
		flag.String(),
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			config.SetActiveProfile(cmd, args, false)
			return completeFlagFromMelt(flag, cmd, args, toComplete)
		})
}

func registerOptimizerCompletion(command *cobra.Command, flag optimizerFlag) {
	_ = command.RegisterFlagCompletionFunc(
		flag.String(),
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			config.SetActiveProfile(cmd, args, false)
			return completeFlagFromKS(flag, cmd, args, toComplete)
		})
}

func completeFlagFromKS(flag optimizerFlag, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	objStoreUrl := getKnowledgeURL(cmd, "optimizer", "data")

	headers := getOrionTenantHeaders()

	httpOptions := &api.Options{Headers: headers}

	var result api.CollectionResult[configJsonStoreItem]
	err := api.JSONGetCollection[configJsonStoreItem](objStoreUrl, &result, httpOptions)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var ids []string

	// Filter completion results by toComplete
	for _, s := range result.Items {
		val := flag.ValueFromObject(&s.Data)
		if strings.HasPrefix(val, toComplete) {
			ids = append(ids, val)
		}
	}
	return ids, cobra.ShellCompDirectiveNoFileComp

}

func completeFlagFromMelt(flag profilerReportFlag, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	filterSegments := []string{
		fmt.Sprintf(`attributes(%s) ~ "%s*"`, flag.K8sAttribute(), toComplete),
	}
	flags := cmd.Flags()
	if flags != nil {
		profileFlags := []profilerReportFlag{profilerReportFlagCluster, profilerReportFlagNamespace, profilerReportFlagWorkloadName}
		for _, f := range profileFlags {
			val, _ := flags.GetString(f.String())
			if val != "" {
				filterSegments = append(
					filterSegments,
					fmt.Sprintf(`attributes(%s) = "%s"`, f.K8sAttribute(), val),
				)
			}
		}
	}

	filter := fmt.Sprintf("[%s]", strings.Join(filterSegments, " && "))

	fetchField := flag.ReportAttribute()
	query := fmt.Sprintf(`
				SINCE
					-3d
				FETCH
					id,
					events(k8sprofiler:report){attributes(%s)}
				FROM
					entities(k8s:deployment)%s
				LIMITS
					events.count(1)`, fetchField, filter)

	resp, err := uql.ClientV1.ExecuteQuery(&uql.Query{Str: query})
	if err != nil || resp.HasErrors() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	m := resp.Main()
	if m == nil || len(m.Data) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var results []string
	for _, d := range m.Data {
		if len(d) < 2 {
			continue
		}
		eventDataSet, ok := d[1].(*uql.DataSet)
		if !ok {
			continue
		}
		if len(eventDataSet.Data) < 1 || len(eventDataSet.Data[0]) < 1 {
			continue
		}
		s, ok := eventDataSet.Data[0][0].(string)
		if ok {
			results = append(results, s)
		}
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}
