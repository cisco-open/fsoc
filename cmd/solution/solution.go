// Copyright 2022 Cisco Systems, Inc.
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

package solution

import (
	"net/url"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
)

// solutionCmd represents the login command
var solutionCmd = &cobra.Command{
	Use:   "solution",
	Short: "Perform solution operations",
	Long: `Perform solution lifecycle and control operations.

For more information on platform solutions, see https://developer.cisco.com/docs/cisco-observability-platform/#!create-a-solution-introduction`,
	Example:          `  fsoc solution list`,
	TraverseChildren: true,
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func NewSubCmd() *cobra.Command {
	solutionCmd.AddCommand(getSolutionListCmd())
	solutionCmd.AddCommand(getInitSolutionCmd())
	solutionCmd.AddCommand(getSubscribeSolutionCmd())
	solutionCmd.AddCommand(getUnsubscribeSolutionCmd())
	solutionCmd.AddCommand(getSolutionExtendCmd())
	solutionCmd.AddCommand(getSolutionFixCmd())
	solutionCmd.AddCommand(getSolutionPackageCmd())
	solutionCmd.AddCommand(getSolutionPushCmd())
	solutionCmd.AddCommand(getSolutionDownloadCmd())
	solutionCmd.AddCommand(getSolutionValidateCmd())
	solutionCmd.AddCommand(GetSolutionForkCommand())
	solutionCmd.AddCommand(getSolutionCheckCmd())
	solutionCmd.AddCommand(getSolutionStatusCmd())
	solutionCmd.AddCommand(getSolutionDescribeCmd())
	solutionCmd.AddCommand(getSolutionShowCmd())
	solutionCmd.AddCommand(getSolutionBumpCmd())
	solutionCmd.AddCommand(getSolutionTestCmd())
	solutionCmd.AddCommand(getSolutionTestStatusCmd())
	solutionCmd.AddCommand(getsolutionIsolateCmd())
	solutionCmd.AddCommand(getSolutionZapCmd())
	solutionCmd.AddCommand(getSolutionDeleteCommand())
	solutionListCmd.Flags().StringP("output", "o", "", "Output format (human*, json, yaml)")

	return solutionCmd
}

// getSolutionNameFromArgs gets the solution name from the command line, either from
// the first positional argument or from a flag (deprecated but kepts for backward compatibility).
// The flagName is optional (use "" to omit).
// Prints error message and terminates if the name is missing/empty
func getSolutionNameFromArgs(cmd *cobra.Command, args []string, flagName string) string {
	// get solution name from a flag, if provided (deprecated but kept for backward compatibility)
	var nameFromFlag string
	if flagName != "" {
		var err error
		nameFromFlag, err = cmd.Flags().GetString(flagName)
		if err != nil {
			log.Fatalf("Error parsing flag %q: %v", flagName, err)
		}
	}

	// get solution name from the first positional argument and
	// return it (or fail if flag was provided as well)
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	if name != "" {
		if nameFromFlag != "" {
			log.Fatal("Solution name must be specified either as a positional argument or with a flag but not both")
		}

		return name
	}

	// return the solution name from flag, if provided
	if nameFromFlag != "" {
		return nameFromFlag
	}

	// fail
	log.Fatal("A non-empty <solution-name> argument is required.")
	return "" // unreachable
}

// getSolutionObjectUrl returns the tenant-relative URL path to the solution object
// for a given solution (solution ID). If the solutionId is empty, then the root
// is returned (which can then be used to query the table)
func getSolutionObjectUrl(solutionId string) string {
	// nb: JoinPath doesn't add '/' for empty elements; PathEscape doesn't change the empty string
	url, err := url.JoinPath("knowledge-store/v1/objects/extensibility:solution", url.PathEscape(solutionId))
	if err != nil {
		log.Fatalf("(bug) unexpected failure to construct path for solution ID %q", solutionId)
	}
	return url
}

// getHeaders returns the tenant-level headers required for accessing solution objects
func getHeaders() map[string]string {
	cfg := config.GetCurrentContext()

	return map[string]string{
		"layer-type": "TENANT",
		"layer-id":   cfg.Tenant,
	}
}
