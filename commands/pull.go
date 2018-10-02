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

	"github.com/google/git-appraise/fork"
	"github.com/google/git-appraise/repository"
	"golang.org/x/sync/errgroup"
)

var (
	pullFlagSet      = flag.NewFlagSet("pull", flag.ExitOnError)
	pullIncludeForks = pullFlagSet.Bool("include-forks", true, "Also pull reviews and comments from forks.")
)

// pull updates the local git-notes used for reviews with those from a remote repo.
func pull(repo repository.Repo, args []string) error {
	pullFlagSet.Parse(args)
	args = pullFlagSet.Args()

	if len(args) > 1 {
		return errors.New("Only pulling from one remote at a time is supported.")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	if !*pullIncludeForks {
		return repo.PullNotesAndArchive(remote, notesRefPattern, archiveRefPattern)
	}
	if err := repo.PullNotesForksAndArchive(remote, notesRefPattern, fork.Ref, archiveRefPattern); err != nil {
		return fmt.Errorf("failure pulling review metadata from the remote %q: %v", remote, err)
	}
	forks, err := fork.List(repo)
	if err != nil {
		return fmt.Errorf("failure listing the forks: %v", err)
	}
	var g errgroup.Group
	for _, f := range forks {
		func(f *fork.Fork) {
			g.Go(func() error {
				if err := f.Fetch(repo); err != nil {
					return fmt.Errorf("failure pulling from the fork %q: %v", f.Name, err)
				}
				return nil
			})
		}(f)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	for _, f := range forks {
		if err := f.Merge(repo); err != nil {
			return fmt.Errorf("failure merging from the fork %q: %v", f.Name, err)
		}
	}
	return nil
}

var pullCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s pull [<option>...] [<remote>]\n\nOptions:\n", arg0)
		pullFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return pull(repo, args)
	},
}
