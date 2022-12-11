# Development workflow

Forgejo is a soft fork, i.e. a set of commits applied to the Gitea development branch and the stable branches. On a regular basis those commits are rebased and modified if necessary to keep working. All Forgejo commits are merged into a branch from which binary releases and packages are created and distributed. The development workflow is a set of conventions Forgejo developers are expected to follow to work together.

Discussions on how the workflow should evolve happen [in the isssue tracker](https://codeberg.org/forgejo/forgejo/issues?type=all&state=open&labels=&milestone=0&assignee=0&q=%5BWORKFLOW%5D).

## Naming conventions

### Development

* Gitea: main
* Forgejo: forgejo
* Feature branches: forgejo-feature-name

### Stable

* Gitea: release/vX.Y
* Forgejo: vX.Y/forgejo
* Feature branches: vX.Y/forgejo-feature-name

### Soft fork history

Before rebasing on top of Gitea, all branches are copied to `soft-fork/YYYY-MM-DD/<branch>` for safekeeping.

## Rebasing

### *Feature branch*

The *Gitea* branches are mirrored with the Gitea development and stable branches.

On a regular basis, each *Feature branch* is rebased against the base *Gitea* branch.

### forgejo branch

The latest *Gitea* branch resets the *forgejo* branch and all *Feature branches* are merged into it.

If tests pass after pushing *forgejo* to the https://codeberg.org/forgejo-integration/forgejo repository, it can be pushed to the https://codeberg.org/forgejo/forgejo repository.

If tests do not pass, an issue is filed to the *Feature branch* that fails the test. Once the issue is resolved, another round of rebasing starts.

### Cherry picking and rebasing

Because Forgejo is a soft fork of Gitea, the commits in feature branches need to be cherry-picked on top of their base branch. They cannot be rebased using `git rebase`, because their base branch has been rebased.

Here is how the commits in the `forgejo-f3` branch can be cherry-picked on top of the latest `forgejo-development` branch:

```
$ git fetch --all
$ git remote get-url forgejo
git@codeberg.org:forgejo/forgejo.git
$ git checkout -b forgejo/forgejo-f3
$ git reset --hard forgejo/forgejo-development
$ git cherry-pick $(git rev-list --reverse forgejo/soft-fork/2022-12-10/forgejo-development..forgejo/soft-fork/2022-12-10/forgejo-f3)
$ git push --force forgejo-f3 forgejo/forgejo-f3
```

## Releasing

When a tag is set to a *Stable* *Forgejo* branch, the CI pipeline creates and uploads binaries and packages.

## Feature branches

All *Feature branches* are based on the {vX.Y/,}forgejo-development branch which provides and other development tools and documenation.

The \*forgejo-development branch is based on the {vX.Y/,}forgejo-ci branch which provides the Woodpecker CI configuration.

The purpose of each *Feature branch* is documented below:

### General purpose

* [forgejo-ci](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-ci) based on [main](https://codeberg.org/forgejo/forgejo/src/branch/main)
  Woodpecker CI configuration, including the release process.
  * Backports: [v1.18/forgejo-ci](https://codeberg.org/forgejo/forgejo/src/branch/v1.18/forgejo-ci)

* [forgejo-development](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-development) based on [forgejo-ci](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-ci)
  Forgejo development tools and documentation.
  * Backports: [v1.18/forgejo-development](https://codeberg.org/forgejo/forgejo/src/branch/v1.18/forgejo-development)

### [Federation](https://codeberg.org/forgejo/forgejo/issues?labels=79349)

* [forgejo-federation](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-federation) based on [forgejo-development](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-development)
  Federation support for Forgejo

* [forgejo-f3](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-f3) based on [forgejo-development](https://codeberg.org/forgejo/forgejo/src/branch/forgejo-development)
  [F3](https://lab.forgefriends.org/friendlyforgeformat/gof3) support for Forgejo

## Pull requests and feature branches

Most people who are used to contributing will be familiar with the workflow of sending a pull request against the default branch. When that happens the reviewer should change the base branch to the appropriate *Feature branch* instead. If the pull request does not fit in any *Feature branch*, the reviewer needs to make decision to either:

* Decline the pull request because it is best contributed to Gitea
* Create a new *Feature branch*

Returning contributors can figure out which *Feature branch* to base their pull request on using the list of *Feature branches*.

## Granularity

*Feature branches* can contain a number of commits grouped together, for instance for branding the documentation, the landing page and the footer. It makes it convenient for people working on that topic to get the big picture without browsing multiple branches. Creating a new *Feature branch* for each individual commit, while possible, is likely to be difficult to work with.

Observing the granularity of the existing *Feature branches* is the best way to figure out what works and what does not. It requires adjustments from time to time depending on the number of contributors and the complexity of the Forgejo codebase that sits on top of Gitea.
