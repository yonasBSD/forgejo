#!/bin/bash

set -e

echo Update the options/license directory with licenses that have been verified to be Free Software
echo See https://spdx.org/licenses/

if ! test -f tarball ; then
    wget -c https://api.github.com/repos/spdx/license-list-data/tarball
fi
dir=$(pwd)
tmp=$(mktemp -d)
cd $tmp
tar -zxv --wildcards --strip-components 1 -f $dir/tarball spdx-license-list-data-*/text spdx-license-list-data-*/json/licenses.json
rm -r $dir/options/license
mkdir $dir/options/license

#
# Verified by either OSI or FSF
#
jq --raw-output '.licenses[] | select(.isDeprecatedLicenseId == false and (.isFsfLibre == true or .isOsiApproved == true)) | .licenseId' < json/licenses.json | while read licenseid ; do
    mv text/$licenseid.txt $dir/options/license/$licenseid
done

#
# See https://codeberg.org/forgejo/forgejo/pulls/1409 for a discussion on why it is removed
#
rm $dir/options/license/Jam

#
# Free Software licenses from Creative Commons
#
for license in text/CC-{BY,BY-SA}-?.?.txt ; do
    licenseid=$(basename $license .txt)
    mv text/$licenseid.txt $dir/options/license/$licenseid
done
