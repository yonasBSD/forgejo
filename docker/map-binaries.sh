#!/bin/bash

errcho () {
  echo "${@}" >&2
}

if [ ${#@} -lt 3 ]; then
  errcho "not enough arguments provided"
  echo "syntax: ${0} <path directly containing the binary> <base name of the binary> <target name or false> [<version tag as in the file name>]"
  exit 2
fi

BINARYLOOKUPPATH="${1}"
FILEBASENAME="${2}"
RENAME="${3}"

if [ ${#@} -gt 3 ] && ! [ "${4}" == "" ]; then
  echo "Tag provided: ${4}"
  FILEVERSION="${4}"
else
  echo "No tag provided defaulting to tag ${DEFAULTTAGNAME:-main}"
  FILEVERSION="${DEFAULTTAGNAME:-main}"
fi

OUTFILENAME=""
if [ "${RENAME}" == "false" ]; then
  echo "Rename deactivated outputting as ${FILEBASENAME}"
  OUTFILENAME="${FILEBASENAME}"
elif [ "${RENAME}" == "true" ]; then
  OUTFILENAME="gitea"
  echo "Rename set to true, renaming to ${OUTFILENAME}"
else
  echo "Rename target name provided, renaming to ${RENAME}"
  OUTFILENAME="${RENAME}"
fi

FILE_WIHTOUT_PLATFORM="${FILEBASENAME}-${FILEVERSION}"

DOCKERDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit 3; pwd -P )"
WS_BASE="${DOCKERDIR%/docker}"

if [ ! -f "${DOCKERDIR}/archmap.txt" ]; then
  errcho "no archmap found"
  exit 4
fi

if [ ! -d "${DOCKERDIR}/bin" ]; then
  echo "Folder ${DOCKERDIR}/bin missing, creating it"
  mkdir -p "${DOCKERDIR}/bin"
fi

map () {
  local file="${1}"
  local bin_platform="${file##"./${FILE_WIHTOUT_PLATFORM}-"}"
  local platform
  echo "handling ${file} with detected bin_platform $bin_platform"
  if ! grep -q "$bin_platform" "${DOCKERDIR}/archmap.txt"; then
    errcho "no matching platform for $file found"
    return 0
  fi

  platform="$(grep "$bin_platform" "${DOCKERDIR}/archmap.txt" | cut -d':' -f2)"
  echo "Docker platform parsed: $platform"
  if [ ! -d "${DOCKERDIR}/bin/$platform" ]; then
    echo "First item for this platform, creating directory \"${DOCKERDIR}/bin/$platform\""
    mkdir -p "${DOCKERDIR}/bin/$platform"
  fi
  echo "Mapped ${file} to $platform, copying to \"${DOCKERDIR}/bin/$platform/${OUTFILENAME}\""
  cp "${file}" "${DOCKERDIR}/bin/$platform/${OUTFILENAME}"
}

cd "$WS_BASE/${BINARYLOOKUPPATH}" || exit 5

while IFS= read -r -d $'\0' infile; do
  map "$infile"
done < <(find . -maxdepth 1 -type f -name "${FILE_WIHTOUT_PLATFORM}-*" -print0)

cd "$WS_BASE"
