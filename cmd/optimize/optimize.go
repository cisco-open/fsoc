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

package optimize

import (
	"github.com/spf13/cobra"
)

// optimizeCmd represents the optimize command
var optimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Perform optimize interactions",
	Long: `Interact with optimize components. Currently only workload profile reports are available
via the report subcommand.`,
	Example:          `  fsoc optimize report "frontend"`,
	TraverseChildren: true,
}

func NewSubCmd() *cobra.Command {
	optimizeCmd.AddCommand(NewCmdConfigure())
	optimizeCmd.AddCommand(NewCmdReport())

	return optimizeCmd
}
