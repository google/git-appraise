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
	"sort"
	"strings"

	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review/analyses"
	"github.com/google/git-appraise/review/ci"
	"github.com/google/git-appraise/review/comment"
	"github.com/google/git-appraise/review/gpg"
	"github.com/google/git-appraise/review/request"
)

const archiveRef = "refs/devtools/archives/reviews"

var emptyTree = repository.NewTree(map[string]repository.TreeChild{})

// CommentThread represents the tree-based hierarchy of comments.
//
// The Resolved field represents the aggregate status of the entire thread. If
// it is set to false, then it indicates that there is an unaddressed comment
// in the thread. If it is unset, then that means that the root comment is an
// FYI only, and that there are no unaddressed comments. If it is set to true,
// then that means that there are no unaddressed comments, and that the root
// comment has its resolved bit set to true.
type CommentThread struct {
	Hash     string             `json:"hash,omitempty"`
	Comment  comment.Comment    `json:"comment"`
	Original *comment.Comment   `json:"original,omitempty"`
	Edits    []*comment.Comment `json:"edits,omitempty"`
	Children []CommentThread    `json:"children,omitempty"`
	Resolved *bool              `json:"resolved,omitempty"`
	Edited   bool               `json:"edited,omitempty"`
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

type commentsByTimestamp []*comment.Comment

// Interface methods for sorting comment threads by timestamp
func (cs commentsByTimestamp) Len() int      { return len(cs) }
func (cs commentsByTimestamp) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs commentsByTimestamp) Less(i, j int) bool {
	return cs[i].Timestamp < cs[j].Timestamp
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

type summariesWithNewestRequestsFirst []Summary

// Interface methods for sorting review summaries in reverse chronological order
func (summaries summariesWithNewestRequestsFirst) Len() int { return len(summaries) }
func (summaries summariesWithNewestRequestsFirst) Swap(i, j int) {
	summaries[i], summaries[j] = summaries[j], summaries[i]
}
func (summaries summariesWithNewestRequestsFirst) Less(i, j int) bool {
	return summaries[i].Request.Timestamp > summaries[j].Request.Timestamp
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

// Verify verifies the signature on a comment.
func (thread *CommentThread) Verify() error {
	err := gpg.Verify(&thread.Comment)
	if err != nil {
		hash, _ := thread.Comment.Hash()
		return fmt.Errorf("verification of comment [%s] failed: %s", hash, err)
	}
	for _, child := range thread.Children {
		err = child.Verify()
		if err != nil {
			return err
		}
	}
	return nil
}

// mutableThread is an internal-only data structure used to store partially constructed comment threads.
type mutableThread struct {
	Hash     string
	Comment  comment.Comment
	Edits    []*comment.Comment
	Children []*mutableThread
}

// fixMutableThread is a helper method to finalize a mutableThread struct
// (partially constructed comment thread) as a CommentThread struct
// (fully constructed comment thread).
func fixMutableThread(mutableThread *mutableThread) CommentThread {
	var children []CommentThread
	edited := len(mutableThread.Edits) > 0
	for _, mutableChild := range mutableThread.Children {
		child := fixMutableThread(mutableChild)
		if (!edited) && child.Edited {
			edited = true
		}
		children = append(children, child)
	}
	comment := &mutableThread.Comment
	if len(mutableThread.Edits) > 0 {
		sort.Stable(commentsByTimestamp(mutableThread.Edits))
		comment = mutableThread.Edits[len(mutableThread.Edits)-1]
	}

	return CommentThread{
		Hash:     mutableThread.Hash,
		Comment:  *comment,
		Original: &mutableThread.Comment,
		Edits:    mutableThread.Edits,
		Children: children,
		Edited:   edited,
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
		if thread.Comment.Original != "" {
			original, ok := threadsByHash[thread.Comment.Original]
			if ok {
				original.Edits = append(original.Edits, &thread.Comment)
			}
		} else if thread.Comment.Parent == "" {
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

// getCommentsFromNotes parses the log-structured sequence of comments for a commit,
// and then builds the corresponding tree-structured comment threads.
func getCommentsFromNotes(repo repository.Repo, revision string, commentNotes []repository.Note) ([]CommentThread, *bool) {
	commentsByHash := comment.ParseAllValid(commentNotes)
	comments := buildCommentThreads(commentsByHash)
	resolved := updateThreadsStatus(comments)
	return comments, resolved
}

func getSummaryFromNotes(repo repository.Repo, revision string, requestNotes, commentNotes []repository.Note) (*Summary, error) {
	requests := request.ParseAllValid(requestNotes)
	if requests == nil {
		return nil, fmt.Errorf("Could not find any review requests for %q", revision)
	}
	sort.Stable(requestsByTimestamp(requests))
	reviewSummary := Summary{
		Repo:        repo,
		Revision:    revision,
		Request:     requests[len(requests)-1],
		AllRequests: requests,
	}
	comments, resolved := getCommentsFromNotes(repo, revision, commentNotes)
	reviewSummary.Comments = comments
	reviewSummary.Resolved = resolved
	return &reviewSummary, nil
}

func GetComments(repo repository.Repo, revision string) ([]CommentThread, error) {
	commentNotes := repo.GetNotes(comment.Ref, revision)
	c, _ := getCommentsFromNotes(repo, revision, commentNotes)
	return c, nil
}

// GetSummary returns the summary of the code review specified by its revision
// and the references which contain that reviews summary and comments.
//
// If no review request exists, the returned review summary is nil.
func GetSummaryViaRefs(repo repository.Repo, requestRef, commentRef, revision string) (*Summary, error) {
	if err := repo.VerifyCommit(revision); err != nil {
		return nil, fmt.Errorf("Could not find a commit named %q", revision)
	}
	requestNotes := repo.GetNotes(requestRef, revision)
	commentNotes := repo.GetNotes(commentRef, revision)
	summary, err := getSummaryFromNotes(repo, revision, requestNotes, commentNotes)
	if err != nil {
		return nil, err
	}
	currentCommit := revision
	if summary.Request.Alias != "" {
		currentCommit = summary.Request.Alias
	}

	if !summary.IsAbandoned() {
		submitted, err := repo.IsAncestor(currentCommit, summary.Request.TargetRef)
		if err != nil {
			return nil, err
		}
		summary.Submitted = submitted
	}
	return summary, nil
}

// GetSummary returns the summary of the specified code review.
//
// If no review request exists, the returned review summary is nil.
func GetSummary(repo repository.Repo, revision string) (*Summary, error) {
	return GetSummaryViaRefs(repo, request.Ref, comment.Ref, revision)
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

// IsAbandoned returns whether or not the given review has been abandoned.
func (r *Summary) IsAbandoned() bool {
	return r.Request.TargetRef == ""
}

// IsOpen returns whether or not the given review is still open (neither submitted nor abandoned).
func (r *Summary) IsOpen() bool {
	return !r.Submitted && !r.IsAbandoned()
}

// Verify returns whether or not a summary's comments are a) signed, and b)
/// that those signatures are verifiable.
func (r *Summary) Verify() error {
	err := gpg.Verify(&r.Request)
	if err != nil {
		return fmt.Errorf("couldn't verify request targeting: %q: %s",
			r.Request.TargetRef, err)
	}
	for _, thread := range r.Comments {
		err := thread.Verify()
		if err != nil {
			return err
		}
	}
	return nil
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

func getIsSubmittedCheck(repo repository.Repo) func(ref, commit string) bool {
	refCommitsMap := make(map[string]map[string]bool)

	getRefCommitsMap := func(ref string) map[string]bool {
		commitsMap, ok := refCommitsMap[ref]
		if ok {
			return commitsMap
		}
		commitsMap = make(map[string]bool)
		for _, commit := range repo.ListCommits(ref) {
			commitsMap[commit] = true
		}
		refCommitsMap[ref] = commitsMap
		return commitsMap
	}

	return func(ref, commit string) bool {
		return getRefCommitsMap(ref)[commit]
	}
}

func unsortedListAll(repo repository.Repo) []Summary {
	reviewNotesMap, err := repo.GetAllNotes(request.Ref)
	if err != nil {
		return nil
	}
	discussNotesMap, err := repo.GetAllNotes(comment.Ref)
	if err != nil {
		return nil
	}

	isSubmittedCheck := getIsSubmittedCheck(repo)
	var reviews []Summary
	for commit, notes := range reviewNotesMap {
		summary, err := getSummaryFromNotes(repo, commit, notes, discussNotesMap[commit])
		if err != nil {
			continue
		}
		if !summary.IsAbandoned() {
			summary.Submitted = isSubmittedCheck(summary.Request.TargetRef, summary.getStartingCommit())
		}
		reviews = append(reviews, *summary)
	}
	return reviews
}

// ListAll returns all reviews stored in the git-notes.
func ListAll(repo repository.Repo) []Summary {
	reviews := unsortedListAll(repo)
	sort.Stable(summariesWithNewestRequestsFirst(reviews))
	return reviews
}

// ListOpen returns all reviews that are not yet incorporated into their target refs.
func ListOpen(repo repository.Repo) []Summary {
	var openReviews []Summary
	for _, review := range unsortedListAll(repo) {
		if review.IsOpen() {
			openReviews = append(openReviews, review)
		}
	}
	sort.Stable(summariesWithNewestRequestsFirst(openReviews))
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

// GetCommentsJSON returns the pretty printed JSON for a slice of comment threads.
func GetCommentsJSON(cs []CommentThread) (string, error) {
	jsonBytes, err := json.Marshal(cs)
	if err != nil {
		return "", err
	}
	return prettyPrintJSON(jsonBytes)
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
func (r *Review) findLastCommit(startingCommit, latestCommit string, commentThreads []CommentThread) string {
	isLater := func(commit string) bool {
		if err := r.Repo.VerifyCommit(commit); err != nil {
			return false
		}
		if t, e := r.Repo.IsAncestor(latestCommit, commit); e == nil && t {
			return true
		}
		if t, e := r.Repo.IsAncestor(startingCommit, commit); e == nil && !t {
			return false
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
		updateLatest(r.findLastCommit(startingCommit, latestCommit, commentThread.Children))
	}
	return latestCommit
}

func (r *Summary) getStartingCommit() string {
	if r.Request.Alias != "" {
		return r.Request.Alias
	}
	return r.Revision
}

// LocalReviewRef finds the local ref that most closely matches the review ref.
//
// For refs outside of refs/heads/ (i.e. non-branches), the local ref has to
// match the review ref exactly.
//
// For branches, the following are checked in order:
// 1. An exact match in the local branches.
// 2. A matching branch in the `origin` remote.
// 3. A matching branch in one of the other remotes, checked in alphabetical order.
// 4. A matching branch in a matching fork (e.g. if the branch starts with "somefork/", then look under "refs/forks/somefork/refs/heads/somefork/...").
func (r *Review) LocalReviewRef() (string, error) {
	if !strings.HasPrefix(r.Request.ReviewRef, "refs/heads/") {
		return r.Request.ReviewRef, nil
	}
	if hasRef, err := r.Repo.HasRef(r.Request.ReviewRef); err != nil {
		return "", err
	} else if hasRef {
		return r.Request.ReviewRef, nil
	}
	branchName := strings.TrimPrefix(r.Request.ReviewRef, "refs/heads/")
	originBranch := "refs/remotes/origin/" + branchName
	if hasRef, err := r.Repo.HasRef(originBranch); err != nil {
		return "", err
	} else if hasRef {
		return originBranch, nil
	}
	remotes, err := r.Repo.Remotes()
	if err != nil {
		return "", err
	}
	for _, remote := range remotes {
		remoteBranch := "refs/remotes/" + remote + "/" + branchName
		if hasRef, err := r.Repo.HasRef(remoteBranch); err != nil {
			return "", err
		} else if hasRef {
			return remoteBranch, nil
		}
	}
	if strings.Index(branchName, "/") <= 0 {
		return r.Request.ReviewRef, nil
	}
	forkName := branchName[0:strings.Index(branchName, "/")]
	forkBranch := "refs/forks/" + forkName + "/refs/heads/" + branchName
	if hasRef, err := r.Repo.HasRef(forkBranch); err != nil {
		return "", err
	} else if hasRef {
		return forkBranch, nil
	}
	return r.Request.ReviewRef, nil
}

// GetHeadCommit returns the latest commit in a review.
func (r *Review) GetHeadCommit() (string, error) {
	currentCommit := r.getStartingCommit()
	if r.Request.ReviewRef == "" {
		return currentCommit, nil
	}

	if r.Submitted {
		// The review has already been submitted.
		// Go through the list of comments and find the last commented upon commit.
		return r.findLastCommit(currentCommit, currentCommit, r.Comments), nil
	}

	// It is possible that the review ref is no longer an ancestor of the starting
	// commit (e.g. if a rebase left us in a detached head), in which case we have to
	// find the head commit without using it.
	reviewRef, err := r.LocalReviewRef()
	if err != nil {
		return "", err
	}
	useReviewRef, err := r.Repo.IsAncestor(currentCommit, reviewRef)
	if err != nil {
		return "", err
	}
	if useReviewRef {
		return r.Repo.ResolveRefCommit(reviewRef)
	}

	return r.findLastCommit(currentCommit, currentCommit, r.Comments), nil
}

// GetBaseCommit returns the commit against which a review should be compared.
func (r *Review) GetBaseCommit() (string, error) {
	if !r.IsOpen() {
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

// ListCommits lists the commits included in a review.
func (r *Review) ListCommits() ([]string, error) {
	baseCommit, err := r.GetBaseCommit()
	if err != nil {
		return nil, err
	}
	headCommit, err := r.GetHeadCommit()
	if err != nil {
		return nil, err
	}
	return r.Repo.ListCommitsBetween(baseCommit, headCommit)
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

// Rebase performs an interactive rebase of the review onto its target ref.
//
// If the 'archivePrevious' argument is true, then the previous head of the
// review will be added to the 'refs/devtools/archives/reviews' ref prior
// to being rewritten. That ensures the review history is kept from being
// garbage collected.
func (r *Review) Rebase(archivePrevious bool) error {
	if archivePrevious {
		orig, err := r.GetHeadCommit()
		if err != nil {
			return err
		}
		if err := r.Repo.ArchiveRef(orig, archiveRef); err != nil {
			return err
		}
	}
	if err := r.Repo.SwitchToRef(r.Request.ReviewRef); err != nil {
		return err
	}

	err := r.Repo.RebaseRef(r.Request.TargetRef)
	if err != nil {
		return err
	}

	alias, err := r.Repo.GetCommitHash("HEAD")
	if err != nil {
		return err
	}
	r.Request.Alias = alias
	newNote, err := r.Request.Write()
	if err != nil {
		return err
	}
	return r.Repo.AppendNote(request.Ref, r.Revision, newNote)
}

// RebaseAndSign performs an interactive rebase of the review onto its
// target ref. It signs the result of the rebase as well as (re)signs
// the review request itself.
//
// If the 'archivePrevious' argument is true, then the previous head of the
// review will be added to the 'refs/devtools/archives/reviews' ref prior
// to being rewritten. That ensures the review history is kept from being
// garbage collected.
func (r *Review) RebaseAndSign(archivePrevious bool) error {
	if archivePrevious {
		orig, err := r.GetHeadCommit()
		if err != nil {
			return err
		}
		if err := r.Repo.ArchiveRef(orig, archiveRef); err != nil {
			return err
		}
	}
	if err := r.Repo.SwitchToRef(r.Request.ReviewRef); err != nil {
		return err
	}

	err := r.Repo.RebaseAndSignRef(r.Request.TargetRef)
	if err != nil {
		return err
	}

	alias, err := r.Repo.GetCommitHash("HEAD")
	if err != nil {
		return err
	}
	r.Request.Alias = alias

	key, err := r.Repo.GetUserSigningKey()
	if err != nil {
		return err
	}
	err = gpg.Sign(key, &r.Request)
	if err != nil {
		return err
	}

	newNote, err := r.Request.Write()
	if err != nil {
		return err
	}
	return r.Repo.AppendNote(request.Ref, r.Revision, newNote)
}

func wellKnownCommitForPath(repo repository.Repo, path string, archive bool) (string, error) {
	commitDetails := &repository.CommitDetails{
		Author:         "nobody",
		AuthorEmail:    "nobody",
		AuthorTime:     "100000000 +0000",
		Committer:      "nobody",
		CommitterEmail: "nobody",
		Time:           "100000000 +0000",
		Summary:        path,
	}
	commitHash, err := repo.CreateCommitWithTree(commitDetails, emptyTree)
	if err != nil {
		return "", err
	}
	if !archive {
		return commitHash, nil
	}
	if err := repo.ArchiveRef(commitHash, archiveRef); err != nil {
		return "", err
	}
	return commitHash, nil
}

func AddDetachedComment(repo repository.Repo, c *comment.Comment) error {
	path := c.Location.Path
	wellKnownCommit, err := wellKnownCommitForPath(repo, path, true)
	if err != nil {
		return fmt.Errorf("Failure finding the well-known commit for detached comments on %q: %v", path, err)
	}
	commentNote, err := c.Write()
	if err != nil {
		return err
	}
	return repo.AppendNote(comment.Ref, wellKnownCommit, commentNote)
}

func GetDetachedComments(repo repository.Repo, path string) ([]CommentThread, error) {
	wellKnownCommit, err := wellKnownCommitForPath(repo, path, false)
	if err != nil {
		return nil, fmt.Errorf("Failure finding the well-known commit for detached comments on %q: %v", path, err)
	}
	return GetComments(repo, wellKnownCommit)
}
