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
	"strings"

	"github.com/google/git-appraise/repository"
	"github.com/google/git-appraise/review/comment"
	"github.com/google/git-appraise/review/gpg"
	"github.com/google/git-appraise/review/request"
	"golang.org/x/sync/errgroup"
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

func localRefForFork(remoteRef, forkName string) (string, error) {
	if strings.Contains(remoteRef, ":") || strings.Contains(remoteRef, "+") {
		return "", fmt.Errorf("invalid remote ref %q", remoteRef)
	}
	if strings.HasPrefix(remoteRef, "refs/notes/") {
		// Older versions of git require all notes refs to be under "refs/notes"
		return fmt.Sprintf("refs/notes/forks/%s/%s", forkName, remoteRef), nil
	}
	return fmt.Sprintf("refs/forks/%s/%s", forkName, remoteRef), nil
}

func filteredRefForFork(remoteRef, forkName string) (string, error) {
	if strings.Contains(remoteRef, ":") || strings.Contains(remoteRef, "+") {
		return "", fmt.Errorf("invalid remote ref %q", remoteRef)
	}
	if !strings.HasPrefix(remoteRef, "refs/notes/devtools/") {
		return "", fmt.Errorf("only devtools notes refs can be filtered")
	}
	return fmt.Sprintf("refs/notes/filteredForks/%s/%s", forkName, remoteRef), nil
}

func forkPathFromName(forkName string) []string {
	forkHash := fmt.Sprintf("%x", sha1.Sum([]byte(forkName)))
	return []string{forkHash[0:1], forkHash[1:2], forkHash[2:]}
}

func forkPath(fork *Fork) []string {
	return forkPathFromName(fork.Name)
}

func encodeListAsHashedFiles(items []string) *repository.Tree {
	contents := make(map[string]repository.TreeChild)
	for _, item := range items {
		blob := repository.NewBlob(item)
		path := fmt.Sprintf("%x", sha1.Sum([]byte(item)))
		contents[path] = blob
	}
	return repository.NewTree(contents)
}

func addNestedTree(parent *repository.Tree, descendant repository.TreeChild, path []string) *repository.Tree {
	if len(path) == 0 {
		return parent
	}
	var contents map[string]repository.TreeChild
	if parent == nil {
		contents = make(map[string]repository.TreeChild)
	} else {
		contents = parent.Contents()
	}

	if len(path) == 1 {
		contents[path[0]] = descendant
		return repository.NewTree(contents)
	}

	childName := path[0]
	childPath := path[0:1]
	nestedPath := path[1:]
	var childTree *repository.Tree
	childObj, ok := contents[childName]
	if ok {
		childTree, ok = childObj.(*repository.Tree)
		if !ok {
			// One of the intermediary path components points
			// at a BLOB, so we are overwriting it.
			childTree = nil
		}
	}
	childTree = addNestedTree(childTree, descendant, nestedPath)
	return addNestedTree(parent, childTree, childPath)
}

// Add adds the given fork to the repository, replacing any existing forks with the same name.
func Add(repo repository.Repo, fork *Fork) error {
	contents := make(map[string]repository.TreeChild)
	contents[nameFilePath] = repository.NewBlob(fork.Name)
	contents[urlsDirPath] = encodeListAsHashedFiles(fork.URLS)
	contents[ownersDirPath] = encodeListAsHashedFiles(fork.Owners)
	contents[refsDirPath] = encodeListAsHashedFiles(fork.Refs)
	newForkTree := repository.NewTree(contents)

	var previousCommitHash string
	var forksTree *repository.Tree
	if hasRef, err := repo.HasRef(Ref); err != nil {
		return fmt.Errorf("failure checking the existence of the forks ref: %v", err)
	} else if hasRef {
		previousCommitHash, err = repo.GetCommitHash(Ref)
		if err != nil {
			return fmt.Errorf("failure reading the forks ref commit: %v", err)
		}
		forksTree, err = repo.ReadTree(previousCommitHash)
		if err != nil {
			return fmt.Errorf("failure reading the forks ref: %v", err)
		}
	}

	path := forkPath(fork)
	forksTree = addNestedTree(forksTree, newForkTree, path)
	var commitParents []string
	if previousCommitHash != "" {
		commitParents = append(commitParents, previousCommitHash)
	}
	commitDetails := &repository.CommitDetails{
		Summary: fmt.Sprintf("Adding the fork: %q", fork.Name),
		Parents: commitParents,
	}
	commitHash, err := repo.CreateCommitWithTree(commitDetails, forksTree)
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
		childObj, ok := currentLevel.Contents()[childName]
		if !ok {
			return fmt.Errorf("the specified fork, %q, does not exist", name)
		}
		childTree, ok := childObj.(*repository.Tree)
		if !ok {
			return fmt.Errorf("the specified fork, %q, does not exist", name)
		}
		currentLevel = childTree
	}
	delete(currentLevel.Contents(), path[0])
	commitDetails := &repository.CommitDetails{
		Summary: fmt.Sprintf("Deleting the fork: %q", name),
		Parents: []string{previousCommitHash},
	}
	commitHash, err := repo.CreateCommitWithTree(commitDetails, forksTree)
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
		contents := blob.Contents()
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
	contents := t.Contents()
	nameFile, ok := contents[nameFilePath]
	if !ok {
		return nil, fmt.Errorf("fork missing a NAME file")
	}
	nameBlob, ok := nameFile.(*repository.Blob)
	if !ok {
		return nil, fmt.Errorf("fork NAME file is not actually a file")
	}
	ownersFile, ok := contents[ownersDirPath]
	if !ok {
		return nil, fmt.Errorf("fork missing an OWNERS subdirectory")
	}
	ownersDir, ok := ownersFile.(*repository.Tree)
	if !ok {
		return nil, fmt.Errorf("fork OWNERS subdirectory is not actually a directory")
	}
	refsFile, ok := contents[refsDirPath]
	if !ok {
		return nil, fmt.Errorf("fork missing a REFS subdirectory")
	}
	refsDir, ok := refsFile.(*repository.Tree)
	if !ok {
		return nil, fmt.Errorf("fork REFS subdirectory is not actually a directory")
	}
	urlsFile, ok := contents[urlsDirPath]
	if !ok {
		return nil, fmt.Errorf("fork missing a URLS subdirectory")
	}
	urlsDir, ok := urlsFile.(*repository.Tree)
	if !ok {
		return nil, fmt.Errorf("fork URLS subdirectory is not actually a directory")
	}
	fork := &Fork{
		Name:   nameBlob.Contents(),
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

func (fork *Fork) isOwner(email string) bool {
	for _, owner := range fork.Owners {
		if owner == email {
			return true
		}
	}
	return false
}

func createMergeCommit(repo repository.Repo, ref, message string, commitsToMerge ...string) error {
	refCommit, err := repo.GetCommitHash(ref)
	if err != nil {
		return fmt.Errorf("failure reading the commit for the ref %q: %v", ref, err)
	}
	refDetails, err := repo.GetCommitDetails(refCommit)
	if err != nil {
		return fmt.Errorf("failure reading the commit %q: %v", refCommit, err)
	}
	parents := append([]string{refCommit}, commitsToMerge...)
	commitDetails := &repository.CommitDetails{
		Parents: parents,
		Summary: message,
		Tree:    refDetails.Tree,
	}
	mergeCommit, err := repo.CreateCommit(commitDetails)
	if err != nil {
		return fmt.Errorf("failure creating a merge commit for %q: %v", ref, err)
	}
	return repo.SetRef(ref, mergeCommit, refCommit)
}

func filterNewNotes(repo repository.Repo, destinationRef, sourceRef string, allow func(obj string, note repository.Note) (bool, error)) (map[string][]repository.Note, error) {
	sourceNotesMap, err := repo.GetAllNotes(sourceRef)
	if err != nil {
		return nil, err
	}
	if len(sourceNotesMap) == 0 {
		return nil, nil
	}
	existingNoteHashesMap := make(map[string]map[string]struct{})
	destinationNotesMap, err := repo.GetAllNotes(destinationRef)
	if err == nil {
		// Assume the opposite just means the destination ref does not exist
		for obj, notes := range destinationNotesMap {
			existingNoteHashes := make(map[string]struct{})
			existingNoteHashesMap[obj] = existingNoteHashes
			for _, note := range notes {
				existingNoteHashes[note.Hash()] = struct{}{}
			}
		}
	}
	newNotesMap := make(map[string][]repository.Note)
	for obj, objNotes := range sourceNotesMap {
		if existingNoteHashes, ok := existingNoteHashesMap[obj]; ok {
			var newNotes []repository.Note
			for _, note := range objNotes {
				if len(note) == 0 {
					continue
				}
				if _, ok := existingNoteHashes[note.Hash()]; ok {
					continue
				}
				newNotes = append(newNotes, note)
			}
			objNotes = newNotes
		}
		var filteredNotes []repository.Note
		for _, note := range objNotes {
			if allow == nil {
				filteredNotes = append(filteredNotes, note)
			} else if isAllowed, err := allow(obj, note); err != nil {
				return nil, err
			} else if isAllowed {
				filteredNotes = append(filteredNotes, note)
			}
		}
		if len(filteredNotes) > 0 {
			newNotesMap[obj] = filteredNotes
		}

	}
	return newNotesMap, nil
}

func mergeNewFilteredNotes(repo repository.Repo, destinationRef, sourceRef, mergeMessage string, allow func(obj string, note repository.Note) (bool, error)) error {
	sourceCommit, err := repo.GetCommitHash(sourceRef)
	if err != nil {
		// There are no notes to merge
		return nil
	}
	parentCommits := []string{sourceCommit}
	destinationCommit, err := repo.GetCommitHash(destinationRef)
	if err == nil {
		if isAncestor, err := repo.IsAncestor(sourceCommit, destinationCommit); err != nil {
			return err
		} else if isAncestor {
			// The notes have already been merged
			return nil
		}
	}
	notesToAddMap, err := filterNewNotes(repo, destinationRef, sourceRef, allow)
	if err != nil {
		return err
	}
	if len(notesToAddMap) == 0 {
		return nil
	}
	for obj, notesToAdd := range notesToAddMap {
		combinedNote := combineNotes(notesToAdd)
		if err := repo.AppendNote(destinationRef, obj, combinedNote); err != nil {
			return fmt.Errorf("failure merging in new notes to the ref %q: %v", destinationRef, err)
		}
	}
	return createMergeCommit(repo, destinationRef, mergeMessage, parentCommits...)
}

func (fork *Fork) filterOwnerRequests(repo repository.Repo, verifySignatures bool) error {
	forkRequestsRef, err := localRefForFork(request.Ref, fork.Name)
	if err != nil {
		return err
	}
	filteredForkRequestsRef, err := filteredRefForFork(request.Ref, fork.Name)
	if err != nil {
		return err
	}
	return mergeNewFilteredNotes(repo, filteredForkRequestsRef, forkRequestsRef,
		fmt.Sprintf("merging in requests from the fork %q", fork.Name),
		func(obj string, note repository.Note) (bool, error) {
			commitDetails, err := repo.GetCommitDetails(obj)
			if err != nil {
				// Ignore requests for unknown commits
				return false, nil
			}
			if !fork.isOwner(commitDetails.CommitterEmail) {
				// Ignore requests to review someone else's commit
				return false, nil
			}
			r, err := request.Parse(note)
			if err != nil {
				return false, nil
			}
			if !fork.isOwner(r.Requester) {
				return false, nil
			}
			if !verifySignatures {
				return true, nil
			}
			if err := gpg.Verify(&r); err != nil {
				// Ignore requests that do not have a verified signature
				return false, nil
			}
			return true, nil
		})
}

func (fork *Fork) filterOwnerComments(repo repository.Repo, verifySignatures bool) error {
	forkCommentsRef, err := localRefForFork(comment.Ref, fork.Name)
	if err != nil {
		return err
	}
	filteredForkCommentsRef, err := filteredRefForFork(comment.Ref, fork.Name)
	if err != nil {
		return err
	}
	return mergeNewFilteredNotes(repo, filteredForkCommentsRef, forkCommentsRef,
		fmt.Sprintf("merging in comments from the fork %q", fork.Name),
		func(obj string, note repository.Note) (bool, error) {
			c, err := comment.Parse(note)
			if err != nil {
				return false, nil
			}
			if c.Original != "" {
				// Ignore comment edits.
				// TODO(ojarjur): Also support pulling comment edits from forks
				return false, nil
			}
			if !fork.isOwner(c.Author) {
				// Ignore comments that aren't from the repository owner.
				return false, nil
			}
			if c.Location != nil && c.Location.Check(repo) != nil {
				// Ignore comments at non-existant locations.
				return false, nil
			}
			if !verifySignatures {
				return true, nil
			}
			if err := gpg.Verify(&c); err != nil {
				// Ignore comments that do not have a verified signature
				return false, nil
			}
			return true, nil
		})
}

func combineNotes(notes []repository.Note) repository.Note {
	var noteStrings []string
	for _, note := range notes {
		noteStrings = append(noteStrings, string(note))
	}
	return repository.Note([]byte(strings.Join(noteStrings, "\n")))
}

func appendAllNotes(repo repository.Repo, destination string, sources ...string) error {
	for _, source := range sources {
		mergeMessage := fmt.Sprintf("merging in filtered notes from the ref %q", source)
		if err := mergeNewFilteredNotes(repo, destination, source, mergeMessage, nil); err != nil {
			return err
		}
	}
	return nil
}

func (fork *Fork) Fetch(repo repository.Repo, verifySignatures bool) (bool, error) {
	initialHash, err := repo.GetRepoStateHash()
	if err != nil {
		return false, err
	}
	for _, url := range fork.URLS {
		var refSpecs []string
		for _, ref := range fork.Refs {
			localRef, err := localRefForFork(ref, fork.Name)
			if err != nil {
				return false, err
			}
			refSpec := fmt.Sprintf("+%s:%s", ref, localRef)
			refSpecs = append(refSpecs, refSpec)
		}
		if err := repo.Fetch(url, refSpecs...); err != nil {
			return false, fmt.Errorf("failure fetching from the fork: %v", err)
		}
		if updatedHash, err := repo.GetRepoStateHash(); err != nil {
			return false, err
		} else if updatedHash == initialHash {
			continue
		}
		var g errgroup.Group
		g.Go(func() error {
			if err := fork.filterOwnerRequests(repo, verifySignatures); err != nil {
				return fmt.Errorf("failure merging the review requests: %v", err)
			}
			return nil
		})
		g.Go(func() error {
			if err := fork.filterOwnerComments(repo, verifySignatures); err != nil {
				return fmt.Errorf("failure merging the comments: %v", err)
			}
			return nil
		})
		if err := g.Wait(); err != nil {
			return false, err
		}
	}
	updatedHash, err := repo.GetRepoStateHash()
	if err != nil {
		return false, err
	}
	return updatedHash != initialHash, nil
}

func MergeAll(repo repository.Repo, forks []*Fork) error {
	var filteredForkCommentsRefs []string
	var filteredForkRequestsRefs []string
	for _, f := range forks {
		filteredForkCommentsRef, err := filteredRefForFork(comment.Ref, f.Name)
		if err != nil {
			return err
		}
		filteredForkRequestsRef, err := filteredRefForFork(request.Ref, f.Name)
		if err != nil {
			return err
		}
		filteredForkCommentsRefs = append(filteredForkCommentsRefs, filteredForkCommentsRef)
		filteredForkRequestsRefs = append(filteredForkRequestsRefs, filteredForkRequestsRef)
	}
	if err := appendAllNotes(repo, comment.Ref, filteredForkCommentsRefs...); err != nil {
		return err
	}
	if err := appendAllNotes(repo, request.Ref, filteredForkRequestsRefs...); err != nil {
		return err
	}
	return nil
}

func Pull(repo repository.Repo, verifySignatures bool) error {
	forks, err := List(repo)
	if err != nil {
		return fmt.Errorf("failure listing the forks: %v", err)
	}
	var g errgroup.Group
	newlyFetchedForksChan := make(chan *Fork, len(forks))
	for _, f := range forks {
		func(f *Fork) {
			g.Go(func() error {
				if newData, err := f.Fetch(repo, verifySignatures); err != nil {
					return fmt.Errorf("failure pulling from the fork %q: %v", f.Name, err)
				} else if newData {
					newlyFetchedForksChan <- f
				}
				return nil
			})
		}(f)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	close(newlyFetchedForksChan)
	if len(newlyFetchedForksChan) == 0 {
		return nil
	}
	var forksWithNewData []*Fork
	for f := range newlyFetchedForksChan {
		forksWithNewData = append(forksWithNewData, f)
	}
	if err := MergeAll(repo, forksWithNewData); err != nil {
		return fmt.Errorf("failure merging from the forks: %v", err)
	}
	return nil
}
