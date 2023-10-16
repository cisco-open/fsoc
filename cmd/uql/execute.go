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
	"encoding/json"
	"fmt"
	"strings"
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

type uqlService interface {
	Execute(query *Query, apiVersion ApiVersion) (parsedResponse, error)
	Continue(link *Link) (parsedResponse, error)
}

// Default clients that can be used as `uql.Client` from other packages
// - Client should be used to go with the default API version (possibly overridden in fsoc config)
// - ClientV1 should be used when `v1` is required
var Client UqlClient = NewClient()
var ClientV1 UqlClient = NewClient(WithClientApiVersion(ApiVersion1))

func executeUqlQuery(query *Query, apiVersion ApiVersion, backend uqlService) (*Response, error) {
	if query == nil || strings.Trim(query.Str, "") == "" {
		return nil, fmt.Errorf("uql query missing")
	}

	if apiVersion == "" {
		apiVersion = ApiVersionDefault
		if GlobalConfig.ApiVersion != nil && *GlobalConfig.ApiVersion != "" {
			apiVersion = *GlobalConfig.ApiVersion // from fsoc config file
		}
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
