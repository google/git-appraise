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

// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"crypto/sha1"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Note represents the contents of a git-note
type Note []byte

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	return strings.Trim(string(out), "\n"), err
}

func runGitCommandOrDie(args ...string) string {
	out, err := runGitCommand(args...)
	if err != nil {
		log.Print("git", args)
		log.Fatal(out)
	}
	return out
}

// IsGitRepo determines if the current working directory is inside of a git repository.
func IsGitRepo() bool {
	_, err := runGitCommand("rev-parse")
	if err == nil {
		return true
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false
	}
	log.Fatal(err)
	return false
}

// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
func GetRepoStateHash() string {
	stateSummary := runGitCommandOrDie("show-ref")
	return fmt.Sprintf("%x", sha1.Sum([]byte(stateSummary)))
}

// GetUserEmail returns the email address that the user has used to configure git.
func GetUserEmail() string {
	return runGitCommandOrDie("config", "user.email")
}

// HasUncommittedChanges returns true if there are local, uncommitted changes.
func HasUncommittedChanges() bool {
	out := runGitCommandOrDie("status", "--porcelain")
	if len(out) > 0 {
		return true
	}
	return false
}

// VerifyGitRefOrDie verifies that the supplied ref points to a known commit.
func VerifyGitRefOrDie(ref string) {
	runGitCommandOrDie("show-ref", "--verify", ref)
}

// GetHeadRef returns the ref that is the current HEAD.
func GetHeadRef() string {
	return runGitCommandOrDie("symbolic-ref", "HEAD")
}

// GetCommitMessage returns the message stored in the commit pointed to by the given ref.
func GetCommitMessage(ref string) string {
	return runGitCommandOrDie("show", "-s", "--format=%B", ref)
}

// IsAncestor determins if the first argument points to a commit that is an ancestor of the second.
func IsAncestor(ancestor, descendant string) bool {
	_, err := runGitCommand("merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false
	}
	log.Fatal(err)
	return false
}

// ListCommitsBetween returns the list of commits between the two given revisions.
//
// The "from" parameter is the starting point (exclusive), and the "to" parameter
// is the ending point (inclusive). If the commit pointed to by the "from" parameter
// is not an ancestor of the commit pointed to by the "to" parameter, then the
// merge base of the two is used as the starting point.
//
// The generated list is in chronological order (with the oldest commit first).
func ListCommitsBetween(from, to string) []string {
	out := runGitCommandOrDie("rev-list", "--reverse", "--ancestry-path", from+".."+to)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// GetNotes uses the "git" command-line tool to read the notes from the given ref for a given revision.
func GetNotes(notesRef, revision string) []Note {
	var notes []Note
	rawNotes, err := runGitCommand("notes", "--ref", notesRef, "show", revision)
	if err != nil {
		// We just assume that this means there are no notes
		return nil
	}
	for _, line := range strings.Split(rawNotes, "\n") {
		notes = append(notes, Note([]byte(line)))
	}
	return notes
}

// AppendNote appends a note to a revision under the given ref.
func AppendNote(notesRef, revision string, note Note) {
	runGitCommandOrDie("notes", "--ref", notesRef, "append", "-m", string(note), revision)
}

// ListNotedRevisions returns the collection of revisions that are annotated by notes in the given ref.
func ListNotedRevisions(notesRef string) []string {
	var revisions []string
	notesList := strings.Split(runGitCommandOrDie("notes", "--ref", notesRef, "list"), "\n")
	for _, notePair := range notesList {
		noteParts := strings.SplitN(notePair, " ", 2)
		if len(noteParts) == 2 {
			objHash := noteParts[1]
			objType, err := runGitCommand("cat-file", "-t", objHash)
			// If a note points to an object that we do not know about (yet), then err will not
			// be nil. We can safely just ignore those notes.
			if err == nil && objType == "commit" {
				revisions = append(revisions, objHash)
			}
		}
	}
	return revisions
}
