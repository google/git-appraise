package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"unicode"
)

func GetDate(timestamp string) (*time.Time, error) {
	gitAuthorDate := os.Getenv("GIT_AUTHOR_DATE")
	gitCommiterDate := os.Getenv("GIT_COMMITTER_DATE")

	realGetDate := func(timestampStr string) (*time.Time, error) {
		for _, char := range timestampStr {
			if !unicode.IsDigit(char) {
				return nil, fmt.Errorf("Invalid timestamp: %s", timestampStr)
			}
		}

		intTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return nil, err
		}
		date := time.Unix(intTimestamp, 0)
		return &date, nil
	}

	if len(timestamp) > 0 {
		return realGetDate(timestamp)
	} else if len(gitAuthorDate) > 0 {
		return realGetDate(gitAuthorDate)
	} else if len(gitCommiterDate) > 0 {
		return realGetDate(gitCommiterDate)
	}
	return nil, nil
}

func FormatDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return strconv.FormatInt(date.Unix(), 10)
}
