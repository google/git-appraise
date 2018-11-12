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

	"github.com/google/git-appraise/commands/input"
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review"
	"github.com/google/git-appraise/review/comment"
	"github.com/google/git-appraise/review/gpg"
	"github.com/google/git-appraise/review/request"
)

var abandonFlagSet = flag.NewFlagSet("abandon", flag.ExitOnError)

var (
	abandonMessageFile = abandonFlagSet.String("F", "", "Take the comment from the given file. Use - to read the message from the standard input")
	abandonMessage     = abandonFlagSet.String("m", "", "Message to attach to the review")

	abandonSign = abandonFlagSet.Bool("S", false,
		"Sign the contents of the abandonment")
)

// abandonReview adds an NMW comment to the current code review.
func abandonReview(repo repository.Repo, args []string) error {
	abandonFlagSet.Parse(args)
	args = abandonFlagSet.Args()

	var r *review.Review
	var err error
	if len(args) > 1 {
		return errors.New("Only abandon a single review is supported.")
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

	if *abandonMessageFile != "" && *abandonMessage == "" {
		*abandonMessage, err = input.FromFile(*abandonMessageFile)
		if err != nil {
			return err
		}
	}
	if *abandonMessageFile == "" && *abandonMessage == "" {
		*abandonMessage, err = input.LaunchEditor(repo, commentFilename)
		if err != nil {
			return err
		}
	}

	abandonedCommit, err := r.GetHeadCommit()
	if err != nil {
		return err
	}
	location := comment.Location{
		Commit: abandonedCommit,
	}
	resolved := false
	userEmail, err := repo.GetUserEmail()
	if err != nil {
		return err
	}
	c := comment.New(userEmail, *abandonMessage)
	c.Location = &location
	c.Resolved = &resolved

	var key string
	if *abandonSign {
		key, err := repo.GetUserSigningKey()
		if err != nil {
			return err
		}
		err = gpg.Sign(key, &c)
		if err != nil {
			return err
		}
	}

	err = r.AddComment(c)
	if err != nil {
		return err
	}

	// Empty target ref indicates that request was abandoned
	r.Request.TargetRef = ""
	// (re)sign the request after clearing out `TargetRef'.
	if *abandonSign {
		err = gpg.Sign(key, &r.Request)
		if err != nil {
			return err
		}
	}

	note, err := r.Request.Write()
	if err != nil {
		return err
	}

	return repo.AppendNote(request.Ref, r.Revision, note)
}

// abandonCmd defines the "abandon" subcommand.
var abandonCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s abandon [<option>...] [<commit>]\n\nOptions:\n", arg0)
		abandonFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return abandonReview(repo, args)
	},
}
