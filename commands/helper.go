package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetDate(timestamp string) (*time.Time, error) {
	gitAuthorDate := os.Getenv("GIT_AUTHOR_DATE")
	gitCommiterDate := os.Getenv("GIT_COMMITTER_DATE")
	layouts := [...]string{time.RFC1123Z, time.RFC3339,
		"2006-01-02 15:04:05", "2006.01.02T15:04:05",
		"2005.04.07 15:04:05", "01/02/2006T15:04:05",
		"01/02/2006 15:04:05", "02.01.2006T15:04:05",
		"02.01.2006 15:04:05",
	}

	realGetDate := func(timestampStr string) (*time.Time, error) {
		var date time.Time
		var err error

		// <unix timestamp> <time zone offset>
		ary := strings.Split(timestampStr, " ")
		if len(ary) == 2 {
			unixTimestamp := ary[0]
			intTimestamp, innerErr := strconv.ParseInt(unixTimestamp, 10, 64)
			if innerErr == nil {
				timeZoneOffset := ary[1]
				var loc time.Time
				loc, innerErr = time.Parse("-0700", timeZoneOffset)
				if innerErr != nil {
					return nil, fmt.Errorf("unsupported timestamp format: %s", timestampStr)
				}
				tmpDate := time.Unix(intTimestamp, 0)
				date = tmpDate.In(loc.Location())
				return &date, nil
			}
		}

		for _, layout := range layouts {
			date, err = time.Parse(layout, timestampStr)
			if err == nil {
				break
			} else {
				fmt.Println(err)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("unsupported timestamp format: %s", timestampStr)
		}
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
