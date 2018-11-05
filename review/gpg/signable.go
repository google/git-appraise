// Package gpg provides an interface and an abstraction with which to sign and
// verify review requests and comments.
package gpg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

const placeholder = "gpgsig"

// Sig provides an abstraction around shelling out to GPG to sign the
// content it's given.
type Sig struct {
	// Sig holds an object's content's signature.
	Sig string `json:"signature,omitempty"`
}

// Signable is an interfaces which provides the pointer to the signable
// object's stringified signature.
//
// This pointer is used by `Sign` and `Verify` to replace its contents with
// `placeholder` or the signature itself for the purposes of signing or
// verifying.
type Signable interface {
	Signature() *string
}

// Signature is `Sig`'s implementation of `Signable`. Through this function, an
// object which needs to implement `Signable` need only embed `Sig`
// anonymously. See, e.g., review/request.go.
func (s *Sig) Signature() *string {
	return &s.Sig
}

// Sign uses gpg to sign the contents of a request and deposit it into the
// signature key of the request.
func Sign(key string, s Signable) error {
	// First we retrieve the pointer and write `placeholder` as its value.
	sigPtr := s.Signature()
	*sigPtr = placeholder

	// Marshal the content and sign it.
	content, err := json.Marshal(s)
	if err != nil {
		return err
	}
	sig, err := signContent(key, content)
	if err != nil {
		return err
	}

	// Write the signature as the new value at the pointer.
	*sigPtr = sig.String()
	return nil
}

func signContent(key string, content []byte) (*bytes.Buffer,
	error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gpg", "-u", key, "--detach-sign", "--armor")
	cmd.Stdin = bytes.NewReader(content)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return &stdout, err
}

// Verify verifies the signatures on the request and its comments with the
// given key.
func Verify(s Signable) error {
	// Retrieve the pointer.
	sigPtr := s.Signature()
	// Copy its contents.
	sig := *sigPtr
	// Overwrite the value with the placeholder.
	*sigPtr = placeholder

	defer func() { *sigPtr = sig }()

	// 1. Marshal the content into JSON.
	// 2. Write the signature and the content to temp files.
	// 3. Use gpg to verify the signature.
	content, err := json.Marshal(s)
	if err != nil {
		return err
	}
	sigFile, err := ioutil.TempFile("", "sig")
	if err != nil {
		return err
	}
	defer os.Remove(sigFile.Name())
	_, err = sigFile.Write([]byte(sig))
	if err != nil {
		return err
	}
	err = sigFile.Close()
	if err != nil {
		return err
	}

	contentFile, err := ioutil.TempFile("", "content")
	if err != nil {
		return err
	}
	defer os.Remove(contentFile.Name())
	_, err = contentFile.Write(content)
	if err != nil {
		return err
	}
	err = contentFile.Close()
	if err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gpg", "--verify", sigFile.Name(), contentFile.Name())
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%s", stderr.String())
	}
	return nil
}
