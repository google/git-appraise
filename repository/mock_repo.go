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

package repository

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// Constants used for testing.
// We initialize our mock repo with two branches (one of which holds a pending review),
// and commit history that looks like this:
//
//  Master Branch:    A--B--D--E--F--J
//                     \   /    \  \
//                       C       \  \
//                                \  \
//  Review Branch:                 G--H--I
//
// Where commits "B" and "D" represent reviews that have been submitted, and "G"
// is a pending review.
const (
	TestTargetRef   = "refs/heads/master"
	TestReviewRef   = "refs/heads/ojarjur/mychange"
	TestRequestsRef = "refs/notes/devtools/reviews"
	TestCommentsRef = "refs/notes/devtools/discuss"

	TestCommitA = "A"
	TestCommitB = "B"
	TestCommitC = "C"
	TestCommitD = "D"
	TestCommitE = "E"
	TestCommitF = "F"
	TestCommitG = "G"
	TestCommitH = "H"
	TestCommitI = "I"
	TestCommitJ = "J"

	TestRequestB = `{"timestamp": "0000000001", "reviewRef": "refs/heads/ojarjur/mychange", "targetRef": "refs/heads/master", "requester": "ojarjur", "reviewers": ["ojarjur"], "description": "B"}`
	TestRequestD = `{"timestamp": "0000000002", "reviewRef": "refs/heads/ojarjur/mychange", "targetRef": "refs/heads/master", "requester": "ojarjur", "reviewers": ["ojarjur"], "description": "D"}`
	TestRequestG = `{"timestamp": "0000000004", "reviewRef": "refs/heads/ojarjur/mychange", "targetRef": "refs/heads/master", "requester": "ojarjur", "reviewers": ["ojarjur"], "description": "G"}`

	TestDiscussB = `{"timestamp": "0000000001", "author": "ojarjur", "location": {"commit": "B"}, "resolved": true}`
	TestDiscussD = `{"timestamp": "0000000003", "author": "ojarjur", "location": {"commit": "E"}, "resolved": true}`
)

type mockCommit struct {
	Message string   `json:"message,omitempty"`
	Time    string   `json:"time,omitempty"`
	Parents []string `json:"parents,omitempty"`
}

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {
	Head    string
	Refs    map[string]string            `json:"refs,omitempty"`
	Commits map[string]mockCommit        `json:"commits,omitempty"`
	Notes   map[string]map[string]string `json:"notes,omitempty"`
}

func NewMockRepoForTest() Repo {
	commitA := mockCommit{
		Message: "First commit",
		Time:    "0",
		Parents: nil,
	}
	commitB := mockCommit{
		Message: "Second commit",
		Time:    "1",
		Parents: []string{TestCommitA},
	}
	commitC := mockCommit{
		Message: "No, I'm the second commit",
		Time:    "1",
		Parents: []string{TestCommitA},
	}
	commitD := mockCommit{
		Message: "Fourth commit",
		Time:    "2",
		Parents: []string{TestCommitB, TestCommitC},
	}
	commitE := mockCommit{
		Message: "Fifth commit",
		Time:    "3",
		Parents: []string{TestCommitD},
	}
	commitF := mockCommit{
		Message: "Sixth commit",
		Time:    "4",
		Parents: []string{TestCommitE},
	}
	commitG := mockCommit{
		Message: "No, I'm the sixth commit",
		Time:    "4",
		Parents: []string{TestCommitE},
	}
	commitH := mockCommit{
		Message: "Seventh commit",
		Time:    "5",
		Parents: []string{TestCommitG, TestCommitF},
	}
	commitI := mockCommit{
		Message: "Eighth commit",
		Time:    "6",
		Parents: []string{TestCommitH},
	}
	commitJ := mockCommit{
		Message: "No, I'm the eighth commit",
		Time:    "6",
		Parents: []string{TestCommitF},
	}
	return mockRepoForTest{
		Head: TestTargetRef,
		Refs: map[string]string{
			TestTargetRef: TestCommitJ,
			TestReviewRef: TestCommitI,
		},
		Commits: map[string]mockCommit{
			TestCommitA: commitA,
			TestCommitB: commitB,
			TestCommitC: commitC,
			TestCommitD: commitD,
			TestCommitE: commitE,
			TestCommitF: commitF,
			TestCommitG: commitG,
			TestCommitH: commitH,
			TestCommitI: commitI,
			TestCommitJ: commitJ,
		},
		Notes: map[string]map[string]string{
			TestRequestsRef: map[string]string{
				TestCommitB: TestRequestB,
				TestCommitD: TestRequestD,
				TestCommitG: TestRequestG,
			},
			TestCommentsRef: map[string]string{
				TestCommitB: TestDiscussB,
				TestCommitD: TestDiscussD,
			},
		},
	}
}

// GetPath returns the path to the repo.
func (r mockRepoForTest) GetPath() string { return "~/mockRepo/" }

// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
func (r mockRepoForTest) GetRepoStateHash() string {
	repoJson, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(repoJson)))
}

// GetUserEmail returns the email address that the user has used to configure git.
func (r mockRepoForTest) GetUserEmail() string { return "user@example.com" }

// HasUncommittedChanges returns true if there are local, uncommitted changes.
func (r mockRepoForTest) HasUncommittedChanges() bool { return false }

func (r mockRepoForTest) resolveLocalRef(ref string) (string, error) {
	if commit, ok := r.Refs[ref]; ok {
		return commit, nil
	}
	if _, ok := r.Commits[ref]; ok {
		return ref, nil
	}
	return "", fmt.Errorf("The ref %q does not exist", ref)
}

// VerifyGitRef verifies that the supplied ref points to a known commit.
func (r mockRepoForTest) VerifyGitRef(ref string) error {
	_, err := r.resolveLocalRef(ref)
	return err
}

// VerifyGitRefOrDie verifies that the supplied ref points to a known commit.
func (r mockRepoForTest) VerifyGitRefOrDie(ref string) {
	if err := r.VerifyGitRef(ref); err != nil {
		log.Fatal(err)
	}
}

// GetHeadRef returns the ref that is the current HEAD.
func (r mockRepoForTest) GetHeadRef() string { return r.Head }

// GetCommitHash returns the hash of the commit pointed to by the given ref.
func (r mockRepoForTest) GetCommitHash(ref string) string {
	r.VerifyGitRefOrDie(ref)
	return r.Refs[ref]
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
func (r mockRepoForTest) ResolveRefCommit(ref string) (string, error) {
	if commit, err := r.resolveLocalRef(ref); err == nil {
		return commit, err
	}
	return r.resolveLocalRef(strings.Replace(ref, "refs/heads/", "refs/remotes/origin/", 1))
}

func (r mockRepoForTest) getCommit(ref string) (mockCommit, error) {
	commit, err := r.resolveLocalRef(ref)
	return r.Commits[commit], err
}

func (r mockRepoForTest) getCommitOrDie(ref string) mockCommit {
	commit, err := r.getCommit(ref)
	if err != nil {
		log.Fatal(err)
	}
	return commit
}

// GetCommitMessage returns the message stored in the commit pointed to by the given ref.
func (r mockRepoForTest) GetCommitMessage(ref string) string {
	return r.getCommitOrDie(ref).Message
}

// GetCommitTime returns the commit time of the commit pointed to by the given ref.
func (r mockRepoForTest) GetCommitTime(ref string) string {
	return r.getCommitOrDie(ref).Time
}

// GetLastParent returns the last parent of the given commit (as ordered by git).
func (r mockRepoForTest) GetLastParent(ref string) (string, error) {
	commit, err := r.getCommit(ref)
	if len(commit.Parents) > 0 {
		return commit.Parents[len(commit.Parents)-1], err
	}
	return "", err
}

// GetCommitDetails returns the details of a commit's metadata.
func (r mockRepoForTest) GetCommitDetails(ref string) (*CommitDetails, error) {
	commit, err := r.getCommit(ref)
	if err != nil {
		return nil, err
	}
	var details CommitDetails
	details.Author = "Test Author"
	details.AuthorEmail = "author@example.com"
	details.Summary = commit.Message
	details.Time = commit.Time
	details.Parents = commit.Parents
	return &details, nil
}

// ancestors returns the breadth-first traversal of a commit's ancestors
func (r mockRepoForTest) ancestors(commit string) []string {
	queue := []string{commit}
	var ancestors []string
	for queue != nil {
		var nextQueue []string
		for _, c := range queue {
			parents := r.getCommitOrDie(c).Parents
			nextQueue = append(nextQueue, parents...)
			ancestors = append(ancestors, parents...)
		}
		queue = nextQueue
	}
	return ancestors
}

// IsAncestor determines if the first argument points to a commit that is an ancestor of the second.
func (r mockRepoForTest) IsAncestor(ancestor, descendant string) bool {
	if ancestor == descendant {
		return true
	}
	for _, parent := range r.getCommitOrDie(descendant).Parents {
		if r.IsAncestor(ancestor, parent) {
			return true
		}
	}
	return false
}

// MergeBase determines if the first commit that is an ancestor of the two arguments.
func (r mockRepoForTest) MergeBase(a, b string) string {
	for _, ancestor := range r.ancestors(a) {
		if r.IsAncestor(ancestor, b) {
			return ancestor
		}
	}
	return ""
}

// Diff computes the diff between two given commits.
func (r mockRepoForTest) Diff(left, right string, diffArgs ...string) string {
	return fmt.Sprintf("Diff between %q and %q", left, right)
}

// SwitchToRef changes the currently-checked-out ref.
func (r mockRepoForTest) SwitchToRef(ref string) {
	r.Head = ref
}

// MergeRef merges the given ref into the current one.
//
// The ref argument is the ref to merge, and fastForward indicates that the
// current ref should only move forward, as opposed to creating a bubble merge.
func (r mockRepoForTest) MergeRef(ref string, fastForward bool) {}

// RebaseRef rebases the given ref into the current one.
func (r mockRepoForTest) RebaseRef(ref string) {}

// ListCommitsBetween returns the list of commits between the two given revisions.
//
// The "from" parameter is the starting point (exclusive), and the "to" parameter
// is the ending point (inclusive). If the commit pointed to by the "from" parameter
// is not an ancestor of the commit pointed to by the "to" parameter, then the
// merge base of the two is used as the starting point.
//
// The generated list is in chronological order (with the oldest commit first).
func (r mockRepoForTest) ListCommitsBetween(from, to string) []string { return nil }

// GetNotes reads the notes from the given ref that annotate the given revision.
func (r mockRepoForTest) GetNotes(notesRef, revision string) []Note {
	notesText := r.Notes[notesRef][revision]
	var notes []Note
	for _, line := range strings.Split(notesText, "\n") {
		notes = append(notes, Note(line))
	}
	return notes
}

// AppendNote appends a note to a revision under the given ref.
func (r mockRepoForTest) AppendNote(ref, revision string, note Note) {
	existingNotes := r.Notes[ref][revision]
	newNotes := existingNotes + "\n" + string(note)
	r.Notes[ref][revision] = newNotes
}

// ListNotedRevisions returns the collection of revisions that are annotated by notes in the given ref.
func (r mockRepoForTest) ListNotedRevisions(notesRef string) []string {
	var revisions []string
	for revision := range r.Notes[notesRef] {
		if _, ok := r.Commits[revision]; ok {
			revisions = append(revisions, revision)
		}
	}
	return revisions
}

// PushNotes pushes git notes to a remote repo.
func (r mockRepoForTest) PushNotes(remote, notesRefPattern string) error { return nil }

// PullNotes fetches the contents of the given notes ref from a remote repo,
// and then merges them with the corresponding local notes using the
// "cat_sort_uniq" strategy.
func (r mockRepoForTest) PullNotes(remote, notesRefPattern string) {}
