#!/bin/bash
set -e
#
#
# Gets dist/zip_dirty created by Goreleaser and reorganize inside files
#
#
INTEGRATION=$1
ARCH=$2
TAG=$3

# REPO here is only necessary for forks. It can be removed when this is merged into the original repo
REPO=$4

echo "Publishing MSI to repo ${REPO_FULL_NAME}..."
gh release upload ${TAG} "build/package/windows/nri-${ARCH}-installer/bin/Release/nri-${INTEGRATION}-${ARCH}.${TAG:1}.msi" -R ${REPO}