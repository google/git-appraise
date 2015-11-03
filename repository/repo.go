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

// Package repository contains helper methods for working with a Git repo.
package repository

// Note represents the contents of a git-note
type Note []byte

type CommitDetails struct {
	Author      string   `json:"author,omitempty"`
	AuthorEmail string   `json:"authorEmail,omitempty"`
	Tree        string   `json:"tree,omitempty"`
	Time        string   `json:"time,omitempty"`
	Parents     []string `json:"parents,omitempty"`
	Summary     string   `json:"summary,omitempty"`
}

// Repo represents a source code repository.
type Repo interface {
	// GetPath returns the path to the repo.
	GetPath() string

	// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
	GetRepoStateHash() string

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() string

	// HasUncommittedChanges returns true if there are local, uncommitted changes.
	HasUncommittedChanges() bool

	// VerifyGitRef verifies that the supplied ref points to a known commit.
	VerifyGitRef(ref string) error

	// VerifyGitRefOrDie verifies that the supplied ref points to a known commit.
	VerifyGitRefOrDie(ref string)

	// GetHeadRef returns the ref that is the current HEAD.
	GetHeadRef() string

	// GetCommitHash returns the hash of the commit pointed to by the given ref.
	GetCommitHash(ref string) string

	// ResolveRefCommit returns the commit pointed to by the given ref, which may be a remote ref.
	//
	// This differs from GetCommitHash which only works on exact matches, in that it will try to
	// intelligently handle the scenario of a ref not existing locally, but being known to exist
	// in a remote repo.
	//
	// This method should be used when a command may be performed by either the reviewer or the
	// reviewee, while GetCommitHash should be used when the encompassing command should only be
	// performed by the reviewee.
	ResolveRefCommit(ref string) (string, error)

	// GetCommitMessage returns the message stored in the commit pointed to by the given ref.
	GetCommitMessage(ref string) string

	// GetCommitTime returns the commit time of the commit pointed to by the given ref.
	GetCommitTime(ref string) string

	// GetLastParent returns the last parent of the given commit (as ordered by git).
	GetLastParent(ref string) (string, error)

	// GetCommitDetails returns the details of a commit's metadata.
	GetCommitDetails(ref string) (*CommitDetails, error)

	// MergeBase determines if the first commit that is an ancestor of the two arguments.
	MergeBase(a, b string) string

	// IsAncestor determines if the first argument points to a commit that is an ancestor of the second.
	IsAncestor(ancestor, descendant string) bool

	// Diff computes the diff between two given commits.
	Diff(left, right string, diffArgs ...string) string

	// SwitchToRef changes the currently-checked-out ref.
	SwitchToRef(ref string)

	// MergeRef merges the given ref into the current one.
	//
	// The ref argument is the ref to merge, and fastForward indicates that the
	// current ref should only move forward, as opposed to creating a bubble merge.
	MergeRef(ref string, fastForward bool)

	// RebaseRef rebases the given ref into the current one.
	RebaseRef(ref string)

	// ListCommitsBetween returns the list of commits between the two given revisions.
	//
	// The "from" parameter is the starting point (exclusive), and the "to" parameter
	// is the ending point (inclusive). If the commit pointed to by the "from" parameter
	// is not an ancestor of the commit pointed to by the "to" parameter, then the
	// merge base of the two is used as the starting point.
	//
	// The generated list is in chronological order (with the oldest commit first).
	ListCommitsBetween(from, to string) []string

	// GetNotes reads the notes from the given ref that annotate the given revision.
	GetNotes(notesRef, revision string) []Note

	// AppendNote appends a note to a revision under the given ref.
	AppendNote(ref, revision string, note Note)

	// ListNotedRevisions returns the collection of revisions that are annotated by notes in the given ref.
	ListNotedRevisions(notesRef string) []string

	// PushNotes pushes git notes to a remote repo.
	PushNotes(remote, notesRefPattern string) error

	// PullNotes fetches the contents of the given notes ref from a remote repo,
	// and then merges them with the corresponding local notes using the
	// "cat_sort_uniq" strategy.
	PullNotes(remote, notesRefPattern string)
}
