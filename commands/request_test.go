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

package commands

import (
	"testing"
)

func TestBuildRequestFromFlags(t *testing.T) {
	args := []string{"-m", "Request message", "-r", "Me, Myself, \nAnd I "}
	requestFlagSet.Parse(args)
	r, err := buildRequestFromFlags("user@hostname.com")
	if err != nil {
		t.Fatal(err)
	}
	if r.Description != "Request message" {
		t.Fatalf("Unexpected request description: '%s'", r.Description)
	}
	if r.Reviewers == nil || len(r.Reviewers) != 3 || r.Reviewers[0] != "Me" || r.Reviewers[1] != "Myself" || r.Reviewers[2] != "And I" {
		t.Fatalf("Unexpected reviewers list: '%v'", r.Reviewers)
	}
}
