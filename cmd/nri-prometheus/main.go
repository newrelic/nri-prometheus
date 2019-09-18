// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
)

func loadConfig() (*scraper.Config, error) {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath("/etc/nri-prometheus/")
	cfg.AddConfigPath(".")
	scraper.LoadViperDefaults(cfg)

	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var scraperCfg scraper.Config
	scraper.BindViperEnv(cfg, scraperCfg)
	err = cfg.Unmarshal(&scraperCfg)
	if err != nil {
		return nil, err
	}
	return &scraperCfg, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("while getting configuration options")
	}

	if cfg.LicenseKey == "" {
		logrus.Fatal("LICENSE_KEY is required and can't be empty")
	}

	scraper.Run(cfg)
}
