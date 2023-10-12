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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"

	"github.com/cisco-open/fsoc/platform/api"
)

// Query represents a UQL request body
type Query struct {
	Str string `json:"query"`
}

// parsedResponse contains unchanged response JSON and list of parsed data chunks
type parsedResponse struct {
	chunks  []parsedChunk
	rawJson *json.RawMessage
}

// parsedChunk is a union of all fields on all UQL response dataset types
type parsedChunk struct {
	Type     string              `json:"type"`
	Model    json.RawMessage     `json:"model"`
	Metadata map[string]any      `json:"metadata"`
	Dataset  string              `json:"dataset"`
	Data     [][]json.RawMessage `json:"data"`
	Links    map[string]rawLink  `json:"_links"`
	Error    *Error              `json:"error"`
}

type rawLink struct {
	Href string `json:"href"`
}

type modelRef struct {
	JsonPath string `json:"$jsonPath"`
	Model    string `json:"$model"`
}

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

type uqlService interface {
	Execute(query *Query, apiVersion ApiVersion) (parsedResponse, error)
	Continue(link *Link) (parsedResponse, error)
}

type defaultBackend struct {
	apiOptions *api.Options
}

func (b defaultBackend) Execute(query *Query, apiVersion ApiVersion) (parsedResponse, error) {
	log.WithFields(log.Fields{"query": query.Str, "apiVersion": apiVersion}).Info("executing UQL query")

	var rawJson json.RawMessage
	err := api.JSONPost(GetAPIEndpoint(apiVersion), query, &rawJson, b.apiOptions)
	if err != nil {
		if problem, ok := err.(api.Problem); ok {
			return parsedResponse{}, makeUqlProblem(problem)
		}
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed to execute UQL Query: '%s'", query.Str))
	}
	var chunks []parsedChunk
	err = json.Unmarshal(rawJson, &chunks)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed to parse response for UQL Query: '%s'", query.Str))
	}
	return parsedResponse{
		chunks:  chunks,
		rawJson: &rawJson,
	}, nil
}

func (b defaultBackend) Continue(link *Link) (parsedResponse, error) {
	log.WithFields(log.Fields{"query": link.Href}).Info("continuing UQL query")

	var rawJson json.RawMessage
	err := api.JSONGet(link.Href, &rawJson, b.apiOptions)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed follow link: '%s'", link.Href))
	}
	var chunks []parsedChunk
	err = json.Unmarshal(rawJson, &chunks)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed to parse response for link: '%s'", link.Href))
	}
	return parsedResponse{
		chunks:  chunks,
		rawJson: &rawJson,
	}, nil
}

type UqlClient interface {
	// ExecuteQuery sends an execute request to the UQL service
	ExecuteQuery(query *Query, apiVersion ApiVersion) (*Response, error)

	// ContinueQuery sends a continue request to the UQL service
	ContinueQuery(dataSet *DataSet, rel string) (*Response, error)
}

type defaultClient struct {
	backend uqlService
}

func (c defaultClient) ExecuteQuery(query *Query, apiVersion ApiVersion) (*Response, error) {
	return executeUqlQuery(query, apiVersion, c.backend)
}

func (c defaultClient) ContinueQuery(dataSet *DataSet, rel string) (*Response, error) {
	return continueUqlQuery(dataSet, rel, c.backend)
}

func MakeBackendClient(options *api.Options) UqlClient {
	return &defaultClient{
		backend: &defaultBackend{
			apiOptions: options,
		},
	}
}

var Client UqlClient = MakeBackendClient(nil)

func executeUqlQuery(query *Query, apiVersion ApiVersion, backend uqlService) (*Response, error) {
	if query == nil || strings.Trim(query.Str, "") == "" {
		return nil, fmt.Errorf("uql query missing")
	}

	if apiVersion == "" {
		return nil, fmt.Errorf("uql API version missing")
	}

	response, err := backend.Execute(query, apiVersion)
	if err != nil {
		return nil, err
	}

	return processResponse(response)
}

func continueUqlQuery(dataSet *DataSet, rel string, backend uqlService) (*Response, error) {
	link := extractLink(dataSet, rel)

	if link == nil {
		return nil, fmt.Errorf("link with rel '%s' not found in dataset", rel)
	}

	response, err := backend.Continue(link)
	if err != nil {
		return nil, err
	}

	return processResponse(response)
}

func extractLink(dataSet *DataSet, rel string) *Link {
	if dataSet == nil {
		return nil
	}

	if len(dataSet.Links) == 0 {
		return nil
	}

	link, ok := dataSet.Links[rel]
	if !ok {
		return nil
	}

	return &link
}

func processResponse(response parsedResponse) (*Response, error) {
	var model *Model
	var dataSets = make(map[string]*DataSet)
	var errorSets []*Error
	var modelIndex map[string]*Model

	for _, dataset := range response.chunks {
		switch dataset.Type {
		case "model":
			err := json.Unmarshal(dataset.Model, &model)
			if err != nil {
				return nil, err
			}
			modelIndex = createModelIndex(model)
		case "data":
			var modelRef modelRef
			err := json.Unmarshal(dataset.Model, &modelRef)
			if err != nil {
				return nil, err
			}
			var dataSetModel = modelIndex[modelRef.Model]
			values, err := processValues(dataset.Data, dataSetModel)
			if err != nil {
				return nil, err
			}
			dataSets[dataset.Dataset] = &DataSet{
				Name:      dataset.Dataset,
				DataModel: dataSetModel,
				Metadata:  dataset.Metadata,
				Data:      values,
				Links:     parseLinks(dataset.Links),
			}
		case "error":
			errorSets = append(errorSets, dataset.Error)
		}
	}

	resp := &Response{
		model:       model,
		mainDataSet: resolveRefs(dataSets["d:main"], dataSets),
		errors:      errorSets,
		raw:         response.rawJson,
	}
	return resp, nil
}

func processValues(values [][]json.RawMessage, model *Model) ([][]any, error) {
	var processedData [][]any
	for rowIndex := range values {
		var row []any
		for columnIndex, field := range model.Fields {
			if field.IsReference() {
				value, err := dataSetRefDeserializer(values[rowIndex][columnIndex])
				if err != nil {
					return nil, err
				}
				row = append(row, value)
			} else {
				switch field.Type {
				case "number":
					value, err := longDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						value, err := doubleDeserializer(values[rowIndex][columnIndex])
						if err != nil {
							return nil, err
						}
						row = append(row, value)
						continue
					}
					row = append(row, value)
				case "long":
					value, err := longDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				case "double":
					value, err := doubleDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				case "string", "csv":
					value, err := stringDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				case "boolean":
					value, err := booleanDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				case "timestamp":
					value, err := timestampDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				case "complex", "timeseries": // inlined complex types
					var value [][]json.RawMessage
					err := json.Unmarshal(values[rowIndex][columnIndex], &value)
					if err != nil {
						return nil, err
					}
					processedValues, err := processValues(value, field.Model)
					if err != nil {
						return nil, err
					}
					row = append(row, ComplexData{
						DataModel: field.Model,
						Data:      processedValues,
					})
				case "object": // unknown, mixed types
					value, err := jsonScalarDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				case "json": // unknown json
					value, err := jsonObjectDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				default: // unknown types
					value, err := jsonScalarDeserializer(values[rowIndex][columnIndex])
					if err != nil {
						return nil, err
					}
					row = append(row, value)
				}
			}
		}
		processedData = append(processedData, row)
	}
	return processedData, nil
}

func parseLinks(rawLinks map[string]rawLink) map[string]Link {
	links := make(map[string]Link)
	for key, value := range rawLinks {
		links[key] = Link(value)
	}
	return links
}

func createModelIndex(model *Model) map[string]*Model {
	var index = make(map[string]*Model)
	appendModelToIndex(model, index)
	return index
}

func appendModelToIndex(model *Model, index map[string]*Model) {
	index[model.Name] = model
	for _, field := range model.Fields {
		if field.Model != nil {
			appendModelToIndex(field.Model, index)
		}
	}
}

func resolveRefs(dataSet *DataSet, dataSets map[string]*DataSet) *DataSet {
	if dataSet == nil {
		return nil
	}
	for r, row := range dataSet.Data {
		for c, val := range row {
			switch ref := val.(type) {
			case DataSetRef:
				referenced := resolveRefs(dataSets[ref.Dataset], dataSets)
				dataSet.Data[r][c] = referenced
			}
		}
	}
	return dataSet
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
