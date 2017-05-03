# Getting started with git-appraise

This file gives an example code-review workflow using git-appraise. It starts
with cloning a repository and goes all the way through to browsing
your submitted commits.

The git-appraise tool is largely agnostic of what workflow you use, so feel
free to change things to your liking, but this particular workflow should help
you get started.

## Cloning your repository

Since you're using a code review tool, we'll assume that you have a URL that
you can push to and pull from in order to collaborate with the rest of your team.

First we'll create our local clone of the repository:
```shell
git clone ${URL} example-repo
cd example-repo
```

If you are starting from an empty repository, then it's a good practice to add a
README file explaining the purpose of the repository:

```shell
echo '# Example Repository' > README.md
git add README.md
git commit -m 'Added a README file to the repo'
git push
```

## Creating our first review

Generally, reviews in git-appraise are used to decide if the code in one branch
(called the "source") is ready to merge into another branch (called the
"target"). The meaning of each branch and the policies around merging into a
branch vary from team to team, but for this example we'll use a simple practice
called [GitHub Flow](https://guides.github.com/introduction/flow/).

Specifically, we'll create a new branch for a particular feature, review the
changes to that branch against our master branch, and then delete the feature
branch once we are done.

### Creating our change

Create the feature branch:
```shell
git checkout -b ${USER}/getting-started
git push --set-upstream origin ${USER}/getting-started
```

... And make some changes to it:
```shell
echo "This is an example repository used for coming up to speed" >> README.md
git commit -a -m "Added an explanation to the README file"
git push
```

### Requesting the review

Up to this point we've only used the regular commands that come with git. Now,
we will use git-appraise to perform a review:

Request a review:
```shell
git appraise request
```

The output of this will be a summary of the newly requested review:
```
Review requested:
Commit: 1e6eb14c8014593843c5b5f29377585e4ed55304
Target Ref: refs/heads/master
Review Ref: refs/heads/ojarjur/getting-started
Message: "Added an explanation to the README file"
```

Show the details of the current review:
```shell
git appraise show
```

```
[pending] 1e6eb14c8014
  Added an explanation to the README file
  "refs/heads/ojarjur/getting-started" -> "refs/heads/master"
  reviewers: ""
  requester: "ojarjur@google.com"
  build status: unknown
  analyses:  No analyses available
  comments (0 threads):
```

Show the changes included in the review:
```shell
git appraise show --diff
```

```
diff --git a/README.md b/README.md
index 08fde78..85c4208 100644
--- a/README.md
+++ b/README.md
@@ -1 +1,2 @@
 # Example Repository
+This is an example repository used for coming up to speed
```

### Sending our updates to the remote repository

Before a teammate can review our change, we have to make it available to them.
This involves pushing both our commits, and our code review data to the remote
repository:
```shell
git push
git appraise pull
git appraise push
```

The command `git appraise pull` is used to make sure that our local code review
data includes everything from the remote repo before we try to push our changes
back to it. If you forget to run this command, then the subsequent call to
`git appraise push` might fail with a message that the push was rejected. If
that happens, simply run `git appraise pull` and try again.

## Reviewing the change

Your teammates can review your changes using the same tool.

Fetch the current data from the remote repository:
```shell
git fetch origin
git appraise pull
```

List the open reviews:
```shell
git appraise list
```

The output of this command will be a list of entries formatted like this:
```
Loaded 1 open reviews:
[pending] 1e6eb14c8014
  Added an explanation to the README file
```

The text within the square brackets is the status of a review, and for open
reviews will be one of "pending", "accepted", or "rejected". The text which
follows the status is the hash of the first commit in the review. This is
used to uniquely identify reviews, and most git-appraise commands will accept
this hash as an argument in order to select the review to handle.

For instance, we can see the details of a specific review using the "show"
subcommand:
```shell
git appraise show 1e6eb14c8014
```

```
[pending] 1e6eb14c8014
  Added an explanation to the README file
  "refs/heads/ojarjur/getting-started" -> "refs/heads/master"
  reviewers: ""
  requester: "ojarjur@google.com"
  build status: unknown
  analyses:  No analyses available
  comments (0 threads):
```

... or, we can see the diff of the changes under review:
```shell
git appraise show --diff 1e6eb14c8014
```

```
diff --git a/README.md b/README.md
index 08fde78..85c4208 100644
--- a/README.md
+++ b/README.md
@@ -1 +1,2 @@
 # Example Repository
+This is an example repository used for coming up to speed
```

Comments can be added either for the entire review, or on individual lines:
```shell
git appraise comment -f README.md -l 2 -m "Ah, so that's what this is" 1e6eb14c8014
```

These comments then show up in the output of `git appraise show`:
```shell
git appraise show 1e6eb14c8014
```

```
[pending] 1e6eb14c8014
  Added an explanation to the README file
  "refs/heads/ojarjur/getting-started" -> "refs/heads/master"
  reviewers: ""
  requester: "ojarjur@google.com"
  build status: unknown
  analyses:  No analyses available
  comments (1 threads):
    "README.md"@1e6eb14c8014
    |# Example Repository
    |This is an example repository used for coming up to speed
    comment: bd4c11ecafd443c9d1dde6035e89804160cd7487
      author: ojarjur@google.com
      time:   Fri Dec 18 10:58:54 PST 2015
      status: fyi
      Ah, so that's what this is
```

Comments initially only exist in your local repository, so to share them
with the rest of your team you have to push your review changes back:

```shell
git appraise pull
git appraise push
```

When the change is ready to be merged, you indicate that by accepting the
review:

```shell
git appraise accept 1e6eb14c8014
git appraise pull
git appraise push
```

The updated status of the review will be visible in the output of "show":
```shell
git appraise show 1e6eb14c8014
```

```
[accepted] 1e6eb14c8014
  Added an explanation to the README file
  "refs/heads/ojarjur/getting-started" -> "refs/heads/master"
  reviewers: ""
  requester: "ojarjur@google.com"
  build status: unknown
  analyses:  No analyses available
  comments (2 threads):
    "README.md"@1e6eb14c8014
    |# Example Repository
    |This is an example repository used for coming up to speed
    comment: bd4c11ecafd443c9d1dde6035e89804160cd7487
      author: ojarjur@google.com
      time:   Fri Dec 18 10:58:54 PST 2015
      status: fyi
      Ah, so that's what this is
    comment: 4034c60e6ed6f24b01e9a581087d1ab86d376b81
      author: ojarjur@google.com
      time:   Fri Dec 18 11:02:45 PST 2015
      status: fyi
```

## Submitting the change

Once a review has been accepted, you can merge it with the tool:

```shell
git appraise submit --merge 1e6eb14c8014
git push
```

The submit command will pop up a text editor where you can edit the default
merge message. That message will be used to create a new commit that is a
merge of the previous commit on the master branch, and the history of all
of your changes to the review. You can see what this looks like using
the `git log --graph` command:

```
*   commit 3a4d1b8cd264b921c858185f2c36aac283b45e49
|\  Merge: b404fa3 1e6eb14
| | Author: Omar Jarjur <ojarjur@google.com>
| | Date:   Fri Dec 18 11:06:24 2015 -0800
| | 
| |     Submitting review 1e6eb14c8014
| |     
| |     Added an explanation to the README file
| |   
| * commit 1e6eb14c8014593843c5b5f29377585e4ed55304
|/  Author: Omar Jarjur <ojarjur@google.com>
|   Date:   Fri Dec 18 10:49:56 2015 -0800
|   
|       Added an explanation to the README file
|  
* commit b404fa39ae98950d95ab06012191f58507e51d12
  Author: Omar Jarjur <ojarjur@google.com>
  Date:   Fri Dec 18 10:48:06 2015 -0800
  
      Added a README file to the repo
```

This is sometimes called a "merge bubble". When the review is simply accepted
as is, these do not add much value. However, reviews often go through several
rounds of changes before they are accepted. By using these merge commits, we
can preserve both the full history of individual reviews, and the high-level
(review-based) history of the repository.

This can be seen with the history of git-appraise itself. We can see the high
level review history using `git log --first-parent`:

```
commit 83c4d770cfde25c943de161c0cac54d714b7de38
Merge: 9a607b8 931d1b4
Author: Omar Jarjur <ojarjur@google.com>
Date:   Fri Dec 18 09:46:10 2015 -0800

    Submitting review 8cb887077783
    
    Fix a bug where requesting a review would fail with an erroneous message.
    
    We were figuring out the set of commits to include in a review by
    listing the commits between the head of the target ref and the head of
    the source ref. However, this only works if the source ref is a
    fast-forward of the target ref.
    
    This commit changes it so that we use the merge-base of the target and
    source refs as the starting point instead of the target ref.

commit 9a607b8529d7483e5b323303c73da05843ff3ca9
Author: Harry Lawrence <hazbo@gmx.com>
Date:   Fri Dec 18 10:24:00 2015 +0000

    Added links to Eclipse and Jenkins plugins
    
    As suggested in #11

commit 8876cfff2ed848d50cb559c05d44e11b95ca791c
Merge: 00c0e82 1436c83
Author: Omar Jarjur <ojarjur@google.com>
Date:   Thu Dec 17 12:46:32 2015 -0800

    Submitting review 09aecba64027
    
    Force default git editor when omitting -m
    For review comments, the absence of the -m flag will now attempt to load the
    user's default git editor.
    
    i.e. git appraise comment c0a643ff39dd
    
    An initial draft as discussed in #8
    
    I'm still not sure whether or not the file that is saved is in the most appropriate place or not. I like the idea of it being relative to the project although it could have gone in `/tmp` I suppose.

commit 00c0e827e5b86fb9d200f474d4f65f43677cbc6c
Merge: 31209ce 41fde0b
Author: Omar Jarjur <ojarjur@google.com>
Date:   Wed Dec 16 17:10:06 2015 -0800

    Submitting review 2c9bff89f0f8
    
    Improve the error messages returned when a git command fails.
    
    Previously, we were simply cascading the error returned by the instance
    of exec.Command. However, that winds up just being something of the form
    "exit status 128", with all of the real error message going to the
    Stderr field.
    
    As such, this commit changes the behavior to save the data written to
    stderr, and use it to construct a new error to return.

...
```

Here you see a linear view of the reviews that have been submitted, but if we
run the command `git log --oneline --graph`, then we can see that the full
history of each individual review is also available:

```
*   83c4d77 Submitting review 8cb887077783
|\  
| *   931d1b4 Merge branch 'master' into ojarjur/fix-request-bug
| |\  
| |/  
|/|   
* | 9a607b8 Added links to Eclipse and Jenkins plugins
| *   c7be567 Merge branch 'master' into ojarjur/fix-request-bug
| |\  
| |/  
|/|   
* |   8876cff Submitting review 09aecba64027
|\ \  
| * | 1436c83 Using git var GIT_EDITOR rather than git config
| * | 09aecba Force default git editor when omitting -m
|/ /  
| * 8cb8870 Fix a bug where requesting a review would fail with an erroneous message.
|/  
*   00c0e82 Submitting review 2c9bff89f0f8
...
```

## Cleaning up

Now that our feature branch has been merged into master, we can delete it:

```shell
git branch -d ${USER}/getting-started
git push origin --delete ${USER}/getting-started
```
