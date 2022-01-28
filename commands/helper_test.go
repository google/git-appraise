package commands

import (
	"os"
	"testing"
)

func TestGetDate(t testing.T) {
	_, err := GetDate("aaaa")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	_, err = GetDate("1488452400")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	os.Setenv("GIT_AUTHOR_DATE", "1488452400")
	_, err = GetDate("")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	os.Setenv("GIT_COMMITTER_DATE", "1488452400")
	os.Setenv("GIT_AUTHOR_DATE", "")
	_, err = GetDate("")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
