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
package fork

import (
	"errors"
	"fmt"

	"github.com/google/git-appraise/repository"
)

const Ref = "refs/devtools/forks"

type Fork struct {
	Name       string   `json:"name,omitempty"`
	URL        string   `json:"url,omitempty"`
	Owners     []string `json:"owners,omitempty"`
	FetchSpecs []string `json:"fetchSpecs,omitempty"`
}

func New(name, url string, owners []string) *Fork {
	return &Fork{
		Name:   name,
		URL:    url,
		Owners: owners,
		FetchSpecs: []string{
			fmt.Sprintf("refs/heads/%s/*:refs/forks/%s/heads/%s/*", name, name, name),
			fmt.Sprintf("refs/devtools/*:refs/forks/%s/devtools/%s/*", name, name),
			fmt.Sprintf("refs/notes/devtools/*:refs/notes/forks/%s/devtools/*", name),
		},
	}
}

// Add adds the given fork to the repository, replacing any existing forks with the same name.
func Add(repo repository.Repo, fork *Fork) error {
	return errors.New("Not yet implemented.")
}

// Get gets the given fork from the repository.
func Get(repo repository.Repo, name string) (*Fork, error) {
	return nil, errors.New("Not yet implemented.")
}

// Delete delets the given fork from the repository.
func Delete(repo repository.Repo, name string) error {
	return errors.New("Not yet implemented.")
}

// List lists the forks recorded in the repository.
func List(repo repository.Repo) ([]*Fork, error) {
	return nil, errors.New("Not yet implemented.")
}

func Pull(repo repository.Repo, fork *Fork) error {
	return errors.New("Not yet implemented.")
}
