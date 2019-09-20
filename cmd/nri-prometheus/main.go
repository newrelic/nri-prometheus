// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("while loading configuration")
	}

	scraper.Run(cfg)
}
