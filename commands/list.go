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
	"source.developers.google.com/id/0tH0wAQFren.git/review"
)

// listReviews lists all extant reviews.
// TODO(ojarjur): Add flags for filtering the output (e.g. to just open reviews).
func listReviews(args []string) {
	reviews := review.ListAll()
	fmt.Printf("Loaded %d reviews:\n", len(reviews))
	for _, review := range review.ListAll() {
		review.PrintSummary()
	}
}

// listCmd defines the "list" subcommand.
var listCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s list\n", arg0)
	},
	RunMethod: func(args []string) {
		listReviews(args)
	},
}
