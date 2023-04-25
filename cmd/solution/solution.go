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
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var solutionCmd = &cobra.Command{
	Use:   "solution",
	Short: "Perform solution lifecycle operations",
	Long: `Perform solution lifecycle operations with the FSO Platform.

For more information on FSO solutions, see https://developer.cisco.com/docs/fso/#!create-a-solution-introduction`,
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
	solutionCmd.AddCommand(getSubscribeListCmd())
	solutionCmd.AddCommand(getInitSolutionCmd())
	solutionCmd.AddCommand(getSubscribeSolutionCmd())
	solutionCmd.AddCommand(getUnsubscribeSolutionCmd())
	solutionCmd.AddCommand(getSolutionExtendCmd())
	solutionCmd.AddCommand(getSolutionPackageCmd())
	solutionCmd.AddCommand(getSolutionPushCmd())
	solutionCmd.AddCommand(getAuthorCmd())
	solutionCmd.AddCommand(getSolutionDownloadCmd())
	solutionCmd.AddCommand(getSolutionValidateCmd())
	solutionCmd.AddCommand(GetSolutionForkCommand())
	solutionCmd.AddCommand(getSolutionCheckCmd())
	solutionCmd.AddCommand(getSolutionStatusCmd())
	solutionCmd.AddCommand(getSolutionDescribeCmd())
	solutionCmd.AddCommand(getSolutionBumpCmd())
	solutionListCmd.Flags().StringP("output", "o", "", "Output format (human*, json, yaml)")

	return solutionCmd
}
