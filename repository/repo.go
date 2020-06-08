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

import (
	"crypto/sha1"
	"fmt"
)

// Note represents the contents of a git-note
type Note []byte

// Hash returns a hash of the given note
func (n Note) Hash() string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(n)))
}

// CommitDetails represents the contents of a commit.
type CommitDetails struct {
	Author         string   `json:"author,omitempty"`
	AuthorEmail    string   `json:"authorEmail,omitempty"`
	AuthorTime     string   `json:"authorTime,omitempty"`
	Committer      string   `json:"committer,omitempty"`
	CommitterEmail string   `json:"committerEmail,omitempty"`
	Tree           string   `json:"tree,omitempty"`
	Time           string   `json:"time,omitempty"`
	Parents        []string `json:"parents,omitempty"`
	Summary        string   `json:"summary,omitempty"`
}

type TreeChild interface {
	// Type returns the type of the child object (e.g. "blob" vs. "tree").
	Type() string

	// Store writes the object to the repository and returns its hash.
	Store(repo Repo) (string, error)
}

// Blob represents a (non-directory) file stored in a repository.
//
// Blob objects are immutable.
type Blob struct {
	savedHashes map[Repo]string
	contents    string
}

// NewBlob returns a new *Blob object tied to the given repo with the given contents.
func NewBlob(contents string) *Blob {
	savedHashes := make(map[Repo]string)
	return &Blob{
		savedHashes: savedHashes,
		contents:    contents,
	}
}

func (b *Blob) Type() string {
	return "blob"
}

func (b *Blob) Store(repo Repo) (string, error) {
	if savedHash := b.savedHashes[repo]; savedHash != "" {
		return savedHash, nil
	}
	savedHash, err := repo.StoreBlob(b.Contents())
	if err == nil && savedHash != "" {
		b.savedHashes[repo] = savedHash
	}
	return savedHash, nil
}

// Contents returns the contents of the blob
func (b *Blob) Contents() string {
	return b.contents
}

// Tree represents a directory stored in a repository.
//
// Tree objects are immutable.
type Tree struct {
	savedHashes map[Repo]string
	contents    map[string]TreeChild
}

// NewTree constructs a new *Tree object tied to the given repo with the given contents.
func NewTree(contents map[string]TreeChild) *Tree {
	immutableContents := make(map[string]TreeChild)
	for k, v := range contents {
		immutableContents[k] = v
	}
	savedHashes := make(map[Repo]string)
	return &Tree{
		savedHashes: savedHashes,
		contents:    immutableContents,
	}
}

func (t *Tree) Type() string {
	return "tree"
}

func (t *Tree) Store(repo Repo) (string, error) {
	if savedHash := t.savedHashes[repo]; savedHash != "" {
		return savedHash, nil
	}
	savedHash, err := repo.StoreTree(t.Contents())
	if err == nil && savedHash != "" {
		t.savedHashes[repo] = savedHash
	}
	return savedHash, nil
}

// Contents returns a map of the child elements of the tree.
//
// The returned map is mutable, but changes made to it have no
// effect on the underly Tree object.
func (t *Tree) Contents() map[string]TreeChild {
	result := make(map[string]TreeChild)
	for k, v := range t.contents {
		result[k] = v
	}
	return result
}

// Repo represents a source code repository.
type Repo interface {
	// GetPath returns the path to the repo.
	GetPath() string

	// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
	GetRepoStateHash() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetUserSigningKey returns the key id the user has configured for
	// sigining git artifacts.
	GetUserSigningKey() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// GetSubmitStrategy returns the way in which a review is submitted
	GetSubmitStrategy() (string, error)

	// HasUncommittedChanges returns true if there are local, uncommitted changes.
	HasUncommittedChanges() (bool, error)

	// HasRef checks whether the specified ref exists in the repo.
	HasRef(ref string) (bool, error)

	// HasObject returns whether or not the repo contains an object with the given hash.
	HasObject(hash string) (bool, error)

	// VerifyCommit verifies that the supplied hash points to a known commit.
	VerifyCommit(hash string) error

	// VerifyGitRef verifies that the supplied ref points to a known commit.
	VerifyGitRef(ref string) error

	// GetHeadRef returns the ref that is the current HEAD.
	GetHeadRef() (string, error)

	// GetCommitHash returns the hash of the commit pointed to by the given ref.
	GetCommitHash(ref string) (string, error)

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
	GetCommitMessage(ref string) (string, error)

	// GetCommitTime returns the commit time of the commit pointed to by the given ref.
	GetCommitTime(ref string) (string, error)

	// GetLastParent returns the last parent of the given commit (as ordered by git).
	GetLastParent(ref string) (string, error)

	// GetCommitDetails returns the details of a commit's metadata.
	GetCommitDetails(ref string) (*CommitDetails, error)

	// MergeBase determines if the first commit that is an ancestor of the two arguments.
	MergeBase(a, b string) (string, error)

	// IsAncestor determines if the first argument points to a commit that is an ancestor of the second.
	IsAncestor(ancestor, descendant string) (bool, error)

	// Diff computes the diff between two given commits.
	Diff(left, right string, diffArgs ...string) (string, error)

	// Show returns the contents of the given file at the given commit.
	Show(commit, path string) (string, error)

	// SwitchToRef changes the currently-checked-out ref.
	SwitchToRef(ref string) error

	// ArchiveRef adds the current commit pointed to by the 'ref' argument
	// under the ref specified in the 'archive' argument.
	//
	// Both the 'ref' and 'archive' arguments are expected to be the fully
	// qualified names of git refs (e.g. 'refs/heads/my-change' or
	// 'refs/archive/devtools').
	//
	// If the ref pointed to by the 'archive' argument does not exist
	// yet, then it will be created.
	ArchiveRef(ref, archive string) error

	// MergeRef merges the given ref into the current one.
	//
	// The ref argument is the ref to merge, and fastForward indicates that the
	// current ref should only move forward, as opposed to creating a bubble merge.
	// The messages argument(s) provide text that should be included in the default
	// merge commit message (separated by blank lines).
	MergeRef(ref string, fastForward bool, messages ...string) error

	// MergeAndSignRef merges the given ref into the current one and signs the
	// merge.
	//
	// The ref argument is the ref to merge, and fastForward indicates that the
	// current ref should only move forward, as opposed to creating a bubble merge.
	// The messages argument(s) provide text that should be included in the default
	// merge commit message (separated by blank lines).
	MergeAndSignRef(ref string, fastForward bool, messages ...string) error

	// RebaseRef rebases the current ref onto the given one.
	RebaseRef(ref string) error

	// RebaseAndSignRef rebases the current ref onto the given one and signs
	// the result.
	RebaseAndSignRef(ref string) error

	// ListCommits returns the list of commits reachable from the given ref.
	//
	// The generated list is in chronological order (with the oldest commit first).
	//
	// If the specified ref does not exist, then this method returns an empty result.
	ListCommits(ref string) []string

	// ListCommitsBetween returns the list of commits between the two given revisions.
	//
	// The "from" parameter is the starting point (exclusive), and the "to"
	// parameter is the ending point (inclusive).
	//
	// The "from" commit does not need to be an ancestor of the "to" commit. If it
	// is not, then the merge base of the two is used as the starting point.
	// Admittedly, this makes calling these the "between" commits is a bit of a
	// misnomer, but it also makes the method easier to use when you want to
	// generate the list of changes in a feature branch, as it eliminates the need
	// to explicitly calculate the merge base. This also makes the semantics of the
	// method compatible with git's built-in "rev-list" command.
	//
	// The generated list is in chronological order (with the oldest commit first).
	ListCommitsBetween(from, to string) ([]string, error)

	// StoreBlob writes the given file contents to the repository and returns its hash.
	StoreBlob(contents string) (string, error)

	// StoreTree writes the given file tree contents to the repository and returns its hash.
	StoreTree(contents map[string]TreeChild) (string, error)

	// ReadTree reads the file tree pointed to by the given ref or hash from the repository.
	ReadTree(ref string) (*Tree, error)

	// CreateCommit creates a commit object and returns its hash.
	CreateCommit(details *CommitDetails) (string, error)

	// CreateCommitWithTree creates a commit object with the given tree and returns its hash.
	CreateCommitWithTree(details *CommitDetails, t *Tree) (string, error)

	// SetRef sets the commit pointed to by the specified ref to `newCommitHash`,
	// iff the ref currently points `previousCommitHash`.
	SetRef(ref, newCommitHash, previousCommitHash string) error

	// GetNotes reads the notes from the given ref that annotate the given revision.
	GetNotes(notesRef, revision string) []Note

	// GetAllNotes reads the contents of the notes under the given ref for every commit.
	//
	// The returned value is a mapping from commit hash to the list of notes for that commit.
	//
	// This is the batch version of the corresponding GetNotes(...) method.
	GetAllNotes(notesRef string) (map[string][]Note, error)

	// AppendNote appends a note to a revision under the given ref.
	AppendNote(ref, revision string, note Note) error

	// ListNotedRevisions returns the collection of revisions that are annotated by notes in the given ref.
	ListNotedRevisions(notesRef string) []string

	// Remotes returns a list of the remotes.
	Remotes() ([]string, error)

	// Fetch fetches from the given remote using the supplied refspecs.
	Fetch(remote string, refspecs ...string) error

	// PushNotes pushes git notes to a remote repo.
	PushNotes(remote, notesRefPattern string) error

	// PullNotes fetches the contents of the given notes ref from a remote repo,
	// and then merges them with the corresponding local notes using the
	// "cat_sort_uniq" strategy.
	PullNotes(remote, notesRefPattern string) error

	// PushNotesAndArchive pushes the given notes and archive refs to a remote repo.
	PushNotesAndArchive(remote, notesRefPattern, archiveRefPattern string) error

	// PullNotesAndArchive fetches the contents of the notes and archives refs from
	// a remote repo, and merges them with the corresponding local refs.
	//
	// For notes refs, we assume that every note can be automatically merged using
	// the 'cat_sort_uniq' strategy (the git-appraise schemas fit that requirement),
	// so we automatically merge the remote notes into the local notes.
	//
	// For "archive" refs, they are expected to be used solely for maintaining
	// reachability of commits that are part of the history of any reviews,
	// so we do not maintain any consistency with their tree objects. Instead,
	// we merely ensure that their history graph includes every commit that we
	// intend to keep.
	PullNotesAndArchive(remote, notesRefPattern, archiveRefPattern string) error

	// MergeNotes merges in the remote's state of the archives reference into
	// the local repository's.
	MergeNotes(remote, notesRefPattern string) error

	// MergeArchives merges in the remote's state of the archives reference
	// into the local repository's.
	MergeArchives(remote, archiveRefPattern string) error

	// MergeForks merges in the remote's state of the forks reference
	// into the local repository's.
	MergeForks(remote, forksRef string) error

	// FetchAndReturnNewReviewHashes fetches the notes "branches" and then
	// susses out the IDs (the revision the review points to) of any new
	// reviews, then returns that list of IDs.
	//
	// This is accomplished by determining which files in the notes tree have
	// changed because the _names_ of these files correspond to the revisions
	// they point to.
	FetchAndReturnNewReviewHashes(remote, notesRefPattern string, devtoolsRefPatterns ...string) ([]string, error)

	// PullNotesForksAndArchive fetches the contents of the notes, forks, and archives
	// refs from  a remote repo, and merges them with the corresponding local refs.
	//
	// For notes refs, we assume that every note can be automatically merged using
	// the 'cat_sort_uniq' strategy (the git-appraise schemas fit that requirement),
	// so we automatically merge the remote notes into the local notes.
	//
	// For the forks ref, we assume that we can merge using the recursive, `ours`,
	// merge strategy.
	//
	// For "archive" refs, they are expected to be used solely for maintaining
	// reachability of commits that are part of the history of any reviews,
	// so we do not maintain any consistency with their tree objects. Instead,
	// we merely ensure that their history graph includes every commit that we
	// intend to keep.
	//
	// The returned slice contains a list of all objects for which new notes were
	// fetched from the remote.
	PullNotesForksAndArchive(remote, notesRefPattern, forksRef, archiveRefPattern string) ([]string, error)

	// Push pushes the given refs to a remote repo.
	Push(remote string, refPattern ...string) error
}
