---

name: "Pull Request Template"
about: "Template for all Pull Requests"
labels:

- test/needed

---

## Checklist

The [developer guide](https://forgejo.org/docs/next/developer/) contains information that will be helpful to first time contributors. You are also welcome to join the [Forgejo development chatroom](https://matrix.to/#/#forgejo-development:matrix.org).

- I added tests coverage for Go changes...
  - [ ] in their respective `*_test.go` for unit tests.
  - [ ] in the `tests/integration` directory if it involves interactions with a live Forgejo server.
- I added tests coverage for JavaScript changes...
  - [ ] in `web_src/js/*.test.js` if it can be unit tested.
  - [ ] in `tests/e2e/*.test.e2e.js` if it requires interactions with a live Forgejo server (see also the [developer guide for JavaScript testing](https://codeberg.org/forgejo/forgejo/src/branch/forgejo/tests/e2e/README.md#end-to-end-tests)).
- I added documentation because there are changes related to the...
  - [ ] [User eXperience](https://forgejo.org/docs/next/user/) in the [user guide](https://codeberg.org/forgejo/docs/src/branch/next/docs/user).
  - [ ] [installation and administration](https://forgejo.org/docs/next/admin/) of an instance in the [administrator guide](https://codeberg.org/forgejo/docs/src/branch/next/docs/admin).
- [ ] I want the title to show in the release notes with a link to this pull request and I used a [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/) prefix.
- [ ] I want the content of the `release-notes/<pull request number>.md` file to show in the release notes instead of the title.
