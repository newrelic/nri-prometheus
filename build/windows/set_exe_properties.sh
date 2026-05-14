#!/bin/bash
set -e
#
#
# Create the metadata for the exe's files, called by .goreleser as a hook in the build section
#
#
TAG=$1
INTEGRATION=$2

if [ -n "$1" ]; then
  echo "===> Tag is ${TAG}"
else
  # todo: exit here with error?
  echo "===> Tag not specified will be 0.0.0"
  TAG='0.0.0'
fi

MajorVersion=$(echo ${TAG:1} | cut -d "." -f 1)
MinorVersion=$(echo ${TAG:1} | cut -d "." -f 2)
PatchVersion=$(echo ${TAG:1} | cut -d "." -f 3)
BuildVersion='0'

Year=$(date +"%Y")
INTEGRATION_EXE="nri-${INTEGRATION}.exe"

mkdir -p ./winres

sed \
  -e "s/{MajorVersion}/$MajorVersion/g" \
  -e "s/{MinorVersion}/$MinorVersion/g" \
  -e "s/{PatchVersion}/$PatchVersion/g" \
  -e "s/{BuildVersion}/$BuildVersion/g" \
  -e "s/{Year}/$Year/g" \
  -e "s/{Integration}/nri-$INTEGRATION/g" \
  -e "s/{IntegrationExe}/$INTEGRATION_EXE/g" \
   ./build/windows/winres.json.template > ./winres/winres.json

go-winres make --arch 386,amd64 --out ./cmd/nri-prometheus/rsrc