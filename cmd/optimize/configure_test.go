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

package optimize

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/test"
)

var validPaths []string = []string{
	"/monitoring/v1/query/execute",
	"/knowledge-store/v1/objects/:optimizer",
	"knowledge-store/v1/objects/:optimizer?filter=data.target.k8sDeployment.workloadId+eq+%22VfJUeLlJOUyRrgi8ABDBMQ%22",
}

var WORKLOAD_UQL_RESP_FILE string = "testdata/configure_test_workload.json"
var REPORT_UQL_RESP_FILE string = "testdata/configure_test_report.json"

func TestConfigureMultipleMatches(t *testing.T) {
	workloadResponse, err := os.ReadFile(WORKLOAD_UQL_RESP_FILE)
	if err != nil {
		t.Fatalf("failed to load test data from %v: %v", WORKLOAD_UQL_RESP_FILE, err)
		return
	}
	reportResponse, err := os.ReadFile(REPORT_UQL_RESP_FILE)
	if err != nil {
		t.Fatalf("failed to load test data from %v: %v", REPORT_UQL_RESP_FILE, err)
		return
	}
	// HTTP server hit by calls made duringtest
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !slices.Contains(validPaths, r.URL.Path) {
			t.Errorf("Unexpected path requeested: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unable to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bodyStr := string(body)
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/knowledge-store/v1/objects/:optimizer" {
			if r.Method == "GET" {
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}
		var respErr error
		if r.Method == "GET" {
			_, respErr = w.Write([]byte(`{"items": [], "total": 0}`))
		} else if strings.Contains(bodyStr, "FETCH id") {
			_, respErr = w.Write([]byte(workloadResponse))
		} else if strings.Contains(bodyStr, "FETCH events") {
			_, respErr = w.Write([]byte(reportResponse))
		} else {
			t.Errorf("unrecognized query: %v", bodyStr)
			w.WriteHeader(http.StatusBadRequest)
		}
		if respErr != nil {
			t.Errorf("error writing response: %v", respErr)
		}
	}))

	defer test.SetActiveConfigProfileServer(testServer.URL)()

	testFlags := &configureFlags{}
	testFlags.Cluster = "optimize-c1-qe"
	testFlags.Namespace = "bofa-24-02"
	testFlags.WorkloadName = "frontend"
	testFlags.create = true
	testFlags.overrideSoftBlockers = true
	createFunc := configureOptimizer(testFlags)
	if err := createFunc(&cobra.Command{}, nil); err != nil {
		t.Fatalf("failed to confiugre multimatch workload: %v", err)
	}
}
