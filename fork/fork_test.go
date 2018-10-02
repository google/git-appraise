/*
Copyright 2018 Google Inc. All rights reserved.

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

package fork

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	localUserName  = "Local Git User For Test"
	localUserEmail = "test-local-user@example.com"
	forkUserName   = "Fork Git User For Test"
	forkUserEmail  = "test-fork-user@example.com"
)

func createTestRepository() (string, error) {
	dir, err := ioutil.TempDir("", "test-git-repo")
	if err != nil {
		return "", err
	}
	initCmd := exec.Command("git", "init")
	initCmd.Dir = dir
	err = initCmd.Run()
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func runGitCommandInRepo(repo string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run the git command %v. Stdout: %q, Stderr: %q, Error: %v", args, stdout.String(), stderr.String(), err)
	}
	return stdout.String(), nil
}

type testFork struct {
	Name  string
	User  string
	Email string
	Dir   string
}

func newTestFork(remoteRepo, forkName string) (f *testFork, err error) {
	forkUser := fmt.Sprintf("Test user for %s", forkName)
	forkEmail := fmt.Sprintf("%s-owner@example.com", forkName)
	dir, err := createTestRepository()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	remoteAddCmd := []string{"remote", "add", "origin", remoteRepo}
	pullCmd := []string{"pull", "origin", "master"}
	if _, err := runGitCommandInRepo(dir, remoteAddCmd); err != nil {
		return nil, fmt.Errorf("Failed to set up the remote for the test fork repo %q: %v", forkName, err)
	}
	if _, err := runGitCommandInRepo(dir, pullCmd); err != nil {
		return nil, fmt.Errorf("Failed to pull the contents of the remote for the test fork repo %q: %v", forkName, err)
	}
	if _, err := runGitCommandInRepo(dir, []string{"config", "--local", "--add", "user.name", forkUser}); err != nil {
		return nil, fmt.Errorf("Failed to set the git user name for the test fork repo %q: %v", forkName, err)
	}
	if _, err := runGitCommandInRepo(dir, []string{"config", "--local", "--add", "user.email", forkEmail}); err != nil {
		return nil, fmt.Errorf("Failed to set the git user email for the test fork repo %q: %v", forkName, err)
	}
	if _, err := runGitCommandInRepo(remoteRepo, []string{"appraise", "fork", "add", "-o", forkEmail, forkName, dir}); err != nil {
		return nil, fmt.Errorf("Failed to add the fork repo %q as a fork: %v", forkName, err)
	}
	return &testFork{
		Name:  forkName,
		User:  forkUser,
		Email: forkEmail,
		Dir:   dir,
	}, nil
}

func (f *testFork) Remove() error {
	return os.RemoveAll(f.Dir)
}

func (f *testFork) WriteFile(filename, contents string) error {
	return ioutil.WriteFile(filepath.Join(f.Dir, filename), []byte(contents), 0644)
}

func (f *testFork) RunGitCommand(args ...string) (string, error) {
	return runGitCommandInRepo(f.Dir, args)
}

func TestPullingFromForks(t *testing.T) {
	remoteRepo, err := createTestRepository()
	if err != nil {
		t.Fatalf("Failed to create the test remote repository: %v", err)
	}
	defer os.RemoveAll(remoteRepo)
	if err := ioutil.WriteFile(filepath.Join(remoteRepo, "README.md"), []byte("# Test Repository"), 0644); err != nil {
		t.Fatalf("Failed to initialize the contents of the test remote repo: %v", err)
	}
	if _, err := runGitCommandInRepo(remoteRepo, []string{"add", "README.md"}); err != nil {
		t.Fatalf("Failed to add the initial contents of the test remote repo: %v", err)
	}
	if _, err := runGitCommandInRepo(remoteRepo, []string{"commit", "-a", "-m", "Initial commit"}); err != nil {
		t.Fatalf("Failed to commit the initial contents of the test remote repo: %v", err)
	}

	localRepo, err := newTestFork(remoteRepo, "local")
	if err != nil {
		t.Fatalf("Failed to create the test local repository: %v", err)
	}
	defer localRepo.Remove()

	for i := 0; i < 10; i++ {
		forkName := fmt.Sprintf("fork-%d", i)
		forkRepo, err := newTestFork(remoteRepo, forkName)
		if err != nil {
			t.Fatalf("Failed to create the test fork repository %q: %v", forkName, err)
		}
		defer forkRepo.Remove()
		forkBranch := fmt.Sprintf("%s/test-branch", forkName)
		if _, err := forkRepo.RunGitCommand("checkout", "-b", forkBranch); err != nil {
			t.Fatalf("Failed to checkout the feature branch of the test fork %q: %v", forkName, err)
		}
		if err := forkRepo.WriteFile("Fork.md", fmt.Sprintf("# File written from the fork %q", forkName)); err != nil {
			t.Fatalf("Failed to initialize the contents of the test fork repo %q feature branch: %v", forkName, err)
		}
		if _, err := forkRepo.RunGitCommand("add", "Fork.md"); err != nil {
			t.Fatalf("Failed to add the contents of the feature branch of the test fork repo: %v", err)
		}
		if _, err := forkRepo.RunGitCommand("commit", "-a", "-m", fmt.Sprintf("Add a file from the fork %q", forkName)); err != nil {
			t.Fatalf("Failed to commit the feature branch of the test fork repo: %v", err)
		}
		if _, err := forkRepo.RunGitCommand("appraise", "request"); err != nil {
			t.Fatalf("Failed to create the review request in the fork repo: %v", err)
		}
	}

	if _, err := localRepo.RunGitCommand("appraise", "pull", "origin"); err != nil {
		t.Fatalf("Failed to pull the review metadata from the remote: %v", err)
	}
	if listed, err := localRepo.RunGitCommand("appraise", "list"); err != nil {
		t.Errorf("Error listing the open reviews: %v", err)
	} else if len(listed) == 0 {
		t.Errorf("Failed to list the open reviews")
	} else if !strings.Contains(listed, "Loaded 10 open reviews") {
		t.Errorf("Unexpected result from listing the open reviews from forks: %q", listed)
	}
}
