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

// Package commands contains the assorted sub commands supported by the git-appraise tool.
package commands

import (
	"fmt"

	"github.com/google/git-appraise/repository"
)

const notesRefPattern = "refs/notes/devtools/*"
const archiveRefPattern = "refs/devtools/archives/*"
const commentFilename = "APPRAISE_COMMENT_EDITMSG"

// Command represents the definition of a single command.
type Command struct {
	Usage     func(string)
	RunMethod func(repository.Repo, []string) error
}

// Run executes a command, given its arguments.
//
// The args parameter is all of the command line args that followed the
// subcommand.
func (cmd *Command) Run(repo repository.Repo, args []string) error {
	return cmd.RunMethod(repo, args)
}

// CommandMap defines all of the available (sub)commands.
var CommandMap = map[string]*Command{
	"abandon":     abandonCmd,
	"accept":      acceptCmd,
	"comment":     commentCmd,
	"fork add":    addForkCmd,
	"fork list":   listForksCmd,
	"fork remove": removeForkCmd,
	"list":        listCmd,
	"pull":        pullCmd,
	"push":        pushCmd,
	"rebase":      rebaseCmd,
	"reject":      rejectCmd,
	"request":     requestCmd,
	"show":        showCmd,
	"submit":      submitCmd,
}

// FindSubcommand parses the subcommand from the list of arguments.
//
// The args parameter is the list of command line args after the program name.
//
// The return result are the matching command (if found), whether or not the
// command was found, and the list of remaining command line arguments that
// followed the subcommand.
func FindSubcommand(args []string) (*Command, bool, []string) {
	if len(args) < 1 {
		subcommand, ok := CommandMap["list"]
		return subcommand, ok, []string{}
	}
	subcommand, ok := CommandMap[args[0]]
	if ok {
		return subcommand, ok, args[1:]
	}
	if len(args) > 1 {
		subcommand, ok := CommandMap[fmt.Sprintf("%s %s", args[0], args[1])]
		return subcommand, ok, args[2:]
	}
	return nil, false, []string{}
}
