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

package review

import (
	"github.com/google/git-appraise/review/comment"
	"sort"
	"testing"
)

func TestCommentSorting(t *testing.T) {
	sampleThreads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp:   "012400",
				Description: "Fourth",
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp:   "012346",
				Description: "Second",
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp:   "012345",
				Description: "First",
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp:   "012347",
				Description: "Third",
			},
		},
	}
	sort.Sort(byTimestamp(sampleThreads))
	descriptions := []string{}
	for _, thread := range sampleThreads {
		descriptions = append(descriptions, thread.Comment.Description)
	}
	if !(descriptions[0] == "First" && descriptions[1] == "Second" && descriptions[2] == "Third" && descriptions[3] == "Fourth") {
		t.Fatalf("Comment thread ordering failed. Got %s", sampleThreads)
	}
}

func validateUnresolved(t *testing.T, resolved *bool) {
	if resolved != nil {
		t.Fatalf("Expected resolved status to be unset, but instead it was %v", *resolved)
	}
}

func validateAccepted(t *testing.T, resolved *bool) {
	if resolved == nil {
		t.Fatal("Expected resolved status to be true, but it was unset")
	}
	if !*resolved {
		t.Fatal("Expected resolved status to be true, but it was false")
	}
}

func validateRejected(t *testing.T, resolved *bool) {
	if resolved == nil {
		t.Fatal("Expected resolved status to be false, but it was unset")
	}
	if *resolved {
		t.Fatal("Expected resolved status to be false, but it was true")
	}
}

func (commentThread *CommentThread) validateUnresolved(t *testing.T) {
	validateUnresolved(t, commentThread.Resolved)
}

func (commentThread *CommentThread) validateAccepted(t *testing.T) {
	validateAccepted(t, commentThread.Resolved)
}

func (commentThread *CommentThread) validateRejected(t *testing.T) {
	validateRejected(t, commentThread.Resolved)
}

func TestSimpleAcceptedThreadStatus(t *testing.T) {
	resolved := true
	simpleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &resolved,
		},
	}
	simpleThread.updateResolvedStatus()
	simpleThread.validateAccepted(t)
}

func TestSimpleRejectedThreadStatus(t *testing.T) {
	resolved := false
	simpleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &resolved,
		},
	}
	simpleThread.updateResolvedStatus()
	simpleThread.validateRejected(t)
}

func TestFYIThenAcceptedThreadStatus(t *testing.T) {
	accepted := true
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: nil,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  &accepted,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateUnresolved(t)
}

func TestFYIThenFYIThreadStatus(t *testing.T) {
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: nil,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  nil,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateUnresolved(t)
}

func TestFYIThenRejectedThreadStatus(t *testing.T) {
	rejected := false
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: nil,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  &rejected,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateRejected(t)
}

func TestAcceptedThenAcceptedThreadStatus(t *testing.T) {
	accepted := true
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &accepted,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  &accepted,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateAccepted(t)
}

func TestAcceptedThenFYIThreadStatus(t *testing.T) {
	accepted := true
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &accepted,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  nil,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateAccepted(t)
}

func TestAcceptedThenRejectedThreadStatus(t *testing.T) {
	accepted := true
	rejected := false
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &accepted,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  &rejected,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateRejected(t)
}

func TestRejectedThenAcceptedThreadStatus(t *testing.T) {
	accepted := true
	rejected := false
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &rejected,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  &accepted,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateUnresolved(t)
}

func TestRejectedThenFYIThreadStatus(t *testing.T) {
	rejected := false
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &rejected,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  nil,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateRejected(t)
}

func TestRejectedThenRejectedThreadStatus(t *testing.T) {
	rejected := false
	sampleThread := CommentThread{
		Comment: comment.Comment{
			Resolved: &rejected,
		},
		Children: []CommentThread{
			CommentThread{
				Comment: comment.Comment{
					Timestamp: "012345",
					Resolved:  &rejected,
				},
			},
		},
	}
	sampleThread.updateResolvedStatus()
	sampleThread.validateRejected(t)
}

func TestRejectedThenAcceptedThreadsStatus(t *testing.T) {
	accepted := true
	rejected := false
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  &rejected,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  &accepted,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateRejected(t, status)
}

func TestRejectedThenFYIThreadsStatus(t *testing.T) {
	rejected := false
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  &rejected,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  nil,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateRejected(t, status)
}

func TestRejectedThenRejectedThreadsStatus(t *testing.T) {
	rejected := false
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  &rejected,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  &rejected,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateRejected(t, status)
}

func TestAcceptedThenAcceptedThreadsStatus(t *testing.T) {
	accepted := true
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  &accepted,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  &accepted,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateAccepted(t, status)
}

func TestAcceptedThenFYIThreadsStatus(t *testing.T) {
	accepted := true
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  &accepted,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  nil,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateAccepted(t, status)
}

func TestAcceptedThenRejectedThreadsStatus(t *testing.T) {
	accepted := true
	rejected := false
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  &accepted,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  &rejected,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateRejected(t, status)
}

func TestFYIThenAcceptedThreadsStatus(t *testing.T) {
	accepted := true
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  nil,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  &accepted,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateAccepted(t, status)
}

func TestFYIThenFYIThreadsStatus(t *testing.T) {
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  nil,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  nil,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateUnresolved(t, status)
}

func TestFYIThenRejectedThreadsStatus(t *testing.T) {
	rejected := false
	threads := []CommentThread{
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012345",
				Resolved:  nil,
			},
		},
		CommentThread{
			Comment: comment.Comment{
				Timestamp: "012346",
				Resolved:  &rejected,
			},
		},
	}
	status := updateThreadsStatus(threads)
	validateRejected(t, status)
}

func TestBuildCommentThreads(t *testing.T) {
	rejected := false
	accepted := true
	root := comment.Comment{
		Timestamp:   "012345",
		Resolved:    nil,
		Description: "root",
	}
	rootHash, err := root.Hash()
	if err != nil {
		t.Fatal(err)
	}
	child := comment.Comment{
		Timestamp:   "012346",
		Resolved:    &rejected,
		Parent:      rootHash,
		Description: "child",
	}
	childHash, err := child.Hash()
	if err != nil {
		t.Fatal(err)
	}
	leaf := comment.Comment{
		Timestamp:   "012347",
		Resolved:    &accepted,
		Parent:      childHash,
		Description: "leaf",
	}
	leafHash, err := leaf.Hash()
	if err != nil {
		t.Fatal(err)
	}
	commentsByHash := map[string]comment.Comment{
		rootHash:  root,
		childHash: child,
		leafHash:  leaf,
	}
	threads := buildCommentThreads(commentsByHash)
	if len(threads) != 1 {
		t.Fatal("Unexpected threads: %v", threads)
	}
	rootThread := threads[0]
	if rootThread.Comment.Description != "root" {
		t.Fatal("Unexpected root thread: %v", rootThread)
	}
	if len(rootThread.Children) != 1 {
		t.Fatal("Unexpected root children: %v", rootThread.Children)
	}
	rootChild := rootThread.Children[0]
	if rootChild.Comment.Description != "child" {
		t.Fatal("Unexpected child: %v", rootChild)
	}
	if len(rootChild.Children) != 1 {
		t.Fatal("Unexpected leaves: %v", rootChild.Children)
	}
	threadLeaf := rootChild.Children[0]
	if threadLeaf.Comment.Description != "leaf" {
		t.Fatal("Unexpected leaf: %v", threadLeaf)
	}
	if len(threadLeaf.Children) != 0 {
		t.Fatal("Unexpected leaf children: %v", threadLeaf.Children)
	}
}

func TestGetHeadCommit(t *testing.T) {
	// TODO(ojarjur): It's pretty terrible that this relies on running within the git repo of
	// the tool and then using the tool's own review history as test data. We should change this
	// to use a mock git repo.
	submittedMergeReview := Get("fcc9b48925b8a880813275fa29b43426b5f1fccd")
	submittedMergeReviewBase, err := submittedMergeReview.GetBaseCommit()
	if err != nil {
		t.Fatal("Unable to compute the base commit for a known review of a merge commit.")
	}
	if submittedMergeReviewBase != "5c2b1d1e12eae76a85eb1b586c58d60e8c9ce388" {
		t.Fatal("Unexpected base commit computed for a known review of a merge commit.")
	}

	submittedModifiedReview := Get("62f1f51aea3b59829071c58ad2189231b6505fd3")
	submittedModifiedReviewBase, err := submittedModifiedReview.GetBaseCommit()
	if err != nil {
		t.Fatal("Unable to compute the base commit for a known, multi-commit review.")
	}
	if submittedModifiedReviewBase != "b346936104f9bb4532d31abd085b531109e0b19c" {
		t.Fatal("Unexpected base commit for a known, multi-commit review.")
	}
}
