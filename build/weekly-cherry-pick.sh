#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 Gergely Nagy
# SPDX-FileContributor: Gergely Nagy
#
# SPDX-License-Identifier: EUPL-1.2

set -euo pipefail

STATE_DIR="${XDG_STATE_HOME:-${HOME:-~}/.local/state}/forgejo-wcp"
STATE_FILE="${STATE_DIR}/current"
mkdir -p "${STATE_DIR}"

# Display a very barebones help screen.
cmd_help() {
    cat <<EOF
$0 -- weekly cherry pick helper

Usage:
  $0 <COMMAND> [<OPTIONS...>]

Commands:

  cherry pick session:

    start [<FROM>] [<remote/branch>]
    stop
    view
    edit
    update | push
    sync

  current session:

    list-cps [--mark=[question|cherries|fast_forward|bulb]]
    pending

    next [--text]
    skip [<COMMIT>]
    port [<COMMIT>]
    pick | cp | cherry-pick [<COMMIT>]
EOF
}

# Turn a commit id into a markdown link to a commit on Gitea's GitHub repo.
gitea_commit_link() {
    _commit_id="$1"
    _short_id="$(echo -n "${_commit_id}" | cut -c 1-10)"

    echo -n "[\`gitea@${_short_id}\`](https://github.com/go-gitea/gitea/commit/${_commit_id})"
}

# Turn the PR suffix of cherry picked commits into a markdown link pointing to
# the Gitea PR.
gitea_commit_message_pr_linkify() {
    _message="$1"

    # shellcheck disable=SC2001
    echo "${_message}" | sed -e 's%(#\([0-9]*\))$%([gitea#\1](https://github.com/go-gitea/gitea/pull/\1))%'
}

# Attempt to find the previous weekly cherry pick PR.
#
# Looks at recently closed PRs with the "forgejo/furnace-cleanup" (116080)
# label, and finds the most recent gitea cherry pick.
#
# Returns the PR's number.
forgejo_find_previous_wcp() {
    curl -s 'https://codeberg.org/api/v1/repos/forgejo/forgejo/pulls?state=closed&labels=116080' \
         -H 'accept: application/json' \
    | jq -r '[ .[] | select(.title | test("\\[gitea\\].*cherry-pick")) ] | .[0].number'
}

# Attempt to find the current weekly cherry pick PR.
#
# Same as `forgejo_find_previous_wcp`, except it looks for open PRs.
forgejo_find_current_wcp() {
    curl -s 'https://codeberg.org/api/v1/repos/forgejo/forgejo/pulls?state=open&labels=116080' \
         -H 'accept: application/json' \
    | jq -r '[ .[] | select(.title | test("\\[gitea\\].*cherry-pick")) ] | .[0].number'
}

# Attempt to find the last picked commit.
#
# First looks at a "Last commit considered:" line in the previous PR, and picks
# the commit from there, if any. If none are found, looks at the Commits
# section, and picks the topmost entry.
forgejo_find_last_picked_commit() {
    _last_pr="${1:-$(forgejo_find_previous_wcp)}"
    _body="$(curl -s "https://codeberg.org/api/v1/repos/forgejo/forgejo/pulls/${_last_pr}" \
                  -H "accept: application/json" \
             | jq -r '.body')"

    if echo "${_body}" | grep -q "^Last commit considered: "; then
        _commit="$(echo "${_body}" \
            | grep "^Last commit considered: " \
            | sed -e "s#.*https://github.com/go-gitea/gitea/commit/\(.*\))#\1#")"
    else
        _commit="$(echo "${_body}" \
            | grep "^## Commits" -A 2 \
            | tail -n 1 \
            | sed -e "s,^- [^ ]* \([0-9a-zA-Z]*\) .*,\1,")"
    fi

    echo "${_commit}"
}

# Add the weekly commit PR preamble.
forgejo_plan_preamble() {
    _last_pr="$(forgejo_find_previous_wcp)"
    _last_cp="$(forgejo_find_last_picked_commit "${_last_pr}")"
    _last_cp_short="$(echo -n "${_last_cp}" | cut -c 1-10)"

    _head_sha="$(git rev-parse "${1}")"
    _head_sha_short="$(echo -n "${_head_sha}" | cut -c 1-10)"

    cat <<EOF
## Checklist

- [ ] if there are significant changes in translations \`options/locale\` (crowdin commits)
  - [merge translations](https://forgejo.org/docs/v1.21/developer/localization-admin/) ([LINK])
- [ ] check the PRs that are of [particular interest to someone](https://pad.gusted.xyz/B2CXwfxvTh6I2FGAp_xOvw?view)
- [ ] go to the last cherry-pick PR to figure out how far it went: forgejo/forgejo#${_last_pr}
  - [\`gitea#${_last_cp_short}\`](https://github.com/go-gitea/gitea/commits/${_last_cp})
- [ ] cherry-pick and open PR in WIP ([LINK])
- [ ] have the PR pass the CI
- end-to-end (specially important if there are actions related changes)
  - [ ] add run-end-to-end-label: [LINK]
  - [ ] check the result  [LINK]
- [ ] remove WIP
- [ ] assign reviewers
- [ ] 48h later, last call [LINK]
- merge 1 hour after the last call
  - [ ] [Merge translations](https://forgejo.org/docs/v1.21/developer/localization-admin/) before merging the PR and keep it locked
  - [ ] merge the PR
  - [ ] reset the translations
  - [ ] unlock the translations
  - [ ] reload the admin page
  - [ ] verify there are no pending changes

## Notes

- start \`cherry-pick -x\`
    - try to understand and resolve all conflicts to the extent that they are trivial
    - if they are not trivial abort the cherry-pick, it is better to schedule a port with the person who was last active in the same area
- Conflict resolution
  - All changes in \`docs/\` are silently resolved in favor of the commit
  - All \`options/locale\` changes are resolved in favor of Forgejo
- All Gitea database migrations are imported unmodified even if they are part of a commit that is skipped
- Test fail resolution
  - If they require Forgejo specific change or non trivial modifications of Gitea files, do them in a commit with the same title and the suffix (followup) and move the commit next to the one they relate to
  - If they require changing the Gitea commit in a trivial way amend the corresponding commit and make sure the comment the "Conflict" section of the commit message
- Skip commits and mark them "- Gitea specific"
  - Only change
    - \`docs/\`
    - \`.github/\`
    - \`.gitea/\`
- Run deadcode and add the change in a \`[DEADCODE] update\`
- To speed up the debug loop, run lint & tests locally to verify there are no hidden / non syntactic conflicts and resolve them in new commits
- When a commit is skipped, the comment that goes with it should:
  - tag the person who is most familiar with this area of the codebase
    - so they are notified in case they do not already watch the PR
    - to record that in case their expertise is needed later on
  - include the name of a related Forgejo commit when
    - there is a conflict
    - it is already implemented

## Legend

- :question: - No decision about the commit has been made.
- :cherries: - The commit has been cherry picked.
- :fast_forward: - The commit has been skipped.
- :bulb: - The commit has been skipped, but should be ported to Forgejo.

Last commit considered: [\`gitea@${_head_sha_short}\`](https://github.com/go-gitea/gitea/commits/${_head_sha})

## Commits

EOF
}

# Start a weekly cherry picking session.
#
# Looks at the commits between the last commit (either as given on the
# commandline, or discovered) and the given remote (defaulting to `gitea/main`),
# and transforms the list to a markdown-formatted plan.
#
# Also prepends that with the output of `forgejo_plan_preamble`.
cmd_start() {
    _from="${1:-}"
    _remote_head="${2:-gitea/main}"

    if [ -z "${_from}" ]; then
        _from="$(forgejo_find_last_picked_commit)"
    fi

    if [ -e "${STATE_FILE}" ]; then
        echo "Cherry picking already started!" >&2
        echo "Use \`$0 stop\` to stop, and then start again." >&2
        exit 1
    fi

    _commits="$(git log --format=oneline "${_from}...${_remote_head}")"

    forgejo_plan_preamble "${_remote_head}" >"${STATE_FILE}"

    IFS_SAVE="$IFS"
    IFS="
"
    (for line in $_commits; do
        _commit_id="$(echo "${line}" | cut -d " " -f 1)"
        _commit_msg="$(echo "${line}" | cut -d " " -f 2-)"

        echo "- :question: $(gitea_commit_link "${_commit_id}") $(gitea_commit_message_pr_linkify "${_commit_msg}")"
    done) >> "${STATE_FILE}"
    IFS="${IFS_SAVE}"

    echo "Plan made, view with: $0 view"
}

# Stop the weekly cherry picking.
#
# When the PR is final, and has been merged, or when you want to restart from
# scratch.
cmd_stop() {
    rm -f "${STATE_FILE}"
}

# View the current plan.
cmd_view() {
    if [ ! -e "${STATE_FILE}" ]; then
        echo "cherry picking session not started yet." >&2
        exit 1
    fi

    ${PAGER:-less} <"${STATE_FILE}"
}

# Edit the current plan.
cmd_edit() {
    if [ ! -e "${STATE_FILE}" ]; then
        echo "cherry picking session not started yet." >&2
        exit 1
    fi

    ${EDITOR:-vim} "${STATE_FILE}"
}

# List commits considered for the current session.
#
# Lists all commits by default, but can filter by mark with the `--mark=STRING`
# option.
cmd_list_cps() {
    _opts="${1:-}"

    if [ ! -e "${STATE_FILE}" ]; then
        echo "cherry picking session not started yet." >&2
        exit 1
    fi

    _filter="cat"
    case "${_opts}" in
        "--mark="*)
            _filter="grep :${_opts#--mark=}:"
            ;;
    esac

    grep "^- :[a-z0-9]*: \[\`gitea@" "${STATE_FILE}" \
        | ${_filter} \
        | sed -e "s,^.*gitea/commit/\([^)]*\)) \(.*\),\1 \2," \
        | ${PAGER:-less}
}

# Displays pending commits in reverse chronological order.
#
# Optionally limits to the first N commits if an argument is given to it.
cmd_pending() {
    limit="cat"
    if [ -n "${1:-}" ]; then
        limit="head -n ${1}"
    fi
    cmd_list_cps --mark=question | tac | ${limit}
}

# Display the next pending commit's PR in a browser.
#
# If `--text` is specified, display the commit info on stdout instead.
cmd_next() {
    next="$(cmd_pending 1)"

    case "${1:-}" in
        --text)
            echo "${next}"
            return
            ;;
        *)
            # shellcheck disable=SC2001
            xdg-open "$(echo "${next}" | sed -e 's#.*\(https://.*\)))#\1#')"
            return
            ;;
    esac
}

_next_commit() {
    cmd_next | cut -d" " -f1
}

# Mark the next commit as skipped or to-port.
_skip() {
    _cid="${1:-$(_next_commit)}"
    _mark="${2:-fast_forward}"

    if [ ! -e "${STATE_FILE}" ]; then
        echo "cherry picking session not started yet." >&2
        exit 1
    fi

    if [ -z "${_cid}" ]; then
        cmd_help
        exit 1
    fi

    # Find the line the commit id appears on
    line="$(grep -n "${_cid}" "${STATE_FILE}" | cut -d: -f 1)"

    # Change the mark from `:question:` to the given mark
    sed -i "/- :question: .*gitea\/commit\/${_cid}/ { s/:question:/:${_mark}:/ }" "${STATE_FILE}"

    # Open an editor at the commit's line
    ${EDITOR:-vim} "+${line}" "${STATE_FILE}"
}

# Skip the next commit
cmd_skip() {
    _skip "${1:-}"
}

# Mark the next commit as to-be-ported
cmd_port() {
    _skip "${1:-}" "bulb"
}

# Cherry pick the next commit
cmd_cp() {
    _cid="${1:-$(_next_commit)}"

    if [ ! -e "${STATE_FILE}" ]; then
        echo "cherry picking session not started yet." >&2
        exit 1
    fi

    if [ -z "${_cid}" ]; then
        cmd_help
        exit 1
    fi

    # Find the line that references the commit
    line="$(grep -n "${_cid}" "${STATE_FILE}" | cut -d: -f 1)"

    # See if the commit has been cherry picked already.
    #
    # If the cherry pick fails, the cherry pick can be amended, committed, and
    # then running the same `weekly-cherry-pick.sh cp` command again will skip
    # the cherry pick part, and move on to editing the plan.
    is_picked="$(git show --format=format:true -q --grep "cherry picked from commit ${_cid}")"
    if [ -z "${is_picked}" ]; then
        # ...if not, cherry pick it.
        git cherry-pick -x "${_cid}"
    fi

    # Change the mark from `:question:` to `:cherries:`.
    sed -i "/- :question: .*gitea\/commit\/${_cid}/ { s/:question:/:cherries:/ }" "${STATE_FILE}"

    # Open an editor at the commit's line.
    ${EDITOR:-vim} "+${line}" "${STATE_FILE}"
}

# Push the current state to Codeberg.
#
# Updates both the branch, and the initial comment.
cmd_update() {
    if [ -z "${CODEBERG_TOKEN:-}" ]; then
        echo 'CODEBERG_TOKEN not set, cannot update the current pull request.' >&2
        exit 1
    fi

    # Push the commits we have queued
    git push "$@"

    cmd_sync
}

# Sync the current state to Codeberg.
#
# Updates the initial comment only.
cmd_sync() {
    if [ -z "${CODEBERG_TOKEN:-}" ]; then
        echo 'CODEBERG_TOKEN not set, cannot update the current pull request.' >&2
        exit 1
    fi

    # Find the current PR#
    _current_pr="$(forgejo_find_current_wcp)"

    # Turn the statefile into JSON
    tmpfile="$(mktemp)"
    jq -R -s '. as $body | {body: $body}' "${STATE_FILE}" >"${tmpfile}"
    # Update the initial comment of the PR with the current plan.
    curl -s -X PATCH "https://codeberg.org/api/v1/repos/forgejo/forgejo/pulls/${_current_pr}" \
         -H "content-type: application/json" \
         -H "authorization: Bearer ${CODEBERG_TOKEN}" \
         --data "@${tmpfile}" >/dev/null
    rm -f "${tmpfile}"
}

cmd="${1:-}"
shift || true

case "${cmd}" in
    help)
        cmd_help
        exit 0
        ;;
    start)
        cmd_start "$@"
        ;;
    stop)
        cmd_stop "$@"
        ;;
    view)
        cmd_view "$@"
        ;;
    edit)
        cmd_edit "$@"
        ;;
    list-cps)
        cmd_list_cps "$@"
        ;;
    pending)
        cmd_pending "$@"
        ;;
    skip)
        cmd_skip "$@"
        ;;
    port)
        cmd_port "$@"
        ;;
    next)
        cmd_next "$@"
        ;;
    pick|cp|cherry-pick)
        cmd_cp "$@"
        ;;
    update|push)
        cmd_update "$@"
        ;;
    sync)
        cmd_sync "$@"
        ;;
    *)
        cmd_help
        exit 1
        ;;
esac
