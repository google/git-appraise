/*
Copyright 2016 Google Inc. All rights reserved.

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

	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review"
)

var rebaseFlagSet = flag.NewFlagSet("rebase", flag.ExitOnError)

var (
	rebaseArchive = rebaseFlagSet.Bool("archive", true, "Prevent the original commit from being garbage collected.")
	rebaseSign    = rebaseFlagSet.Bool("S", false,
		"Sign the contents of the request after the rebase")
)

// Validate that the user's request to rebase a review makes sense.
//
// This checks both that the request is well formed, and that the
// corresponding review is in a state where rebasing is appropriate.
func validateRebaseRequest(repo repository.Repo, args []string) (*review.Review, error) {
	var r *review.Review
	var err error
	if len(args) > 1 {
		return nil, errors.New("Only rebasing a single review is supported.")
	}
	if len(args) == 1 {
		r, err = review.Get(repo, args[0])
	} else {
		r, err = review.GetCurrent(repo)
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to load the review: %v\n", err)
	}
	if r == nil {
		return nil, errors.New("There is no matching review.")
	}

	if r.Submitted {
		return nil, errors.New("The review has already been submitted.")
	}

	if r.Request.TargetRef == "" {
		return nil, errors.New("The review was abandoned.")
	}

	target := r.Request.TargetRef
	if err := repo.VerifyGitRef(target); err != nil {
		return nil, err
	}

	return r, nil
}

// Rebase the current code review.
//
// The "args" parameter contains all of the command line arguments that followed the subcommand.
func rebaseReview(repo repository.Repo, args []string) error {
	rebaseFlagSet.Parse(args)
	args = rebaseFlagSet.Args()

	r, err := validateRebaseRequest(repo, args)
	if err != nil {
		return err
	}
	if *rebaseSign {
		return r.RebaseAndSign(*rebaseArchive)
	}
	return r.Rebase(*rebaseArchive)
}

// rebaseCmd defines the "rebase" subcommand.
var rebaseCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s rebase [<option>...] [<review-hash>]\n\nOptions:\n", arg0)
		rebaseFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return rebaseReview(repo, args)
	},
}
