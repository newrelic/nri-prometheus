// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/sirupsen/logrus"
)

//go:generate go run -ldflags "-X main.majorVersion=$MAJOR_VERSION -X main.minorVersion=$MINOR_VERSION" ../../tools/deploy-yaml/main.go
func main() {
	cfg, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("while loading configuration")
	}

	err = scraper.Run(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("error occurred while running scraper")
	}
}
