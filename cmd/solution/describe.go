package solution

import (
	"github.com/apex/log"
	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/cmdkit"
	"github.com/spf13/cobra"
)

var solutionDescribeCmd = &cobra.Command{
	Use:   "describe --name=<solution>",
	Short: "",
	Long:  ``,
	Run:   solutionDescribe,
}

func getSolutionDescribeCmd() *cobra.Command {
	solutionDescribeCmd.Flags().
		String("solution", "", "The name of the solution to describe")
	_ = solutionDescribeCmd.MarkFlagRequired("solution")

	return solutionDescribeCmd
}

func solutionDescribe(cmd *cobra.Command, args []string) {
	log.Info("Fetching the list of solutions...")

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	// get data and display
	cmdkit.FetchAndPrint(cmd, getSolutionDescribeUrl(), &cmdkit.FetchAndPrintOptions{Headers: headers, IsCollection: true})
}

func getSolutionDescribeUrl() string {
	return "objstore/v1beta/objects/extensibility"
}
