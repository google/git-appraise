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
	"github.com/google/git-appraise/review"
)

var (
	pullFlagSet = flag.NewFlagSet("pull", flag.ExitOnError)
	pullVerify  = pullFlagSet.Bool("verify-signatures", false,
		"verify the signatures of pulled reviews")
	pullIncludeForks = pullFlagSet.Bool("include-forks", true, "Also pull reviews and comments from forks.")
)

// pull updates the local git-notes used for reviews with those from a remote
// repo.
func pull(repo repository.Repo, args []string) error {
	pullFlagSet.Parse(args)
	pullArgs := pullFlagSet.Args()

	if len(pullArgs) > 1 {
		return errors.New("Only pulling from one remote at a time is supported.")
	}

	remote := "origin"
	if len(pullArgs) == 1 {
		remote = pullArgs[0]
	}

	if !*pullVerify && !*pullIncludeForks {
		return repo.PullNotesAndArchive(remote, notesRefPattern, archiveRefPattern)
	}
	if !*pullVerify {
		if _, err := repo.PullNotesForksAndArchive(remote, notesRefPattern, fork.Ref, archiveRefPattern); err != nil {
			return fmt.Errorf("failure pulling review metadata from the remote %q: %v", remote, err)
		}
		return fork.Pull(repo, *pullVerify)
	}

	// We collect the fetched reviewed revisions (their hashes), get
	// their reviews, and then one by one, verify them. If we make it through
	// the set, _then_ we merge the remote reference into the local branch.
	revisions, err := repo.FetchAndReturnNewReviewHashes(remote, notesRefPattern, archiveRefPattern)
	if err != nil {
		return err
	}
	for _, revision := range revisions {
		rvw, err := review.GetSummaryViaRefs(repo,
			"refs/notes/"+remote+"/devtools/reviews",
			"refs/notes/"+remote+"/devtools/discuss", revision)
		if err != nil {
			return err
		}
		err = rvw.Verify()
		if err != nil {
			return err
		}
		fmt.Println("verified review:", revision)
	}
	if err := repo.MergeNotes(remote, notesRefPattern); err != nil {
		return err
	}
	if err := repo.MergeArchives(remote, archiveRefPattern); err != nil {
		return err
	}
	if !*pullIncludeForks {
		return nil
	}
	if err := repo.MergeForks(remote, fork.Ref); err != nil {
		return err
	}
	return fork.Pull(repo, *pullVerify)
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
