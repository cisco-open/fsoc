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

package solution

type SolutionTestObjects struct {
	Tests         []SolutionTestObject `json:"tests"`
	InitialDelay  int                  `json:"initialDelay,omitempty"`
	MaxRetryCount int                  `json:"retryCount,omitempty"`
	RetryDelay    int                  `json:"retryDelay,omitempty"`
}

type SolutionTestObject struct {
	Name        string                  `json:"name,omitempty"`
	Type        string                  `json:"type,omitempty"`
	Description string                  `json:"description,omitempty"`
	Setup       SolutionTestSetup       `json:"setup"`
	Assertions  []SolutionTestAssertion `json:"assertions"`
}

type SolutionTestSetup struct {
	Type     string      `json:"type"`
	Input    interface{} `json:"input,omitempty"`
	Location string      `json:"location,omitempty"`
}

type SolutionTestAssertion struct {
	UQL        string                           `json:"uql"`
	Transforms []SolutionTestAssertionTransform `json:"transforms"`
}

type SolutionTestAssertionTransform struct {
	Type       string `json:"type"`
	Expression string `json:"expression,omitempty"`
	Message    string `json:"message,omitempty"`
	Location   string `json:"location,omitempty"`
}

type SolutionTestResult struct {
	ID string `json:"testRunId"`
}

type SolutionTestStatusResult struct {
	Complete       bool            `json:"completed"`
	Status         string          `json:"status"`
	StatusMessages []StatusMessage `json:"statusMessages"`
}

type StatusMessage struct {
	Timestamp string   `json:"timestamp"`
	Message   string   `json:"message,omitempty"`
	Statuses  []string `json:"statuses,omitempty"`
}
