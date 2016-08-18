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
	"github.com/google/git-appraise/repository"
)

// push pushes the local git-notes used for reviews to a remote repo.
func push(repo repository.Repo, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported.")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	if err := repo.PushNotesAndArchive(remote, notesRefPattern, archiveRefPattern); err != nil {
		return err
	}
	return nil
}

var pushCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s push [<remote>]\n", arg0)
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return push(repo, args)
	},
}
