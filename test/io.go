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

package test

import (
	"fmt"
	"io"
	"os"
	"testing"
)

// CaptureConsoleOutput - captures the console output as a string and return
// useful in test cases
func CaptureConsoleOutput(f func(), t *testing.T) string {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout
	return string(out)
}

// ReadFileToString - Read a file and return contents as string
func ReadFileToString(path string) (string, error) {
	dat, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return string(dat), nil
}
