/*
Copyright 2018 Google Inc. All rights reserved.

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
	"strings"
	"testing"
)

func TestFindSubcommandBuiltins(t *testing.T) {
	for name, cmd := range CommandMap {
		additionalArg := "foo"
		matchingArgs := append(strings.Split(name, " "), additionalArg)
		subcommand, ok, remainingArgs := FindSubcommand(matchingArgs)
		if !ok {
			t.Errorf("Failed to find the built-in subcommand %q", name)
		} else if subcommand != cmd {
			t.Errorf("Return the wrong subcommand for %q", name)
		} else if len(remainingArgs) != 1 || remainingArgs[0] != additionalArg {
			t.Errorf("Failed to return the remaining arguments for %q", name)
		}
	}
}

func TestFindSubcommandEmpty(t *testing.T) {
	subcommand, ok, remaining := FindSubcommand([]string{})
	if !ok {
		t.Fatalf("Failed to return a default subcommand")
	}
	if subcommand != CommandMap["list"] {
		t.Fatalf("Failed to return `list` as the default subcommand")
	}
	if len(remaining) != 0 {
		t.Fatalf("Unexpected remaining arguments for an empty command: %q", remaining)
	}
}
