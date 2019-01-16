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
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/git-appraise/commands/output"
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review"
)

var listFlagSet = flag.NewFlagSet("list", flag.ExitOnError)

var (
	listAll        = listFlagSet.Bool("a", false, "List all reviews (not just the open ones).")
	listJSONOutput = listFlagSet.Bool("json", false, "Format the output as JSON")
)

// listReviews lists all extant reviews.
// TODO(ojarjur): Add more flags for filtering the output (e.g. filtering by reviewer or status).
func listReviews(repo repository.Repo, args []string) error {
	listFlagSet.Parse(args)
	var reviews []review.Summary
	if *listAll {
		reviews = review.ListAll(repo)
		if !*listJSONOutput {
			fmt.Printf("Loaded %d reviews:\n", len(reviews))
		}
	} else {
		reviews = review.ListOpen(repo)
		if !*listJSONOutput {
			fmt.Printf("Loaded %d open reviews:\n", len(reviews))
		}
	}
	if *listJSONOutput {
		b, err := json.MarshalIndent(reviews, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	for _, r := range reviews {
		output.PrintSummary(&r)
	}
	return nil
}

// listCmd defines the "list" subcommand.
var listCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s list [<option>...]\n\nOptions:\n", arg0)
		listFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return listReviews(repo, args)
	},
}
