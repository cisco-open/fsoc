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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests whether the model of the main dataset is returned when calling Model()
func TestUqlResponse_Model_HappyDay(t *testing.T) {
	// given

	eventModel := model("m:events-1", timestampField("timestamp"), stringField("raw"))
	mainModel := model("m:main", longField("count"), timeSeriesField("events", eventModel, nil))

	eventsDataSet := &DataSet{
		Name:      "d:events-1",
		DataModel: eventModel,
		Data: [][]any{
			{time.Time{}, "INFO hello world"},
		},
	}
	response := Response{
		model: mainModel,
		mainDataSet: &DataSet{
			Name:      "d:main",
			DataModel: mainModel,
			Data: [][]any{
				{10, eventsDataSet},
			},
		},
	}

	// when
	actualModel := response.Model()

	// then
	assert.Equal(t, response.Main().Model(), actualModel, "response.Model() does not return the main dataset model")
}

// Tests that that calling Model() works even if the response contains errors
func TestUqlResponse_Model_ErrorResponse(t *testing.T) {
	// given
	mainModel := model("m:main", longField("count"))
	response := Response{
		model:       mainModel,
		mainDataSet: nil,
		errors: []*Error{
			{Type: "does not matter", Title: "does not matter", Detail: "does not matter"},
		},
	}

	// when
	actualModel := response.Model()

	// then
	assert.Equal(t, mainModel, actualModel, "response.Model() does not return the main dataset model")
}

// Tests that Main() returns the correct dataset i.e. "d:main"
func TestUqlResponse_Main(t *testing.T) {
	// given
	eventModel := model("m:events-1", timestampField("timestamp"), stringField("raw"))
	mainModel := model("m:main", longField("count"), timeSeriesField("events", eventModel, nil))
	eventDataSet := &DataSet{
		Name:      "d:events-1",
		DataModel: eventModel,
		Data: [][]any{
			{time.Time{}, "INFO hello world"},
		},
	}
	mainDataSet := &DataSet{
		Name:      "d:main",
		DataModel: mainModel,
		Data: [][]any{
			{10, eventDataSet},
		},
	}

	response := Response{
		model:       mainModel,
		mainDataSet: mainDataSet,
	}

	// when
	actualMainDataSet := response.Main()

	// then
	assert.Equal(t, mainDataSet, actualMainDataSet, "response.Main() does not return the main dataset")
}

// Tests that HasErrors() returns true if there are any errors
func TestUqlResponse_HasErrors(t *testing.T) {
	// given
	response := Response{
		model:       model("m:main", longField("count")),
		mainDataSet: nil,
		errors: []*Error{
			{Type: "does not matter", Title: "does not matter", Detail: "does not matter"},
		},
	}

	// when
	hasErrors := response.HasErrors()

	// then
	assert.True(t, hasErrors, "response.HasErrors() is not true for response containing errors")
}

func numberField(alias string) ModelField {
	return numberFieldH(alias, &Hint{})
}

func numberFieldH(alias string, hints *Hint) ModelField {
	return inlineField(alias, "number", hints)
}

func longField(alias string) ModelField {
	return longFieldH(alias, &Hint{})
}

func longFieldH(alias string, hints *Hint) ModelField {
	return inlineField(alias, "long", hints)
}

func timestampField(alias string) ModelField {
	return timestampFieldH(alias, &Hint{})
}

func timestampFieldH(alias string, hints *Hint) ModelField {
	return inlineField(alias, "timestamp", hints)
}

func stringField(alias string) ModelField {
	return stringFieldH(alias, &Hint{})
}

func stringFieldH(alias string, hints *Hint) ModelField {
	return inlineField(alias, "string", hints)
}

func timeSeriesField(alias string, model *Model, hints *Hint) ModelField {
	return referenceField(alias, "timeseries", model, hints)
}

func inlineField(alias, fieldType string, hints *Hint) ModelField {
	return modelField(alias, fieldType, "", hints, nil)
}

func referenceField(alias, fieldType string, model *Model, hints *Hint) ModelField {
	return modelField(alias, fieldType, "reference", hints, model)
}

func modelField(alias, fieldType, form string, hints *Hint, model *Model) ModelField {
	return ModelField{
		Alias: alias,
		Type:  fieldType,
		Form:  form,
		Hints: hints,
		Model: model,
	}
}

func model(name string, fields ...ModelField) *Model {
	return &Model{
		Name:   name,
		Fields: fields,
	}
}
