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

// Package review contains the data structures used to represent code reviews.
package review

import (
	"fmt"
	"repository"
	"review/comment"
	"review/request"
	"sort"
)

// CommentThread represents the tree-based hierarchy of comments.
type CommentThread struct {
	Comment  comment.Comment
	Children []CommentThread
	Resolved *bool
}

// Review represents the entire state of a code review.
//
// Reviews have two status fields which are orthogonal:
// 1. Resolved indicates if a reviewer has accepted or rejected the change.
// 2. Submitted indicates if the change has been incorporated into the target.
type Review struct {
	Revision  string
	Request   request.Request
	Comments  []CommentThread
	Resolved  *bool
	Submitted bool
}

type byTimestamp []CommentThread

// Interface methods for sorting comment threads by timestamp
func (threads byTimestamp) Len() int      { return len(threads) }
func (threads byTimestamp) Swap(i, j int) { threads[i], threads[j] = threads[j], threads[i] }
func (threads byTimestamp) Less(i, j int) bool {
	return threads[i].Comment.Timestamp < threads[j].Comment.Timestamp
}

// updateThreadsStatus calculates the aggregate status of a sequence of comment threads.
//
// This has the side-effect of setting the "Resolved" field of all descendant comment threads.
func updateThreadsStatus(threads []CommentThread) *bool {
	sort.Sort(sort.Reverse(byTimestamp(threads)))
	for _, thread := range threads {
		thread.updateResolvedStatus()
		if thread.Resolved != nil {
			return thread.Resolved
		}
	}
	return nil
}

// updateResolvedStatus calculates the aggregate status of a single comment thread,
// and updates the "Resolved" field of that thread accordingly.
func (thread *CommentThread) updateResolvedStatus() {
	resolved := updateThreadsStatus(thread.Children)
	if resolved == nil {
		resolved = thread.Comment.Resolved
	}
	thread.Resolved = resolved
}

// loadComments reads in the log-structured sequence of comments for a review,
// and then builds the corresponding tree-structured comment threads.
func (review *Review) loadComments() []CommentThread {
	commentNotes := repository.GetNotes(comment.Ref, review.Revision)
	commentsByHash := comment.ParseAllValid(commentNotes)
	threadsByHash := make(map[string]CommentThread)
	for hash, comment := range commentsByHash {
		thread, ok := threadsByHash[hash]
		if !ok {
			thread = CommentThread{
				Comment: comment,
			}
			threadsByHash[hash] = thread
		}
	}
	var threads []CommentThread
	for _, thread := range threadsByHash {
		if thread.Comment.Parent == "" {
			threads = append(threads, thread)
		} else {
			parent, ok := threadsByHash[thread.Comment.Parent]
			if ok {
				parent.Children = append(parent.Children, thread)
			}
		}
	}
	return threads
}

// ListAll returns all reviews stored in the git-notes.
func ListAll() []Review {
	var reviews []Review
	for _, revision := range repository.ListNotedRevisions(request.Ref) {
		requestNotes := repository.GetNotes(request.Ref, revision)
		for _, req := range request.ParseAllValid(requestNotes) {
			review := Review{
				Revision: revision,
				Request:  req,
			}
			review.Comments = review.loadComments()
			review.Resolved = updateThreadsStatus(review.Comments)
			review.Submitted = repository.IsAncestor(revision, req.TargetRef)
			reviews = append(reviews, review)
		}
	}
	return reviews
}

// ListOpen returns all reviews that are not yet incorporated into their target refs.
func ListOpen() []Review {
	var openReviews []Review
	for _, review := range ListAll() {
		if !review.Submitted {
			openReviews = append(openReviews, review)
		}
	}
	return openReviews
}

// GetCurrent returns the current, open code review.
//
// If there are multiple matching reviews, then an error is returned.
func GetCurrent() (*Review, error) {
	reviewRef := repository.GetHeadRef()
	var matchingReviews []Review
	for _, review := range ListOpen() {
		if review.Request.ReviewRef == reviewRef {
			matchingReviews = append(matchingReviews, review)
		}
	}
	if matchingReviews == nil {
		return nil, nil
	}
	if len(matchingReviews) != 1 {
		return nil, fmt.Errorf("There are %d open reviews for the ref \"%s\"", len(matchingReviews), reviewRef)
	}
	return &matchingReviews[0], nil
}
