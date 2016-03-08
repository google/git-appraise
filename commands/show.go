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
	"flag"
	"fmt"
	"github.com/google/git-appraise/commands/output"
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review"
	"strings"
)

var showFlagSet = flag.NewFlagSet("show", flag.ExitOnError)

var (
	showJSONOutput  = showFlagSet.Bool("json", false, "Format the output as JSON")
	showDiffOutput  = showFlagSet.Bool("diff", false, "Show the current diff for the review")
	showDiffOptions = showFlagSet.String("diff-opts", "", "Options to pass to the diff tool; can only be used with the --diff option")
)

// showReview prints the current code review.
func showReview(repo repository.Repo, args []string) error {
	showFlagSet.Parse(args)
	args = showFlagSet.Args()
	if *showDiffOptions != "" && !*showDiffOutput {
		return errors.New("The --diff-opts flag can only be used if the --diff flag is set.")
	}

	var r *review.Review
	var err error
	if len(args) > 1 {
		return errors.New("Only showing a single review is supported.")
	}

	if len(args) == 1 {
		r, err = review.Get(repo, args[0])
	} else {
		r, err = review.GetCurrent(repo)
	}

	if err != nil {
		return fmt.Errorf("Failed to load the review: %v\n", err)
	}
	if r == nil {
		return errors.New("There is no matching review.")
	}
	if *showJSONOutput {
		return output.PrintJSON(r)
	}
	if *showDiffOutput {
		var diffArgs []string
		if *showDiffOptions != "" {
			diffArgs = strings.Split(*showDiffOptions, ",")
		}
		return output.PrintDiff(r, diffArgs...)
	}
	return output.PrintDetails(r)
}

// showCmd defines the "show" subcommand.
var showCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s show [<option>...] [<commit>]\n\nOptions:\n", arg0)
		showFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return showReview(repo, args)
	},
}
