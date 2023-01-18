package uql

import (
	"bytes"
	"encoding/json"
	"testing"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTransformForJsonOutput_ComplexData(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": {
      "name": "m:main",
      "fields": [
        { "alias": "id", "type": "string", "hints": { "kind": "entity", "field": "id", "type": "infra:container" } },
        { "alias": "metrics", "type": "complex", "hints": { "kind": "metric", "type": "infra:container.cpu.system.utilization" }, "form": "reference",
          "model": {
            "name": "m:metrics",
            "fields": [
              { "alias": "source", "type": "string", "hints": { "kind": "metric", "field": "source" } },
              { "alias": "metrics", "type": "timeseries", "hints": { "kind": "metric", "type": "infra:container.cpu.system.utilization" },
                "form": "inline",
                "model": {
                  "name": "m:metrics_2",
                  "fields": [
                    { "alias": "timestamp", "type": "timestamp", "hints": { "kind": "metric", "field": "timestamp", "type": "infra:container.cpu.system.utilization" } },
                    { "alias": "value", "type": "number", "hints": { "kind": "metric", "field": "value", "type": "infra:container.cpu.system.utilization" } }
                  ] }
              }
            ] }
        },
        { "alias": "events", "type": "timeseries", "hints": { "kind": "event", "type": "logs:generic_record" }, "form": "reference",
          "model": {
            "name": "m:events",
            "fields": [
              { "alias": "timestamp", "type": "timestamp", "hints": { "kind": "event", "field": "timestamp", "type": "logs:generic_record" } },
              { "alias": "raw", "type": "string", "hints": { "kind": "event", "field": "raw", "type": "logs:generic_record" } }
            ] }
        }
      ] }
  }, {
    "type": "data",
    "model": {
      "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]",
      "$model": "m:main"
    },
    "dataset": "d:main",
    "data": [
      [ "infra:container:kLRMaC54NpSEJ9SEoFZuqA", { "$dataset": "d:metrics-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-1')]" }, { "$dataset": "d:events-2", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:events-2')]" } ],
      [ "infra:container:nSQcvyuEPkumAipUZTgOJQ", { "$dataset": "d:metrics-3", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-3')]" }, { "$dataset": "d:events-4", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:events-4')]" } ],
      [ "infra:container:tu0AV/jvNT6HxPf+4NFwDQ", { "$dataset": "d:metrics-5", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-5')]" }, { "$dataset": "d:events-6", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:events-6')]" } ]
    ]
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics')]", "$model": "m:metrics" },
    "dataset": "d:metrics-1",
    "data": [ [ "opentelemetry",
     [ [ "2023-01-04T14:35Z", 0.03720745700301489 ], [ "2023-01-04T14:36Z", 0.01031480793850134 ], [ "2023-01-04T14:37Z", 0.4074337863009272 ],[ "2023-01-04T14:38Z", 0.057308243138152616 ] ] ] ]
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:events')]", "$model": "m:events" },
    "dataset": "d:events-2",
    "data": [
      [ "2023-01-04T14:37:51.314Z", "io.jaegertracing.internal.exceptions.SenderException: Failed to flush spans." ],
      [ "2023-01-04T14:37:51.314Z", "\tat io.jaegertracing.thrift.internal.senders.ThriftSender.flush(ThriftSender.java:115)" ]
    ]
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics')]", "$model": "m:metrics" },
    "dataset": "d:metrics-3",
    "data": []
  },
  {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:events')]", "$model": "m:events" },
    "dataset": "d:events-4",
    "data": [
      [ "2023-01-04T14:42:02.115Z", "2023-01-04T14:42:02 ERROR LogExample [main] MultiLine IllArg Error : \njava.lang.IllegalArgumentException\n\tat ExceptionsClass.throwIllegalArgumentException(ExceptionsClass.java:8)\n\tat ExceptionClassD.throwIllegalArgumentException(ExceptionClassD.java:9)\n\tat ExceptionClassC.throwIllegalArgumentException(ExceptionClassC.java:9)\n\tat ExceptionClassB.throwIllegalArgumentException(ExceptionClassB.java:9)\n\tat ExceptionClassA.throwIllegalArgumentException(ExceptionClassA.java:9)\n\tat LogExample.main(LogExample.java:49)" ]
    ]
  },
  {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics')]", "$model": "m:metrics" },
    "dataset": "d:metrics-5",
    "data": [ [ "opentelemetry", [ [ "2023-01-04T14:35Z", 0.05877454922543631 ], [ "2023-01-04T14:36Z", 0.040730745299347976 ], [ "2023-01-04T14:37Z", 0.0451670159824317 ] ] ] ]
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:events')]", "$model": "m:events" },
    "dataset": "d:events-6",
    "data": [
      [ "2023-01-04T14:40:28.487Z", "2023-01-04T14:40:32 ERROR LogExample [main] MultiLine Arth Error : \njava.lang.ArithmeticException\n\tat ExceptionsClass.throwArithmeticException(ExceptionsClass.java:4)\n\tat ExceptionClassD.throwArithmeticException(ExceptionClassD.java:4)\n\tat ExceptionClassC.throwArithmeticException(ExceptionClassC.java:4)\n\tat ExceptionClassB.throwArithmeticException(ExceptionClassB.java:4)\n\tat ExceptionClassA.throwArithmeticException(ExceptionClassA.java:4)\n\tat LogExample.main(LogExample.java:44)" ],
      [ "2023-01-04T14:40:02.071Z", "2023-01-04T14:40:02 ERROR LogExample [main] This is an Error message for Levitate" ]
    ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	transformed, err := transformForJsonOutput(response)

	// Then
	assert.Nil(t, err)
	expected := `{"model":{"id":"string","metrics":{"source":"string","metrics":{"timestamp":"string","value":"number"}},"events":{"timestamp":"string","raw":"string"}},"data":[{"id":"infra:container:kLRMaC54NpSEJ9SEoFZuqA","metrics":[{"source":"opentelemetry","metrics":[{"timestamp":"2023-01-04T14:35:00Z","value":0.03720745700301489},{"timestamp":"2023-01-04T14:36:00Z","value":0.01031480793850134},{"timestamp":"2023-01-04T14:37:00Z","value":0.4074337863009272},{"timestamp":"2023-01-04T14:38:00Z","value":0.057308243138152616}]}],"events":[{"timestamp":"2023-01-04T14:37:51.314Z","raw":"io.jaegertracing.internal.exceptions.SenderException: Failed to flush spans."},{"timestamp":"2023-01-04T14:37:51.314Z","raw":"\tat io.jaegertracing.thrift.internal.senders.ThriftSender.flush(ThriftSender.java:115)"}]},{"id":"infra:container:nSQcvyuEPkumAipUZTgOJQ","metrics":[],"events":[{"timestamp":"2023-01-04T14:42:02.115Z","raw":"2023-01-04T14:42:02 ERROR LogExample [main] MultiLine IllArg Error : \njava.lang.IllegalArgumentException\n\tat ExceptionsClass.throwIllegalArgumentException(ExceptionsClass.java:8)\n\tat ExceptionClassD.throwIllegalArgumentException(ExceptionClassD.java:9)\n\tat ExceptionClassC.throwIllegalArgumentException(ExceptionClassC.java:9)\n\tat ExceptionClassB.throwIllegalArgumentException(ExceptionClassB.java:9)\n\tat ExceptionClassA.throwIllegalArgumentException(ExceptionClassA.java:9)\n\tat LogExample.main(LogExample.java:49)"}]},{"id":"infra:container:tu0AV/jvNT6HxPf+4NFwDQ","metrics":[{"source":"opentelemetry","metrics":[{"timestamp":"2023-01-04T14:35:00Z","value":0.05877454922543631},{"timestamp":"2023-01-04T14:36:00Z","value":0.040730745299347976},{"timestamp":"2023-01-04T14:37:00Z","value":0.0451670159824317}]}],"events":[{"timestamp":"2023-01-04T14:40:28.487Z","raw":"2023-01-04T14:40:32 ERROR LogExample [main] MultiLine Arth Error : \njava.lang.ArithmeticException\n\tat ExceptionsClass.throwArithmeticException(ExceptionsClass.java:4)\n\tat ExceptionClassD.throwArithmeticException(ExceptionClassD.java:4)\n\tat ExceptionClassC.throwArithmeticException(ExceptionClassC.java:4)\n\tat ExceptionClassB.throwArithmeticException(ExceptionClassB.java:4)\n\tat ExceptionClassA.throwArithmeticException(ExceptionClassA.java:4)\n\tat LogExample.main(LogExample.java:44)"},{"timestamp":"2023-01-04T14:40:02.071Z","raw":"2023-01-04T14:40:02 ERROR LogExample [main] This is an Error message for Levitate"}]}]}`
	byteArray, _ := json.Marshal(transformed)
	assert.Equal(t, expected, string(byteArray))
}

func TestTransformForJsonOutput_MissingVsEmptyDatasets(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": {
      "name": "m:main",
      "fields": [
        { "alias": "id", "type": "string", "hints": { "kind": "entity", "field": "id", "type": "k8s:pod" } },
        { "alias": "metrics", "type": "timeseries", "hints": { "kind": "metric", "type": "alerting:health.status" }, "form": "reference",
          "model": {
            "name": "m:metrics",
            "fields": [
              { "alias": "timestamp", "type": "timestamp", "hints": { "kind": "metric", "field": "timestamp", "type": "alerting:health.status" } },
              { "alias": "value", "type": "number", "hints": { "kind": "metric", "field": "value", "type": "alerting:health.status" } }
            ] } }
      ] }
  },
  {
    "type": "data",
    "model": {
      "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]",
      "$model": "m:main"
    },
    "dataset": "d:main",
    "data": [
      [ "entity-with-data", { "$dataset": "d:metrics-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-1')]" } ],
      [ "entity-no-data", { "$dataset": "d:metrics-2", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-2')]" } ],
      [ "entity-missing-data", { "$dataset": "missing", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'missing')]" } ]
    ]
  },
  {
    "type": "data",
    "model": {
      "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics')]",
      "$model": "m:metrics"
    },
    "dataset": "d:metrics-1",
    "data": [ [ "2023-01-03T11:16Z", 1 ], [ "2023-01-03T11:17Z", 1 ] ] },
  {
    "type": "data",
    "model": {
      "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics')]",
      "$model": "m:metrics"
    },
    "dataset": "d:metrics-2",
    "data": []
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	transformed, err := transformForJsonOutput(response)

	// Then
	assert.Nil(t, err)
	expected := `{"model":{"id":"string","metrics":{"timestamp":"string","value":"number"}},"data":[{"id":"entity-with-data","metrics":[{"timestamp":"2023-01-03T11:16:00Z","value":1},{"timestamp":"2023-01-03T11:17:00Z","value":1}]},{"id":"entity-no-data","metrics":[]},{"id":"entity-missing-data","metrics":null}]}`
	byteArray, _ := json.Marshal(transformed)
	assert.Equal(t, expected, string(byteArray))
}

func TestTransformForJsonOutput_SpacesAndNonAlphaAliases(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": { "name": "m:main", "fields": [ { "alias": "foo @*(&!)?#//", "type": "string", "hints": { "kind": "entity", "field": "id" } } ] }
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
    "dataset": "d:main",
    "data": [ [ "apm:service:oTHR/29IOh+/AiyhjzQhyQ" ] ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	transformed, err := transformForJsonOutput(response)

	// Then
	assert.Nil(t, err)
	expected := `{"model":{"foo @*(\u0026!)?#//":"string"},"data":[{"foo @*(\u0026!)?#//":"apm:service:oTHR/29IOh+/AiyhjzQhyQ"}]}`
	bytes, _ := json.Marshal(transformed)
	assert.Equal(t, expected, string(bytes))
}

func TestTransformForJsonOutput_AliasIsGoKeyword(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": { "name": "m:main", "fields": [ { "alias": "struct", "type": "string", "hints": { "kind": "entity", "field": "id" } } ] }
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
    "dataset": "d:main",
    "data": [ [ "apm:service:oTHR/29IOh+/AiyhjzQhyQ" ] ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	transformed, err := transformForJsonOutput(response)

	// Then
	assert.Nil(t, err)
	expected := `{"model":{"struct":"string"},"data":[{"struct":"apm:service:oTHR/29IOh+/AiyhjzQhyQ"}]}`
	byteArray, _ := json.Marshal(transformed)
	assert.Equal(t, expected, string(byteArray))
}

func TestTransformForJsonOutput_DataTypes(t *testing.T) {
	// Given
	// language=json
	serverResponseTemplate := template.Must(template.New("response-template").Parse(`
	[ {
		"type": "model",
		"model": { "name": "m:main", "fields": [ { "alias": "{{ .Alias }}", "type": "{{ .Type }}", "hints": {} } ] }
	  }, {
		"type": "data",
		"model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
		"dataset": "d:main",
		"data": [ [ {{if .Formatted}}{{.Formatted}}{{else}}{{.Value}}{{end}} ] ]
	  } ]`))
	outputTemplate := template.Must(template.New("output-template").Parse(`{"model":{"{{.Alias}}":"{{.JsonType}}"},"data":[{"{{.Alias}}":{{if .Formatted}}{{.Formatted}}{{else}}{{.Value}}{{end}}}]}`))

	type params struct {
		Alias     string
		Type      string
		Value     any
		Formatted string
		JsonType  string
	}

	// Please note: When changed, please also change TestExecuteUqlQuery_DataTypes in execute_test.go
	cases := []params{
		{Alias: "int-as-number", Type: "number", Value: 123, JsonType: "number"},
		{Alias: "double-as-number", Type: "number", Value: 45.47, JsonType: "number"},
		{Alias: "long", Type: "long", Value: 10000, JsonType: "number"},
		{Alias: "double", Type: "double", Value: 10.01, JsonType: "number"},
		{Alias: "string", Type: "string", Value: "service", Formatted: `"service"`, JsonType: "string"},
		{Alias: "boolean", Type: "boolean", Value: true, JsonType: "boolean"},
		{Alias: "timestamp", Type: "timestamp", Value: time.Date(2022, time.December, 5, 0, 30, 0, 0, time.UTC), Formatted: `"2022-12-05T00:00:00Z"`, JsonType: "string"},
		{Alias: "timestamp-iso8601", Type: "timestamp", Value: time.Date(2022, time.December, 5, 0, 0, 0, 0, time.UTC), Formatted: `"2022-12-05T00:00:00Z"`, JsonType: "string"},
		{Alias: "unknown", Type: "unknown", Value: "unknown", Formatted: `"unknown"`, JsonType: "string"},
		{Alias: "int-as-object", Type: "object", Value: 123, JsonType: "undefined"},
		{Alias: "double-as-object", Type: "object", Value: 45.47, JsonType: "undefined"},
		{Alias: "boolean-as-object", Type: "object", Value: true, JsonType: "undefined"},
		{Alias: "string-as-object", Type: "object", Value: "service", Formatted: `"service"`, JsonType: "undefined"},
		{Alias: "timestamp-as-object", Type: "object", Value: `2022-12-05T00:30:00Z`, Formatted: `"2022-12-05T00:30:00Z"`, JsonType: "undefined"},
		{Alias: "json-object", Type: "json", Value: `{"answer":42}`, JsonType: "undefined"},
		{Alias: "json-array", Type: "json", Value: `[1,2,"Fizz"]`, JsonType: "undefined"},
		{Alias: "csv", Type: "csv", Value: "foo,bar", Formatted: `"foo,bar"`, JsonType: "string"},
		{Alias: "duration", Type: "duration", Value: "PT0.000515106S", Formatted: `"PT0.000515106S"`, JsonType: "string"},
	}

	for _, c := range cases {
		t.Run(c.Alias, func(t *testing.T) {
			var renderedResp bytes.Buffer
			err := serverResponseTemplate.Execute(&renderedResp, c)
			check := assert.New(t)
			check.NoError(err, "failed to compile template")
			response, err := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(renderedResp.String()))
			check.NoError(err, "failed to parse mocked response")
			var renderedExpectation bytes.Buffer
			err = outputTemplate.Execute(&renderedExpectation, c)
			check.NoError(err, "failed to compile template")

			// when
			output, err := transformForJsonOutput(response)

			// then
			check.NoError(err, "transformation to json output should not generate an error")
			actualOutput := asJson(output)
			check.EqualValues(renderedExpectation.String(), actualOutput, "Formatted data as JSON should match expectation")
		})
	}
}

func asJson(value any) string {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		panic(errors.Wrap(err, "Unexpectedly failed to marshal data to JSON"))
	}
	return string(jsonBytes)
}
