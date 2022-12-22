# Release management

## Release numbering

The Forgejo release numbers look like Gitea release numbers but are compliant with [semver](https://semver.org/).

* Gitea **v1.18.0** will be Forgejo **v118.0.0**, **v118.0.1**, etc.

The Gitea release candidates (or [pre-release](https://semver.org/#spec-item-9) in the semver parlance) are suffixed with **-rcN** which is handled as a special case for packaging: although **X.Y.Z** is lexicographically lower than **X.Y.Z-rc1** is is considered greater. The Forgejo release is appended after the **-rcN** suffix.

* Gitea **v1.18.0-rc0** will be Forgejo **v118.0.0-rc0.0**, **v118.0.0-rc0.1**, etc.
* Gitea **v1.18.0-rc1** will be Forgejo **v118.0.0-rc1.0**, **v118.0.0-rc1.1**, **v118.0.0-rc1.2**, etc.
* Gitea **v1.18.0** will be Forgejo **v118.0.0**, **v118.0.1**, **v118.0.2**, etc.
* etc.

## Release process

### Integration

* Reset the vX.Y/forgejo branch to the Gitea tag vX.Y.Z
* Merge all feature branches into the vX.Y/forgejo branch

### Testing

When Forgejo is released, artefacts (packages, binaries, etc.) are first published by the CI/CD pipelines in the https://codeberg.org/forgejo-integration organization, to be downloaded and verified to work. When modifying the CI/CD pipelines, there is a chance that these verification steps fail and that the published artefacts override previous ones or worse. During this debugging phase, a fork of Forgejo must be used.

* Push the vX.Y.Z tag to a Forgejo fork (e.g. https://codeberg.org/someuser/forgejo)
* Verify the release is published in https://codeberg.org/forgejo-integration
* Verify the release is published in https://codeberg.org/someuser

### Publication

* Push the vX.Y.Z tag to https://codeberg.org/forgejo/forgejo
* [Binaries](https://codeberg.org/forgejo/forgejo/releases) are built, signed and uploaded by the CI.
* [Container images](https://codeberg.org/forgejo/-/packages/container/forgejo/versions) are built and uploaded by the CI.

### Website update

* Restart the last CI build at https://codeberg.org/forgejo/website/src/branch/main/
* Verify https://forgejo.org/download/ points to the expected release
* Manually try the instructions to work

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
* create [an issue](https://codeberg.org/forgejo/forgejo/issues) to schedule the renewal

#### 2023

* gpg --export --armor F7CBF02094E7665E17ED6C44E381BF3E50D53707 > 2023-release-team-gpg.pub
* gpg --export-secret-keys --armor F7CBF02094E7665E17ED6C44E381BF3E50D53707 > 2023-release-team-gpg
* commit to the secrets repository
* renewal issue https://codeberg.org/forgejo/forgejo/issues/58

### CI configuration

In the Woodpecker CI configuration the following secrets must be set:

* `releaseteamgpg` is the secret GPG key used to sign the releases
* `releaseteamuser` is the user name to authenticate with the Forgejo API and publish the releases
* `releaseteamtoken` is the token to authenticate `releaseteamuser` with the Forgejo API and publish the releases
* `domain` is `codeberg.org`

## Users, organizations and repositories

## Shared user: release-team

The [release-team](https://codeberg.org/release-team) user publishes and signs all releases. The associated email is mailto:release@forgejo.org.

The public GPG key used to sign the releases is [EB114F5E6C0DC2BCDD183550A4B61A2DC5923710](https://codeberg.org/release-team.gpg) `Forgejo Releases <release@forgejo.org>`

## Integration organization

The https://codeberg.org/forgejo-integration organization is dedicated to integration testing. Its purpose is to ensure all artefacts can effectively be published and retrieved by the CI/CD pipelines. The `release-team` user as well as all Forgejo contributors working on the CI/CD pipeline should be owners of the `forgejo-integration` organization. Assuming `someuser` is such a user, they can use this organization to verify a modified CI/CD pipeline behaves as expected before actually trying to publish anything for real at https://codeberg.org/forgejo.

* Modify files in the `.woodpecker` directory
* Set a tag (e.g. v10.0.0)
* Push the tag to `https://codeberg.org/someouser/forgejo`
* After the CI/CD pipeline completes the artefacts (release, package, etc.) must be available and identical at https://codeberg.org/someouser/forgejo and https://codeberg.org/forgejo-integration/forgejo
