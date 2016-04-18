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
	"github.com/google/git-appraise/review/analyses"
	"github.com/google/git-appraise/review/ci"
	"github.com/google/git-appraise/review/comment"
	"github.com/google/git-appraise/review/request"
	"sort"
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

// Summary represents the high-level state of a code review.
//
// This high-level state corresponds to the data that can be quickly read
// directly from the repo, so other methods that need to operate on a lot
// of reviews (such as listing the open reviews) should prefer operating on
// the summary rather than the details.
//
// Review summaries have two status fields which are orthogonal:
// 1. Resolved indicates if a reviewer has accepted or rejected the change.
// 2. Submitted indicates if the change has been incorporated into the target.
type Summary struct {
	Repo        repository.Repo   `json:"-"`
	Revision    string            `json:"revision"`
	Request     request.Request   `json:"request"`
	AllRequests []request.Request `json:"-"`
	Comments    []CommentThread   `json:"comments,omitempty"`
	Resolved    *bool             `json:"resolved,omitempty"`
	Submitted   bool              `json:"submitted"`
}

// Review represents the entire state of a code review.
//
// This extends Summary to also include a list of reports for both the
// continuous integration status, and the static analysis runs. Those reports
// correspond to either the current commit in the review ref (for pending
// reviews), or to the last commented-upon commit (for submitted reviews).
type Review struct {
	*Summary
	Reports  []ci.Report       `json:"reports,omitempty"`
	Analyses []analyses.Report `json:"analyses,omitempty"`
}

type byTimestamp []CommentThread

// Interface methods for sorting comment threads by timestamp
func (threads byTimestamp) Len() int      { return len(threads) }
func (threads byTimestamp) Swap(i, j int) { threads[i], threads[j] = threads[j], threads[i] }
func (threads byTimestamp) Less(i, j int) bool {
	return threads[i].Comment.Timestamp < threads[j].Comment.Timestamp
}

type requestsByTimestamp []request.Request

// Interface methods for sorting review requests by timestamp
func (requests requestsByTimestamp) Len() int { return len(requests) }
func (requests requestsByTimestamp) Swap(i, j int) {
	requests[i], requests[j] = requests[j], requests[i]
}
func (requests requestsByTimestamp) Less(i, j int) bool {
	return requests[i].Timestamp < requests[j].Timestamp
}

// updateThreadsStatus calculates the aggregate status of a sequence of comment threads.
//
// The aggregate status is the conjunction of all of the non-nil child statuses.
//
// This has the side-effect of setting the "Resolved" field of all descendant comment threads.
func updateThreadsStatus(threads []CommentThread) *bool {
	sort.Stable(byTimestamp(threads))
	noUnresolved := true
	var result *bool
	for i := range threads {
		thread := &threads[i]
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
func (r *Summary) loadComments() []CommentThread {
	commentNotes := r.Repo.GetNotes(comment.Ref, r.Revision)
	commentsByHash := comment.ParseAllValid(commentNotes)
	return buildCommentThreads(commentsByHash)
}

// GetSummary returns the summary of the specified code review.
//
// If no review request exists, the returned review summary is nil.
func GetSummary(repo repository.Repo, revision string) (*Summary, error) {
	requestNotes := repo.GetNotes(request.Ref, revision)
	requests := request.ParseAllValid(requestNotes)
	if requests == nil {
		return nil, nil
	}
	sort.Stable(requestsByTimestamp(requests))
	reviewSummary := Summary{
		Repo:        repo,
		Revision:    revision,
		Request:     requests[len(requests)-1],
		AllRequests: requests,
	}
	reviewSummary.Comments = reviewSummary.loadComments()
	reviewSummary.Resolved = updateThreadsStatus(reviewSummary.Comments)
	submitted, err := repo.IsAncestor(revision, reviewSummary.Request.TargetRef)
	if err != nil {
		return nil, err
	}
	reviewSummary.Submitted = submitted
	return &reviewSummary, nil
}

// Details returns the detailed review for the given summary.
func (r *Summary) Details() (*Review, error) {
	review := Review{
		Summary: r,
	}
	currentCommit, err := review.GetHeadCommit()
	if err == nil {
		review.Reports = ci.ParseAllValid(review.Repo.GetNotes(ci.Ref, currentCommit))
		review.Analyses = analyses.ParseAllValid(review.Repo.GetNotes(analyses.Ref, currentCommit))
	}
	return &review, nil
}

// Get returns the specified code review.
//
// If no review request exists, the returned review is nil.
func Get(repo repository.Repo, revision string) (*Review, error) {
	summary, err := GetSummary(repo, revision)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}
	return summary.Details()
}

// ListAll returns all reviews stored in the git-notes.
func ListAll(repo repository.Repo) []Summary {
	var reviews []Summary
	for _, revision := range repo.ListNotedRevisions(request.Ref) {
		review, err := GetSummary(repo, revision)
		if err == nil && review != nil {
			reviews = append(reviews, *review)
		}
	}
	return reviews
}

// ListOpen returns all reviews that are not yet incorporated into their target refs.
func ListOpen(repo repository.Repo) []Summary {
	var openReviews []Summary
	for _, review := range ListAll(repo) {
		if !review.Submitted {
			openReviews = append(openReviews, review)
		}
	}
	return openReviews
}

// GetCurrent returns the current, open code review.
//
// If there are multiple matching reviews, then an error is returned.
func GetCurrent(repo repository.Repo) (*Review, error) {
	reviewRef, err := repo.GetHeadRef()
	if err != nil {
		return nil, err
	}
	var matchingReviews []Summary
	for _, review := range ListOpen(repo) {
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
	return matchingReviews[0].Details()
}

// GetBuildStatusMessage returns a string of the current build-and-test status
// of the review, or "unknown" if the build-and-test status cannot be determined.
func (r *Review) GetBuildStatusMessage() string {
	statusMessage := "unknown"
	ciReport, err := ci.GetLatestCIReport(r.Reports)
	if err != nil {
		return fmt.Sprintf("unknown: %s", err)
	}
	if ciReport != nil {
		statusMessage = fmt.Sprintf("%s (%q)", ciReport.Status, ciReport.URL)
	}
	return statusMessage
}

// GetBuildStatusOutput returns a string of the current build-and-test output
// of the review, or "unknown" if the build-and-test output cannot be determined.
func (r *Review) GetBuildStatusOutput() string {
	statusOutput := ""
	ciReport, err := ci.GetLatestCIReport(r.Reports)
	if err != nil {
		return fmt.Sprintf("unknown: %s", err)
	}
	if ciReport != nil {
		statusOutput = ciReport.Output
	}
	return statusOutput
}

// GetAnalysesNotes returns all of the notes from the most recent static
// analysis run recorded in the git notes.
func (r *Review) GetAnalysesNotes() ([]analyses.Note, error) {
	latestAnalyses, err := analyses.GetLatestAnalysesReport(r.Analyses)
	if err != nil {
		return nil, err
	}
	if latestAnalyses == nil {
		return nil, fmt.Errorf("No analyses available")
	}
	return latestAnalyses.GetNotes()
}

// GetAnalysesMessage returns a string summarizing the results of the
// most recent static analyses.
func (r *Review) GetAnalysesMessage() string {
	latestAnalyses, err := analyses.GetLatestAnalysesReport(r.Analyses)
	if err != nil {
		return err.Error()
	}
	if latestAnalyses == nil {
		return "No analyses available"
	}
	status := latestAnalyses.Status
	if status != "" && status != analyses.StatusNeedsMoreWork {
		return status
	}
	analysesNotes, err := latestAnalyses.GetNotes()
	if err != nil {
		return err.Error()
	}
	if analysesNotes == nil {
		return "passed"
	}
	return fmt.Sprintf("%d warnings\n", len(analysesNotes))
	// TODO(ojarjur): Figure out the best place to display the actual notes
}

func prettyPrintJSON(jsonBytes []byte) (string, error) {
	var prettyBytes bytes.Buffer
	err := json.Indent(&prettyBytes, jsonBytes, "", "  ")
	if err != nil {
		return "", err
	}
	return prettyBytes.String(), nil
}

// GetJSON returns the pretty printed JSON for a review summary.
func (r *Summary) GetJSON() (string, error) {
	jsonBytes, err := json.Marshal(*r)
	if err != nil {
		return "", err
	}
	return prettyPrintJSON(jsonBytes)
}

// GetJSON returns the pretty printed JSON for a review.
func (r *Review) GetJSON() (string, error) {
	jsonBytes, err := json.Marshal(*r)
	if err != nil {
		return "", err
	}
	return prettyPrintJSON(jsonBytes)
}

// findLastCommit returns the later (newest) commit from the union of the provided commit
// and all of the commits that are referenced in the given comment threads.
func (r *Review) findLastCommit(latestCommit string, commentThreads []CommentThread) string {
	isLater := func(commit string) bool {
		if err := r.Repo.VerifyCommit(commit); err != nil {
			return false
		}
		if t, e := r.Repo.IsAncestor(latestCommit, commit); e == nil && t {
			return true
		}
		if t, e := r.Repo.IsAncestor(commit, latestCommit); e == nil && t {
			return false
		}
		ct, err := r.Repo.GetCommitTime(commit)
		if err != nil {
			return false
		}
		lt, err := r.Repo.GetCommitTime(latestCommit)
		if err != nil {
			return true
		}
		return ct > lt
	}
	updateLatest := func(commit string) {
		if commit == "" {
			return
		}
		if isLater(commit) {
			latestCommit = commit
		}
	}
	for _, commentThread := range commentThreads {
		comment := commentThread.Comment
		if comment.Location != nil {
			updateLatest(comment.Location.Commit)
		}
		updateLatest(r.findLastCommit(latestCommit, commentThread.Children))
	}
	return latestCommit
}

// GetHeadCommit returns the latest commit in a review.
func (r *Review) GetHeadCommit() (string, error) {
	if r.Request.ReviewRef == "" {
		return r.Revision, nil
	}

	if r.Submitted {
		// The review has already been submitted.
		// Go through the list of comments and find the last commented upon commit.
		return r.findLastCommit(r.Revision, r.Comments), nil
	}

	return r.Repo.ResolveRefCommit(r.Request.ReviewRef)
}

// GetBaseCommit returns the commit against which a review should be compared.
func (r *Review) GetBaseCommit() (string, error) {
	if r.Submitted {
		if r.Request.BaseCommit != "" {
			return r.Request.BaseCommit, nil
		}

		// This means the review has been submitted, but did not specify a base commit.
		// In this case, we have to treat the last parent commit as the base. This is
		// usually what we want, since merging a target branch into a feature branch
		// results in the previous commit to the feature branch being the first parent,
		// and the latest commit to the target branch being the second parent.
		return r.Repo.GetLastParent(r.Revision)
	}

	targetRefHead, err := r.Repo.ResolveRefCommit(r.Request.TargetRef)
	if err != nil {
		return "", err
	}
	leftHandSide := targetRefHead
	rightHandSide := r.Revision
	if r.Request.ReviewRef != "" {
		if reviewRefHead, err := r.Repo.ResolveRefCommit(r.Request.ReviewRef); err == nil {
			rightHandSide = reviewRefHead
		}
	}

	return r.Repo.MergeBase(leftHandSide, rightHandSide)
}

// GetDiff returns the diff for a review.
func (r *Review) GetDiff(diffArgs ...string) (string, error) {
	var baseCommit, headCommit string
	baseCommit, err := r.GetBaseCommit()
	if err == nil {
		headCommit, err = r.GetHeadCommit()
	}
	if err == nil {
		return r.Repo.Diff(baseCommit, headCommit, diffArgs...)
	}
	return "", err
}

// AddComment adds the given comment to the review.
func (r *Review) AddComment(c comment.Comment) error {
	commentNote, err := c.Write()
	if err != nil {
		return err
	}

	r.Repo.AppendNote(comment.Ref, r.Revision, commentNote)
	return nil
}
