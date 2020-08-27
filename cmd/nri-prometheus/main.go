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

	if cfg.Standalone {
		err = scraper.Run(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("error occurred while running scraper")
		}
	} else {
		// todo create a proper emitter that add metrics to an integration and prints them to stdout once scraping and processing is performed
		cfg.Emitters = []string{"integrationSDK4"}
		err = scraper.RunOnce(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("error occurred while running scraper")
		}
	}
}
