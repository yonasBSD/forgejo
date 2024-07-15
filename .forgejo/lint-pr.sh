#!/bin/bash

set -e

RELEASE_NOTES_DIR="release-notes"
URL_DOCUMENTATION="forgejo.org/docs/"
LABEL_DOCUMENTATION="forgejo/documentation"
LABEL_GITEA="dependency/Gitea"
LABEL_WORTH_A_RELEASE_NOTE="worth a release-note"
ERROR_MISSING=101
ERROR_DOCUMENTATION=102
ERROR_LINE_COUNT=103
ERROR_CONVENTIONAL_COMMIT=104

function debug() {
  set -x
  PS4='${BASH_SOURCE[0]}:$LINENO: ${FUNCNAME[0]}:  '
}

function dependencies() {
  if test $(id -u) != 0; then
    return
  fi
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get -q install -qq -y jq
}

function info() {
  if ! test -f "$GITHUB_EVENT_PATH"; then
    echo 'expected $GITHUB_EVENT_PATH="'$GITHUB_EVENT_PATH'" to contain the JSON payload of the event but the file does not exist'
    return 1
  fi
  NUMBER="$(jq --raw-output '.pull_request.number' <$GITHUB_EVENT_PATH)"
  LABELS="$(jq --raw-output '.pull_request.labels[].name' <$GITHUB_EVENT_PATH)"
  TITLE="$(jq --raw-output '.pull_request.title' <$GITHUB_EVENT_PATH)"
  RELEASE_NOTE_FILE=$RELEASE_NOTES_DIR/$NUMBER.md
}

function lint_release_note() {
  if echo "$LABELS" | grep --quiet $LABEL_DOCUMENTATION; then
    if ! test -f $RELEASE_NOTE_FILE; then
      echo "the $RELEASE_NOTE_FILE file is missing"
      return $ERROR_MISSING
    fi

    if ! grep --quiet "$URL_DOCUMENTATION" $RELEASE_NOTE_FILE; then
      echo "$RELEASE_NOTE_FILE does not contain a URL to the documentation ($URL_DOCUMENTATION)"
      echo "cat $RELEASE_NOTE_FILE"
      cat $RELEASE_NOTE_FILE
      return $ERROR_DOCUMENTATION
    fi
  fi

  if ! test -f $RELEASE_NOTE_FILE; then
    return
  fi

  if echo "$LABELS" | grep --quiet $LABEL_GITEA; then
    return
  fi

  local line_count=$(wc -l <$RELEASE_NOTE_FILE)
  if test "$line_count" -gt 1; then
    echo "$RELEASE_NOTE_FILE must be a single line but wc -l says it has $line_count"
    return $ERROR_LINE_COUNT
  fi
}

function lint_title() {
  local conventional_commit='^[a-z()!]+: '
  if echo "$TITLE" | grep --extended-regexp --quiet "$conventional_commit"; then
    return
  else
    echo "'$TITLE' the title of the pull request is not compatible with [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/#summary) because it does not match '$conventional_commit'. Hint: start with feat: or fix:."
    return $ERROR_CONVENTIONAL_COMMIT
  fi
}

function lint() {
  info
  if lint_title; then
    case "$LABELS" in
    *$LABEL_WORTH_A_RELEASE_NOTE*)
      lint_release_note
      ;;
    esac
  else
    return $?
  fi
}

function main_test() {
  TMPDIR=$(mktemp -d)

  trap "rm -fr $TMPDIR" EXIT

  EVENT_PATH=$TMPDIR/event.json
  RELEASE_NOTES_DIR=$TMPDIR

  debug
  dependencies

  GITHUB_EVENT_PATH=$TMPDIR/nonexist
  ! info

  GITHUB_EVENT_PATH=$EVENT_PATH

  #
  # OK: nothing
  #
  local number=0
  cat >$EVENT_PATH <<EOF
{ "pull_request":
  {
    "title": "feat: feature01",
    "number": $number,
    "labels": []
  }
}
EOF
  lint

  #
  # OK: gitea dependencies can have multiline release notes
  #
  local number=1
  (
    echo "line 1"
    echo "line 2"
  ) >$RELEASE_NOTES_DIR/$number.md
  cat >$EVENT_PATH <<EOF
{ "pull_request":
  {
    "title": "feat: feature01",
    "number": $number,
    "labels": [
      {
        "name": "$LABEL_GITEA"
      },
      {
        "name": "$LABEL_WORTH_A_RELEASE_NOTE"
      }
    ]
  }
}
EOF
  lint

  #
  # OK: no release notes file but a good title
  #
  local number=2
  cat >$EVENT_PATH <<EOF
{ "pull_request":
  {
    "title": "feat: a feature",
    "number": $number,
    "labels": [
      {
        "name": "$LABEL_WORTH_A_RELEASE_NOTE"
      }
    ]
  }
}
EOF
  lint

  #
  # OK: the release note file has precedence over the title
  #
  local number=3
  echo "good release notes with link to $URL_DOCUMENTATION" >$RELEASE_NOTES_DIR/$number.md
  cat >$EVENT_PATH <<EOF
{ "pull_request":
  {
    "title": "feat: feature01",
    "number": $number,
    "labels": [
      {
        "name": "$LABEL_DOCUMENTATION"
      },
      {
        "name": "$LABEL_WORTH_A_RELEASE_NOTE"
      }
    ]
  }
}
EOF
  lint

  #
  # BAD: missing link to documentation
  #
  echo "good release notes with no link to documentation" >$RELEASE_NOTES_DIR/$number.md
  if lint; then
    return 1
  elif test $? != $ERROR_DOCUMENTATION; then
    return 1
  fi

  #
  # BAD: wrong line count
  #
  (
    echo "good release notes with link to $URL_DOCUMENTATION"
    echo "too many lines"
  ) >$RELEASE_NOTES_DIR/$number.md
  if lint; then
    return 1
  elif test $? != $ERROR_LINE_COUNT; then
    return 1
  fi

  #
  # BAD: no release notes file
  #
  number=4
  cat >$EVENT_PATH <<EOF
{ "pull_request":
  {
    "title": "feat: feature01",
    "number": $number,
    "labels": [
      {
        "name": "$LABEL_DOCUMENTATION"
      },
      {
        "name": "$LABEL_WORTH_A_RELEASE_NOTE"
      }
    ]
  }
}
EOF
  if lint; then
    return 1
  elif test $? != $ERROR_MISSING; then
    return 1
  fi

  #
  # BAD: the title is not conventional commit
  #
  number=5
  cat >$EVENT_PATH <<EOF
{ "pull_request":
  {
    "title": "something not conventional",
    "number": $number,
    "labels": []
  }
}
EOF
  if lint; then
    return 1
  elif test $? != $ERROR_CONVENTIONAL_COMMIT; then
    return 1
  fi
}

function main() {
  dependencies
  lint
}

"${@:-main}"
