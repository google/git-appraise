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

package commands

import (
	"fmt"
	"review"
	"strconv"
	"time"
)

const (
	showTemplate = `[%s] %s
  "%s"
`
	threadTemplate = `  [%s] %s %s "%s"
`
)

func reformatTimestamp(timestamp string) string {
	parsedTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		// The timestamp is an unexpected format, so leave it alone
		return timestamp
	}
	t := time.Unix(parsedTimestamp, 0)
	return t.Format(time.UnixDate)
}

func showThread(thread review.CommentThread, indent string) {
	comment := thread.Comment
	timestamp := reformatTimestamp(comment.Timestamp)
	statusString := "fyi"
	if comment.Resolved != nil {
		if *comment.Resolved {
			statusString = "lgtm"
		} else {
			statusString = "needs work"
		}
	}
	fmt.Printf(threadTemplate, timestamp, comment.Author, statusString, comment.Description)
	for _, child := range thread.Children {
		showThread(child, indent+"  ")
	}
}

func showReview(args []string) {
	r, err := review.GetCurrent()
	if err != nil {
		fmt.Printf("Failed to load the current review: %v\n", err)
		return
	}
	if r == nil {
		fmt.Println("There is no current review.")
		return
	}
	statusString := "pending"
	if r.Resolved != nil {
		if *r.Resolved {
			statusString = "accepted"
		} else {
			statusString = "rejected"
		}
	}
	fmt.Printf(showTemplate, statusString, r.Revision, r.Request.Description)
	for _, thread := range r.Comments {
		showThread(thread, "")
	}
}

var showCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s show\n", arg0)
	},
	RunMethod: func(args []string) {
		showReview(args)
	},
}
