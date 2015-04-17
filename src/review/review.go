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

func updateResolvedStatus(thread CommentThread) {
	resolved := false
	anyResolved := false
	for _, child := range thread.Children {
		updateResolvedStatus(child)
		if child.Resolved != nil {
			if *child.Resolved == false {
				thread.Resolved = &resolved
				return
			} else {
				anyResolved = true
			}
		}
	}
	if anyResolved {
		resolved = true
		thread.Resolved = &resolved
	} else {
		thread.Resolved = nil
	}
}

func updateReviewStatus(review Review) {
	resolved := false
	anyResolved := false
	for _, thread := range review.Comments {
		updateResolvedStatus(thread)
		if thread.Resolved != nil {
			if *thread.Resolved == false {
				review.Resolved = &resolved
				return
			} else {
				anyResolved = true
			}
		}
	}
	if anyResolved {
		resolved = true
		review.Resolved = &resolved
	} else {
		review.Resolved = nil
	}
}

func loadComments(review Review) []CommentThread {
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
	for _, thread := range threads {
		updateResolvedStatus(thread)
	}
	return threads
}

func ListAll() []Review {
	var reviews []Review
	for _, revision := range repository.ListNotedRevisions(request.Ref) {
		requestNotes := repository.GetNotes(request.Ref, revision)
		for _, req := range request.ParseAllValid(requestNotes) {
			reviews = append(reviews, Review{
				Revision: revision,
				Request:  req,
			})
		}
	}
	for _, review := range reviews {
		review.Comments = loadComments(review)
		updateReviewStatus(review)
	}
	return reviews
}
