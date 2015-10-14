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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review/ci"
	"github.com/google/git-appraise/review/comment"
	"github.com/google/git-appraise/review/request"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Template for printing the summary of a code review.
	reviewTemplate = `[%s] %s
  "%s"
`
	// Template for printing a single comment.
	commentTemplate = `[%s] %s
  %s %s "%s"
`
)

// CommentThread represents the tree-based hierarchy of comments.
//
// The Resolved field represents the aggregate status of the entire thread. If
// it is set to false, then it indicates that there is an unaddressed comment
// in the thread. If it is unset, then that means that the root comment is an
// FYI only, and that there are no unaddressed comments. If it is set to true,
// then that means that there are no unaddressed comments, and that the root
// comment has its resolved bit set to true.
type CommentThread struct {
	Hash     string          `json:"hash,omitempty"`
	Comment  comment.Comment `json:"comment"`
	Children []CommentThread `json:"children,omitempty"`
	Resolved *bool           `json:"resolved,omitempty"`
}

// Review represents the entire state of a code review.
//
// Reviews have two status fields which are orthogonal:
// 1. Resolved indicates if a reviewer has accepted or rejected the change.
// 2. Submitted indicates if the change has been incorporated into the target.
//
// Reviews also include a list of build-and-test status reports. Those
// correspond to either the current commit in the review ref (for pending
// reviews), or to the last commented-upon commit (for submitted reviews).
type Review struct {
	Revision  string          `json:"revision"`
	Request   request.Request `json:"request"`
	Comments  []CommentThread `json:"comments,omitempty"`
	Resolved  *bool           `json:"resolved,omitempty"`
	Submitted bool            `json:"submitted"`
	Reports   []ci.Report     `json:"reports,omitempty"`
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
// The aggregate status is the conjunction of all of the non-nil child statuses.
//
// This has the side-effect of setting the "Resolved" field of all descendant comment threads.
func updateThreadsStatus(threads []CommentThread) *bool {
	sort.Sort(byTimestamp(threads))
	noUnresolved := true
	var result *bool
	for _, thread := range threads {
		thread.updateResolvedStatus()
		if thread.Resolved != nil {
			noUnresolved = noUnresolved && *thread.Resolved
			result = &noUnresolved
		}
	}
	return result
}

// updateResolvedStatus calculates the aggregate status of a single comment thread,
// and updates the "Resolved" field of that thread accordingly.
func (thread *CommentThread) updateResolvedStatus() {
	resolved := updateThreadsStatus(thread.Children)
	if resolved == nil {
		thread.Resolved = thread.Comment.Resolved
		return
	}

	if !*resolved {
		thread.Resolved = resolved
		return
	}

	if thread.Comment.Resolved == nil || !*thread.Comment.Resolved {
		thread.Resolved = nil
		return
	}

	thread.Resolved = resolved
}

// mutableThread is an internal-only data structure used to store partially constructed comment threads.
type mutableThread struct {
	Hash     string
	Comment  comment.Comment
	Children []*mutableThread
}

// fixMutableThread is a helper method to finalize a mutableThread struct
// (partially constructed comment thread) as a CommentThread struct
// (fully constructed comment thread).
func fixMutableThread(mutableThread *mutableThread) CommentThread {
	var children []CommentThread
	for _, mutableChild := range mutableThread.Children {
		children = append(children, fixMutableThread(mutableChild))
	}
	return CommentThread{
		Hash:     mutableThread.Hash,
		Comment:  mutableThread.Comment,
		Children: children,
	}
}

// This function builds the comment thread tree from the log-based list of comments.
//
// Since the comments can be processed in any order, this uses an internal mutable
// data structure, and then converts it to the proper CommentThread structure at the end.
func buildCommentThreads(commentsByHash map[string]comment.Comment) []CommentThread {
	threadsByHash := make(map[string]*mutableThread)
	for hash, comment := range commentsByHash {
		thread, ok := threadsByHash[hash]
		if !ok {
			thread = &mutableThread{
				Hash:    hash,
				Comment: comment,
			}
			threadsByHash[hash] = thread
		}
	}
	var rootHashes []string
	for hash, thread := range threadsByHash {
		if thread.Comment.Parent == "" {
			rootHashes = append(rootHashes, hash)
		} else {
			parent, ok := threadsByHash[thread.Comment.Parent]
			if ok {
				parent.Children = append(parent.Children, thread)
			}
		}
	}
	var threads []CommentThread
	for _, hash := range rootHashes {
		threads = append(threads, fixMutableThread(threadsByHash[hash]))
	}
	return threads
}

// loadComments reads in the log-structured sequence of comments for a review,
// and then builds the corresponding tree-structured comment threads.
func (r *Review) loadComments() []CommentThread {
	commentNotes := repository.GetNotes(comment.Ref, r.Revision)
	commentsByHash := comment.ParseAllValid(commentNotes)
	return buildCommentThreads(commentsByHash)
}

// Get returns the specified code review.
//
// If no review request exists, the returned review is nil.
func Get(revision string) *Review {
	requestNotes := repository.GetNotes(request.Ref, revision)
	requests := request.ParseAllValid(requestNotes)
	if requests == nil {
		return nil
	}
	review := Review{
		Revision: revision,
		Request:  requests[len(requests)-1],
	}
	review.Comments = review.loadComments()
	review.Resolved = updateThreadsStatus(review.Comments)
	review.Submitted = repository.IsAncestor(revision, review.Request.TargetRef)
	// TODO(ojarjur): Optionally fetch the CI status of the last commit
	// in the review for which there are comments.
	return &review
}

// ListAll returns all reviews stored in the git-notes.
func ListAll() []Review {
	var reviews []Review
	for _, revision := range repository.ListNotedRevisions(request.Ref) {
		review := Get(revision)
		if review != nil {
			reviews = append(reviews, *review)
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
	currentCommit := repository.GetCommitHash(reviewRef)
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
	r := &matchingReviews[0]
	reports := ci.ParseAllValid(repository.GetNotes(ci.Ref, currentCommit))
	r.Reports = reports
	return r, nil
}

// PrintSummary prints a single-line summary of a review.
func (r *Review) PrintSummary() {
	statusString := "pending"
	if r.Resolved != nil {
		if *r.Resolved {
			statusString = "accepted"
		} else {
			statusString = "rejected"
		}
	}
	fmt.Printf(reviewTemplate, statusString, r.Revision, r.Request.Description)
}

// reformatTimestamp takes a timestamp string of the form "0123456789" and changes it
// to the form "Mon Jan _2 13:04:05 UTC 2006".
//
// Timestamps that are not in the format we expect are left alone.
func reformatTimestamp(timestamp string) string {
	parsedTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		// The timestamp is an unexpected format, so leave it alone
		return timestamp
	}
	t := time.Unix(parsedTimestamp, 0)
	return t.Format(time.UnixDate)
}

// showThread prints the given comment thread, indented by the given prefix string.
func showThread(thread CommentThread, indent string) error {
	comment := thread.Comment
	threadHash, err := comment.Hash()
	if err != nil {
		return err
	}

	timestamp := reformatTimestamp(comment.Timestamp)
	statusString := "fyi"
	if comment.Resolved != nil {
		if *comment.Resolved {
			statusString = "lgtm"
		} else {
			statusString = "needs work"
		}
	}

	threadDetails := fmt.Sprintf(commentTemplate, timestamp, threadHash, comment.Author, statusString, comment.Description)
	fmt.Print(indent + strings.Replace(threadDetails, "\n", "\n"+indent, 1))
	for _, child := range thread.Children {
		err := showThread(child, indent+"  ")
		if err != nil {
			return err
		}
	}
	return nil
}

// PrintDetails prints a multi-line overview of a review, including all comments.
func (r *Review) PrintDetails() error {
	r.PrintSummary()
	for _, thread := range r.Comments {
		err := showThread(thread, "  ")
		if err != nil {
			return err
		}
	}
	// TODO(ojarjur): If there are CI status reports for the last commit, then show them.
	return nil
}

// PrintJson pretty prints a review (including comments) formatted as JSON.
func (r *Review) PrintJson() error {
	jsonBytes, err := json.Marshal(*r)
	if err != nil {
		return err
	}
	var prettyBytes bytes.Buffer
	err = json.Indent(&prettyBytes, jsonBytes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(prettyBytes.String())
	return nil
}

// AddComment adds the given comment to the review.
func (r *Review) AddComment(c comment.Comment) error {
	commentNote, err := c.Write()
	if err != nil {
		return err
	}

	repository.AppendNote(comment.Ref, r.Revision, commentNote)
	return nil
}
