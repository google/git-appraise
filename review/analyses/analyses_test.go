/*
Copyright 2015 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package analyses

import (
	"fmt"
	"github.com/google/git-appraise/repository"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	mockOldReport = `{"timestamp": "0", "url": "https://this-url-does-not-exist.test/analysis.json"}`
	mockNewReport = `{"timestamp": "1", "url": "%s"}`
	mockResults   = `{
  "analyze_response": [{
    "note": [{
      "location": {
        "path": "file.txt",
        "range": {
          "start_line": 5
        }
      },
      "category": "test",
      "description": "This is a test"
    }]
  }]
}`
)

func mockHandler(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t.Log(r)
		fmt.Fprintln(w, mockResults)
		w.WriteHeader(http.StatusOK)
	}
}

func TestGetLatestResult(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(mockHandler(t)))
	defer mockServer.Close()

	reports := ParseAllValid([]repository.Note{
		repository.Note([]byte(mockOldReport)),
		repository.Note([]byte(fmt.Sprintf(mockNewReport, mockServer.URL))),
	})

	report, err := GetLatestAnalysesReport(reports)
	if err != nil {
		t.Fatal("Unexpected error while parsing analysis reports", err)
	}
	if report == nil {
		t.Fatal("Unexpected nil report")
	}
	reportResult, err := report.GetLintReportResult()
	if err != nil {
		t.Fatal("Unexpected error while reading the latest report's results", err)
	}
	if len(reportResult) != 1 {
		t.Fatal("Unexpected report result", reportResult)
	}
}
