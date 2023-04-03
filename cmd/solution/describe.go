package solution

import (
	"fmt"
	"net/url"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionDescribeCmd = &cobra.Command{
	Use:   "describe <solution-name>",
	Short: "",
	Long:  ``,
	Run:   solutionDescribe,
}

type Solution struct {
	ID             string `json:"id"`
	LayerID        string `json:"layerId"`
	LayerType      string `json:"layerType"`
	ObjectMimeType string `json:"objectMimeType"`
	TargetObjectId string `json:"targetObjectId"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	DisplayName    string `json:"displayName"`
}

func getSolutionDescribeCmd() *cobra.Command {
	solutionDescribeCmd.Flags().
		String("solution", "", "The name of the solution to describe")
	_ = solutionDescribeCmd.Flags().MarkDeprecated("solution", "The --solution flag is deprecated, please use argument instead.")
	//_ = solutionDescribeCmd.MarkFlagRequired("solution")

	return solutionDescribeCmd
}

func solutionDescribe(cmd *cobra.Command, args []string) {
	log.Info("Fetching the details of the specified solutions...")
	solution, _ := cmd.Flags().GetString("solution")
	if len(args) > 0 {
		solution = args[0]
	} else {
		if len(solution) == 0 {
			log.Fatal("A non-empty \"solution-name\" argument is required.")
		}
	}

	cfg := config.GetCurrentContext()
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	log.Infof("Getting details of the '%s' solution", solution)
	var res Solution
	_ = api.JSONGet(getSolutionDescribeUrl(url.PathEscape(solution)), &res, &api.Options{Headers: headers})
	fmt.Printf("ID: %s\n", res.ID)
	fmt.Printf("LayerID: %s\n", res.LayerID)
	fmt.Printf("layerType: %s\n", res.LayerType)
	fmt.Printf("ObjectMimeType: %s\n", res.ObjectMimeType)
	fmt.Printf("TargetObjectId: %s\n", res.TargetObjectId)
	fmt.Printf("CreatedAt: %s\n", res.CreatedAt)
	fmt.Printf("UpdatedAt: %s\n", res.UpdatedAt)
}

func getSolutionDescribeUrl(id string) string {
	//println(id)
	return "objstore/v1beta/objects/extensibility:solution/" + id
}
