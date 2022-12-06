# Development workflow

Forgejo is a soft fork, i.e. a set of commits applied to the Gitea development branch and the stable branches. On a regular basis those commits are rebased and modified if necessary to keep working. All Forgejo commits are merged into a branch from which binary releases and packages are created and distributed. The development workflow is a set of conventions Forgejo developers are expected to follow to work together.

Discussions on how the workflow should evolve happen [in the isssue tracker](https://codeberg.org/forgejo/forgejo/issues?type=all&state=open&labels=&milestone=0&assignee=0&q=%5BWORKFLOW%5D).

## Naming conventions

### Development

* Gitea: main
* Forgejo: forgejo
* Integration: forgejo-integration
* Feature branches: forgejo-feature-name

### Stable

* Gitea: release/vX.Y
* Forgejo: vX.Y/forgejo
* Integration: vX.Y/forgejo-integration
* Feature branches: vX.Y/forgejo-feature-name

## Rebasing

### *Feature branch*

The *Gitea* branches are mirrored with the Gitea development and stable branches.

On a regular basis, each *Feature branch* is rebased against the base *Gitea* branch.

### *Integration* and *Forgejo*

The latest *Gitea* branch resets the *Integration* branch and all *Feature branches* are merged into it. 

If tests pass, the *Forgejo* branch is reset to the tip of the *Integration* branch.

If tests do not pass, an issue is filed to the *Feature branch* that fails the test. Once the issue is resolved, another round of rebasing starts.

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
