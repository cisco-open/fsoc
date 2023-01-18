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

package logs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/uql"
)

var (
	followFlag        bool
	messageFilterFlag []string
	formatFlag        string
	messageCountFlag  int
	minSeverityFlag   string
	logsCmd           = &cobra.Command{
		Use:   "logs",
		Short: "Display solution logs",
		Long: `Display solution logs

This command allows you to either fetch a certain number of logs or to follow the logs as they are produced.

You can specify:
- which sources of logs you are interested in
- how many messages you want displayed (--count/-n)
- filters over log messages (--message-filter/-m)
- the minimum severity level (--severity/-l)
- format for the resulting output (--format/-t)

Specifying log sources
- for all sources do not pass in any argument
- for logs from concrete entities pass as argument: 'infra:container:123,infra:container:456'
- for logs from all entities of a certain type pass as argument: 'infra:container'
- power users can use the whole UQL traversal: 'entities(k8s:pod)[attributes(k8s.namespace.name)="default"].out.to(infra:container)'

Specifying message filters
- for all messages do not include the --message-filter/-m flag
- for messages that contain an exact phrase: --message-filter='<exact-phrase>'
- for messages approximate phrases: -m 'Hello*'
- for messages that contain multiple phrases: -m 'hello' -m 'world' or -m 'hello,world'

Formatting output
- to change the output format specify the --format/-t flag
- available fields: Timestamp, Severity, Message, EntityId, SpanId, TraceId
- to color a field in your template write the name of a color before the field e.g. {{yellow Timestamp}}
- available colors: red, green, yellow, blue, purple, cyan, gray, white`,
		Example: `
Fetch the last 50 logs containing the string 'ERROR'
  fsoc logs -n 50 -l "ERROR" 

Fetch logs from all pods in my namespace
  fsoc logs 'entities(k8s:pod)[attributes(k8s.namespace.name)="default"].out.to(infra:container)'

Follow logs from my pods
  fsoc logs -f "k8s:pod:123"

Coloring output
  fsoc logs -t '{{yellow .Timestamp}} {{red .Severity}} {{purple .EntityId}} {{.Message}}'`,
		Args:             cobra.MaximumNArgs(1),
		RunE:             fetchLogs,
		TraverseChildren: true,
	}
)

func NewSubCmd() *cobra.Command {
	cmd := logsCmd
	cmd.Flags().IntVarP(&messageCountFlag, "count", "n", 30, "number of log messages to fetch")
	cmd.Flags().BoolVarP(&followFlag, "follow", "f", false, "follow output")
	cmd.Flags().StringSliceVarP(&messageFilterFlag, "message-filter", "m", nil, "message filter")
	cmd.Flags().StringVarP(&formatFlag, "format", "t", `{{.Message}}`, "format individual rows (Go template), may refer to: Message, Timestamp, Severity, EntityId, SpanId, TraceId")
	cmd.Flags().StringVarP(&minSeverityFlag, "severity", "l", "", "minimum severity level")
	cmd.MarkFlagsMutuallyExclusive("count", "follow")
	return cmd
}

func fetchLogs(cmd *cobra.Command, args []string) error {
	from := "everything"
	if len(args) == 1 {
		from = args[0]
	}
	log.WithFields(log.Fields{"from": from}).Info("fetch logs")

	variables, err := resolveTemplateVariables(cmd, args)
	if err != nil {
		return err
	}

	rowFormat, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}

	query, err := renderTemplate(uqlQueryTemplate, variables)
	if err != nil {
		return err
	}

	formatter, err := createRowFormatter(rowFormat)
	if err != nil {
		return err
	}

	query = prettifyUqlQuery(query)

	resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
	if err != nil {
		return err
	}

	if resp.HasErrors() {
		return uql.Errors(resp.Errors())
	}

	values := resp.DataSet(resp.Main().Values[0][0].(uql.DataSetRef)).Values
	for i := len(values) - 1; i >= 0; i-- {
		printRow(values[i], formatter)
	}

	return nil
}

func resolveTemplateVariables(cmd *cobra.Command, args []string) (*templateVariables, error) {
	countValue, err := cmd.Flags().GetInt("count")
	if err != nil {
		return nil, err
	}

	var fromValue string
	if len(args) == 1 {
		fromValue = args[0]
	}

	messageFilterValue, err := cmd.Flags().GetStringSlice("message-filter")
	if err != nil {
		return nil, err
	}

	severityValue, err := cmd.Flags().GetString("severity")
	if err != nil {
		return nil, err
	}

	if severityValue != "" && !validLevel(severityValue) {
		return nil, fmt.Errorf("unknown error level: %s, must be one of: %s", severityValue, allLevelsNames())
	}

	return &templateVariables{
		Count:      countValue,
		FromClause: resolveFromClause(fromValue),
		RawFilter:  messageFilterValue,
		Severities: findLowerOrEqualLevels(severityValue),
	}, nil
}

func resolveFromClause(value string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "entities(") {
		return value
	}
	return fmt.Sprintf("entities(%s)", value)
}

var redundantWhitespaces = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

func prettifyUqlQuery(query string) string {
	return redundantWhitespaces.ReplaceAllString(strings.ReplaceAll(strings.TrimSpace(query), "\n", " "), " ")
}
