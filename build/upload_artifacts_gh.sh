#!/bin/bash
#
#
# Upload dist artifacts to GH Release assets
#
#

cd dist
for package in $(find . -regex ".*\.\(msi\|rpm\|deb\|zip\|tar.gz\)");do
  echo "===> Uploading package: '${package}'"
  gh release upload ${TAG} ${package}
done
