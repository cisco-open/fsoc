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

package uql

import (
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	fsoc "github.com/cisco-open/fsoc/output"
)

var outputFlag string
var rawFlag bool

const (
	availableFormats string = "auto, table, json, yaml"
)

// uqlCmd represents the uql command
var uqlCmd = &cobra.Command{
	Use:   "uql",
	Short: "Perform UQL query",
	Long: `Perform UQL query of MELT data for a tenant. 

See https://developer.cisco.com/docs/fso/#!data-query-using-unified-query-language
for more information on the UQL query language for FSO.

Parsed response data are displayed in a table by default.
Available output formats: ` + availableFormats + `.
If the "raw" flag is provided, the actual response from the backend API is displayed instead.`,
	Example: `# Get parsed results
  fsoc uql "FETCH id, type, attributes FROM entities(k8s:workload)"`,
	Args:             cobra.ExactArgs(1),
	RunE:             uqlQuery,
	TraverseChildren: true,
}

type format int

const (
	tableFormat format = iota
	autoFormat
	rawFormat
	jsonFormat
	yamlFormat
)

func init() {
	uqlCmd.Flags().StringVarP(&outputFlag, "output", "o", "table", "overridden")
	uqlCmd.Flags().BoolVar(&rawFlag, "raw", false, "Display actual response from the backend. Cannot be used together with the output flag.")
	uqlCmd.MarkFlagsMutuallyExclusive("output", "raw")
	uqlCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		changeFlagUsage(cmd.Parent())
		cmd.Parent().HelpFunc()(cmd, args)
	})
	uqlCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		changeFlagUsage(cmd.Parent())
		return cmd.Parent().UsageFunc()(cmd)
	})
}

func NewSubCmd() *cobra.Command {
	return uqlCmd
}

func uqlQuery(cmd *cobra.Command, args []string) error {
	log.WithFields(log.Fields{"command": cmd.Name(), "args": args[0]}).Info("Performing UQL query")

	output, err := outputFormat(outputFlag, rawFlag)
	if err != nil {
		return err
	}
	queryStr := args[0]
	response, err := runQuery(queryStr)
	if err != nil {
		if problem, ok := err.(uqlProblem); ok {
			printProblemDescription(cmd, problem, queryStr)
			os.Exit(1)
		} else {
			log.Fatal(err.Error())
		}
	}
	if response.HasErrors() {
		log.Error("Execution of query encountered errors. Returned data are not complete!")
		for _, e := range response.Errors() {
			log.Errorf("%s: %s", e.Title, e.Detail)
		}
	}
	err = printResponse(cmd, response, output)
	if err != nil {
		return err
	}
	return nil
}

func outputFormat(output string, useRaw bool) (format, error) {
	if useRaw {
		return rawFormat, nil
	}
	switch strings.ToLower(output) {
	case "auto":
		return autoFormat, nil
	case "table":
		return tableFormat, nil
	case "json":
		return jsonFormat, nil
	case "yaml":
		return yamlFormat, nil

	default:
		return -1, fmt.Errorf(
			"unsupported output format %s for sub-command uql. This sub-command supports only following formats: [%s]",
			output,
			availableFormats,
		)
	}
}

func runQuery(query string) (*Response, error) {
	log.Info("fetch data")

	resp, err := ExecuteQuery(&Query{Str: query}, ApiVersion1)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func printResponse(cmd *cobra.Command, response *Response, output format) error {
	switch output {
	case tableFormat, autoFormat:
		t := makeFlatTable(response)
		cmd.Println(t.Render())
	case jsonFormat:
		json, err := transformForJsonOutput(response)
		if err != nil {
			return err
		}
		return fsoc.PrintJson(cmd, json)
	case yamlFormat:
		json, err := transformForJsonOutput(response)
		if err != nil {
			return err
		}
		return fsoc.PrintYaml(cmd, json)
	case rawFormat:
		filter := fsoc.CreateFilter("", []int{})
		fsoc.PrintCmdOutput(cmd, string(*response.raw), filter)
	}
	return nil
}

func changeFlagUsage(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "output" {
			flag.Usage = fmt.Sprintf("output format (%s)", availableFormats)
		}
	})
}

func printProblemDescription(cmd *cobra.Command, problem uqlProblem, inputQuery string) {
	cmd.Printf("%s\n%s\n\n", problem.title, problem.detail)
	if len(problem.errorDetails) != 0 {
		var query string
		// Sometimes, the query is not reported back in the problem json
		if problem.query == "" {
			query = inputQuery
		} else {
			query = problem.query
		}
		cmd.Printf("Error in the query:\n%s\n\n", highlightError(query, problem.errorDetails[0]))
	}
	if problem.errorDetails != nil && len(problem.errorDetails) > 0 {
		printErrorDetail(cmd, problem.errorDetails[0])
	}
}

func printErrorDetail(cmd *cobra.Command, detail errorDetail) {
	msg := strings.Builder{}
	msg.WriteString(detail.message)
	if detail.fixSuggestion != "" {
		msg.WriteString(". ")
		msg.WriteString(detail.fixSuggestion)
	}
	if len(detail.fixPossibilities) != 0 {
		msg.WriteString("\nPossible fixes: \n(")
		msg.WriteString(strings.Join(detail.fixPossibilities, ", "))
		msg.WriteString(")")
	}
	cmd.Println(msg.String())
}

func highlightError(query string, detail errorDetail) string {
	background := termenv.BackgroundColor()
	foreground := termenv.ForegroundColor()
	switched := lipgloss.NewStyle().
		Foreground(lipgloss.Color(fmt.Sprint(background))). // Unfortunately, there is no better conversion in the API
		Background(lipgloss.Color(fmt.Sprint(foreground))).
		Bold(true)
	lines := strings.Split(query, "\n")
	from := detail.errorFrom
	to := detail.errorTo
	noPosition := position{}
	if from == noPosition && to == noPosition {
		return query
	}
	start := from.column
	for l := from.line - 1; l < to.line-1 && l < len(lines) && l >= 0; l++ {
		line := lines[l]
		lines[l] = fmt.Sprintf("%s%s", line[:min(start, len(line))], switched.Render(line[min(start, len(line)):]))
	}
	lastErrorLine := to.line - 1
	if lastErrorLine < len(lines) && lastErrorLine >= 0 {
		line := lines[lastErrorLine]
		toColumn := to.column
		if detail.errorType == "SEMANTIC" {
			toColumn++ // There is a bug in the reporting of the error position for the semantic errors.
		}
		lines[lastErrorLine] = fmt.Sprintf(
			"%s%s%s",
			line[:min(start, len(line))],
			switched.Render(line[min(start, len(line)):min(toColumn, len(line))]),
			line[min(toColumn, len(line)):],
		)
	}
	return strings.Join(lines, "\n")
}

func min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}
