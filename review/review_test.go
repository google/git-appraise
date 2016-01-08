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
	"github.com/google/git-appraise/repository"
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
		t.Fatalf("Comment thread ordering failed. Got %v", sampleThreads)
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
		t.Fatalf("Unexpected threads: %v", threads)
	}
	rootThread := threads[0]
	if rootThread.Comment.Description != "root" {
		t.Fatalf("Unexpected root thread: %v", rootThread)
	}
	if len(rootThread.Children) != 1 {
		t.Fatalf("Unexpected root children: %v", rootThread.Children)
	}
	rootChild := rootThread.Children[0]
	if rootChild.Comment.Description != "child" {
		t.Fatalf("Unexpected child: %v", rootChild)
	}
	if len(rootChild.Children) != 1 {
		t.Fatalf("Unexpected leaves: %v", rootChild.Children)
	}
	threadLeaf := rootChild.Children[0]
	if threadLeaf.Comment.Description != "leaf" {
		t.Fatalf("Unexpected leaf: %v", threadLeaf)
	}
	if len(threadLeaf.Children) != 0 {
		t.Fatalf("Unexpected leaf children: %v", threadLeaf.Children)
	}
}

func TestGetHeadCommit(t *testing.T) {
	repo := repository.NewMockRepoForTest()

	submittedSimpleReview, err := Get(repo, repository.TestCommitB)
	if err != nil {
		t.Fatal(err)
	}
	submittedSimpleReviewHead, err := submittedSimpleReview.GetHeadCommit()
	if err != nil {
		t.Fatal("Unable to compute the head commit for a known review of a simple commit: ", err)
	}
	if submittedSimpleReviewHead != repository.TestCommitB {
		t.Fatal("Unexpected head commit computed for a known review of a simple commit.")
	}

	submittedModifiedReview, err := Get(repo, repository.TestCommitD)
	if err != nil {
		t.Fatal(err)
	}
	submittedModifiedReviewHead, err := submittedModifiedReview.GetHeadCommit()
	if err != nil {
		t.Fatal("Unable to compute the head commit for a known, multi-commit review: ", err)
	}
	if submittedModifiedReviewHead != repository.TestCommitE {
		t.Fatal("Unexpected head commit for a known, multi-commit review.")
	}

	pendingReview, err := Get(repo, repository.TestCommitG)
	if err != nil {
		t.Fatal(err)
	}
	pendingReviewHead, err := pendingReview.GetHeadCommit()
	if err != nil {
		t.Fatal("Unable to compute the head commit for a known review of a merge commit: ", err)
	}
	if pendingReviewHead != repository.TestCommitI {
		t.Fatal("Unexpected head commit computed for a pending review.")
	}
}

func TestGetBaseCommit(t *testing.T) {
	repo := repository.NewMockRepoForTest()

	submittedSimpleReview, err := Get(repo, repository.TestCommitB)
	if err != nil {
		t.Fatal(err)
	}
	submittedSimpleReviewBase, err := submittedSimpleReview.GetBaseCommit()
	if err != nil {
		t.Fatal("Unable to compute the base commit for a known review of a simple commit: ", err)
	}
	if submittedSimpleReviewBase != repository.TestCommitA {
		t.Fatal("Unexpected base commit computed for a known review of a simple commit.")
	}

	submittedMergeReview, err := Get(repo, repository.TestCommitD)
	if err != nil {
		t.Fatal(err)
	}
	submittedMergeReviewBase, err := submittedMergeReview.GetBaseCommit()
	if err != nil {
		t.Fatal("Unable to compute the base commit for a known review of a merge commit: ", err)
	}
	if submittedMergeReviewBase != repository.TestCommitC {
		t.Fatal("Unexpected base commit computed for a known review of a merge commit.")
	}

	pendingReview, err := Get(repo, repository.TestCommitG)
	if err != nil {
		t.Fatal(err)
	}
	pendingReviewBase, err := pendingReview.GetBaseCommit()
	if err != nil {
		t.Fatal("Unable to compute the base commit for a known review of a merge commit: ", err)
	}
	if pendingReviewBase != repository.TestCommitF {
		t.Fatal("Unexpected base commit computed for a pending review.")
	}
}
