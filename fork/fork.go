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
	"crypto/sha1"
	"fmt"

	"github.com/google/git-appraise/repository"
)

const (
	Ref = "refs/devtools/forks"

	nameFilePath  = "NAME"
	ownersDirPath = "OWNERS"
	refsDirPath   = "REFS"
	urlsDirPath   = "URLS"
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

func forkPathFromName(forkName string) []string {
	forkHash := fmt.Sprintf("%x", sha1.Sum([]byte(forkName)))
	return []string{forkHash[0:1], forkHash[1:2], forkHash[2:]}
}

func forkPath(fork *Fork) []string {
	return forkPathFromName(fork.Name)
}

func encodeListAsHashedFiles(items []string) *repository.Tree {
	t := repository.NewTree()
	for _, item := range items {
		blob := &repository.Blob{Contents: item}
		path := fmt.Sprintf("%x", sha1.Sum([]byte(item)))
		t.Add(path, blob)
	}
	return t
}

// Add adds the given fork to the repository, replacing any existing forks with the same name.
func Add(repo repository.Repo, fork *Fork) error {
	t := repository.NewTree()
	t.Add(nameFilePath, &repository.Blob{Contents: fork.Name})
	t.Add(urlsDirPath, encodeListAsHashedFiles(fork.URLS))
	t.Add(ownersDirPath, encodeListAsHashedFiles(fork.Owners))
	t.Add(refsDirPath, encodeListAsHashedFiles(fork.Refs))

	var previousCommitHash string
	var forksTree *repository.Tree
	hasRef, err := repo.HasRef(Ref)
	if err != nil {
		return fmt.Errorf("failure checking the existence of the forks ref: %v", err)
	}
	if hasRef {
		previousCommitHash, err = repo.GetCommitHash(Ref)
		if err != nil {
			return fmt.Errorf("failure reading the forks ref commit: %v", err)
		}
		forksTree, err = repo.ReadTree(previousCommitHash)
		if err != nil {
			return fmt.Errorf("failure reading the forks ref: %v", err)
		}
	} else {
		forksTree = repository.NewTree()
	}

	currentLevel := forksTree
	path := forkPath(fork)
	for len(path) > 1 {
		childName := path[0]
		path = path[1:]
		var childTree *repository.Tree
		childObj, ok := currentLevel.Get(childName)
		if ok {
			childTree, ok = childObj.(*repository.Tree)
		}
		if !ok {
			childTree = repository.NewTree()
			currentLevel.Add(childName, childTree)
		}
		currentLevel = childTree
	}
	currentLevel.Add(path[0], t)
	var commitParents []string
	if previousCommitHash != "" {
		commitParents = append(commitParents, previousCommitHash)
	}
	commitHash, err := repo.CreateCommit(forksTree, commitParents, fmt.Sprintf("Adding the fork: %q", fork.Name))
	if err != nil {
		return fmt.Errorf("failure creating a commit to add the fork %q", fork.Name)
	}
	return repo.SetRef(Ref, commitHash, previousCommitHash)
}

// Delete deletes the given fork from the repository.
func Delete(repo repository.Repo, name string) error {
	if hasRef, err := repo.HasRef(Ref); err != nil {
		return fmt.Errorf("failure checking the existence of the forks ref: %v", err)
	} else if !hasRef {
		return fmt.Errorf("the specified fork, %q, does not exist", name)
	}
	previousCommitHash, err := repo.GetCommitHash(Ref)
	if err != nil {
		return fmt.Errorf("failure reading the forks ref commit: %v", err)
	}
	forksTree, err := repo.ReadTree(previousCommitHash)
	if err != nil {
		return fmt.Errorf("failure reading the forks ref: %v", err)
	}

	currentLevel := forksTree
	path := forkPathFromName(name)
	for len(path) > 1 {
		childName := path[0]
		path = path[1:]
		childObj, ok := currentLevel.Get(childName)
		if !ok {
			return fmt.Errorf("the specified fork, %q, does not exist", name)
		}
		childTree, ok := childObj.(*repository.Tree)
		if !ok {
			return fmt.Errorf("the specified fork, %q, does not exist", name)
		}
		currentLevel = childTree
	}
	currentLevel.Delete(path[0])
	commitHash, err := repo.CreateCommit(forksTree, []string{previousCommitHash},
		fmt.Sprintf("Deleting the fork: %q", name))
	if err != nil {
		return fmt.Errorf("failure creating a commit to delete the fork %q", name)
	}
	return repo.SetRef(Ref, commitHash, previousCommitHash)
}

func readHashedFiles(t *repository.Tree) []string {
	var results []string
	for path, obj := range t.Contents() {
		blob, ok := obj.(*repository.Blob)
		if !ok {
			// we are not interested in subdirectories
			continue
		}
		contents := blob.Contents
		hash := fmt.Sprintf("%x", sha1.Sum([]byte(contents)))
		if path != hash {
			// we are not interested in non-hash-named files
			continue
		}
		results = append(results, contents)
	}
	return results
}

func parseForkTree(t *repository.Tree) (*Fork, error) {
	nameFile, ok := t.Get(nameFilePath)
	if !ok {
		return nil, fmt.Errorf("fork missing a NAME file")
	}
	nameBlob, ok := nameFile.(*repository.Blob)
	if !ok {
		return nil, fmt.Errorf("fork NAME file is not actually a file")
	}
	ownersFile, ok := t.Get(ownersDirPath)
	if !ok {
		return nil, fmt.Errorf("fork missing an OWNERS subdirectory")
	}
	ownersDir, ok := ownersFile.(*repository.Tree)
	if !ok {
		return nil, fmt.Errorf("fork OWNERS subdirectory is not actually a directory")
	}
	refsFile, ok := t.Get(refsDirPath)
	if !ok {
		return nil, fmt.Errorf("fork missing a REFS subdirectory")
	}
	refsDir, ok := refsFile.(*repository.Tree)
	if !ok {
		return nil, fmt.Errorf("fork REFS subdirectory is not actually a directory")
	}
	urlsFile, ok := t.Get(urlsDirPath)
	if !ok {
		return nil, fmt.Errorf("fork missing a URLS subdirectory")
	}
	urlsDir, ok := urlsFile.(*repository.Tree)
	if !ok {
		return nil, fmt.Errorf("fork URLS subdirectory is not actually a directory")
	}
	fork := &Fork{
		Name:   nameBlob.Contents,
		Owners: readHashedFiles(ownersDir),
		Refs:   readHashedFiles(refsDir),
		URLS:   readHashedFiles(urlsDir),
	}
	return fork, nil
}

// Flatten the given number of levels of the specified tree.
//
// The resulting value is a map from paths in the tree to the nested tree
// at each path, and is stored in the `results` parameter.
//
// Any child objects that are not subtrees at the specified level are ignored.
func flattenTree(t *repository.Tree, levelsToFlatten int, pathPrefix string, results map[string]*repository.Tree) {
	if levelsToFlatten < 1 {
		results[pathPrefix] = t
		return
	}
	for path, obj := range t.Contents() {
		childTree, ok := obj.(*repository.Tree)
		if !ok {
			continue
		}
		flattenTree(childTree, levelsToFlatten-1, pathPrefix+path, results)
	}
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
	forksTree, err := repo.ReadTree(Ref)
	if err != nil {
		return nil, err
	}
	forkTrees := make(map[string]*repository.Tree)
	flattenTree(forksTree, 3, "", forkTrees)
	forkHashesMap := make(map[string]*Fork)
	for forkPath, forkTree := range forkTrees {
		fork, err := parseForkTree(forkTree)
		if err != nil {
			continue
		}
		forkHashesMap[forkPath] = fork
	}
	var forks []*Fork
	for _, fork := range forkHashesMap {
		forks = append(forks, fork)
	}
	return forks, nil
}

func Pull(repo repository.Repo, fork *Fork) error {
	for _, url := range fork.URLS {
		var refSpecs []string
		for _, ref := range fork.Refs {
			refSpecs = append(refSpecs, fmt.Sprintf("+%s:refs/forks/%s/%s", ref, fork.Name, ref))
		}
		if err := repo.Fetch(url, refSpecs); err != nil {
			return err
		}
	}
	return nil
}
