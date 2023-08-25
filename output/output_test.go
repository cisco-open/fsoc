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

package output

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cisco-open/fsoc/test"
)

type testStruct struct {
	Field1 string
	Field2 uint
	Field3 bool
}

func TestPrintJSONAndYaml(t *testing.T) {

	obj := testStruct{
		Field1: "hello",
		Field2: 100,
		Field3: true,
	}

	tests := []struct {
		format  string
		fixture string
	}{
		{format: "json", fixture: "./fixtures/output_json.txt"},
		{format: "yaml", fixture: "./fixtures/output_yaml.txt"},
	}

	for _, tt := range tests {
		pr := &PrintRequest{Format: tt.format}
		outExpected, err := test.ReadFileToString(tt.fixture)
		require.Nil(t, err)
		outActual := test.CaptureConsoleOutput(func() { printCmdOutputCustom(pr, obj) }, t)
		require.Equal(t, outExpected, outActual)
	}
}

func TestPrintSimple(t *testing.T) {
	pr := &PrintRequest{Format: ""}

	// simple
	outExpected, err := test.ReadFileToString("./fixtures/output_text.txt")
	require.Nil(t, err)
	outActual := test.CaptureConsoleOutput(func() { printCmdOutputCustom(pr, "test string") }, t)
	require.Equal(t, outExpected, outActual)
}

func TestPrintCmdStatus(t *testing.T) {
	// simple
	outExpected := "test string"
	outActual := test.CaptureConsoleOutput(func() { PrintCmdStatus(nil, "test string") }, t)
	require.Equal(t, outExpected, outActual)
}

func TestPrintTable(t *testing.T) {
	table := &Table{
		Headers: []string{"Field1", "Field2", "Field3"},
	}
	for i := 1; i <= 5; i++ {
		rowString := []string{fmt.Sprintf("Row%d-Field1", i), fmt.Sprintf("%d", i), strconv.FormatBool(true)}
		table.Lines = append(table.Lines, rowString)
	}

	pr := &PrintRequest{Format: "", Table: table}
	outExpected, err := test.ReadFileToString("./fixtures/output_table.txt")
	require.Nil(t, err)
	outActual := test.CaptureConsoleOutput(func() { printCmdOutputCustom(pr, nil) }, t)
	require.Equal(t, outExpected, outActual)
}
