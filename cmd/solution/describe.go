package solution

import (
	"net/url"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionDescribeCmd = &cobra.Command{
	Use:     "describe <solution-name>",
	Args:    cobra.MaximumNArgs(1),
	Short:   "Describe solution",
	Long:    `Obtain metadata about a solution`,
	Example: `  fsoc solution describe spacefleet`,
	Run:     solutionDescribe,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
	},
}

func getSolutionDescribeCmd() *cobra.Command {
	solutionDescribeCmd.Flags().
		String("solution", "", "The name of the solution to describe")
	_ = solutionDescribeCmd.Flags().MarkDeprecated("solution", "please use argument instead.")

	return solutionDescribeCmd
}

func solutionDescribe(cmd *cobra.Command, args []string) {
	solution := getSolutionNameFromArgs(cmd, args, "solution")

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	log.WithField("solution", solution).Info("Getting solution details")
	var res Solution
	err := api.JSONGet(getSolutionDescribeUrl(url.PathEscape(solution)), &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Cannot get solution details: %v", err)
	}
	filter := output.CreateFilter("", []int{})

	output.PrintCmdOutput(cmd, res, filter)
}

func getSolutionDescribeUrl(id string) string {
	return "objstore/v1beta/objects/extensibility:solution/" + id
}
