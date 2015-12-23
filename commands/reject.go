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
	"io/ioutil"
	"os"
	"os/exec"
)

var rejectFlagSet = flag.NewFlagSet("reject", flag.ExitOnError)
var rejectFilename = "APPRAISE_COMMENT_EDITMSG"

var (
	rejectMessage = rejectFlagSet.String("m", "", "Message to attach to the review")
)

// rejectReview adds an NMW comment to the current code review.
func rejectReview(repo repository.Repo, args []string) error {
	rejectFlagSet.Parse(args)
	args = rejectFlagSet.Args()

	var r *review.Review
	var err error
	if len(args) > 1 {
		return errors.New("Only rejecting a single review is supported.")
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

	if *rejectMessage == "" {
		editor, err := repo.GetCoreEditor()
		if err != nil {
			return fmt.Errorf("Unable to detect default git editor: %v\n", err)
		}

		path := fmt.Sprintf("%s/.git/%s", repo.GetPath(), rejectFilename)

		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("Unable to start editor: %v\n", err)
		}

		err = cmd.Wait()
		if err != nil {
			return fmt.Errorf("Editing finished with error: %v\n", err)
		}

		comment, err := ioutil.ReadFile(path)
		if err != nil {
			os.Remove(path)
			return fmt.Errorf("Error reading comment file: %v\n", err)
		}
		*rejectMessage = string(comment)
		os.Remove(path)
	}

	rejectedCommit, err := r.GetHeadCommit()
	if err != nil {
		return err
	}
	location := comment.Location{
		Commit: rejectedCommit,
	}
	resolved := false
	userEmail, err := repo.GetUserEmail()
	if err != nil {
		return err
	}
	c := comment.New(userEmail, *rejectMessage)
	c.Location = &location
	c.Resolved = &resolved
	return r.AddComment(c)
}

// rejectCmd defines the "reject" subcommand.
var rejectCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s reject [<option>...] [<commit>]\n\nOptions:\n", arg0)
		rejectFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return rejectReview(repo, args)
	},
}
