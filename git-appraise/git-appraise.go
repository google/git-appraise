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

// Command git-appraise manages code reviews stored as git-notes in the source repo.
//
// To install, run:
//
//    $ go get github.com/google/git-appraise/git-appraise
//
// And for usage information, run:
//
//    $ git-appraise help
package main

import (
	"fmt"
	"github.com/google/git-appraise/commands"
	"github.com/google/git-appraise/repository"
	"os"
	"sort"
	"strings"
)

const usageMessageTemplate = `Usage: %s <command>

Where <command> is one of:
  %s

For individual command usage, run:
  %s help <command>
`

func usage() {
	command := os.Args[0]
	var subcommands []string
	for subcommand := range commands.CommandMap {
		subcommands = append(subcommands, subcommand)
	}
	sort.Strings(subcommands)
	fmt.Printf(usageMessageTemplate, command, strings.Join(subcommands, "\n  "), command)
}

func help() {
	if len(os.Args) < 3 {
		usage()
		return
	}
	subcommand, ok := commands.CommandMap[os.Args[2]]
	if !ok {
		fmt.Printf("Unknown command %q\n", os.Args[2])
		usage()
		return
	}
	subcommand.Usage(os.Args[0])
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "help" {
		help()
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Unable to get the current working directory: %q\n", err)
		return
	}
	repo, err := repository.NewGitRepo(cwd)
	if err != nil {
		fmt.Printf("%s must be run from within a git repo.\n", os.Args[0])
		return
	}
	if len(os.Args) < 2 {
		subcommand, ok := commands.CommandMap["list"]
		if !ok {
			fmt.Printf("Unable to list reviews")
			return
		}
		subcommand.Run(repo, []string{})
		return
	}
	subcommand, ok := commands.CommandMap[os.Args[1]]
	if !ok {
		fmt.Printf("Unknown command: %q\n", os.Args[1])
		usage()
		return
	}
	if err := subcommand.Run(repo, os.Args[2:]); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
