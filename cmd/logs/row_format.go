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
	"bytes"
	"fmt"
	"text/template"
	"time"
)

type row struct {
	Message   any
	Severity  any
	Timestamp any
	EntityId  any
	SpanId    any
	TraceId   any
}

var (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	purple = "\033[35m"
	cyan   = "\033[36m"
	gray   = "\033[37m"
	white  = "\033[97m"
)

type rowFormatter func(*row) (string, error)

func printRow(rowValues []any, formatter rowFormatter) {
	row := &row{
		Timestamp: (rowValues[0]).(time.Time).Format(time.RFC3339),
		Message:   rowValues[1],
		Severity:  rowValues[2],
		EntityId:  rowValues[3],
		SpanId:    rowValues[4],
		TraceId:   rowValues[5],
	}
	formattedRow, err := formatter(row)
	if err != nil {
		fmt.Printf("cannot format: %s\n", row)
	}
	fmt.Println(formattedRow)
}

func createRowFormatter(rowFormat string) (rowFormatter, error) {
	messageTemplate, err := template.New("row-format").Funcs(template.FuncMap{
		"red":    color(red),
		"green":  color(green),
		"yellow": color(yellow),
		"blue":   color(blue),
		"purple": color(purple),
		"cyan":   color(cyan),
		"gray":   color(gray),
		"grey":   color(gray),
		"white":  color(white),
	}).Parse(rowFormat)
	if err != nil {
		return nil, err
	}

	return func(r *row) (string, error) {
		return renderTemplate(messageTemplate, r)
	}, nil
}

func color(color string) func(value any) string {
	return func(value any) string {
		return color + fmt.Sprintf("%s", value) + reset
	}
}

func renderTemplate(template *template.Template, vars any) (string, error) {
	var rendered bytes.Buffer
	err := template.Execute(&rendered, vars)
	if err != nil {
		return "", err
	}
	return rendered.String(), nil
}
