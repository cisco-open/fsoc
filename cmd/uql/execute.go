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
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"

	"github.com/cisco-open/fsoc/platform/api"
)

type ApiVersion string

const (
	ApiVersion1     ApiVersion = "v1"
	ApiVersion1Beta ApiVersion = "v1beta"
)

// Query represents a UQL request body
type Query struct {
	Str string `json:"query"`
}

type rawResponse []rawChunk

// rawChunk is a union of all fields on all UQL response dataset types
type rawChunk struct {
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

type uqlService func(query *Query, url string) (rawResponse, error)

// ExecuteQuery sends an execute request to the UQL service
func ExecuteQuery(query *Query, apiVersion ApiVersion) (*Response, error) {
	log.WithFields(log.Fields{"query": query.Str, "apiVersion": apiVersion}).Info("executing UQL query")

	return executeUqlQuery(query, apiVersion, func(query *Query, url string) (rawResponse, error) {
		var response rawResponse
		err := api.JSONPost(url, query, &response, nil)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to execute UQL Query: '%s'", query.Str))
		}
		return response, nil
	})
}

func executeUqlQuery(query *Query, apiVersion ApiVersion, backend uqlService) (*Response, error) {
	if query == nil || strings.Trim(query.Str, "") == "" {
		return nil, fmt.Errorf("uql query missing")
	}

	if apiVersion == "" {
		return nil, fmt.Errorf("uql API version missing")
	}

	response, err := backend(query, "/monitoring/"+string(apiVersion)+"/query/execute")
	if err != nil {
		return nil, err
	}

	var model *Model
	var dataSets []*DataSet
	var errorSets []*Error
	var modelIndex map[string]*Model

	for _, dataset := range response {
		switch dataset.Type {
		case "model":
			err = json.Unmarshal(dataset.Model, &model)
			if err != nil {
				return nil, err
			}
			modelIndex = createModelIndex(model)
		case "data":
			var modelRef modelRef
			err = json.Unmarshal(dataset.Model, &modelRef)
			if err != nil {
				return nil, err
			}
			var dataSetModel = modelIndex[modelRef.Model]
			var values, err = processValues(dataset.Data, dataSetModel)
			if err != nil {
				return nil, err
			}
			dataSets = append(dataSets, &DataSet{
				Name:     dataset.Dataset,
				Model:    dataSetModel,
				Metadata: dataset.Metadata,
				Values:   values,
			})
		case "error":
			errorSets = append(errorSets, dataset.Error)
		}
	}

	return &Response{model: model, dataSets: createDataSetIndex(dataSets), errors: errorSets}, nil
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
				case "string":
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
					row = append(row, processedValues)
				default: // unknown types
					value, err := stringDeserializer(values[rowIndex][columnIndex])
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

func createDataSetIndex(dataSets []*DataSet) map[string]*DataSet {
	var index = make(map[string]*DataSet)
	for _, dataSet := range dataSets {
		index[dataSet.Name] = dataSet
	}
	return index
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
