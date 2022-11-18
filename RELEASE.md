Release management documentation.

# Shared user: release-team

The [release-team](https://codeberg.org/release-team) user authors and signs all releases. The associated email is release@forgejo.org.

The public GPG key used to sign the releases is [EB114F5E6C0DC2BCDD183550A4B61A2DC5923710](https://codeberg.org/release-team.gpg) `Forgejo Releases <contact@forgejo.org>`

# Release signing keys management

A GPG master key with no expiration date is created and shared with members of the Owners team via encrypted email. A subkey with a one year expiration date is created and stored in the secrets repository, to be used by the CI pipeline. The public master key is stored in the secrets repository and published where relevant.

## Master key creation

* gpg --expert --full-generate-key
* key type: ECC and ECC option with Curve 25519 as curve
* no expiration
* id: Forgejo Releases <contact@forgejo.org>
* gpg --export-secret-keys --armor EB114F5E6C0DC2BCDD183550A4B61A2DC5923710 and send via encrypted email to Owners
* gpg --export --armor EB114F5E6C0DC2BCDD183550A4B61A2DC5923710 > release-team-gpg.pub
* commit to the secret repository

## Subkey creation and renewal

* gpg --expert --edit-key EB114F5E6C0DC2BCDD183550A4B61A2DC5923710
* addkey
* key type: ECC (signature only)
* key validity: one year

### 2023

* gpg --export --armor F7CBF02094E7665E17ED6C44E381BF3E50D53707 > 2023-release-team-gpg.pub
* gpg --export-secret-keys --armor F7CBF02094E7665E17ED6C44E381BF3E50D53707 > 2023-release-team-gpg
* commit to the secret repository

## CI configuration

The `releaseteamgpg` secret in the Woodpecker CI configuration is set with the subkey.

# Release management

* Push a tag, the CI does the rest
