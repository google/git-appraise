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
	"fmt"
	"review"
)

// showReview prints the current code review.
//
// The "args" parameter contains all of the command line arguments that followed the subcommand.
func showReview(args []string) {
	r, err := review.GetCurrent()
	if err != nil {
		fmt.Printf("Failed to load the current review: %v\n", err)
		return
	}
	if r == nil {
		fmt.Println("There is no current review.")
		return
	}
	r.PrintDetails()
}

// showCmd defines the "show" subcommand.
var showCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s show\n", arg0)
	},
	RunMethod: func(args []string) {
		showReview(args)
	},
}
