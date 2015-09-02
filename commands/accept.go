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
	"github.com/google/git-appraise/review/comment"
)

var acceptFlagSet = flag.NewFlagSet("accept", flag.ExitOnError)

var (
	acceptMessage = acceptFlagSet.String("m", "", "Message to attach to the review")
)

// acceptReview adds an LGTM comment to the current code review.
func acceptReview(args []string) error {
	acceptFlagSet.Parse(args)
	args = acceptFlagSet.Args()

	var r *review.Review
	var err error
	if len(args) > 1 {
		return errors.New("Only accepting a single review is supported.")
	}

	if len(args) == 1 {
		r = review.Get(args[0])
	} else {
		r, err = review.GetCurrent()
	}

	if err != nil {
		return fmt.Errorf("Failed to load the review: %v\n", err)
	}
	if r == nil {
		return errors.New("There is no matching review.")
	}

	var acceptedCommit string
	if r.Submitted {
		acceptedCommit = r.Revision
	} else {
		// TODO(ojarjur): This will fail if the user has not fetched the
		// review ref into their local repo. In that case, we should run
		// ls-remote on each of the remote repos until we find a maching
		// ref, and then use that ref's commit.
		acceptedCommit = repository.GetCommitHash(r.Request.ReviewRef)
	}
	location := comment.Location{
		Commit: acceptedCommit,
	}
	resolved := true
	c := comment.New(*acceptMessage)
	c.Location = &location
	c.Resolved = &resolved
	return r.AddComment(c)
}

// acceptCmd defines the "accept" subcommand.
var acceptCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s accept <option>... (<commit>)\n\nOptions:\n", arg0)
		acceptFlagSet.PrintDefaults()
	},
	RunMethod: func(args []string) error {
		return acceptReview(args)
	},
}
