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
	"os"
	"os/exec"
	"strings"
)

const branchRefPrefix = "refs/heads/"

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	Path string
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo.Path
	out, err := cmd.Output()
	return strings.Trim(string(out), "\n"), err
}

// Run the given git command using the same stdin, stdout, and stderr as the review tool.
func (repo *GitRepo) runGitCommandInline(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo.Path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run the given git command using the same stdin, stdout, and stderr as the review tool.
func (repo *GitRepo) runGitCommandInlineOrDie(args ...string) {
	err := repo.runGitCommandInline(args...)
	if err != nil {
		log.Print("git", args)
		log.Fatal(err)
	}
}

// Run the given git command and return its stdout.
func (repo *GitRepo) runGitCommandOrDie(args ...string) string {
	out, err := repo.runGitCommand(args...)
	if err != nil {
		log.Print("git", args)
		log.Fatal(out)
	}
	return out
}

// NewGitRepo determines if the given working directory is inside of a git repository,
// and returns the corresponding GitRepo instance if it is.
func NewGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{Path: path}
	_, err := repo.runGitCommand("rev-parse")
	if err == nil {
		return repo, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return nil, err
	}
	log.Fatal(err)
	return nil, err
}

// GetPath returns the path to the repo.
func (repo *GitRepo) GetPath() string {
	return repo.Path
}

// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
func (repo *GitRepo) GetRepoStateHash() string {
	stateSummary := repo.runGitCommandOrDie("show-ref")
	return fmt.Sprintf("%x", sha1.Sum([]byte(stateSummary)))
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GitRepo) GetUserEmail() string {
	return repo.runGitCommandOrDie("config", "user.email")
}

// HasUncommittedChanges returns true if there are local, uncommitted changes.
func (repo *GitRepo) HasUncommittedChanges() bool {
	out := repo.runGitCommandOrDie("status", "--porcelain")
	if len(out) > 0 {
		return true
	}
	return false
}

// VerifyGitRef verifies that the supplied ref points to a known commit.
func (repo *GitRepo) VerifyGitRef(ref string) error {
	_, err := repo.runGitCommand("show-ref", "--verify", ref)
	return err
}

// VerifyGitRefOrDie verifies that the supplied ref points to a known commit.
func (repo *GitRepo) VerifyGitRefOrDie(ref string) {
	repo.runGitCommandOrDie("show-ref", "--verify", ref)
}

// GetHeadRef returns the ref that is the current HEAD.
func (repo *GitRepo) GetHeadRef() string {
	return repo.runGitCommandOrDie("symbolic-ref", "HEAD")
}

// GetCommitHash returns the hash of the commit pointed to by the given ref.
func (repo *GitRepo) GetCommitHash(ref string) string {
	return repo.runGitCommandOrDie("show", "-s", "--format=%H", ref)
}

// ResolveRefCommit returns the commit pointed to by the given ref, which may be a remote ref.
//
// This differs from GetCommitHash which only works on exact matches, in that it will try to
// intelligently handle the scenario of a ref not existing locally, but being known to exist
// in a remote repo.
//
// This method should be used when a command may be performed by either the reviewer or the
// reviewee, while GetCommitHash should be used when the encompassing command should only be
// performed by the reviewee.
func (repo *GitRepo) ResolveRefCommit(ref string) (string, error) {
	if err := repo.VerifyGitRef(ref); err == nil {
		return repo.GetCommitHash(ref), nil
	}
	if strings.HasPrefix(ref, "refs/heads/") {
		// The ref is a branch. Check if it exists in exactly one remote
		pattern := strings.Replace(ref, "refs/heads", "**", 1)
		matchingOutput := repo.runGitCommandOrDie("for-each-ref", "--format=%(refname)", pattern)
		matchingRefs := strings.Split(matchingOutput, "\n")
		if len(matchingRefs) == 1 && matchingRefs[0] != "" {
			// There is exactly one match
			return repo.GetCommitHash(matchingRefs[0]), nil
		}
		return "", fmt.Errorf("Unable to find a git ref matching the pattern %q", pattern)
	}
	return "", fmt.Errorf("Unknown git ref %q", ref)
}

// GetCommitMessage returns the message stored in the commit pointed to by the given ref.
func (repo *GitRepo) GetCommitMessage(ref string) string {
	return repo.runGitCommandOrDie("show", "-s", "--format=%B", ref)
}

// GetCommitTime returns the commit time of the commit pointed to by the given ref.
func (repo *GitRepo) GetCommitTime(ref string) string {
	return repo.runGitCommandOrDie("show", "-s", "--format=%ct", ref)
}

// GetLastParent returns the last parent of the given commit (as ordered by git).
func (repo *GitRepo) GetLastParent(ref string) (string, error) {
	return repo.runGitCommand("rev-list", "--skip", "1", "-n", "1", ref)
}

// MergeBase determines if the first commit that is an ancestor of the two arguments.
func (repo *GitRepo) MergeBase(a, b string) string {
	return repo.runGitCommandOrDie("merge-base", a, b)
}

// IsAncestor determines if the first argument points to a commit that is an ancestor of the second.
func (repo *GitRepo) IsAncestor(ancestor, descendant string) bool {
	_, err := repo.runGitCommand("merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false
	}
	log.Fatal(err)
	return false
}

// Diff computes the diff between two given commits.
func (repo *GitRepo) Diff(left, right string, diffArgs ...string) string {
	return repo.runGitCommandOrDie("diff", left, right)
}

// SwitchToRef changes the currently-checked-out ref.
func (repo *GitRepo) SwitchToRef(ref string) {
	// If the ref starts with "refs/heads/", then we have to trim that prefix,
	// or else we will wind up in a detached HEAD state.
	if strings.HasPrefix(ref, branchRefPrefix) {
		ref = ref[len(branchRefPrefix):]
	}
	repo.runGitCommandOrDie("checkout", ref)
}

// MergeRef merges the given ref into the current one.
//
// The ref argument is the ref to merge, and fastForward indicates that the
// current ref should only move forward, as opposed to creating a bubble merge.
func (repo *GitRepo) MergeRef(ref string, fastForward bool) {
	args := []string{"merge"}
	if fastForward {
		args = append(args, "--ff", "--ff-only")
	} else {
		args = append(args, "--no-ff")
	}
	args = append(args, ref)
	repo.runGitCommandInlineOrDie(args...)
}

// RebaseRef rebases the given ref into the current one.
func (repo *GitRepo) RebaseRef(ref string) {
	repo.runGitCommandInlineOrDie("rebase", "-i", ref)
}

// ListCommitsBetween returns the list of commits between the two given revisions.
//
// The "from" parameter is the starting point (exclusive), and the "to" parameter
// is the ending point (inclusive). If the commit pointed to by the "from" parameter
// is not an ancestor of the commit pointed to by the "to" parameter, then the
// merge base of the two is used as the starting point.
//
// The generated list is in chronological order (with the oldest commit first).
func (repo *GitRepo) ListCommitsBetween(from, to string) []string {
	out := repo.runGitCommandOrDie("rev-list", "--reverse", "--ancestry-path", from+".."+to)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// GetNotes uses the "git" command-line tool to read the notes from the given ref for a given revision.
func (repo *GitRepo) GetNotes(notesRef, revision string) []Note {
	var notes []Note
	rawNotes, err := repo.runGitCommand("notes", "--ref", notesRef, "show", revision)
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
func (repo *GitRepo) AppendNote(notesRef, revision string, note Note) {
	repo.runGitCommandOrDie("notes", "--ref", notesRef, "append", "-m", string(note), revision)
}

// ListNotedRevisions returns the collection of revisions that are annotated by notes in the given ref.
func (repo *GitRepo) ListNotedRevisions(notesRef string) []string {
	var revisions []string
	notesList := strings.Split(repo.runGitCommandOrDie("notes", "--ref", notesRef, "list"), "\n")
	for _, notePair := range notesList {
		noteParts := strings.SplitN(notePair, " ", 2)
		if len(noteParts) == 2 {
			objHash := noteParts[1]
			objType, err := repo.runGitCommand("cat-file", "-t", objHash)
			// If a note points to an object that we do not know about (yet), then err will not
			// be nil. We can safely just ignore those notes.
			if err == nil && objType == "commit" {
				revisions = append(revisions, objHash)
			}
		}
	}
	return revisions
}

// PushNotes pushes git notes to a remote repo.
func (repo *GitRepo) PushNotes(remote, notesRefPattern string) error {
	refspec := fmt.Sprintf("%s:%s", notesRefPattern, notesRefPattern)

	// The push is liable to fail if the user forgot to do a pull first, so
	// we treat errors as user errors rather than fatal errors.
	err := repo.runGitCommandInline("push", remote, refspec)
	if err != nil {
		return fmt.Errorf("Failed to push to the remote '%s': %v", remote, err)
	}
	return nil
}

func getRemoteNotesRef(remote, localNotesRef string) string {
	relativeNotesRef := strings.TrimPrefix(localNotesRef, "refs/notes/")
	return "refs/notes/" + remote + "/" + relativeNotesRef
}

// PullNotes fetches the contents of the given notes ref from a remote repo,
// and then merges them with the corresponding local notes using the
// "cat_sort_uniq" strategy.
func (repo *GitRepo) PullNotes(remote, notesRefPattern string) {
	remoteNotesRefPattern := getRemoteNotesRef(remote, notesRefPattern)
	fetchRefSpec := fmt.Sprintf("+%s:%s", notesRefPattern, remoteNotesRefPattern)
	repo.runGitCommandInlineOrDie("fetch", remote, fetchRefSpec)

	remoteRefs := repo.runGitCommandOrDie("ls-remote", remote, notesRefPattern)
	for _, line := range strings.Split(remoteRefs, "\n") {
		lineParts := strings.Split(line, "\t")
		if len(lineParts) == 2 {
			ref := lineParts[1]
			remoteRef := getRemoteNotesRef(remote, ref)
			repo.runGitCommandOrDie("notes", "--ref", ref, "merge", remoteRef, "-s", "cat_sort_uniq")
		}
	}
}
