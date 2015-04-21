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
	"flag"
	"fmt"
	"log"
	"repository"
	"review/request"
	"strings"
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
	requestMessage          = requestFlagSet.String("m", "", "Message to attach to the review")
	requestReviewers        = requestFlagSet.String("r", "", "Comma-separated list of reviewers")
	requestSource           = requestFlagSet.String("source", "HEAD", "Revision to review")
	requestTarget           = requestFlagSet.String("target", "refs/heads/master", "Revision against which to review")
	requestQuiet            = requestFlagSet.Bool("quiet", false, "Suppress review summary output")
	requestAllowUncommitted = requestFlagSet.Bool("allow-uncommitted", false, "Allow uncommitted local changes.")
)

// Create a new code review request.
//
// The "args" parameter is all of the command line arguments that followed the subcommand.
func requestReview(args []string) {
	requestFlagSet.Parse(args)

	if !*requestAllowUncommitted {
		// Requesting a code review with uncommited local changes is usually a mistake, so
		// we want to report that to the user instead of creating the request.
		if repository.HasUncommittedChanges() {
			fmt.Println("You have uncommitted or untracked files. Use --allow-uncommitted to ignore those.")
			return
		}
	}

	target := *requestTarget
	source := *requestSource
	if source == "HEAD" {
		source = repository.GetHeadRef()
	}

	repository.VerifyGitRefOrDie(target)
	repository.VerifyGitRefOrDie(source)

	reviewCommits := repository.ListCommitsBetween(target, source)
	if reviewCommits == nil {
		log.Fatal("There are no commits included in the review request")
	}

	description := *requestMessage
	if description == "" {
		description = repository.GetCommitMessage(reviewCommits[0])
	}

	reviewers := make([]string, 0)
	if len(*requestReviewers) > 0 {
		reviewers = strings.Split(*requestReviewers, ",")
	}

	r := request.Request{
		Requester:   repository.GetUserEmail(),
		Reviewers:   reviewers,
		ReviewRef:   source,
		TargetRef:   target,
		Description: description,
	}
	note, err := r.Write()
	if err != nil {
		log.Fatal(err)
	}
	repository.AppendNote(request.Ref, reviewCommits[0], note)
	if !*requestQuiet {
		fmt.Printf(requestSummaryTemplate, reviewCommits[0], target, source, description)
	}
}

// requestCmd defines the "request" subcommand.
var requestCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s request <option>...\n\nOptions:\n", arg0)
		requestFlagSet.PrintDefaults()
	},
	RunMethod: func(args []string) {
		requestReview(args)
	},
}
