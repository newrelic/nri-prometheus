#!/bin/bash
#
#
# Upload dist artifacts to GH Release assets
#
#

# REPO here is only necessary for forks. It can be removed when this is merged into the original repo
REPO=$1

cd dist
for package in $(find  -regex ".*\.\(msi\|rpm\|deb\|zip\|tar.gz\)");do
  echo "===> Uploading to GH $TAG: ${package}"
  gh release upload ${TAG} ${package} -R ${REPO}
done
