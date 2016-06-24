/*
Copyright 2016 Google Inc. All rights reserved.

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

package repository

import (
	"bytes"
	"testing"
)

const (
	simpleBatchCheckOutput = `ddbdcb9d5aa71d35de481789bacece9a2f8138d0 commit
de9ebcdf2a1e93365eefc2739f73f2c68a280c11 commit
def9abf52f9a17d4f168e05bc420557a87a55961 commit
df324616ea2bc9bf6fc7025fc80a373ecec687b6 missing
dfdd159c9c11c08d84c8c050d2a1a4db29147916 commit
e4e48e2b4d76ac305cf76fee1d1c8c0283127d71 commit
e6ae4ed08704fe3c258ab486b07a36e28c3c238a commit
e807a993d1807b154294b9875b9d926b6f246d0c commit
e90f75882526e9bc5a71af64d60ea50092ed0b1d commit`
	simpleBatchCatFileOutput = `c1f5a5f135b171cc963b822d338000d185f1ae4f
342
{"timestamp":"1450315153","v":0,"agent":"Jenkins(1.627) GitNotesJobLogger","url":"https://jenkins-dot-developer-tools-bundle.appspot.com/job/git-appraise/105/"}

{"timestamp":"1450315161","v":0,"agent":"Jenkins(1.627) GitNotesJobLogger","url":"https://jenkins-dot-developer-tools-bundle.appspot.com/job/git-appraise/105/","status":"success"}

31ea4952450bbe5db0d6a7a7903e451925106c0f
141
{"timestamp":"1440202534","url":"https://travis-ci.org/google/git-appraise/builds/76722074","agent":"continuous-integration/travis-ci/push"}

bde25250a9f6dc9c56f16befa5a2d73c8558b472
342
{"timestamp":"1450434854","v":0,"agent":"Jenkins(1.627) GitNotesJobLogger","url":"https://jenkins-dot-developer-tools-bundle.appspot.com/job/git-appraise/112/"}

{"timestamp":"1450434860","v":0,"agent":"Jenkins(1.627) GitNotesJobLogger","url":"https://jenkins-dot-developer-tools-bundle.appspot.com/job/git-appraise/112/","status":"success"}

3128dc6881bf7647aea90fef1f4fbf883df6a8fe
342
{"timestamp":"1457445850","v":0,"agent":"Jenkins(1.627) GitNotesJobLogger","url":"https://jenkins-dot-developer-tools-bundle.appspot.com/job/git-appraise/191/"}

{"timestamp":"1457445856","v":0,"agent":"Jenkins(1.627) GitNotesJobLogger","url":"https://jenkins-dot-developer-tools-bundle.appspot.com/job/git-appraise/191/","status":"success"}

`
)

func TestSplitBatchCheckOutput(t *testing.T) {
	buf := bytes.NewBuffer([]byte(simpleBatchCheckOutput))
	commitsMap, err := splitBatchCheckOutput(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !commitsMap["ddbdcb9d5aa71d35de481789bacece9a2f8138d0"] {
		t.Fatal("Failed to recognize the first commit as valid")
	}
	if !commitsMap["de9ebcdf2a1e93365eefc2739f73f2c68a280c11"] {
		t.Fatal("Failed to recognize the second commit as valid")
	}
	if !commitsMap["e90f75882526e9bc5a71af64d60ea50092ed0b1d"] {
		t.Fatal("Failed to recognize the last commit as valid")
	}
	if commitsMap["df324616ea2bc9bf6fc7025fc80a373ecec687b6"] {
		t.Fatal("Failed to filter out a missing object")
	}
}

func TestSplitBatchCatFileOutput(t *testing.T) {
	buf := bytes.NewBuffer([]byte(simpleBatchCatFileOutput))
	notesMap, err := splitBatchCatFileOutput(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(notesMap["c1f5a5f135b171cc963b822d338000d185f1ae4f"]) != 342 {
		t.Fatal("Failed to parse the contents of the first cat'ed file")
	}
	if len(notesMap["31ea4952450bbe5db0d6a7a7903e451925106c0f"]) != 141 {
		t.Fatal("Failed to parse the contents of the second cat'ed file")
	}
	if len(notesMap["3128dc6881bf7647aea90fef1f4fbf883df6a8fe"]) != 342 {
		t.Fatal("Failed to parse the contents of the last cat'ed file")
	}
}
