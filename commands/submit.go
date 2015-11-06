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
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review"
)

var submitFlagSet = flag.NewFlagSet("submit", flag.ExitOnError)

var (
	submitMerge  = submitFlagSet.Bool("merge", false, "Create a merge of the source and target refs.")
	submitRebase = submitFlagSet.Bool("rebase", false, "Rebase the source ref onto the target ref.")
	submitTBR    = submitFlagSet.Bool("tbr", false, "(To be reviewed) Force the submission of a review that has not been accepted.")
)

// Submit the current code review request.
//
// The "args" parameter contains all of the command line arguments that followed the subcommand.
func submitReview(repo repository.Repo, args []string) error {
	submitFlagSet.Parse(args)

	if *submitMerge && *submitRebase {
		return errors.New("Only one of --merge or --rebase is allowed.")
	}

	r, err := review.GetCurrent(repo)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("There is nothing to submit")
	}

	if !*submitTBR && (r.Resolved == nil || !*r.Resolved) {
		return errors.New("Not submitting as the review has not yet been accepted.")
	}

	target := r.Request.TargetRef
	source := r.Request.ReviewRef
	repo.VerifyGitRefOrDie(target)
	repo.VerifyGitRefOrDie(source)

	if !repo.IsAncestor(target, source) {
		return errors.New("Refusing to submit a non-fast-forward review. First merge the target ref.")
	}

	repo.SwitchToRef(target)
	if *submitMerge {
		repo.MergeRef(source, false)
	} else if *submitRebase {
		repo.RebaseRef(source)
	} else {
		repo.MergeRef(source, true)
	}
	return nil
}

// submitCmd defines the "submit" subcommand.
var submitCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s submit [<option>...]\n\nOptions:\n", arg0)
		submitFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return submitReview(repo, args)
	},
}
