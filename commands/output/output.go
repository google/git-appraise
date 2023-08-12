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

// Package output contains helper methods for pretty-printing code reviews.
package output

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/git-appraise/fork"
	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review"
)

const (
	// Template for printing the summary of a list of reviews.
	reviewListTemplate = `Loaded %d reviews:
`
	// Template for printing the summary of a list of open reviews.
	openReviewListTemplate = `Loaded %d open reviews:
`
	// Template for printing the summary of a list of comment threads.
	commentListTemplate = `Loaded %d comment threads:
`
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
	// Template for printing the summary of the forks.
	forksSummaryTemplate = `%d forks
`
	// Template for printing the summary of a code review.
	forkTemplate = `%q
  owners: %q
  urls: %q
`
)

// getStatusString returns a human friendly string encapsulating both the review's
// resolved status, and its submitted status.
func getStatusString(r *review.Summary) string {
	if r.Resolved == nil && r.Submitted {
		return "tbr"
	}
	if r.Resolved == nil {
		return "pending"
	}
	if *r.Resolved && r.Submitted {
		return "submitted"
	}
	if *r.Resolved {
		return "accepted"
	}
	if r.Submitted {
		return "danger"
	}
	if r.Request.TargetRef == "" {
		return "abandon"
	}
	return "rejected"
}

// PrintSummaries prints single-line summaries of a slice of reviews.
func PrintSummaries(reviews []review.Summary, listAll bool) {
	if listAll {
		fmt.Printf(reviewListTemplate, len(reviews))
	} else {
		fmt.Printf(openReviewListTemplate, len(reviews))
	}
	for _, r := range reviews {
		PrintSummary(&r)
	}
}

// PrintSummary prints a single-line summary of a review.
func PrintSummary(r *review.Summary) {
	statusString := getStatusString(r)
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
func showThread(repo repository.Repo, thread review.CommentThread, indent string) error {
	comment := thread.Comment
	if comment.Location != nil && comment.Location.Path != "" && comment.Location.Range != nil && comment.Location.Range.StartLine > 0 {
		contents, err := repo.Show(comment.Location.Commit, comment.Location.Path)
		if err != nil {
			return err
		}
		lines := strings.Split(contents, "\n")
		err = comment.Location.Check(repo)
		if err != nil {
			return err
		}
		if comment.Location.Range.StartLine <= uint32(len(lines)) {
			firstLine := comment.Location.Range.StartLine
			lastLine := comment.Location.Range.EndLine

			if firstLine == 0 {
				firstLine = 1
			}

			if lastLine == 0 {
				lastLine = firstLine
			}

			if lastLine == firstLine {
				minLine := int(lastLine) - int(contextLineCount)
				if minLine <= 0 {
					minLine = 1
				}
				firstLine = uint32(minLine)
			}

			fmt.Printf(commentLocationTemplate, indent, comment.Location.Path, comment.Location.Commit)
			fmt.Println(indent + "|" + strings.Join(lines[firstLine-1:lastLine], "\n"+indent+"|"))
		}
	}
	return showSubThread(repo, thread, indent)
}

// showSubThread prints the given comment (sub)thread, indented by the given prefix string.
func showSubThread(repo repository.Repo, thread review.CommentThread, indent string) error {
	statusString := "fyi"
	if thread.Resolved != nil {
		if *thread.Resolved {
			statusString = "lgtm"
		} else {
			statusString = "needs work"
		}
	}
	comment := thread.Comment
	threadHash := thread.Hash
	timestamp := reformatTimestamp(comment.Timestamp)
	commentSummary := fmt.Sprintf(indent+commentTemplate, threadHash, comment.Author, timestamp, statusString, comment.Description)
	indent = indent + "  "
	indentedSummary := strings.Replace(commentSummary, "\n", "\n"+indent, -1)
	fmt.Println(indentedSummary)
	for _, child := range thread.Children {
		err := showSubThread(repo, child, indent)
		if err != nil {
			return err
		}
	}
	return nil
}

// printAnalyses prints the static analysis results for the latest commit in the review.
func printAnalyses(r *review.Review) {
	fmt.Println("  analyses: ", r.GetAnalysesMessage())
}

// printCommentsWithIndent prints all of the comment threads with the given indent before each line.
func printCommentsWithIndent(repo repository.Repo, c []review.CommentThread, indent string) error {
	for _, thread := range c {
		err := showThread(repo, thread, indent)
		if err != nil {
			return err
		}
	}
	return nil
}

// PrintComments prints all of the given comment threads.
func PrintComments(repo repository.Repo, c []review.CommentThread) error {
	fmt.Printf(commentListTemplate, len(c))
	return printCommentsWithIndent(repo, c, "  ")
}

// printComments prints all of the comments for the review, with snippets of the preceding source code.
func printComments(r *review.Review) error {
	fmt.Printf(commentSummaryTemplate, len(r.Comments))
	return printCommentsWithIndent(r.Repo, r.Comments, "    ")
}

// PrintDetails prints a multi-line overview of a review, including all comments.
func PrintDetails(r *review.Review) error {
	PrintSummary(r.Summary)
	fmt.Printf(reviewDetailsTemplate, r.Request.ReviewRef, r.Request.TargetRef,
		strings.Join(r.Request.Reviewers, ", "),
		r.Request.Requester, r.GetBuildStatusMessage())
	printAnalyses(r)
	if err := printComments(r); err != nil {
		return err
	}
	return nil
}

// PrintCommentsJSON pretty prints the given review in JSON format.
func PrintCommentsJSON(c []review.CommentThread) error {
	json, err := review.GetCommentsJSON(c)
	if err != nil {
		return err
	}
	fmt.Println(json)
	return nil
}

// PrintJSON pretty prints the given review in JSON format.
func PrintJSON(r *review.Review) error {
	json, err := r.GetJSON()
	if err != nil {
		return err
	}
	fmt.Println(json)
	return nil
}

// PrintDiff prints the diff of the review.
func PrintDiff(r *review.Review, diffArgs ...string) error {
	diff, err := r.GetDiff(diffArgs...)
	if err != nil {
		return err
	}
	fmt.Println(diff)
	return nil
}

// PrintForks prints the list of forks.
func PrintForks(forks []*fork.Fork) {
	fmt.Printf(forksSummaryTemplate, len(forks))
	for _, f := range forks {
		fmt.Printf(forkTemplate, f.Name, f.Owners, f.URLS)
	}
}
