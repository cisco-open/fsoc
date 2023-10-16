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

	"github.com/cisco-open/fsoc/platform/api"
)

// Tests that a proper server response is correctly deserialized into a Response struct
func TestExecuteUqlQuery_HappyDay_Version1(t *testing.T) {
	testExecuteUqlQuery_HappyDay(t, ApiVersion1)
}

func TestExecuteUqlQuery_HappyDay_NoVersion(t *testing.T) {
	testExecuteUqlQuery_HappyDay(t, ApiVersion(""))
}

func testExecuteUqlQuery_HappyDay(t *testing.T, apiVersion ApiVersion) {
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
	response, err := executeUqlQuery(&Query{"fetch count, events(logs:generic_record) from entities"}, apiVersion, mockExecuteResponse(serverResponse))

	// then
	check := assert.New(t)

	check.Nil(err, "parsing of response failed when it should not")
	check.False(response.HasErrors(), "parsed response reports errors when it should not")

	eventModel := model("m:events-1", timestampFieldH("timestamp", &Hint{Kind: "event", Field: "timestamp"}), stringFieldH("raw", &Hint{Kind: "event", Field: "raw"}))
	mainModel := model("m:main", numberField("count"), timeSeriesField("events(logs:generic_record)", eventModel, &Hint{Kind: "event", Type: "logs:generic_record"}))

	check.EqualValues(mainModel, response.Model(), "main model not parsed correctly")
	check.EqualValues(serverResponse, response.Raw())

	eventsDataSet := &DataSet{
		Name:      "d:events-1",
		DataModel: eventModel,
		Data: [][]any{
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
		Links: make(map[string]Link),
	}

	mainDataSet := &DataSet{
		Name:      "d:main",
		DataModel: mainModel,
		Data:      [][]any{{748, eventsDataSet}},
		Links:     make(map[string]Link),
	}
	check.EqualValues(mainDataSet.Data[0][1].(*DataSet).Links, response.Main().Data[0][1].(*DataSet).Links, "Data not parsed correctly")
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
	response, err := executeUqlQuery(&Query{"fetch count from entities"}, ApiVersion1, mockExecuteResponse(serverResponse))

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

func TestExecuteUqlQuery_DataTypes(t *testing.T) {
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

	// Please note: When changed, please also change TestTransformForJsonOutput_DataTypes in json_test.go
	cases := []params{
		{Alias: "int-as-number", Type: "number", Value: 123},
		{Alias: "double-as-number", Type: "number", Value: 45.47},
		{Alias: "long", Type: "long", Value: 10000},
		{Alias: "double", Type: "double", Value: 10.01},
		{Alias: "string", Type: "string", Value: "service", Formatted: `"service"`},
		{Alias: "boolean", Type: "boolean", Value: true},
		{Alias: "timestamp", Type: "timestamp", Value: time.Date(2022, time.December, 5, 0, 30, 0, 0, time.UTC), Formatted: `"2022-12-05T00:30:00Z"`},
		{Alias: "timestamp-iso8601", Type: "timestamp", Value: time.Date(2022, time.December, 5, 0, 0, 0, 0, time.UTC), Formatted: `"2022-12-05"`},
		{Alias: "unknown", Type: "unknown", Value: "unknown", Formatted: `"unknown"`},
		{Alias: "int-as-object", Type: "object", Value: 123},
		{Alias: "double-as-object", Type: "object", Value: 45.47},
		{Alias: "boolean-as-object", Type: "object", Value: true},
		{Alias: "string-as-object", Type: "object", Value: "service", Formatted: `"service"`},
		{Alias: "timestamp-as-object", Type: "object", Value: `2022-12-05T00:30:00Z`, Formatted: `"2022-12-05T00:30:00Z"`},
		{Alias: "json-object", Type: "json", Value: jsonObject(`{ "answer": 42 }`), Formatted: `{ "answer": 42 }`},
		{Alias: "json-array", Type: "json", Value: jsonObject(`[ 1, 2, "Fizz" ]`), Formatted: `[ 1, 2, "Fizz" ]`},
		{Alias: "csv", Type: "csv", Value: "foo,bar", Formatted: `"foo,bar"`},
		{Alias: "duration", Type: "duration", Value: "PT0.000515106S", Formatted: `"PT0.000515106S"`},
		{Alias: "string-tab", Type: "string", Value: "\tinvalid yaml document", Formatted: `"\tinvalid yaml document"`},
		{Alias: "string-colon", Type: "string", Value: "io.SenderException: Failed.", Formatted: `"io.SenderException: Failed."`},
		{Alias: "string-multiline", Type: "string", Value: "line1\nline2", Formatted: `"line1\nline2"`},
	}

	for _, c := range cases {
		t.Run(c.Alias, func(t *testing.T) {
			var rendered bytes.Buffer
			err := serverResponseTemplate.Execute(&rendered, c)

			check := assert.New(t)
			check.NoError(err, "failed to compile server template")

			// when
			response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(rendered.String()))
			check.NoError(err, "parsing of response failed when it should not")

			// then
			check.EqualValues(c.Value, response.Main().Values()[0][0], "")
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
		response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

		check := assert.New(t)
		check.NoError(err, "parsing of response failed when it should not")

		// then
		inlinedComplexValue := response.Main().Values()[0][0].(ComplexData)
		check.EqualValues("inline", inlinedComplexValue.Values()[0][0], "inlined complex value not parsed correctly")
		check.EqualValues(456, inlinedComplexValue.Values()[0][1], "inlined complex value not parsed correctly")
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
		response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

		check := assert.New(t)
		check.NoError(err, "parsing of response failed when it should not")

		// then
		referencedDataset := response.Main().Values()[0][0].(*DataSet)
		check.EqualValues("d:sub-1", referencedDataset.Name, "referenced dataset name not parsed properly")
		check.EqualValues("referenced", referencedDataset.Values()[0][0], "referenced complex value not parsed properly")
		check.EqualValues(789, referencedDataset.Values()[0][1], "referenced complex value not parsed properly")
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
}

func TestContinueQuery_HappyDay(t *testing.T) {
	// given
	// language=json
	serverResponse := `[
	  {
		"type": "model",
		"model": {
		  "name": "m:main",
		  "fields": [
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
		  [ { "$dataset": "d:events-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:events-1')]" } ]
		]
	  },
	  {
		"type": "data",
		"_links": {
			"follow": {
				"href": "/monitoring/monitoring/v1/query/continue?cursor=ewogICJ0eXBlIiA6ICJldmVudCIsCiAgImRvY3VtZW50SWQiIDogIkNOam5ydHZhTUJJekNpbHNiMmR6TFRRM1lUQXhaR1k1TFRVMFlUQXRORGN5WWkwNU5tSTRMVGRqT0dZMk5HVmlOMk5pWmhBQ0dNN0doWjRHSVAvLy8vOFAiLAogICJxdWVyeSIgOiAiRkVUQ0ggZXZlbnRzKGxvZ3M6Z2VuZXJpY19yZWNvcmQpIHsgdGltZXN0YW1wLCByYXcsIGF0dHJpYnV0ZXMoc2V2ZXJpdHkpLCBlbnRpdHlJZCwgc3BhbklkLCB0cmFjZUlkIH0gTElNSVRTIGV2ZW50cy5jb3VudCg1MCkgT1JERVIgZXZlbnRzLmFzYygpIFVOVElMIG5vdygpIFNJTkNFIDIwMjMtMDEtMTNUMTM6NTc6MjAuNDcyWiIKfQ%3D%3D"
			}
		},
		"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:events-logs-generic_record-')]", "$model": "m:events-1" },
		"dataset": "d:events-1",
		"data": [
		  [ "2022-12-05T07:30:56Z", "2022-12-05T07:30:56 DEBUG LogExample [main] This is a Debug message" ],
		  [ "2022-12-05T07:30:56Z", "2022-12-05T07:30:56 ERROR LogExample [main] This is an Error message" ],
		  [ "2022-12-05T07:30:56Z", "2022-12-05T07:30:56 INFO LogExample [main] This is an Info message" ]
		]
	  }
	]`

	initialResponse, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// when
	assert.Nil(t, err)

	_, err = continueUqlQuery(initialResponse.Main().Values()[0][0].(*DataSet), "follow", &mockUqlService{
		executeBehavior: func(query *Query, version ApiVersion) (parsedResponse, error) {
			t.Fail()
			return parsedResponse{}, nil
		},
		continueBehavior: func(link *Link) (parsedResponse, error) {
			assert.Equal(
				t,
				"/monitoring/monitoring/v1/query/continue?cursor=ewogICJ0eXBlIiA6ICJldmVudCIsCiAgImRvY3VtZW50SWQiIDogIkNOam5ydHZhTUJJekNpbHNiMmR6TFRRM1lUQXhaR1k1TFRVMFlUQXRORGN5WWkwNU5tSTRMVGRqT0dZMk5HVmlOMk5pWmhBQ0dNN0doWjRHSVAvLy8vOFAiLAogICJxdWVyeSIgOiAiRkVUQ0ggZXZlbnRzKGxvZ3M6Z2VuZXJpY19yZWNvcmQpIHsgdGltZXN0YW1wLCByYXcsIGF0dHJpYnV0ZXMoc2V2ZXJpdHkpLCBlbnRpdHlJZCwgc3BhbklkLCB0cmFjZUlkIH0gTElNSVRTIGV2ZW50cy5jb3VudCg1MCkgT1JERVIgZXZlbnRzLmFzYygpIFVOVElMIG5vdygpIFNJTkNFIDIwMjMtMDEtMTNUMTM6NTc6MjAuNDcyWiIKfQ%3D%3D",
				link.Href,
			)
			return parsedResponse{}, nil
		},
	})

	// then
	assert.Nil(t, err)
}

func TestMakeUqlProblem(t *testing.T) {
	// given
	apiProblem := api.Problem{
		Type:   "https://developer.cisco.com/docs/appdynamics/errors/#!input-validation",
		Title:  "Query compilation failure!",
		Detail: "Unable to compile due to query semantic errors.",
		Status: 0,
		Extensions: map[string]any{
			"query": "FETCH error",
			"errorDetails": []any{
				map[string]any{
					"message":          "Error message",
					"fixSuggestion":    "Fix suggestion",
					"fixPossibilities": []any{"fix option 1", "fix option 2"},
					"errorType":        "SEMANTIC",
					"errorFrom":        "1:2",
					"errorTo":          "3:4",
				},
			},
		},
	}

	// when
	problem := makeUqlProblem(apiProblem)

	// then
	check := assert.New(t)

	expected := uqlProblem{
		query:  "FETCH error",
		title:  "Query compilation failure!",
		detail: "Unable to compile due to query semantic errors.",
		errorDetails: []errorDetail{
			{
				message:          "Error message",
				fixSuggestion:    "Fix suggestion",
				fixPossibilities: []string{"fix option 1", "fix option 2"},
				errorType:        "SEMANTIC",
				errorFrom: position{
					line:   1,
					column: 2,
				},
				errorTo: position{
					line:   3,
					column: 4,
				},
			},
		},
	}
	check.EqualValues(expected, problem, "uqlProblem struct should be correctly mapped from source data")
}

func TestAsStringOrNothing(t *testing.T) {
	// given
	notString := 12

	// when
	asString := asStringOrNothing(notString)

	// then
	check := assert.New(t)
	check.Zero(asString, "non-strings values should be returned as zero-strings")
}

func TestAsPositionOrNothing(t *testing.T) {
	// given
	type params struct {
		Case     string
		AsString string
		Expected position
	}

	cases := []params{
		{Case: "valid position", AsString: "1:2", Expected: position{line: 1, column: 2}},
		{Case: "invalid value", AsString: "nonsense", Expected: position{}},
		{Case: "half position", AsString: "1:", Expected: position{}},
		{Case: "half position", AsString: "1", Expected: position{}},
		{Case: "negative values", AsString: "-1:2", Expected: position{line: -1, column: 2}},
		{Case: "half invalid", AsString: "1:nonsense", Expected: position{}},
		{Case: "half invalid", AsString: "nonsense:1", Expected: position{}},
		{Case: "too many values", AsString: "1:2:3", Expected: position{}},
	}

	// when - then
	for _, c := range cases {
		t.Run(c.Case, func(t *testing.T) {
			check := assert.New(t)
			actual := asPositionOrNothing(c.AsString)
			check.EqualValues(c.Expected, actual)
		})
	}
}

type mockUqlService struct {
	executeBehavior  func(query *Query, version ApiVersion) (parsedResponse, error)
	continueBehavior func(link *Link) (parsedResponse, error)
}

func (s *mockUqlService) Execute(query *Query, apiVersion ApiVersion) (parsedResponse, error) {
	return s.executeBehavior(query, apiVersion)
}

func (s *mockUqlService) Continue(link *Link) (parsedResponse, error) {
	return s.continueBehavior(link)
}

func emptyResponse() uqlService {
	return mockExecuteResponse("")
}

func mockExecuteResponse(response string) uqlService {
	return &mockUqlService{
		executeBehavior: func(query *Query, version ApiVersion) (parsedResponse, error) {
			rawJson := json.RawMessage(response)
			var chunks []parsedChunk
			err := json.Unmarshal(rawJson, &chunks)
			if err != nil {
				return parsedResponse{}, err
			}
			return parsedResponse{
				chunks:  chunks,
				rawJson: &rawJson,
			}, nil
		},
		continueBehavior: func(link *Link) (parsedResponse, error) {
			panic("continue response not mocked")
		},
	}
}
