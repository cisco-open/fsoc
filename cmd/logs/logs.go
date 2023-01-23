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
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

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
	// followSince controls from how far in the past to start following logs
	followSince = "-2m"
	// followCount controls the maximum number of log records to retrieve per request
	followCount = 50
	// followTimer controls the maximum delay between requests
	followTimer = time.Second * 2
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

	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return err
	}

	if follow {
		variables.Count = followCount
		variables.Since = followSince
		variables.Order = "asc"
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

	resp, err := queryLogs(query)
	if err != nil {
		log.Fatal(err.Error())
	}

	printLogs(resp, formatter, cmd)

	if follow {
		return followLogs(resp, formatter, variables.Count, cmd)
	}

	return nil
}

type eventResult struct {
	data *uql.DataSet
	err  error
}

func followLogs(initialResponse *uql.Response, formatter rowFormatter, limit int, p printer) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	eventResults := make(chan eventResult, 1)
	eventResults <- eventResult{data: extractEventDataSet(initialResponse)}

	for {
		select {
		case <-interrupt:
			return nil
		case followResult := <-eventResults:
			if followResult.err != nil {
				log.Fatal(followResult.err.Error())
			}

			go func() {
				resp, err := followDataSet(followResult.data)
				if err != nil {
					eventResults <- eventResult{err: err}
					return
				}

				printLogs(resp, formatter, p)

				eventsDataSet := extractEventDataSet(resp)

				if len(eventsDataSet.Data) == limit {
					// send another request immediately
					eventResults <- eventResult{data: eventsDataSet}
				} else {
					// wait a while since there probably is not enough data
					go func() {
						time.Sleep(followTimer)
						eventResults <- eventResult{data: eventsDataSet}
					}()
				}
			}()
		}
	}
}

func queryLogs(query string) (*uql.Response, error) {
	resp, err := uql.ExecuteQuery(&uql.Query{Str: query}, uql.ApiVersion1)
	if err != nil {
		return nil, err
	}

	if resp.HasErrors() {
		return nil, uql.Errors(resp.Errors())
	}

	return resp, nil
}

func followDataSet(dataSet *uql.DataSet) (*uql.Response, error) {
	resp, err := uql.ContinueQuery(dataSet, "follow")
	if err != nil {
		return nil, err
	}

	if resp.HasErrors() {
		return nil, uql.Errors(resp.Errors())
	}

	return resp, nil
}

func printLogs(resp *uql.Response, formatter rowFormatter, p printer) {
	rawLogRecords := extractEventDataSet(resp).Values()

	for i := len(rawLogRecords) - 1; i >= 0; i-- {
		printRow(rawLogRecords[i], formatter, p)
	}
}

func extractEventDataSet(resp *uql.Response) *uql.DataSet {
	return resp.Main().Values()[0][0].(*uql.DataSet)
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
		Since:      "-4h",
		Order:      "desc",
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
