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
	"time"

	"github.com/pkg/errors"
	"github.com/relvacode/iso8601"
)

// Response represents a parsed UQL response body
type Response struct {
	model       *Model
	mainDataSet *DataSet
	errors      []*Error
	raw         *json.RawMessage
}

// jsonObject is either a JSON object or an array received as in the UQL API response
type jsonObject json.RawMessage

func (u jsonObject) String() string {
	return string(u)
}

func (u jsonObject) MarshalJSON() ([]byte, error) {
	return u, nil
}

// jsonScalar is a single simple value from a field of a JSON object
type jsonScalar any

func (resp *Response) Model() *Model {
	return resp.model
}

func (resp *Response) Main() *DataSet {
	return resp.mainDataSet
}

func (resp *Response) HasErrors() bool {
	return len(resp.errors) > 0
}

func (resp *Response) Errors() []*Error {
	return resp.errors
}

func (resp *Response) Raw() string {
	data, err := resp.raw.MarshalJSON()
	if err != nil {
		panic(errors.Wrap(err, "Failed to serialize data into JSON format that were already parsed from a JSON:"))
	}
	return string(data)
}

// Model represents the structure of the response data
type Model struct {
	Name   string       `json:"name"`
	Fields []ModelField `json:"fields"`
}

// ModelField is a description of one column of one data set
type ModelField struct {
	Alias string `json:"alias"`
	Type  string `json:"type"`
	Form  string `json:"form"`
	Hints *Hint  `json:"hints"`
	Model *Model `json:"model"`
}

type Complex interface {
	Model() *Model
	Values() [][]any
}

// DataSet holds the result data along with its name, structure (Model) and metadata
type DataSet struct {
	Name      string
	DataModel *Model
	Metadata  map[string]any
	Data      [][]any
	Links     map[string]Link
}

func (d DataSet) Model() *Model {
	return d.DataModel
}

func (d DataSet) Values() [][]any {
	return d.Data
}

type Link struct {
	Href string
}

type ComplexData struct {
	DataModel *Model
	Data      [][]any
}

func (c ComplexData) Model() *Model {
	return c.DataModel
}

func (c ComplexData) Values() [][]any {
	return c.Data
}

// DataSetRef is a reference to another data set within the Response
type DataSetRef struct {
	JsonPath string `json:"$jsonPath"`
	Dataset  string `json:"$dataset"`
}

type Error struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func (mf *ModelField) IsReference() bool {
	return mf.Form == "reference"
}

// Hint provides additional information about a ModelField such as the MELT kind, MELT field and type
type Hint struct {
	Kind  string `json:"kind"`
	Field string `json:"field"`
	Type  string `json:"type"`
}

func Errors(errors []*Error) error {
	var messages []string
	for _, e := range errors {
		messages = append(messages, fmt.Sprintf("%s: %s", e.Title, e.Detail))
	}
	return fmt.Errorf(strings.Join(messages, ", "))
}

type DataType interface {
	int | float64 | string | DataSetRef | bool | time.Time | jsonScalar | jsonObject
}

type valueDeserializer[T DataType] func(json.RawMessage) (T, error)

var (
	longDeserializer valueDeserializer[int] = func(raw json.RawMessage) (int, error) {
		var value int
		if err := json.Unmarshal(raw, &value); err != nil {
			return value, err
		}
		return value, nil
	}
	doubleDeserializer valueDeserializer[float64] = func(raw json.RawMessage) (float64, error) {
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil {
			return value, err
		}
		return value, nil
	}
	stringDeserializer valueDeserializer[string] = func(raw json.RawMessage) (string, error) {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return value, err
		}
		return value, nil
	}
	booleanDeserializer valueDeserializer[bool] = func(raw json.RawMessage) (bool, error) {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return value, err
		}
		return value, nil
	}
	timestampDeserializer valueDeserializer[time.Time] = func(raw json.RawMessage) (time.Time, error) {
		var value iso8601.Time
		if err := json.Unmarshal(raw, &value); err != nil {
			return value.Time, err
		}
		return value.Time, nil
	}
	dataSetRefDeserializer valueDeserializer[DataSetRef] = func(raw json.RawMessage) (DataSetRef, error) {
		var value DataSetRef
		if err := json.Unmarshal(raw, &value); err != nil {
			return value, err
		}
		return value, nil
	}
	jsonScalarDeserializer valueDeserializer[jsonScalar] = func(raw json.RawMessage) (jsonScalar, error) {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	}
	jsonObjectDeserializer valueDeserializer[jsonObject] = func(raw json.RawMessage) (jsonObject, error) {
		return jsonObject(raw), nil
	}
)

// complexIsEmpty checks presence of any data in complex data structures.
func complexIsEmpty(data Complex) bool {
	if complexIsNil(data) {
		return true
	}
	// without pointer cast we cannot compare interface with nil.
	switch typed := data.(type) {
	case *DataSet:
		return len(typed.Values()) == 0
	case ComplexData:
		return len(typed.Values()) == 0
	}
	panic("Unexpected type implementing Complex type. This is a bug")
}

// complexIsNil checks if the complex interface is nil or empty value.
func complexIsNil(data Complex) bool {
	// without pointer cast we cannot compare interface with nil.
	switch typed := data.(type) {
	case *DataSet:
		return typed == nil
	case ComplexData:
		return typed.Values() == nil
	}
	panic("Unexpected type implementing Complex type. This is a bug")
}
