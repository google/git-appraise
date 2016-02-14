# Distributed Code Review For Git

This repo contains a command line tool for performing code reviews on git
repositories.

## Overview

This tool is a *distributed* code review system for git repos.

By "distributed", we mean that code reviews are stored inside of the repository
as git objects. Every developer on your team has their own copy of the review
history that they can push or pull. When pulling, updates from the remote
repo are automatically merged by the tool.

This design removes the need for any sort of server-side setup. As a result,
this tool can work with any git hosting provider, and the only setup required
is installing the client on your workstation.

## Installation

Assuming you have the [Go tools installed](https://golang.org/doc/install), run
the following command:

    go get github.com/google/git-appraise/git-appraise

Then, either make sure that `${GOPATH}/bin` is in your PATH, or explicitly add the
"appraise" git alias by running the following command.

    git config --global alias.appraise '!'"${GOPATH}/bin/git-appraise"

## Requirements

This tool expects to run in an environment with the following attributes:

1.  The git command line tool is installed, and included in the PATH.
2.  The tool is run from within a git repo.
3.  The git command line tool is configured with the credentials it needs to
    push to and pull from the remote repos.

## Usage

Requesting a code review:

    git review request

Pushing code reviews to a remote:

    git review push [<remote>]

Pulling code reviews from a remote:

    git review pull [<remote>]

Listing open code reviews:

    git review
    git review --list

Showing the status of the current review, including comments:

    git review show

Showing the diff of a review:

    git review show -p/-u/--patch [--diff-opts "<diff-options>"] [<review-hash>]

Commenting on a review:

    git review comment -m "<message>" [-f <file> [-l <line>]] [<review-hash>]

Accepting the changes in a review:

    git review accept [-m "<message>"] [<review-hash>]

Submitting the current review:

    git review merge [--ff | --no-ff | --ff-only ]
    git review rebase

A more detailed getting started doc is available [here](docs/tutorial.md).

## Metadata

The code review data is stored in git-notes, using the formats described below.
Each item stored is written as a single line of JSON, and is written with at
most one such item per line. This allows the git notes to be automatically
merged using the "cat\_sort\_uniq" strategy.

Since these notes are not in a human-friendly form, all of the refs used to
track them start with the prefix "refs/notes/git-review". This helps make it
clear that these are meant to be read and written by automated tools.

When a field named "v" appears in one of these notes, it is used to denote
the version of the metadata format being used. If that field is missing, then
it defaults to the value 0, which corresponds to this initial version of the
formats.

### Code Review Requests

Code review requests are stored in the "refs/notes/git-review/requests" ref, and
annotate the first revision in a review. They must conform to the following
schema.
```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "timestamp": {
      "type": "string"
    },
    "reviewRef": {
      "id": "reviewRef",
      "type": "string"
    },
    "targetRef": {
      "id": "targetRef",
      "type": "string"
    },
    "requester": {
      "type": "string"
    },
    "reviewers": {
      "type": "array",
      "items": {
        "type": "string"
      }
    },
    "description": {
      "type": "string"
    },
    "v": {
      "type": "integer",
      "default": 0,
      "enum": [
        null,
        0
      ]
    },
    "baseCommit": {
      "type": "string"
    }
  },
  "required": [
    "targetRef"
  ]
}
```
The "reviewRef" field is used to specify a git ref that tracks the current
revision under review, and the "targetRef" field is used to specify the git ref
that should be updated once the review is approved.

If there are multiple requests for a single commit, then they are sorted by
timestamp and the final request is treated as the current one. This sorting
should be done in a stable manner, so that if there are multiple requests
with the same timestamp, then the last such request in the note is treated
as the current one.

This design allows a user to update a review request by re-running the
`git review request` command.

### Continuous Integration Status

Continuous integration build and test results are stored in the
"refs/notes/git-review/ci" ref, and annotate the revision that was built and
tested. They must conform to the following schema.
```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "timestamp": {
      "type": "string"
    },
    "url": {
      "type": "string"
    },
    "status": {
      "type": "string",
      "enum": [
        null,
        "success",
        "failure"
      ]
    },
    "agent": {
      "type": "string"
    },
    "v": {
      "type": "integer",
      "default": 0,
      "enum": [
        null,
        0
      ]
    }
  },
}
```
The "status" field is for the final status of a build or test. The "agent"
field is a free-form string that identifies the build and test runner.

### Robot Comments

Robot comments are comments generated by static analysis tools. These are
stored in the "refs/notes/git-review/robot" ref, and annotate the revision.
They must conform to the following schema.
```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "timestamp": {
      "type": "string"
    },
    "url": {
      "type": "string"
    },
    "status": {
      "type": "string",
      "enum": [
        null,
        "lgtm",
        "fyi",
        "nmw"
      ]
    },
    "v": {
      "type": "integer",
      "default": 0,
      "enum": [
        null,
        0
      ]
    }
  },
}
```
The "url" field should point to a publicly readable file, which contains JSON
formatted analysis results. Those results should conform to the JSON format of
the ShipshapeResponse protocol buffer message defined
[here](https://github.com/google/shipshape/blob/master/shipshape/proto/shipshape_rpc.proto).

The "status" field represents the overall status of all messages from the
analysis results, with the enum values indicating the following:

*   `lgtm`: This stands for "Looks Good To Me" and indicates the analyses
    produced no messages.
*   `fyi`: This stands for "For your information" and indicates the analyses
    produced some messages, but none of them indicate errors.
*   `nmw`: This stands for "Needs more work" and indicates the analyses
    produced at least one message indicating an error.

### Review Comments

Review comments are comments that were written by a person rather than by a
machine. These are stored in the "refs/notes/devtools/comments" ref, and
annotate the first revision in the review. They must conform to the following
schema.
```json
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "timestamp": {
      "type": "string"
    },
    "author": {
      "type": "string"
    },
    "parent": {
      "type": "string"
    },
    "location": {
      "type": "object",
      "properties": {
        "commit": {
          "type": "string"
        },
        "path": {
          "type": "string"
        },
        "range": {
          "type": "object",
          "properties": {
            "startLine": {
              "type": "integer"
            }
          }
        }
      }
    },
    "description": {
      "type": "string"
    },
    "resolved": {
      "type": "boolean"
    },
    "v": {
      "type": "integer",
      "default": 0,
      "enum": [
        null,
        0
      ]
    }
  }
}
```
When the parent is specified, it must be the SHA1 hash of another comment on
the same revision, and it means this comment is a reply to that comment.

The timestamp field represents the number of seconds since the Unix epoch, and
is formatted as a 10 digit decimal number with zero padding. It should be the
first field written, so that the lexicographical ordering of comments matches
their chronological ordering.

## Plugins

  - [Eclipse](https://github.com/google/git-appraise-eclipse)
  - [Jenkins](https://github.com/jenkinsci/google-git-notes-publisher-plugin)

## Contributing

Please see [the CONTRIBUTING file](CONTRIBUTING.md) for information on contributing to Git Appraise.
