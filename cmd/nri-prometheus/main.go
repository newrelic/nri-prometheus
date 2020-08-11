// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/sirupsen/logrus"
)

//go:generate go run -ldflags "-X main.majorMinorVersion=$MAJOR_MINOR_VERSION -X main.preReleaseVersion=$PRE_RELEASE_VERSION -X main.fullVersion=$FULL_VERSION" ../../tools/deploy-yaml/main.go
func main() {
	cfg, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("while loading configuration")
	}

	if cfg.StandAlone {
		err = scraper.Run(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("error occurred while running scraper")
		}
	} else {
		logrus.Info("New behaviour!")
	}
}
