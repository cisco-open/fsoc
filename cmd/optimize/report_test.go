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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/test"
)

func TestReportsNoEventsData(t *testing.T) {
	reportResponse, err := os.ReadFile("testdata/report_test.json")
	if err != nil {
		t.Fatalf("failed to load test data: %v", err)
		return
	}

	// HTTP server hit by calls made duringtest
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/monitoring/v1/query/execute" {
			t.Errorf("Unexpected path requeested: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(reportResponse)
		if err != nil {
			t.Errorf("error writing response: %v", err)
		}
	}))

	defer test.SetActiveConfigProfileServer(testServer.URL)()

	testFlags := &reportFlags{}
	testFlags.Cluster = "optimize-c1-qe"
	testFlags.Namespace = "bofa-24-02"
	testFlags.WorkloadName = "frontend"
	listFunc := listReports(testFlags)
	if err := listFunc(&cobra.Command{}, nil); err != nil {
		t.Fatalf("failed to confiugre multimatch workload: %v", err)
	}
}
