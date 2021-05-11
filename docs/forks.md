# Working with multiple forks

This file documents how to use `git-appraise` with contributors who do not have
permission to push to the main repository.

In this model, each contributor sets up their own personal fork of the
repository. They then push to their fork instead of the main repository.

The list of these forks is stored inside the repository itself, in a special
ref. The tool uses that list to pull changes and reviews from all the
registered forks.

## Adding a fork

The owner of the main repository can add a fork with the following command,
followed by `git appraise push`:

```shell
git appraise fork add -o <contributor-email>[,<contributor-email>]* <name> <url>
```

## Using a fork to request a review

The owner of a fork can request reviews for any branches they create as long as
the following rules hold:

1. The name of the branch is prefixed with the name of their fork.
2. The committer email field of the commit to be reviewed matches
   one of the owner email addresses of the fork.

To request the review, run the usual `git appraise request ...` command, and
then use `git appraise push ...` to push it to the fork.

## Using a fork to comment on reviews

The owner of a fork can comment on any code review. The only requirement is
that the email address they use for git is listed as one of the owner email
addresses for the fork.

## Pulling code reviews from forks

When running `git appraise pull`, the tool will automatically fetch and merge
reviews from the forks. This behavior can be controlled using the
`--include-forks` flag. Alternatively, it can be configured on a
per-remote basis using the `appraise.remote.<remote>.includeForks` setting.

## Dealing with abusive fork owners

If the owner of a fork is acting inappropriately, both the owner of the main
repository and the cloners of that repository have tools to exclude reviews and
comments from that abusive owner's fork:

### Removing a fork

The owner of the main repository can remove a fork by running the following
command, followed by `git appraise push`:

```shell
git appraise fork remove <name>
```

### Excluding a specific fork

Anyone who runs `git appraise pull --include-forks` can configure a list of
forks to be excluded from the pull. This is controlled by the config settings
`appraise.remote.<remote>.excludeFork` and `appraise.excludeFork`. The first
operates on a per-remote basis while the second applies to every remote.

Either option can be configured with either the fork name or URL.
