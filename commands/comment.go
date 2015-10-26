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
	"strconv"
)

var commentFlagSet = flag.NewFlagSet("comment", flag.ExitOnError)

var (
	commentMessage = commentFlagSet.String("m", "", "Message to attach to the review")
	parent         = commentFlagSet.String("p", "", "Parent comment")
	lgtm           = commentFlagSet.Bool("lgtm", false, "'Looks Good To Me'. Set this to express your approval. This cannot be combined with nmw")
	nmw            = commentFlagSet.Bool("nmw", false, "'Needs More Work'. Set this to express your disapproval. This cannot be combined with lgtm")
)

// commentOnReview adds a comment to the current code review.
func commentOnReview(repo repository.Repo, args []string) error {
	commentFlagSet.Parse(args)
	args = commentFlagSet.Args()
	if *lgtm && *nmw {
		return errors.New("You cannot combine the flags -lgtm and -nmw.")
	}

	r, err := review.GetCurrent(repo)
	if err != nil {
		return fmt.Errorf("Failed to load the current review: %v\n", err)
	}
	if r == nil {
		return errors.New("There is no current review.")
	}

	commentedUponCommit := repo.GetCommitHash(r.Request.ReviewRef)
	location := comment.Location{
		Commit: commentedUponCommit,
	}
	if len(args) > 0 {
		location.Path = args[0]
		if len(args) > 1 {
			startLine, err := strconv.ParseUint(args[1], 0, 32)
			if err != nil {
				return err
			}
			location.Range = &comment.Range{
				StartLine: uint32(startLine),
			}
		}
	}

	c := comment.New(repo.GetUserEmail(), *commentMessage)
	c.Location = &location
	c.Parent = *parent
	if *lgtm || *nmw {
		resolved := *lgtm
		c.Resolved = &resolved
	}
	return r.AddComment(c)
}

// commentCmd defines the "comment" subcommand.
var commentCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s comment <option>... [<file> [<line>]]\n\nOptions:\n", arg0)
		commentFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return commentOnReview(repo, args)
	},
}
