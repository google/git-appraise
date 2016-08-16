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
	"github.com/google/git-appraise/review/request"
)

var rebaseFlagSet = flag.NewFlagSet("rebase", flag.ExitOnError)

var (
	rebaseArchive = rebaseFlagSet.Bool("archive", true, "Prevent the original commit from being garbage collected.")
)

// Rebase the current code review.
//
// The "args" parameter contains all of the command line arguments that followed the subcommand.
func rebaseReview(repo repository.Repo, args []string) error {
	rebaseFlagSet.Parse(args)
	args = rebaseFlagSet.Args()

	var r *review.Review
	var err error
	if len(args) > 1 {
		return errors.New("Only rebasing a single review is supported.")
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

	target := r.Request.TargetRef
	if err := repo.VerifyGitRef(target); err != nil {
		return err
	}
	source, err := r.GetHeadCommit()
	if err != nil {
		return err
	}
	if *rebaseArchive {
		if err := repo.ArchiveRef(source, archiveRef); err != nil {
			return err
		}
	}
	if err := repo.RebaseRef(target); err != nil {
		return err
	}
	source, err = r.GetHeadCommit()
	if err != nil {
		return err
	}

	r.Request.Alias = source
	newNote, err := r.Request.Write()
	if err != nil {
		return err
	}
	repo.AppendNote(request.Ref, r.Revision, newNote)
	return nil
}

// rebaseCmd defines the "rebase" subcommand.
var rebaseCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s rebase [<option>...]\n\nOptions:\n", arg0)
		rebaseFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return rebaseReview(repo, args)
	},
}
