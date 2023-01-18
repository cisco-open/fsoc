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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlatTable_SimplestTable(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": {
      "name": "m:main",
      "fields": [ { "alias": "id", "type": "string", "hints": { "kind": "entity", "field": "id" } } ]
    }
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
    "dataset": "d:main",
    "data": [ [ "apm:database_backend:neDQyzHIO1eOsm0KOvoIYw" ], [ "apm:database_backend:t2G6nwAeP0i/GmbCxAgqxg" ], [ "azure:vm:j5HZ2VavNMi0YmllL0oHNg" ] ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	table := MakeFlatTable(response)
	rendered := table.Render()

	// Then
	expected := ` id                                          
=============================================
 apm:database_backend:neDQyzHIO1eOsm0KOvoIYw 
 apm:database_backend:t2G6nwAeP0i/GmbCxAgqxg 
 azure:vm:j5HZ2VavNMi0YmllL0oHNg             
---------------------------------------------`
	assert.Equal(t, expected, rendered)
}

func TestFlatTable_MultilineCell(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": {
      "name": "m:main",
      "fields": [
        { "alias": "events", "type": "timeseries", "hints": { "kind": "event", "type": "logs:generic_record" }, "form": "reference",
          "model": {
            "name": "m:events",
            "fields": [
              { "alias": "raw", "type": "string", "hints": { "kind": "event", "field": "raw", "type": "logs:generic_record" } },
              { "alias": "entityId", "type": "string", "hints": { "kind": "event", "field": "entityId", "type": "logs:generic_record" } }
            ] }
        }
      ] }
  }, {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
    "dataset": "d:main",
    "data": [ [ { "$dataset": "d:events-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:events-1')]" } ] ]
  }, {
    "type": "data",
    "model": {
      "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:events')]",
      "$model": "m:events"
    },
    "dataset": "d:events-1",
    "data": [ [
        "2023-01-04T14:32:31 ERROR LogExample [main] MultiLine Arth Error : \njava.lang.ArithmeticException\n\tat ExceptionsClass.throwArithmeticException(ExceptionsClass.java:4)\n\tat ExceptionClassD.throwArithmeticException(ExceptionClassD.java:4)\n\tat ExceptionClassC.throwArithmeticException(ExceptionClassC.java:4)\n\tat ExceptionClassB.throwArithmeticException(ExceptionClassB.java:4)\n\tat ExceptionClassA.throwArithmeticException(ExceptionClassA.java:4)\n\tat LogExample.main(LogExample.java:44)",
        "infra:container:nSQcvyuEPkumAipUZTgOJQ"
      ] ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	table := MakeFlatTable(response)
	rendered := table.Render()

	// Then
	expected := ` events                                                                                                           
 raw                                                                     | entityId                               
==================================================================================================================
 2023-01-04T14:32:31 ERROR LogExample [main] MultiLine Arth Error :      | infra:container:nSQcvyuEPkumAipUZTgOJQ 
 java.lang.ArithmeticException                                           |                                        
     at ExceptionsClass.throwArithmeticException(ExceptionsClass.java:4) |                                        
     at ExceptionClassD.throwArithmeticException(ExceptionClassD.java:4) |                                        
     at ExceptionClassC.throwArithmeticException(ExceptionClassC.java:4) |                                        
     at ExceptionClassB.throwArithmeticException(ExceptionClassB.java:4) |                                        
     at ExceptionClassA.throwArithmeticException(ExceptionClassA.java:4) |                                        
     at LogExample.main(LogExample.java:44)                              |                                        
-------------------------------------------------------------------------+----------------------------------------`
	assert.Equal(t, expected, rendered)
}

func TestFlatTable_LongColumnNameExpandsChildren(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": {
      "name": "m:main",
      "fields": [
        { "alias": "metrics", "type": "complex", "hints": { "kind": "metric", "type": "alerting:health.status" }, "form": "reference",
          "model": {
            "name": "m:metrics",
            "fields": [
              { "alias": "source", "type": "string", "hints": { "kind": "metric", "field": "source" } },
              { "alias": "thisNameIsLongerThanSubNames", "type": "timeseries", "hints": { "kind": "metric", "type": "alerting:health.status" }, "form": "inline",
                "model": {
                  "name": "m:metrics_2",
                  "fields": [
                    { "alias": "timestamp", "type": "timestamp", "hints": { "kind": "metric", "field": "timestamp", "type": "alerting:health.status" } },
                    { "alias": "value", "type": "number", "hints": { "kind": "metric", "field": "value", "type": "alerting:health.status" } }
                  ] } }
            ] }
        },
        { "alias": "metrics2", "type": "complex", "hints": { "kind": "metric", "type": "alerting:health.status" }, "form": "reference",
          "model": {
            "name": "m:metrics2",
            "fields": [
              { "alias": "source", "type": "string", "hints": { "kind": "metric", "field": "source" } },
              { "alias": "thisNameIsLongerThanSubNames", "type": "complex", "hints": { "kind": "metric", "type": "alerting:health.status" }, "form": "inline",
                "model": {
                  "name": "m:metrics2_2",
                  "fields": [
                    { "alias": "min", "type": "number", "hints": { "kind": "metric", "field": "value", "type": "alerting:health.status" } },
                    { "alias": "max", "type": "number", "hints": { "kind": "metric", "field": "value", "type": "alerting:health.status" } }
                  ] } }
            ] }
        }
      ] }
  },
  {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]", "$model": "m:main" },
    "dataset": "d:main",
    "data": [ [ { "$dataset": "d:metrics-1", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-1')]" }, { "$dataset": "d:metrics-2", "$jsonPath": "$..[?(@.type == 'data' && @.dataset == 'd:metrics-2')]" } ] ]
  },
  {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics')]", "$model": "m:metrics" },
    "dataset": "d:metrics-1",
    "data": [ [ "alerting", [ [ "2023-01-04T13:25Z", 0.3 ], [ "2023-01-04T13:26Z", 0.6 ], [ "2023-01-04T13:27Z", 0.9 ] ] ] ]
  },
  {
    "type": "data",
    "model": { "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:metrics2')]", "$model": "m:metrics2" },
    "dataset": "d:metrics-2",
    "data": [ [ "alerting", [ [ 0.3, 1000 ] ] ] ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	table := MakeFlatTable(response)
	rendered := table.Render()

	// Then
	expected := ` metrics                                          | metrics2                                
 source   | thisNameIsLongerThanSubNames          | source   | thisNameIsLongerThanSubNames 
          | timestamp                     | value |          | min | max                    
============================================================================================
 alerting | 2023-01-04 13:25:00 +0000 UTC | 0.3   | alerting | 0.3 | 1000                   
          | 2023-01-04 13:26:00 +0000 UTC | 0.6   |----------+-----+------------------------
          | 2023-01-04 13:27:00 +0000 UTC | 0.9   |                                         
----------+-------------------------------+-------+----------+-----+------------------------`
	assert.Equal(t, expected, rendered)
}

func TestFlatTable_EmptyAtomicCell(t *testing.T) {
	// Given
	// language=json
	serverResponse := `[
  {
    "type": "model",
    "model": {
      "name": "m:main",
      "fields": [
        { "alias": "tagA", "type": "string", "hints": {} },
        { "alias": "tagB", "type": "string", "hints": {} }
      ] }
  }, {
    "type": "data",
    "model": {
      "$jsonPath": "$..[?(@.type == 'model')]..[?(@.name == 'm:main')]",
      "$model": "m:main"
    },
    "dataset": "d:main",
    "data": [
      [ "tag[0,0]", null ],
      [ null, "tag[1,1]" ],
      [ "tag[2,0]", "tag[2,1]" ]
    ]
  }
]`
	response, _ := executeUqlQuery(&Query{"ignored"}, ApiVersion1, mockExecuteResponse(serverResponse))

	// When
	table := MakeFlatTable(response)
	rendered := table.Render()

	// Then
	expected := ` tagA     | tagB     
=====================
 tag[0,0] |          
          | tag[1,1] 
 tag[2,0] | tag[2,1] 
----------+----------`
	assert.Equal(t, expected, rendered)
}

func TestFlatTable_EmptyMissingSubTable(t *testing.T) {
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
	table := MakeFlatTable(response)
	rendered := table.Render()

	// Then
	expected := ` id                  | metrics                               
                     | timestamp                     | value 
=============================================================
 entity-with-data    | 2023-01-03 11:16:00 +0000 UTC | 1     
                     | 2023-01-03 11:17:00 +0000 UTC | 1     
---------------------+-------------------------------+-------
 entity-no-data      |                                       
---------------------+-------------------------------+-------
 entity-missing-data |                                       
---------------------+-------------------------------+-------`
	assert.Equal(t, expected, rendered)
}

func TestFlatTable_ComplexFlatTable(t *testing.T) {
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
	table := MakeFlatTable(response)
	rendered := table.Render()

	// Then
	expected := ` id                                     | metrics                                                              | events                                                                                                                        
                                        | source        | metrics                                              | timestamp                         | raw                                                                                       
                                        |               | timestamp                     | value                |                                                                                                                               
===============================================================================================================================================================================================================================================
 infra:container:kLRMaC54NpSEJ9SEoFZuqA | opentelemetry | 2023-01-04 14:35:00 +0000 UTC | 0.03720745700301489  | 2023-01-04 14:37:51.314 +0000 UTC | io.jaegertracing.internal.exceptions.SenderException: Failed to flush spans.              
                                        |               | 2023-01-04 14:36:00 +0000 UTC | 0.01031480793850134  | 2023-01-04 14:37:51.314 +0000 UTC |     at io.jaegertracing.thrift.internal.senders.ThriftSender.flush(ThriftSender.java:115) 
                                        |               | 2023-01-04 14:37:00 +0000 UTC | 0.4074337863009272   |-----------------------------------+-------------------------------------------------------------------------------------------
                                        |               | 2023-01-04 14:38:00 +0000 UTC | 0.057308243138152616 |                                                                                                                               
----------------------------------------+---------------+-------------------------------+----------------------+-----------------------------------+-------------------------------------------------------------------------------------------
 infra:container:nSQcvyuEPkumAipUZTgOJQ |                                                                      | 2023-01-04 14:42:02.115 +0000 UTC | 2023-01-04T14:42:02 ERROR LogExample [main] MultiLine IllArg Error :                      
                                        |                                                                      |                                   | java.lang.IllegalArgumentException                                                        
                                        |                                                                      |                                   |     at ExceptionsClass.throwIllegalArgumentException(ExceptionsClass.java:8)              
                                        |                                                                      |                                   |     at ExceptionClassD.throwIllegalArgumentException(ExceptionClassD.java:9)              
                                        |                                                                      |                                   |     at ExceptionClassC.throwIllegalArgumentException(ExceptionClassC.java:9)              
                                        |                                                                      |                                   |     at ExceptionClassB.throwIllegalArgumentException(ExceptionClassB.java:9)              
                                        |                                                                      |                                   |     at ExceptionClassA.throwIllegalArgumentException(ExceptionClassA.java:9)              
                                        |                                                                      |                                   |     at LogExample.main(LogExample.java:49)                                                
----------------------------------------+---------------+-------------------------------+----------------------+-----------------------------------+-------------------------------------------------------------------------------------------
 infra:container:tu0AV/jvNT6HxPf+4NFwDQ | opentelemetry | 2023-01-04 14:35:00 +0000 UTC | 0.05877454922543631  | 2023-01-04 14:40:28.487 +0000 UTC | 2023-01-04T14:40:32 ERROR LogExample [main] MultiLine Arth Error :                        
                                        |               | 2023-01-04 14:36:00 +0000 UTC | 0.040730745299347976 |                                   | java.lang.ArithmeticException                                                             
                                        |               | 2023-01-04 14:37:00 +0000 UTC | 0.0451670159824317   |                                   |     at ExceptionsClass.throwArithmeticException(ExceptionsClass.java:4)                   
                                        |---------------+-------------------------------+----------------------|                                   |     at ExceptionClassD.throwArithmeticException(ExceptionClassD.java:4)                   
                                        |                                                                      |                                   |     at ExceptionClassC.throwArithmeticException(ExceptionClassC.java:4)                   
                                        |                                                                      |                                   |     at ExceptionClassB.throwArithmeticException(ExceptionClassB.java:4)                   
                                        |                                                                      |                                   |     at ExceptionClassA.throwArithmeticException(ExceptionClassA.java:4)                   
                                        |                                                                      |                                   |     at LogExample.main(LogExample.java:44)                                                
                                        |                                                                      | 2023-01-04 14:40:02.071 +0000 UTC | 2023-01-04T14:40:02 ERROR LogExample [main] This is an Error message for Levitate         
----------------------------------------+---------------+-------------------------------+----------------------+-----------------------------------+-------------------------------------------------------------------------------------------`
	assert.Equal(t, expected, rendered)
}
