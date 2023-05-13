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

var solutionPushCmd = &cobra.Command{
	Use:   "push",
	Args:  cobra.ExactArgs(0),
	Short: "Deploy your solution",
	Long: `This command allows the current tenant specified in the profile to deploy a solution to the FSO Platform.
The solution manifest for the solution must be in the current directory.

Important details on solution tags:
(1) A tag must be associated with the solution being uploaded.  All subsequent solution upload requests should use this same tag
(2) Use caution when supplying the tag value to the solution to upload as typos can result in misleading validation results
(3) 'stable' is a reserved tag value keyword for production-ready versions and hence should be used appropriately
(4) For more info on tags, please visit: https://developer.cisco.com/docs/fso/#!tag-a-solution
`,
	Example: `
  fsoc solution push --tag=stable
  fsoc solution push --wait --tag=dev
  fsoc solution push --bump --wait=60
  fsoc solution push -d mysolution --stable --wait
  fsoc solution push --solution-bundle=mysolution-1.22.3.zip --tag=stable`,
	Run:              pushSolution,
	TraverseChildren: true,
}

func getSolutionPushCmd() *cobra.Command {
	solutionPushCmd.Flags().
		String("tag", "", "Free-form string tag to associate with provided solution")

	solutionPushCmd.Flags().
		Bool("stable", false, "Mark the solution as production-ready.  This is equivalent to supplying --tag=stable")

	solutionPushCmd.Flags().IntP("wait", "w", -1, "Wait (in seconds) for the solution to be deployed")
	solutionPushCmd.Flag("wait").NoOptDefVal = "300"

	solutionPushCmd.Flags().
		BoolP("bump", "b", false, "Increment the patch version before deploying")

	solutionPushCmd.Flags().
		StringP("directory", "d", "", "Path to the solution root directory (defaults to current dir)")

	solutionPushCmd.Flags().
		String("solution-bundle", "", "Path to a prepackaged solution zip bundle")

	solutionPushCmd.Flags().
		String("env-file", "", "Path to the env vars json file with isolation tag and, optionally, dependency tags")

	solutionPushCmd.Flags().
		Bool("no-isolate", false, "Disable fsoc-supported solution isolation")

	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "directory") // either solution dir or prepackaged zip
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "bump")      // cannot modify prepackaged zip
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "wait")      // TODO: allow when extracting manifest data
	solutionPushCmd.MarkFlagsMutuallyExclusive("tag", "stable", "env-file")    // stable is an alias for --tag=stable

	return solutionPushCmd
}

func pushSolution(cmd *cobra.Command, args []string) {
	uploadSolution(cmd, true)
}
