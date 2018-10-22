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
	"strings"

	"github.com/google/git-appraise/commands/input"
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review/gpg"
	"github.com/google/git-appraise/review/request"
)

// Template for the "request" subcommand's output.
const requestSummaryTemplate = `Review requested:
Commit: %s
Target Ref: %s
Review Ref: %s
Message: "%s"
`

var requestFlagSet = flag.NewFlagSet("request", flag.ExitOnError)

var (
	requestMessageFile      = requestFlagSet.String("F", "", "Take the comment from the given file. Use - to read the message from the standard input")
	requestMessage          = requestFlagSet.String("m", "", "Message to attach to the review")
	requestReviewers        = requestFlagSet.String("r", "", "Comma-separated list of reviewers")
	requestSource           = requestFlagSet.String("source", "HEAD", "Revision to review")
	requestTarget           = requestFlagSet.String("target", "refs/heads/master", "Revision against which to review")
	requestQuiet            = requestFlagSet.Bool("quiet", false, "Suppress review summary output")
	requestAllowUncommitted = requestFlagSet.Bool("allow-uncommitted", false, "Allow uncommitted local changes.")
	requestSign             = requestFlagSet.Bool("S", false,
		"GPG sign the content of the request")
)

// Build the template review request based solely on the parsed flag values.
func buildRequestFromFlags(requester string) (request.Request, error) {
	var reviewers []string
	if len(*requestReviewers) > 0 {
		for _, reviewer := range strings.Split(*requestReviewers, ",") {
			reviewers = append(reviewers, strings.TrimSpace(reviewer))
		}
	}
	if *requestMessageFile != "" && *requestMessage == "" {
		var err error
		*requestMessage, err = input.FromFile(*requestMessageFile)
		if err != nil {
			return request.Request{}, err
		}
	}

	return request.New(requester, reviewers, *requestSource, *requestTarget, *requestMessage), nil
}

// Get the commit at which the review request should be anchored.
func getReviewCommit(repo repository.Repo, r request.Request, args []string) (string, string, error) {
	if len(args) > 1 {
		return "", "", errors.New("Only updating a single review is supported.")
	}
	if len(args) == 1 {
		base, err := repo.MergeBase(r.TargetRef, args[0])
		if err != nil {
			return "", "", err
		}
		return args[0], base, nil
	}

	base, err := repo.MergeBase(r.TargetRef, r.ReviewRef)
	if err != nil {
		return "", "", err
	}
	reviewCommits, err := repo.ListCommitsBetween(base, r.ReviewRef)
	if err != nil {
		return "", "", err
	}
	if reviewCommits == nil {
		return "", "", errors.New("There are no commits included in the review request")
	}
	return reviewCommits[0], base, nil
}

// Create a new code review request.
//
// The "args" parameter is all of the command line arguments that followed the subcommand.
func requestReview(repo repository.Repo, args []string) error {
	requestFlagSet.Parse(args)
	args = requestFlagSet.Args()

	if !*requestAllowUncommitted {
		// Requesting a code review with uncommited local changes is usually a mistake, so
		// we want to report that to the user instead of creating the request.
		hasUncommitted, err := repo.HasUncommittedChanges()
		if err != nil {
			return err
		}
		if hasUncommitted {
			return errors.New("You have uncommitted or untracked files. Use --allow-uncommitted to ignore those.")
		}
	}

	userEmail, err := repo.GetUserEmail()
	if err != nil {
		return err
	}
	r, err := buildRequestFromFlags(userEmail)
	if err != nil {
		return err
	}
	if r.ReviewRef == "HEAD" {
		headRef, err := repo.GetHeadRef()
		if err != nil {
			return err
		}
		r.ReviewRef = headRef
	}
	if err := repo.VerifyGitRef(r.TargetRef); err != nil {
		return err
	}
	if err := repo.VerifyGitRef(r.ReviewRef); err != nil {
		return err
	}

	reviewCommit, baseCommit, err := getReviewCommit(repo, r, args)
	if err != nil {
		return err
	}
	r.BaseCommit = baseCommit
	if r.Description == "" {
		description, err := repo.GetCommitMessage(reviewCommit)
		if err != nil {
			return err
		}
		r.Description = description
	}
	if *requestSign {
		key, err := repo.GetUserSigningKey()
		if err != nil {
			return err
		}
		err = gpg.Sign(key, &r)
		if err != nil {
			return err
		}
	}
	note, err := r.Write()
	if err != nil {
		return err
	}
	repo.AppendNote(request.Ref, reviewCommit, note)
	if !*requestQuiet {
		fmt.Printf(requestSummaryTemplate, reviewCommit, r.TargetRef, r.ReviewRef, r.Description)
	}
	return nil
}

// requestCmd defines the "request" subcommand.
var requestCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s request [<option>...] [<review-hash>]\n\nOptions:\n", arg0)
		requestFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return requestReview(repo, args)
	},
}
