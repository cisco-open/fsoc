// Copyright 2023 Cisco Systems, Inc.
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
	"strconv"
	"strings"

	"github.com/cisco-open/fsoc/platform/api"
)

// uqlProblem represents parsed data from an object returned with content-type application/problem+json according to the RFC-7807
// These parsed data are specific for the UQL service.
type uqlProblem struct {
	query        string
	title        string
	detail       string
	errorDetails []errorDetail
}

func (p uqlProblem) Error() string {
	return fmt.Sprintf("%s: %s", p.title, p.detail)
}

// errorDetail contains detailed information about user error in the query
type errorDetail struct {
	message          string
	fixSuggestion    string
	fixPossibilities []string
	errorType        string
	errorFrom        position
	errorTo          position
}

// position is a place in a multi-line string. Numbering of lines starts with 1
// Position before the first character has column value 0
type position struct {
	line   int
	column int
}

func makeUqlProblem(original api.Problem) uqlProblem {
	problem := uqlProblem{
		query:        asStringOrNothing(original.Extensions["query"]),
		title:        original.Title,
		detail:       original.Detail,
		errorDetails: make([]errorDetail, 0),
	}
	switch array := original.Extensions["errorDetails"].(type) {
	case []any:
		for _, values := range array {
			switch asMap := values.(type) {
			case map[string]any:
				problem.errorDetails = append(problem.errorDetails, makeErrorDetail(asMap))
			}
		}
	}
	return problem
}

func makeErrorDetail(values map[string]any) errorDetail {
	detail := errorDetail{
		message:       asStringOrNothing(values["message"]),
		fixSuggestion: asStringOrNothing(values["fixSuggestion"]),
		errorType:     asStringOrNothing(values["errorType"]),
		errorFrom:     asPositionOrNothing(asStringOrNothing(values["errorFrom"])),
		errorTo:       asPositionOrNothing(asStringOrNothing(values["errorTo"])),
	}
	switch typed := values["fixPossibilities"].(type) {
	case []any:
		for _, val := range typed {
			detail.fixPossibilities = append(detail.fixPossibilities, asStringOrNothing(val))
		}
	}
	return detail
}

func asStringOrNothing(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func asPositionOrNothing(value string) position {
	if strings.Count(value, ":") != 1 {
		return position{}
	}
	split := strings.Split(value, ":")
	line, err := strconv.Atoi(split[0])
	if err != nil {
		return position{}
	}
	col, err := strconv.Atoi(split[1])
	if err != nil {
		return position{}
	}
	return position{
		line:   line,
		column: col,
	}
}
