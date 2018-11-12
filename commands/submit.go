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
	submitMerge       = submitFlagSet.Bool("merge", false, "Create a merge of the source and target refs.")
	submitRebase      = submitFlagSet.Bool("rebase", false, "Rebase the source ref onto the target ref.")
	submitFastForward = submitFlagSet.Bool("fast-forward", false, "Create a merge using the default fast-forward mode.")
	submitTBR         = submitFlagSet.Bool("tbr", false, "(To be reviewed) Force the submission of a review that has not been accepted.")
	submitArchive     = submitFlagSet.Bool("archive", true, "Prevent the original commit from being garbage collected; only affects rebased submits.")

	submitSign = submitFlagSet.Bool("S", false,
		"Sign the contents of the submission")
)

// Submit the current code review request.
//
// The "args" parameter contains all of the command line arguments that followed the subcommand.
func submitReview(repo repository.Repo, args []string) error {
	submitFlagSet.Parse(args)
	args = submitFlagSet.Args()

	if *submitMerge && *submitRebase {
		return errors.New("Only one of --merge or --rebase is allowed.")
	}

	var r *review.Review
	var err error
	if len(args) > 1 {
		return errors.New("Only accepting a single review is supported.")
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

	if r.Submitted {
		return errors.New("The review has already been submitted.")
	}

	if !*submitTBR && (r.Resolved == nil || !*r.Resolved) {
		return errors.New("Not submitting as the review has not yet been accepted.")
	}

	target := r.Request.TargetRef
	if err := repo.VerifyGitRef(target); err != nil {
		return err
	}
	source, err := r.GetHeadCommit()
	if err != nil {
		return err
	}

	isAncestor, err := repo.IsAncestor(target, source)
	if err != nil {
		return err
	}
	if !isAncestor {
		return errors.New("Refusing to submit a non-fast-forward review. First merge the target ref.")
	}

	if !(*submitRebase || *submitMerge || *submitFastForward) {
		submitStrategy, err := repo.GetSubmitStrategy()
		if err != nil {
			return err
		}
		if submitStrategy == "merge" && !*submitRebase && !*submitFastForward {
			*submitMerge = true
		}
		if submitStrategy == "rebase" && !*submitMerge && !*submitFastForward {
			*submitRebase = true
		}
		if submitStrategy == "fast-forward" && !*submitRebase && !*submitMerge {
			*submitFastForward = true
		}
	}

	if *submitRebase {
		var err error
		if *submitSign {
			err = r.RebaseAndSign(*submitArchive)
		} else {
			err = r.Rebase(*submitArchive)
		}
		if err != nil {
			return err
		}

		source, err = r.GetHeadCommit()
		if err != nil {
			return err
		}
	}

	if err := repo.SwitchToRef(target); err != nil {
		return err
	}
	if *submitMerge {
		submitMessage := fmt.Sprintf("Submitting review %.12s", r.Revision)
		if *submitSign {
			return repo.MergeAndSignRef(source, false, submitMessage,
				r.Request.Description)
		} else {
			return repo.MergeRef(source, false, submitMessage,
				r.Request.Description)
		}
	} else {
		if *submitSign {
			return repo.MergeAndSignRef(source, true)
		} else {
			return repo.MergeRef(source, true)
		}
	}
}

// submitCmd defines the "submit" subcommand.
var submitCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s submit [<option>...] [<review-hash>]\n\nOptions:\n", arg0)
		submitFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return submitReview(repo, args)
	},
}
