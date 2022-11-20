# Forgejo Contributor Guide

This document explains how to contribute changes to the Forgejo project.
Sensitive security-related issues should be reported to
[security@forgejo.org](mailto:security@forgejo.org).

# Development workflow

Forgejo is a soft fork, i.e. a set of commits applied to the Gitea development branch and the stable branches. On a regular basis those commits are rebased and modified if necessary to keep working. All Forgejo commits are merged into a branch from which binary releases and packages are created and distributed. The development workflow is a set of conventions Forgejo developers are expected to follow to work together.

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

All *Feature branches* are based on the \*forgejo-development branch which provides the Woodpecker CI configuration and other development tools.

The purpose of each *Feature branch* is documented in CONTRIBUTING.md as follows:

* Name of the *Feature branch* and name of the base *Feature branch* (for instance forgejo-federation based on forgejo-development)
    * Backports: list of the versions in which this *Feature branch* is supported (for instance v1.18, v1.19)
    * Description: explains what the focus of the *Feature branch* is (for instance: forge federation features)

## Contributing

Most people who are used to contributing will be familiar with the workflow of sending a pull request against the default branch. When that happens the reviewer should change the base branch to the appropriate *Feature branch* instead. If the pull request does not fit in any *Feature branch*, the reviewer needs to make decision to either:

* Decline the pull request because it is best contributed to Gitea
* Create a new *Feature branch*

Returning contributors can figure out which *Feature branch* to base their pull request on using the list of *Feature branches* found in CONTRIBUTING.md

## Granularity

*Feature branches* can contain a number of commits grouped together, for instance for branding the documentation, the landing page and the footer. It makes it convenient for people working on that topic to get the big picture without browsing multiple branches. Creating a new *Feature branch* for each individual commit, while possible, is likely to be difficult to work with.

Observing the granularity of the existing *Feature branches* is the best way to figure out what works and what does not. It requires adjustments from time to time depending on the number of contributors and the complexity of the Forgejo codebase that sits on top of Gitea.

# Release management

## Shared user: release-team

The [release-team](https://codeberg.org/release-team) user authors and signs all releases. The associated email is release@forgejo.org.

The public GPG key used to sign the releases is [EB114F5E6C0DC2BCDD183550A4B61A2DC5923710](https://codeberg.org/release-team.gpg) `Forgejo Releases <release@forgejo.org>`

## Release process

* Reset the vX.Y/forgejo-integration branch to the Gitea tag vX.Y.Z
* Merge all feature branches into the vX.Y/forgejo-integration branch
* If the CI passes reset the vX.Y/forgejo branch to the tip of vX.Y/forgejo-integration
* Set the vX.Y.Z tag to the tip of the vX.Y/forgejo branch
* [Binaries](https://codeberg.org/forgejo/forgejo/releases) are built, signed and uploaded by the CI.
* [Container images](https://codeberg.org/forgejo/-/packages/container/forgejo/versions) are built and uploaded by the CI.

## Release signing keys management

A GPG master key with no expiration date is created and shared with members of the Owners team via encrypted email. A subkey with a one year expiration date is created and stored in the secrets repository, to be used by the CI pipeline. The public master key is stored in the secrets repository and published where relevant.

### Master key creation

* gpg --expert --full-generate-key
* key type: ECC and ECC option with Curve 25519 as curve
* no expiration
* id: Forgejo Releases <contact@forgejo.org>
* gpg --export-secret-keys --armor EB114F5E6C0DC2BCDD183550A4B61A2DC5923710 and send via encrypted email to Owners
* gpg --export --armor EB114F5E6C0DC2BCDD183550A4B61A2DC5923710 > release-team-gpg.pub
* commit to the secret repository

### Subkey creation and renewal

* gpg --expert --edit-key EB114F5E6C0DC2BCDD183550A4B61A2DC5923710
* addkey
* key type: ECC (signature only)
* key validity: one year

#### 2023

* gpg --export --armor F7CBF02094E7665E17ED6C44E381BF3E50D53707 > 2023-release-team-gpg.pub
* gpg --export-secret-keys --armor F7CBF02094E7665E17ED6C44E381BF3E50D53707 > 2023-release-team-gpg
* commit to the secret repository

### CI configuration

The `releaseteamgpg` secret in the Woodpecker CI configuration is set with the subkey.
