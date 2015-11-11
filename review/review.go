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
	"strconv"
	"strings"
	"time"
)

const (
	// Template for printing the summary of a code review.
	reviewSummaryTemplate = `[%s] %.12s
  %s
`
	// Template for printing the summary of a code review.
	reviewDetailsTemplate = `  %q -> %q
  reviewers: %q
  requester: %q
  build status: %s
`
	// Template for printing the location of an inline comment
	commentLocationTemplate = `%s%q@%.12s
`
	// Template for printing a single comment.
	commentTemplate = `comment: %s
author: %s
time:   %s
status: %s
%s`
	// Template for displaying the summary of the comment threads for a review
	commentSummaryTemplate = `  comments (%d threads):
`
	// Number of lines of context to print for inline comments
	contextLineCount = 5
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
	Repo      repository.Repo   `json:"-"`
	Revision  string            `json:"revision"`
	Request   request.Request   `json:"request"`
	Comments  []CommentThread   `json:"comments,omitempty"`
	Resolved  *bool             `json:"resolved,omitempty"`
	Submitted bool              `json:"submitted"`
	Reports   []ci.Report       `json:"reports,omitempty"`
	Analyses  []analyses.Report `json:"analyses,omitempty"`
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
	commentNotes := r.Repo.GetNotes(comment.Ref, r.Revision)
	commentsByHash := comment.ParseAllValid(commentNotes)
	return buildCommentThreads(commentsByHash)
}

// Get returns the specified code review.
//
// If no review request exists, the returned review is nil.
func Get(repo repository.Repo, revision string) *Review {
	requestNotes := repo.GetNotes(request.Ref, revision)
	requests := request.ParseAllValid(requestNotes)
	if requests == nil {
		return nil
	}
	review := Review{
		Repo:     repo,
		Revision: revision,
		Request:  requests[len(requests)-1],
	}
	review.Comments = review.loadComments()
	review.Resolved = updateThreadsStatus(review.Comments)
	review.Submitted = repo.IsAncestor(revision, review.Request.TargetRef)
	currentCommit, err := review.GetHeadCommit()
	if err == nil {
		review.Reports = ci.ParseAllValid(repo.GetNotes(ci.Ref, currentCommit))
		review.Analyses = analyses.ParseAllValid(repo.GetNotes(analyses.Ref, currentCommit))
	}
	return &review
}

// ListAll returns all reviews stored in the git-notes.
func ListAll(repo repository.Repo) []Review {
	var reviews []Review
	for _, revision := range repo.ListNotedRevisions(request.Ref) {
		review := Get(repo, revision)
		if review != nil {
			reviews = append(reviews, *review)
		}
	}
	return reviews
}

// ListOpen returns all reviews that are not yet incorporated into their target refs.
func ListOpen(repo repository.Repo) []Review {
	var openReviews []Review
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
	reviewRef := repo.GetHeadRef()
	var matchingReviews []Review
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
	r := &matchingReviews[0]
	return r, nil
}

// PrintSummary prints a single-line summary of a review.
func (r *Review) PrintSummary() {
	statusString := "pending"
	if r.Resolved != nil {
		if *r.Resolved {
			if r.Submitted {
				statusString = "submitted"
			} else {
				statusString = "accepted"
			}
		} else {
			if r.Submitted {
				statusString = "danger"
			} else {
				statusString = "rejected"
			}
		}
	} else if r.Submitted {
		statusString = "tbr"
	}
	indentedDescription := strings.Replace(r.Request.Description, "\n", "\n  ", -1)
	fmt.Printf(reviewSummaryTemplate, statusString, r.Revision, indentedDescription)
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

// showThread prints the detailed output for an entire comment thread.
func (r *Review) showThread(thread CommentThread) error {
	comment := thread.Comment
	indent := "    "
	if comment.Location != nil && comment.Location.Path != "" && comment.Location.Range != nil && comment.Location.Range.StartLine > 0 {
		contents, err := r.Repo.Show(comment.Location.Commit, comment.Location.Path)
		if err != nil {
			return err
		}
		lines := strings.Split(contents, "\n")
		if comment.Location.Range.StartLine <= uint32(len(lines)) {
			var firstLine uint32 = 0
			lastLine := comment.Location.Range.StartLine
			if lastLine > contextLineCount {
				firstLine = lastLine - contextLineCount
			}
			fmt.Printf(commentLocationTemplate, indent, comment.Location.Path, comment.Location.Commit)
			fmt.Println(indent + "|" + strings.Join(lines[firstLine:lastLine], "\n"+indent+"|"))
		}
	}
	return r.showSubThread(thread, indent)
}

// showSubThread prints the given comment (sub)thread, indented by the given prefix string.
func (r *Review) showSubThread(thread CommentThread, indent string) error {
	statusString := "fyi"
	if thread.Resolved != nil {
		if *thread.Resolved {
			statusString = "lgtm"
		} else {
			statusString = "needs work"
		}
	}
	comment := thread.Comment
	threadHash, err := comment.Hash()
	if err != nil {
		return err
	}

	timestamp := reformatTimestamp(comment.Timestamp)
	commentSummary := fmt.Sprintf(indent+commentTemplate, threadHash, comment.Author, timestamp, statusString, comment.Description)
	indent = indent + "  "
	indentedSummary := strings.Replace(commentSummary, "\n", "\n"+indent, -1)
	fmt.Println(indentedSummary)
	for _, child := range thread.Children {
		err := r.showSubThread(child, indent)
		if err != nil {
			return err
		}
	}
	return nil
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
	reportResults, err := latestAnalyses.GetLintReportResult()
	if err != nil {
		return nil, err
	}
	var analysesNotes []analyses.Note
	for _, reportResult := range reportResults {
		analysesNotes = append(analysesNotes, reportResult.Notes...)
	}
	return analysesNotes, nil
}

// printAnalyses prints the static analysis results for the latest commit in the review.
func (r *Review) printAnalyses() {
	analysesNotes, err := r.GetAnalysesNotes()
	if err != nil {
		fmt.Println("  analyses: ", err)
		return
	}
	if analysesNotes == nil {
		fmt.Println("  analyses: passed")
		return
	}
	fmt.Printf("  analyses: %d warnings\n", len(analysesNotes))
	// TODO(ojarjur): Print the actual notes
}

// printComments prints all of the comments for the review, with snippets of the preceding source code.
func (r *Review) printComments() error {
	fmt.Printf(commentSummaryTemplate, len(r.Comments))
	for _, thread := range r.Comments {
		err := r.showThread(thread)
		if err != nil {
			return err
		}
	}
	return nil
}

// PrintDetails prints a multi-line overview of a review, including all comments.
func (r *Review) PrintDetails() error {
	r.PrintSummary()
	fmt.Printf(reviewDetailsTemplate, r.Request.ReviewRef, r.Request.TargetRef,
		strings.Join(r.Request.Reviewers, ", "),
		r.Request.Requester, r.GetBuildStatusMessage())
	r.printAnalyses()
	if err := r.printComments(); err != nil {
		return err
	}
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

// findLastCommit returns the later (newest) commit from the union of the provided commit
// and all of the commits that are referenced in the given comment threads.
func (r *Review) findLastCommit(latestCommit string, commentThreads []CommentThread) string {
	isLater := func(commit string) bool {
		if r.Repo.IsAncestor(latestCommit, commit) {
			return true
		}
		if r.Repo.IsAncestor(commit, latestCommit) {
			return false
		}
		return r.Repo.GetCommitTime(commit) > r.Repo.GetCommitTime(latestCommit)
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

	targetRefHead, err := r.Repo.ResolveRefCommit(r.Request.TargetRef)
	if err != nil {
		return "", err
	}

	if r.Repo.IsAncestor(r.Revision, targetRefHead) {
		// The review has already been submitted.
		// Go through the list of comments and find the last commented upon commit.
		return r.findLastCommit(r.Revision, r.Comments), nil
	}

	return r.Repo.ResolveRefCommit(r.Request.ReviewRef)
}

// GetBaseCommit returns the commit against which a review should be compared.
func (r *Review) GetBaseCommit() (string, error) {
	targetRefHead, err := r.Repo.ResolveRefCommit(r.Request.TargetRef)
	if err != nil {
		return "", err
	}
	leftHandSide := targetRefHead

	if r.Repo.IsAncestor(r.Revision, leftHandSide) {
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

	rightHandSide := r.Revision
	if r.Request.ReviewRef != "" {
		if reviewRefHead, err := r.Repo.ResolveRefCommit(r.Request.ReviewRef); err == nil {
			rightHandSide = reviewRefHead
		}
	}

	return r.Repo.MergeBase(leftHandSide, rightHandSide), nil
}

// PrintDiff displays the diff for a review.
func (r *Review) PrintDiff(diffArgs ...string) error {
	var baseCommit, headCommit string
	baseCommit, err := r.GetBaseCommit()
	if err == nil {
		headCommit, err = r.GetHeadCommit()
	}
	if err == nil {
		fmt.Println(r.Repo.Diff(baseCommit, headCommit, diffArgs...))
	}
	return err
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
