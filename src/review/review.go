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
	"repository"
	"review/comment"
	"review/request"
	"sort"
)

type CommentThread struct {
	Comment  comment.Comment
	Children []CommentThread
	Resolved *bool
}

type Review struct {
	Revision string
	Request  request.Request
	Comments []CommentThread
	Resolved *bool
}

type byTimestamp []CommentThread

// Interface methods for sorting comment threads by timestamp
func (threads byTimestamp) Len() int      { return len(threads) }
func (threads byTimestamp) Swap(i, j int) { threads[i], threads[j] = threads[j], threads[i] }
func (threads byTimestamp) Less(i, j int) bool {
	return threads[i].Comment.Timestamp < threads[j].Comment.Timestamp
}

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

func (thread *CommentThread) updateResolvedStatus() {
	resolved := updateThreadsStatus(thread.Children)
	if resolved == nil {
		resolved = thread.Comment.Resolved
	}
	thread.Resolved = resolved
}

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
			reviews = append(reviews, review)
		}
	}
	return reviews
}
