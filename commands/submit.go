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
	"source.developers.google.com/id/0tH0wAQFren.git/repository"
	"source.developers.google.com/id/0tH0wAQFren.git/review"
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
func submitReview(args []string) {
	submitFlagSet.Parse(args)

	if *submitMerge && *submitRebase {
		fmt.Println("Only one of --merge or --rebase is allowed.")
		return
	}

	r, err := review.GetCurrent()
	if err != nil {
		fmt.Println(err)
		return
	}
	if r == nil {
		fmt.Println("There is nothing to submit")
		return
	}

	if !*submitTBR && (r.Resolved == nil || !*r.Resolved) {
		fmt.Println("Not submitting as the review has not yet been accepted.")
		return
	}

	target := r.Request.TargetRef
	source := r.Request.ReviewRef
	repository.VerifyGitRefOrDie(target)
	repository.VerifyGitRefOrDie(source)

	if !repository.IsAncestor(target, source) {
		fmt.Println("Refusing to submit a non-fast-forward review. First merge the target ref.")
		return
	}

	repository.SwitchToRef(target)
	if *submitMerge {
		repository.MergeRef(source, false)
	} else if *submitRebase {
		repository.RebaseRef(source)
	} else {
		repository.MergeRef(source, true)
	}
}

// submitCmd defines the "submit" subcommand.
var submitCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s submit <option>...\n\nOptions:\n", arg0)
		submitFlagSet.PrintDefaults()
	},
	RunMethod: func(args []string) {
		submitReview(args)
	},
}
