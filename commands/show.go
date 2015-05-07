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
	"errors"
	"fmt"
	"source.developers.google.com/id/0tH0wAQFren.git/review"
)

// showReview prints the current code review.
func showReview() error {
	r, err := review.GetCurrent()
	if err != nil {
		return fmt.Errorf("Failed to load the current review: %v\n", err)
	}
	if r == nil {
		return errors.New("There is no current review.")
	}
	r.PrintDetails()
	return nil
}

// showCmd defines the "show" subcommand.
var showCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s show\n", arg0)
	},
	RunMethod: func(args []string) error {
		return showReview()
	},
}
