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
	Long: `This command allows the current tenant specified in the profile to deploy a solution to the platform.
The solution manifest for the solution must be in the current directory.

Important details on solution tags:
  1. A tag must be associated with the solution being uploaded.  All subsequent solution upload requests should use this same tag
  2. Use caution when supplying the tag value to the solution to upload as typos can result in misleading validation results
  3. "stable" is a reserved tag value keyword for production-ready versions and hence should be used appropriately
  4. For more info on tags, please visit: https://developer.cisco.com/docs/cisco-observability-platform/#!tag-a-solution

A tag may be defined in the following ways (in order of precedence):
  1. Specified flag --tag=xyz or --stable: use this tag, ignoring .tag file or env vars
  2. A tag is defined in the FSOC_SOLUTION_TAG environment variable (ignores .tag file)
  3. A tag is defined in the .tag file in the solution directory (usually not version controlled)
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
	addTagFlags(solutionPushCmd) // --tag and --stable

	solutionPushCmd.Flags().IntP("wait", "w", -1, "Wait (in seconds) for the solution to be deployed")
	solutionPushCmd.Flag("wait").NoOptDefVal = "300"

	solutionPushCmd.Flags().
		BoolP("bump", "b", false, "Increment the patch version before deploying solution")

	solutionPushCmd.Flags().
		StringP("directory", "d", "", "Path to the solution root directory (defaults to current dir)")

	solutionPushCmd.Flags().
		String("solution-bundle", "", "Path to a prepackaged solution zip")

	solutionPushCmd.Flags().
		String("env-file", "", "Path to the env vars json file with pseudo-isolation tag and, optionally, dependency tags (DEPRECATED)")

	solutionPushCmd.Flags().
		Bool("no-isolate", false, "Disable fsoc-supported solution pseudo-isolation")

	solutionPushCmd.Flags().
		Bool("subscribe", false, "Subscribe to the solution that you are pushing")

	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "directory") // either solution dir or prepackaged zip
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "bump")      // cannot modify prepackaged zip
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "wait")      // TODO: allow when extracting manifest data
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "subscribe") // TODO: allow when extracting manifest data
	solutionPushCmd.MarkFlagsMutuallyExclusive("tag", "stable", "env-file")

	return solutionPushCmd
}

func pushSolution(cmd *cobra.Command, args []string) {
	uploadSolution(cmd, true)
}
