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

type ErrorItem struct {
	Error  string `json:"error"`
	Source string `json:"source"`
}

type Errors struct {
	Items []ErrorItem `json:"items"`
	Total int         `json:"total"`
}

type Result struct {
	Errors Errors `json:"errors"`
	Valid  bool   `json:"valid"`
}

var solutionValidateCmd = &cobra.Command{
	Use:   "validate",
	Args:  cobra.ExactArgs(0),
	Short: "Validate solution",
	Long:  `This command allows the current tenant specified in the profile to upload the solution in the current directory just to validate its contents.  The --stable flag provides a default value of 'stable' for the tag associated with the given solution bundle.  `,
	Example: `  fsoc solution validate
  fsoc solution validate --bump --tag preprod
  fsoc solution validate --tag dev
  fsoc solution validate --stable
  fsoc solution validate -d mysolution --tag dev
  fsoc solution validate --solution-bundle=mysolution-1.22.3.zip --tag stable`,
	Run:              validateSolution,
	TraverseChildren: true,
}

func getSolutionValidateCmd() *cobra.Command {
	solutionValidateCmd.Flags().
		String("tag", "", "Tag to associate with provided solution.  Ensure tag used for validation & upload are same.")

	solutionValidateCmd.Flags().
		Bool("stable", false, "Automatically associate the 'stable' tag with solution bundle to be validate.  This should only be used for validating solutions uploaded with the 'stable' tag.")

	solutionValidateCmd.Flags().
		BoolP("bump", "b", false, "Increment the patch version before validation")

	solutionValidateCmd.Flags().
		StringP("directory", "d", "", "Path to the solution root directory (defaults to current dir)")

	solutionValidateCmd.Flags().
		String("solution-bundle", "", "Path to a prepackaged solution zip bundle")

	solutionValidateCmd.Flags().
		String("env-file", "", "Path to the env vars json file with isolation tag and, optionally, dependency tags")

	solutionValidateCmd.Flags().
		Bool("no-isolate", false, "Disable fsoc-supported solution isolation")

	solutionValidateCmd.MarkFlagsMutuallyExclusive("solution-bundle", "directory")
	solutionValidateCmd.MarkFlagsMutuallyExclusive("solution-bundle", "bump")

	solutionValidateCmd.MarkFlagsMutuallyExclusive("tag", "stable", "env-file")

	return solutionValidateCmd
}

func validateSolution(cmd *cobra.Command, args []string) {
	uploadSolution(cmd, false)
}
