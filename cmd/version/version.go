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

package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print fsoc version",
	Long:  `Print fsoc version`,
	Run: func(cmd *cobra.Command, args []string) {
		displayVersion(cmd)
	},
	Annotations: map[string]string{config.AnnotationForConfigBypass: ""},
}

func init() {
	versionCmd.PersistentFlags().StringP("output", "o", "human", "Output format (human*, json, yaml)")
	versionCmd.PersistentFlags().BoolP("detail", "d", false, "Show full version detail (incl. git info)")
}

func NewSubCmd() *cobra.Command {
	return versionCmd
}

func displayVersion(cmd *cobra.Command) {
	// determine whether we need short output
	outfmt, _ := cmd.Flags().GetString("output")
	detail, _ := cmd.Flags().GetBool("detail")
	if !detail && (outfmt == "" || outfmt == "human") {
		output.PrintCmdStatus(cmd, fmt.Sprintf("fsoc version %v\n", GetVersionShort()))
		return
	}

	// prepare human output (in case needed)
	titles := []string{}
	values := []string{}
	for _, fieldTuple := range GetVersionDetailsHuman() {
		titles = append(titles, fieldTuple[0])
		values = append(values, fieldTuple[1])
	}
	filter := output.CreateFilter("", []int{})

	output.PrintCmdOutputCustom(cmd, version, &output.Table{
		Headers: titles,
		Lines:   [][]string{values},
		Detail:  true,
	}, filter)
}
