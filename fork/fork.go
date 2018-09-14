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

// Package fork contains the data structures used to represent repository forks.
//
// Forks are stored in a special ref `refs/devtools/forks`, with a tree that
// contains one subtree per fork named based on the hash of the fork's name.
//
// For example, if there is a fork named "omar", then it will have a SHA1 hash
// of "728d67f71db99d4768351e8e7807bfdd1807eadb", and be stored under the
// subtree named "7/2/8d67f71db99d4768351e8e7807bfdd1807eadb".
//
// Each fork subtree will contain one file named "NAME" and three directories
// named "URLS", "OWNERS", and "REFS".
package fork

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/git-appraise/repository"
)

const (
	Ref = "refs/devtools/forks"

	nameFile  = "NAME"
	ownersDir = "OWNERS"
	refsDir   = "REFS"
	urlsDir   = "URLS"
)

type Fork struct {
	Name   string   `json:"name,omitempty"`
	URLS   []string `json:"urls,omitempty"`
	Owners []string `json:"owners,omitempty"`
	Refs   []string `json:"refs,omitempty"`
}

func New(name, url string, owners []string) *Fork {
	return &Fork{
		Name:   name,
		URLS:   []string{url},
		Owners: owners,
		Refs: []string{
			fmt.Sprintf("refs/heads/%s/*", name),
			"refs/devtools/*",
			"refs/notes/devtools/*",
		},
	}
}

// Add adds the given fork to the repository, replacing any existing forks with the same name.
func Add(repo repository.Repo, fork *Fork) error {
	return errors.New("Not yet implemented.")
}

func forkNameFromPath(path string) (string, error) {
	// The path of a fork config item should look like:
	// <FORK_NAME_AS_SUBPATH>/(urls|owners|refs)/<OBJECT_HASH>,
	// ... where <FORK_NAME> may be split into multiple path components
	// to reduce the size of the git tree objects.
	//
	// For example, the fork named "omar" will be represented in
	// a subpath as "o/m/ar", whereas the form named "om" would
	// simply be "o/m".
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 3 {
		// This is not a valid fork entry
		return "", fmt.Errorf("Invalid fork configuration item: %q", path)
	}
	return strings.Join(pathParts[0:len(pathParts)-2], ""), nil
}

// Get gets the given fork from the repository.
func Get(repo repository.Repo, name string) (*Fork, error) {
	return nil, errors.New("Not yet implemented.")
}

// Delete deletes the given fork from the repository.
func Delete(repo repository.Repo, name string) error {
	return errors.New("Not yet implemented.")
}

func forkHashFromPath(path string) (string, error) {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 4 {
		return "", fmt.Errorf("Malformed fork config file path: %q", path)
	}
	return strings.Join(pathParts[0:3], ""), nil
}

// List lists the forks recorded in the repository.
func List(repo repository.Repo) ([]*Fork, error) {
	hasForks, err := repo.HasRef(Ref)
	if err != nil {
		return nil, err
	}
	if !hasForks {
		return nil, nil
	}
	forksTree, err := repo.ShowAll(Ref, "/")
	if err != nil {
		return nil, err
	}
	forkHashesMap := make(map[string]*Fork)
	for path, contents := range forksTree {
		forkHash, err := forkHashFromPath(path)
		if err != nil {
			continue
		}
		fork, ok := forkHashesMap[forkHash]
		if !ok {
			fork = &Fork{}
			forkHashesMap[forkHash] = fork
		}
		pathSuffixParts := strings.Split(path, "/")[3:]
		if len(pathSuffixParts) == 1 && pathSuffixParts[0] == nameFile {
			fork.Name = contents
		}
		if len(pathSuffixParts) != 2 {
			// An unrecognized fork config entry
			continue
		}
		if pathSuffixParts[0] == ownersDir {
			fork.Owners = append(fork.Owners, contents)
		}
		if pathSuffixParts[0] == refsDir {
			fork.Refs = append(fork.Refs, contents)
		}
		if pathSuffixParts[0] == urlsDir {
			fork.URLS = append(fork.URLS, contents)
		}
	}
	var forks []*Fork
	for _, fork := range forkHashesMap {
		forks = append(forks, fork)
	}
	return forks, nil
}

func Pull(repo repository.Repo, fork *Fork) error {
	return errors.New("Not yet implemented.")
}
