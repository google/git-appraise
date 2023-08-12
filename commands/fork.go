/*
Copyright 2018 Google Inc. All rights reserved.

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
	"strings"

	"github.com/google/git-appraise/commands/output"
	"github.com/google/git-appraise/fork"
	"github.com/google/git-appraise/repository"
)

var (
	addForkFlagSet = flag.NewFlagSet("fork add", flag.ExitOnError)
	addForkOwners  = addForkFlagSet.String("o", "", "Comma-separated list of owner email addresses")
)

// addFork updates the local git repository to include the specified fork.
func addFork(repo repository.Repo, args []string) error {
	addForkFlagSet.Parse(args)
	args = addForkFlagSet.Args()

	var owners []string
	if len(*addForkOwners) > 0 {
		for _, owner := range strings.Split(*addForkOwners, ",") {
			owners = append(owners, strings.TrimSpace(owner))
		}
	}
	if len(args) < 2 {
		return errors.New("The name and URL of the fork must be specified.")
	}
	if len(args) > 2 {
		return errors.New("Only the name and URL of the fork may be specified.")
	}
	if len(owners) == 0 {
		return errors.New("You must specify at least one owner.")
	}
	name := args[0]
	url := args[1]
	return fork.Add(repo, fork.New(name, url, owners))
}

// listForks lists the forks registered in the local git repository.
func listForks(repo repository.Repo, args []string) error {
	forks, err := fork.List(repo)
	if err != nil {
		return err
	}
	output.PrintForks(forks)
	return nil
}

// removeFork updates the local git repository to no longer include the specified fork.
func removeFork(repo repository.Repo, args []string) error {
	if len(args) < 1 {
		return errors.New("The name of the fork must be specified.")
	}
	if len(args) > 1 {
		return errors.New("Only the name of the fork may be specified.")
	}
	name := args[0]
	return fork.Delete(repo, name)
}

// addForkCmd defines the `fork add` command.
var addForkCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s fork add [<option>...] <name> <url>\n\nOptions:\n", arg0)
		addForkFlagSet.PrintDefaults()
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return addFork(repo, args)
	},
}

// listForksCmd defines the `fork add` command.
var listForksCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s fork list\n", arg0)
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return listForks(repo, args)
	},
}

// removeForkCmd defines the `fork add` command.
var removeForkCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s fork remove <name>\n", arg0)
	},
	RunMethod: func(repo repository.Repo, args []string) error {
		return removeFork(repo, args)
	},
}
