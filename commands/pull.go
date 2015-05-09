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
	"fmt"
	"source.developers.google.com/id/0tH0wAQFren.git/repository"
	"source.developers.google.com/id/0tH0wAQFren.git/review/comment"
	"source.developers.google.com/id/0tH0wAQFren.git/review/request"
)

// pull updates the local git-notes used for reviews with those from a remote repo.
func pull(args []string) error {
	if len(args) > 1 {
		return errors.New("Only pulling from one remote at a time is supported.")
	}

	remote := "origin"
	if args != nil {
		remote = args[0]
	}

	repository.PullNotes(remote, request.Ref)
	repository.PullNotes(remote, comment.Ref)
	return nil
}

var pullCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s pull [<remote>]", arg0)
	},
	RunMethod: func(args []string) error {
		return pull(args)
	},
}
