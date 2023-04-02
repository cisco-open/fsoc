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

// Package output provides display and formatting capabilities
// to show the result of command's execution (command output)
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/itchyny/gojq"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	// TableFieldsAnnotation is the name of the cobra.Command annotation to use to specify the fields JQ query for table output
	TableFieldsAnnotation = "output/tableFields"

	// DetailFieldsAnnotation is the name of the cobra.Command annotation to use to specify the fields JQ query for detail output
	DetailFieldsAnnotation = "output/detailFields"

	JsonIndent = "    "
)

type printRequest struct {
	cmd         *cobra.Command
	format      string
	fields      string
	annotations map[string]string
}

func print(cmd *cobra.Command, a ...any) {
	if cmd != nil {
		cmd.Print(a...)
	} else {
		fmt.Print(a...)
	}
}

func println(cmd *cobra.Command, a ...any) {
	if cmd != nil {
		cmd.Println(a...)
	} else {
		fmt.Println(a...)
	}
}

func printf(cmd *cobra.Command, format string, a ...any) {
	if cmd != nil {
		cmd.Printf(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

func GetOutWriter(cmd *cobra.Command) io.Writer {
	if cmd != nil {
		return cmd.OutOrStdout()
	} else {
		return os.Stdout
	}
}

func WriteJson(obj interface{}, w io.Writer) error {
	data, err := json.MarshalIndent(obj, "", JsonIndent)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// PrintJson displays the output in prettified JSON
func PrintJson(cmd *cobra.Command, v any) error {
	return WriteJson(v, GetOutWriter(cmd))
}

// PrintYaml displays the output in YAML
func PrintYaml(cmd *cobra.Command, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}

	print(cmd, string(data))
	return nil
}

// PrintCmdStatus displays a single string message to the command output
// Use this only for commands that don't display parseable data (e.g., "config set"),
// for example, to confirm that the operation was completed
func PrintCmdStatus(cmd *cobra.Command, s string) {
	print(cmd, s)
}

type Table struct {
	// table output
	Headers []string
	Lines   [][]string
	Detail  bool // true to print a single line as a name: value multi-line instead of table

	// extract field columns in the same order as headers
	LineBuilder func(v any) []string // use together with Headers and no Lines
}

// PrintCmdOutput displays the output of a command in the user-selected output format. If
// a human display format is selected, PrintCmdOutput automatically converts the value
// to one of the supported formats (within limits); if it cannot be converted, YAML is displayed instead.
// If cmd is not provided or it has no `output` flag, human is assumed
// If human format is requested/assumed but no table is provided, displays YAML
// If the object cannot be converted to the desired format, shows the object in Go's %+v format
func PrintCmdOutput(cmd *cobra.Command, v any) {
	PrintCmdOutputCustom(cmd, v, nil)
}

// PrintCmdOutputCustom displays the output of a command in the user-selected output format
// with the capability to provide custom output for the human display formats.
// If cmd is not provided or it has no `output` flag, human is assumed
// If human format is requested/assumed but no table is provided, displays YAML
// If the object cannot be converted to the desired format, shows the object in Go's %+v format
func PrintCmdOutputCustom(cmd *cobra.Command, v any, table *Table) {
	// extract format, assume default if no command or no -o flag
	format := ""
	if cmd != nil {
		format, _ = cmd.Flags().GetString("output") // if err, leaves format blank
	}

	// select which fields filter specification to use
	// Logic: if the --fields flag is specified, always use it (for all formats)
	//        otherwise
	//        - for human outputs only, get the fields spec from the command annotations (if set)
	//        - for machine formats, don't filter by fields
	fields, _ := cmd.Flags().GetString("fields") // since --fields doesn't have default, non-empty means explicitly set
	pr := printRequest{cmd: cmd, format: format, fields: fields, annotations: cmd.Annotations}
	printCmdOutputCustom(pr, v, table)
}

func printCmdOutputCustom(pr printRequest, v any, table *Table) {
	// if no field spec is given on the command line and built-in specs are available, use them
	if pr.fields == "" && pr.annotations != nil {
		// choose which annotations to use and in what priority order
		annotations := []string{} // names of annotations to use for fields, in priority order
		switch pr.format {
		case "", "auto", "table":
			annotations = []string{TableFieldsAnnotation, DetailFieldsAnnotation}
		case "detail":
			annotations = []string{DetailFieldsAnnotation, TableFieldsAnnotation}
			// all others, keep empty list
		}

		// get the first available fields specification
		for _, name := range annotations {
			if spec := pr.annotations[name]; spec != "" {
				pr.fields = spec
				break
			}
		}
	}

	// adjust format to yaml if not enough info to produce human output (nb: the criteria may change
	// in the future as the auto format capabilities improve)
	if (pr.format == "" || pr.format == "auto") && // format is not explicitly specified
		pr.fields == "" && // no field specification is provided (on the command line or from the command descriptor)
		(table == nil || table.Headers == nil || len(table.Headers) == 0) { // no explicit table form is provided
		// go for YAML output, which is mostly human readable (or, at least, more human-readable than json or go %+v)
		pr.format = "yaml"
	}

	// transform data according to the fields query (if provided and should be used)
	if pr.fields != "" {
		v = transformFields(v, pr.fields)
	}

	// print according to format and presence of table
	switch pr.format {
	case "json":
		if err := PrintJson(pr.cmd, v); err != nil {
			log.Fatalf("Failed to convert output to JSON: %v (%+v)", err, v)
		}
		return
	case "yaml":
		if err := PrintYaml(pr.cmd, v); err != nil {
			log.Fatalf("Failed to convert output to YAML: %v (%+v)", err, v)
		}
		return
	}

	// display simple values
	if strVal, ok := v.(string); ok {
		printSimple(pr.cmd, strVal)
		return
	}

	// prepare lines if builder provided
	if table != nil && table.LineBuilder != nil {
		lines, ok := buildLines(v, table.LineBuilder)
		if ok {
			table = &Table{Headers: table.Headers, Lines: lines, Detail: table.Detail}
		}
	}

	// format table if a transform is provided or there is no custom table
	if pr.fields != "" || table == nil || len(table.Headers) == 0 {
		var err error
		table, err = createTable(v, pr.fields) // replaces the table
		if err != nil {
			log.Warnf("Failed to convert output data to a table: %v; reverting to YAML output", err)
			if err := PrintYaml(pr.cmd, v); err != nil {
				log.Fatalf("Failed to convert output to YAML: %v (%+v)", err, v)
			}
			return
		}
	}

	// display table
	if table.Detail || pr.format == "detail" {
		printDetail(pr.cmd, table)
	} else {
		printTable(pr.cmd, table)
	}
}

func buildLines(in any, builderFunc func(any) []string) ([][]string, bool) {
	// convert to list of a single entry if it's not
	lst, ok := in.([]any)
	if !ok {
		lst = []any{in}
	}

	// build each line
	lines := [][]string{}
	for _, entry := range lst {
		line := builderFunc(entry)
		if line == nil {
			return nil, false // error parsing, give up on human output
		}
		lines = append(lines, line)
	}

	return lines, true
}

// printSimple prints a simple value as a command output.
// It should not be provided with complex values (slices, maps, structs, etc.), but
// it will do its best to print those by letting Go's fmt handle those (but they will not be pretty)
func printSimple(cmd *cobra.Command, v any) {
	println(cmd, v)
}

// printTable prints a table, with header and one or more rows
func printTable(cmd *cobra.Command, t *Table) {
	if t == nil {
		printSimple(cmd, "Nothing to display")
		return
	}
	tw := tablewriter.NewWriter(GetOutWriter(cmd))
	tw.SetBorder(false)
	tw.SetCenterSeparator("")
	tw.SetColumnSeparator("")
	tw.SetRowSeparator("")
	tw.SetHeader(t.Headers)
	tw.AppendBulk(t.Lines)
	tw.Render()
}

// printDetail prints a form-like detail output, with "label: value" pairs on each row
// While printDetail is mostly intended for a single-entry output (one map or struct, not a list)
// if there are multiple entries in t.Lines, it prints each entry as a separate form,
// separating each entry with a blank line
func printDetail(cmd *cobra.Command, t *Table) {
	if t == nil {
		printSimple(cmd, "Nothing to display")
		return
	}

	// determine header max. width
	labelWidth := 0
	for _, v := range t.Headers {
		if l := len(v); l > labelWidth {
			labelWidth = l
		}
	}

	// display first row as entries
	for _, entry := range t.Lines {
		for i := range t.Headers {
			printf(cmd, "%[1]*[2]s: %[3]v\n", labelWidth, t.Headers[i], entry[i])
			//TODO: add support for multi-line values, see Jira ticket FSOC-23
		}
		println(cmd)
	}
}

// createTable automatically creates a table from the structure of the data.
// For now, it relies on a fields specification being provided in the form of a JQ query.
// The fields specification selects what fields should be output, with what names and in what order
func createTable(v any, fields string) (*Table, error) {
	// init empty custom table
	table := Table{Headers: []string{}, Lines: [][]string{}}

	// build order index to undo JQ's alphabetizing
	orderIndex := makeFieldOrderIndex(fields)

	// use JQ to create a list with a header row and data rows where all fields are converted to string
	jqExpression := "(.items[0]|keys),.items[]|to_entries|map(.value|tostring)"
	query, err := gojq.Parse(jqExpression)
	if err != nil {
		log.Fatalf("Failed to parse jq expression: %q: %v; likely a bug; please use --output yaml or json for now", jqExpression, err)
	}
	iter := query.Run(v)
	for index := 0; true; index++ {
		// get next row, break if finished
		row, ok := iter.Next()
		if !ok {
			break
		}

		// skip row and log error if the row failed to convert
		if err, ok := row.(error); ok {
			log.Errorf("error at data row %v : %v; likely a bug; use --output json or yaml for now", index-1, err) // index-1 to compensate for header row
			continue
		}

		// add row to the table
		if index == 0 { // header row
			table.Headers = toArray(row, orderIndex)
		} else {
			table.Lines = append(table.Lines, toArray(row, orderIndex))
		}
	}
	return &table, nil
}

func toArray(a any, order []int) []string {
	_a := a.([]interface{})
	s := make([]string, len(_a))
	for i, e := range _a {
		if order != nil {
			s[order[i]] = fmt.Sprint(e)
		} else {
			s[i] = fmt.Sprint(e)
		}
	}
	return s
}

func makeFieldOrderIndex(fieldsCommaList string) []int {
	if fieldsCommaList == "" {
		return nil
	}

	var order []int
	if strings.TrimSpace(fieldsCommaList) != "*" {
		// extract list of field names, in the desired order
		fields := strings.Split(fieldsCommaList, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		// create an alphabetized version of the list
		alphabetizedFields := make([]string, len(fields))
		copy(alphabetizedFields, fields)
		sort.Strings(alphabetizedFields) //by default jq will alphabetize strings

		// create order index, defining what's the desired position for each field (from alphabetized order)
		order = make([]int, len(fields))
		for i, s := range alphabetizedFields {
			for i2, s2 := range fields {
				if s == s2 {
					order[i] = i2 // now we have recorded where we have to move the ith alphebtized field to
					break
				}
			}
		}
	}
	return order
}

// transformFields transforms the data structure of the entries according to
// a JQ specification.
// Here we use jq to filter the json ".items" array so it only contains the fields we are interested in
// then we reconstitute the original {items, total} object with items now having their fields filtered
// We only need to use this iterator once since it's a single object to single object
// jq expression and we just overwrite the v that came in
func transformFields(v any, fieldsCommaList string) any {
	// canonicalize format (we use JQ, so it must be map[string]interface{})
	v = canonicalizeData(v)

	//default value of fields is "*". We don't mess with anything if the fields
	//requested are "*""
	if strings.TrimSpace(fieldsCommaList) != "*" {
		qStr := fmt.Sprintf(". as $root|.items|{items: map({%s}),total:$root.total}", fieldsCommaList)
		query, err := gojq.Parse(qStr)
		if err != nil {
			log.Fatalf("Failed to parse field list %q as a jq expression %q: %v", fieldsCommaList, qStr, err)
		}
		iter := query.Run(v)

		v, _ = iter.Next()
	}
	return v
}

// canonicalizeData ensures that the data is in a uniform, expected format, converting any possible input
// into the expected .items[] and .total structure, rendered as a map[string]any, as JSON parse would
// produce it given no specific schema/structure to parse into
func canonicalizeData(v any) any {
	// remarshal via JSON if not a map
	var data map[string]any
	data, ok := v.(map[string]any)
	if !ok {
		// convert to JSON, so we can convert it back
		tmp, err := json.Marshal(v)
		if err != nil {
			log.Warnf("failed to convert output data to JSON in preparation for transformations: %v; leaving it as is", err)
			return v
		}

		// convert into a generic map
		if err := json.Unmarshal(tmp, &data); err != nil {
			log.Warnf("failed to convert output data back from JSON in preparation for transformations: %v; leaving it as is", err)
			return v
		}
	}

	// if the map already has the standard FSO API structure, use it
	if _, ok := data["items"]; ok {
		return data
	}

	// force result to be a map with a single item in items array
	out := make(map[string]any)
	out["items"] = []any{data}
	out["total"] = 1

	return out
}
