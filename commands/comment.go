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

var commentFlagSet = flag.NewFlagSet("comment", flag.ExitOnError)
var commentFilename = "APPRAISE_COMMENT_EDITMSG"

var (
	commentMessage = commentFlagSet.String("m", "", "Message to attach to the review")
	commentParent  = commentFlagSet.String("p", "", "Parent comment")
	commentFile    = commentFlagSet.String("f", "", "File being commented upon")
	commentLine    = commentFlagSet.Uint("l", 0, "Line being commented upon; requires that the -f flag also be set")
	commentLgtm    = commentFlagSet.Bool("lgtm", false, "'Looks Good To Me'. Set this to express your approval. This cannot be combined with nmw")
	commentNmw     = commentFlagSet.Bool("nmw", false, "'Needs More Work'. Set this to express your disapproval. This cannot be combined with lgtm")
)

// commentOnReview adds a comment to the current code review.
func commentOnReview(repo repository.Repo, args []string) error {
	commentFlagSet.Parse(args)
	args = commentFlagSet.Args()

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

	if *commentMessage == "" {
		editor, err := repo.GetCoreEditor()
		if err != nil {
			return fmt.Errorf("Unable to detect default git editor: %v\n", err)
		}

		path := fmt.Sprintf("%s/.git/%s", repo.GetPath(), commentFilename)

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
		*commentMessage = string(comment)
		os.Remove(path)
	}
	if *commentLgtm && *commentNmw {
		return errors.New("You cannot combine the flags -lgtm and -nmw.")
	}
	if *commentLine != 0 && *commentFile == "" {
		return errors.New("Specifying a line number with the -l flag requires that you also specify a file name with the -f flag.")
	}

	commentedUponCommit, err := r.GetHeadCommit()
	if err != nil {
		return err
	}
	location := comment.Location{
		Commit: commentedUponCommit,
	}
	if *commentFile != "" {
		location.Path = *commentFile
		if *commentLine != 0 {
			location.Range = &comment.Range{
				StartLine: uint32(*commentLine),
			}
		}
	}

	userEmail, err := repo.GetUserEmail()
	if err != nil {
		return err
	}
	c := comment.New(userEmail, *commentMessage)
	c.Location = &location
	c.Parent = *commentParent
	if *commentLgtm || *commentNmw {
		resolved := *commentLgtm
		c.Resolved = &resolved
	}
	return r.AddComment(c)
}

// commentCmd defines the "comment" subcommand.
var commentCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s comment [<option>...] [<review-hash>]\n\nOptions:\n", arg0)
		commentFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return commentOnReview(repo, args)
	},
}
