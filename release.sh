# Copyright 2019 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#! /bin/bash
if [[ $# -eq 0 ]] ; then
    echo 'Please specify the new release tag'
    exit 0
fi

sed -i -r 's/(## Unreleased)/\1\n\n## '$1'/g' CHANGELOG.md
sed -i -r 's/(Version\s*=\s*\").*$/\1'$1'"/' internal/integration/integration.go
sed -i -r 's/(image\: newrelic\/kps\:).*$/\1'$1'/' deploy/nri-prometheus.yaml
