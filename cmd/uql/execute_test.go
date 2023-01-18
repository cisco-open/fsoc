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
	"bytes"
	"encoding/json"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests that a proper server response is correctly deserialized into an Response struct
func TestExecuteUqlQuery_HappyDay(t *testing.T) {
	// given
	// language=json
	serverResponse := `[
	  {
		"type": "model",
		"model": {
		  "name": "m:main",
		  "fields": [
			{ "alias": "count", "type": "number", "hints": {} },
			{ "alias": "events(logs:generic_record)", "type": "timeseries", "hints": { "kind": "event", "type": "logs:generic_record" }, "form": "reference", "model": {
				"name": "m:events-1",
				"fields": [
				  { "alias": "timestamp", "type": "timestamp", "hints": { "kind": "event", "field": "timestamp" } },
				  { "alias": "raw", "type": "string", "hints": { "kind": "event", "field": "raw"  } }
				]
			  }
			}
		  ]
		}
	  },
	  {
		"type": "data",
		"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
		"dataset": "d:main",
		"data": [
		  [ 748, { "$dataset": "d:events-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:events-1')]" } ]
		]
	  },
	  {
		"type": "data",
		"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:events-logs-generic_record-')]", "$model": "m:events-1" },
		"dataset": "d:events-1",
		"data": [
		  [ "2022-12-05T07:30:56Z", "2022-12-05T07:30:56 DEBUG LogExample [main] This is a Debug message" ],
		  [ "2022-12-05T07:30:56Z", "2022-12-05T07:30:56 ERROR LogExample [main] This is an Error message" ],
		  [ "2022-12-05T07:30:56Z", "2022-12-05T07:30:56 INFO LogExample [main] This is an Info message" ]
		]
	  }
	]`

	// when
	response, err := executeUqlQuery(&Query{"fetch count, events(logs:generic_record) from entities"}, ApiVersion1, mockResponse(serverResponse))

	// then
	check := assert.New(t)

	check.Nil(err, "parsing of response failed when it should not")
	check.False(response.HasErrors(), "parsed response reports errors when it should not")

	eventModel := model("m:events-1", timestampFieldH("timestamp", &Hint{Kind: "event", Field: "timestamp"}), stringFieldH("raw", &Hint{Kind: "event", Field: "raw"}))
	mainModel := model("m:main", numberField("count"), timeSeriesField("events(logs:generic_record)", eventModel, &Hint{Kind: "event", Type: "logs:generic_record"}))

	check.EqualValues(mainModel, response.Model(), "main model not parsed correctly")

	mainDataSet := &DataSet{
		Name:   "d:main",
		Model:  mainModel,
		Values: [][]any{{748, DataSetRef{Dataset: "d:events-1", JsonPath: "$..[?(@.type == 'data' && @.dataset == 'd:events-1')]"}}},
	}
	check.EqualValues(mainDataSet, response.Main(), "main data set not parsed correctly")

	eventsDataSet := &DataSet{
		Name:  "d:events-1",
		Model: eventModel,
		Values: [][]any{
			{
				time.Date(2022, time.December, 5, 7, 30, 56, 0, time.UTC),
				"2022-12-05T07:30:56 DEBUG LogExample [main] This is a Debug message",
			},
			{
				time.Date(2022, time.December, 5, 7, 30, 56, 0, time.UTC),
				"2022-12-05T07:30:56 ERROR LogExample [main] This is an Error message",
			},
			{
				time.Date(2022, time.December, 5, 7, 30, 56, 0, time.UTC),
				"2022-12-05T07:30:56 INFO LogExample [main] This is an Info message",
			},
		},
	}

	responseEvents := response.DataSet(response.Main().Values[0][1].(DataSetRef))
	check.EqualValues(eventsDataSet.Name, responseEvents.Name, "events data set not parsed correctly")
}

func TestExecuteUqlQuery_Errors(t *testing.T) {
	// given
	// language=json
	serverResponse := `[
	  {
		"type": "model",
		"model": {
		  "name": "m:main",
		  "fields": [
			{ "alias": "count", "type": "number", "hints": {} }
		  ]
		}
	  },
	  {
		"type": "error",
		"error": {
			"type": "internal-server-error",
			"title": "downstream failure",
			"detail": "service not available"
		}
	  }
	]`

	// when
	response, err := executeUqlQuery(&Query{"fetch count from entities"}, ApiVersion1, mockResponse(serverResponse))

	// then
	check := assert.New(t)

	check.Nil(err, "parsing of response failed when it should not")

	mainModel := model("m:main", numberField("count"))
	check.EqualValues(mainModel, response.Model(), "parsed response reports errors when it should not")

	check.Nil(response.Main(), "main data set should be null but isn't")
	check.True(response.HasErrors(), "response should contain errors but does not")

	check.Equal(1, len(response.Errors()), "more errors than expected")
	check.EqualValues(&Error{Type: "internal-server-error", Title: "downstream failure", Detail: "service not available"}, response.Errors()[0], "errors do not match")
}

func TestExecuteUqlQuery_SimpleDataTypes(t *testing.T) {
	// given
	serverResponseTemplate := template.Must(template.New("response-template").Parse(`[
	  {
		"type": "model",
		"model": {
		  "name": "m:main",
		  "fields": [
			{ "alias": "{{ .Alias }}", "type": "{{ .Type }}", "hints": {} }
		  ]
		}
	  },
	  {
		"type": "data",
		"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
		"dataset": "d:main",
		"data": [
		  [ {{if .Formatted}}{{.Formatted}}{{else}}{{.Value}}{{end}} ]
		]
	  }
	]`))

	type params struct {
		Alias     string
		Type      string
		Value     any
		Formatted string
	}

	cases := []params{
		{Alias: "int-as-number", Type: "number", Value: 123},
		{Alias: "double-as-number", Type: "number", Value: 45.47},
		{Alias: "long", Type: "long", Value: 10000},
		{Alias: "double", Type: "double", Value: 10.01},
		{Alias: "string", Type: "string", Value: "service", Formatted: "\"service\""},
		{Alias: "boolean", Type: "boolean", Value: true},
		{Alias: "timestamp", Type: "timestamp", Value: time.Date(2022, time.December, 5, 0, 30, 0, 0, time.UTC), Formatted: "\"2022-12-05T00:30:00Z\""},
		{Alias: "timestamp-iso8601", Type: "timestamp", Value: time.Date(2022, time.December, 5, 0, 0, 0, 0, time.UTC), Formatted: "\"2022-12-05\""},
		{Alias: "unknown", Type: "unknown", Value: "unknown", Formatted: "\"unknown\""},
	}

	for _, c := range cases {
		t.Run(c.Alias, func(t *testing.T) {
			var rendered bytes.Buffer
			err := serverResponseTemplate.Execute(&rendered, c)

			check := assert.New(t)
			check.NoError(err, "failed to compile server template")

			// when
			response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockResponse(rendered.String()))
			check.NoError(err, "parsing of response failed when it should not")

			// then
			check.EqualValues(c.Value, response.Main().Values[0][0], "")
		})
	}
}

func TestExecuteUqlQuery_ComplexDataTypes(t *testing.T) {
	t.Run("inlined", func(t *testing.T) {
		// given
		// language=json
		serverResponse := `[
		  {
			"type": "model",
			"model": {
			  "name": "m:main",
			  "fields": [
				{ "alias": "complex-inlined", "type": "complex", "form": "inline", "hints": {}, "model": {
				  "name": "m:sub-1",
				  "fields": [
					{ "alias": "string", "type": "string", "hints": {} },
					{ "alias": "number", "type": "number", "hints": {} }
				  ]
				 }
				}
			  ]
			}
		  },
		  {
			"type": "data",
			"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
			"dataset": "d:main",
			"data": [
			  [ [ [ "inline", 456 ] ] ]
			]
		  }
		]`

		// when
		response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockResponse(serverResponse))

		check := assert.New(t)
		check.NoError(err, "parsing of response failed when it should not")

		// then
		inlinedComplexValue := response.Main().Values[0][0].([][]any)
		check.EqualValues("inline", inlinedComplexValue[0][0], "inlined complex value not parsed correctly")
		check.EqualValues(456, inlinedComplexValue[0][1], "inlined complex value not parsed correctly")
	})

	t.Run("referenced", func(t *testing.T) {
		// given
		// language=json
		serverResponse := `[
		  {
			"type": "model",
			"model": {
			  "name": "m:main",
			  "fields": [
				{ "alias": "complex-referenced", "type": "complex", "form": "reference", "model": {
				  "name": "m:sub-1",
				  "fields": [
					{ "alias": "string", "type": "string", "hints": {} },
					{ "alias": "number", "type": "number", "hints": {} }
				  ]
				}
				}
			  ]
			}
		  },
		  {
			"type": "data",
			"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
			"dataset": "d:main",
			"data": [
			  [ { "$dataset": "d:sub-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:sub-1')]" } ]
			]
		  },
		  {
			"type": "data",
			"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:sub-1')]", "$model": "m:sub-1" },
			"dataset": "d:sub-1",
			"data": [
			  [ "referenced", 789 ]
			]
		  }
		]`

		// when
		response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockResponse(serverResponse))

		check := assert.New(t)
		check.NoError(err, "parsing of response failed when it should not")

		// then
		subRef := response.Main().Values[0][0].(DataSetRef)
		check.EqualValues(DataSetRef{JsonPath: "$..[?(@.type == 'data' && @.dataset == 'd:sub-1')]", Dataset: "d:sub-1"}, subRef, "inlined complex value not parsed correctly")
		check.EqualValues("referenced", response.DataSet(subRef).Values[0][0], "referenced complex value not parsed properly")
		check.EqualValues(789, response.DataSet(subRef).Values[0][1], "referenced complex value not parsed properly")
	})
}

func TestExecuteUqlQuery_Validation(t *testing.T) {
	t.Run("missing query struct", func(t *testing.T) {
		response, err := executeUqlQuery(nil, ApiVersion1, emptyResponse())
		assert.Nil(t, response)
		assert.ErrorContains(t, err, "uql query missing", "no error thrown")
	})
	t.Run("missing query string", func(t *testing.T) {
		response, err := executeUqlQuery(&Query{}, ApiVersion1, emptyResponse())
		assert.Nil(t, response)
		assert.ErrorContains(t, err, "uql query missing", "no error thrown")
	})
	t.Run("empty query string", func(t *testing.T) {
		response, err := executeUqlQuery(&Query{""}, ApiVersion1, emptyResponse())
		assert.Nil(t, response)
		assert.ErrorContains(t, err, "uql query missing", "no error thrown")
	})
	t.Run("missing api version", func(t *testing.T) {
		response, err := executeUqlQuery(&Query{"fetch count from entities"}, "", emptyResponse())
		assert.Nil(t, response)
		assert.ErrorContains(t, err, "uql API version missing", "no error thrown")
	})
}

func TestExecuteUqlQuery_CorrectUrl(t *testing.T) {
	t.Run("correct v1 URL", func(t *testing.T) {
		_, _ = executeUqlQuery(&Query{"fetch count from entities"}, ApiVersion1, func(query *Query, url string) (rawResponse, error) {
			assert.Equal(t, "/monitoring/v1/query/execute", url, "v1 url is incorrect")
			return rawResponse{}, nil
		})
	})
	t.Run("correct v1beta URL", func(t *testing.T) {
		_, _ = executeUqlQuery(&Query{"fetch count from entities"}, ApiVersion1Beta, func(query *Query, url string) (rawResponse, error) {
			assert.Equal(t, "/monitoring/v1beta/query/execute", url, "v1beta url is incorrect")
			return rawResponse{}, nil
		})
	})
}

func emptyResponse() uqlService {
	return mockResponse("")
}

func mockResponse(response string) uqlService {
	return func(query *Query, url string) (rawResponse, error) {
		var resp rawResponse
		err := json.Unmarshal([]byte(response), &resp)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}
