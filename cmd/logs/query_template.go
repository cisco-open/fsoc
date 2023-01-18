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
	"strings"
	"text/template"
)

// template of the query sent to UQL to fetch logs
const queryTemplate = `fetch events(logs:generic_record)
{{if .HasPredicates}}[{{and .Predicates}}]{{end}}
{ timestamp, raw, attributes(severity), entityId, spanId, traceId }
{{if .FromClause}} from {{.FromClause}} {{end}}
order events.desc()
limits events.count({{.Count}})
since -1d
until now()`

func init() {
	uqlQueryTemplate = template.Must(template.New("fetch-logs").Funcs(template.FuncMap{
		"and": func(values []string) string {
			return strings.Join(values, " && ")
		},
	}).Parse(queryTemplate))
}

var uqlQueryTemplate *template.Template

// templateVariables used to fill the queryTemplate
type templateVariables struct {
	Count      int
	FromClause string
	RawFilter  []string
	Severities []string
}

func (tv *templateVariables) HasPredicates() bool {
	return len(tv.RawFilter) > 0 || len(tv.Severities) > 0
}

func (tv *templateVariables) Predicates() []string {
	var predicates []string
	for _, rawFilter := range tv.RawFilter {
		predicates = append(predicates, fmt.Sprintf("raw ~ '%s'", rawFilter))
	}
	if len(tv.Severities) > 0 {
		predicates = append(predicates, fmt.Sprintf("attributes(severity) in [%s]", strings.Join(tv.Severities, ", ")))
	}
	return predicates
}
