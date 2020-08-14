// Package scraper ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package scraper

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLicenseKeyMasking(t *testing.T) {

	const licenseKeyString = "secret"
	licenseKey := LicenseKey(licenseKeyString)

	t.Run("Masks licenseKey in fmt.Sprintf (which uses same logic as Printf)", func(t *testing.T) {
		masked := fmt.Sprintf("%s", licenseKey)
		assert.Equal(t, masked, maskedLicenseKey)
	})

	t.Run("Masks licenseKey in fmt.Sprint (which uses same logic as Print)", func(t *testing.T) {
		masked := fmt.Sprint(licenseKey)
		assert.Equal(t, masked, maskedLicenseKey)
	})

	t.Run("Masks licenseKey in %#v formatting", func(t *testing.T) {
		masked := fmt.Sprintf("%#v", licenseKey)
		if strings.Contains(masked, licenseKeyString) {
			t.Error("found licenseKey in formatted string")
		}
		if !strings.Contains(masked, maskedLicenseKey) {
			t.Error("could not find masked password in formatted string")
		}
	})

	t.Run("Able to convert licenseKey back to string", func(t *testing.T) {
		unmasked := string(licenseKey)
		assert.Equal(t, licenseKeyString, unmasked)
	})
}

func TestLogrusDebugPrintMasksLicenseKey(t *testing.T) {

	const licenseKey = "SECRET_LICENSE_KEY"

	cfg := Config{
		LicenseKey: licenseKey,
	}

	var b bytes.Buffer

	logrus.SetOutput(&b)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debugf("Config: %#v", cfg)

	msg := b.String()
	if strings.Contains(msg, licenseKey) {
		t.Error("Log output contains the license key")
	}
	if !strings.Contains(msg, maskedLicenseKey) {
		t.Error("Log output does not contain the masked licenseKey")
	}
}

func TestConfigParseWithCustomType(t *testing.T) {

	const licenseKey = "MY_LICENSE_KEY"
	cfgStr := []byte(fmt.Sprintf(`LICENSE_KEY: %s`, licenseKey))

	vip := viper.New()
	vip.SetConfigType("yaml")
	err := vip.ReadConfig(bytes.NewBuffer(cfgStr))
	require.NoError(t, err)

	var cfg Config
	err = vip.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, licenseKey, string(cfg.LicenseKey))
}

func TestRunIntegrationOnce(t *testing.T) {
	dat, err := ioutil.ReadFile("./testData/testData.prometheus")
	require.NoError(t, err)
	counter := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write(dat)
		counter++
	}))
	defer srv.Close()

	c := &Config{
		TargetConfigs: []endpoints.TargetConfig{
			{
				URLs: []string{srv.URL, srv.URL},
			},
		},
		Emitters:       []string{"stdout"},
		Standalone:     false,
		Verbose:        true,
		ScrapeDuration: "500ms",
	}
	err = RunOnce(c)
	require.NoError(t, err)
	require.Equal(t, 2, counter, "the scraper should have hit the mock exactly twice")

	//todo once the emitter works properly we should test that the scraped data is the expected one
}

func TestScrapingAnsweringWithError(t *testing.T) {
	counter := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write(nil)
		counter++
	}))

	defer srv.Close()

	c := &Config{
		TargetConfigs: []endpoints.TargetConfig{
			{
				URLs: []string{srv.URL, srv.URL},
			},
		},
		Emitters:       []string{"stdout"},
		Standalone:     false,
		Verbose:        true,
		ScrapeDuration: "500ms",
	}
	err := RunOnce(c)
	// Currently no error is returned in case a scraper does not return any data / err status code
	require.NoError(t, err)
	require.Equal(t, 2, counter, "the scraper should have hit the mock exactly twice")

}

func TestScrapingAnsweringUnexpectedData(t *testing.T) {
	counter := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{not valid string}`n`n\n\n\n Not valid string "))
		counter++
	}))

	defer srv.Close()

	c := &Config{
		TargetConfigs: []endpoints.TargetConfig{
			{
				URLs: []string{srv.URL, srv.URL},
			},
		},
		Emitters:       []string{"stdout"},
		Standalone:     false,
		Verbose:        true,
		ScrapeDuration: "500ms",
	}
	err := RunOnce(c)
	// Currently no error is returned in case a scraper does not return any data / err status code
	require.NoError(t, err)
	require.Equal(t, 2, counter, "the scraper should have hit the mock exactly twice")

}

func TestScrapingNotAnswering(t *testing.T) {

	c := &Config{
		TargetConfigs: []endpoints.TargetConfig{
			{
				URLs: []string{"127.1.1.0:9012"},
			},
		},
		Emitters:       []string{"stdout"},
		Standalone:     false,
		Verbose:        true,
		ScrapeDuration: "500ms",
		ScrapeTimeout:  time.Duration(500) * time.Millisecond,
	}
	err := RunOnce(c)
	// Currently no error is returned in case a scraper does not return any data / err status code
	require.NoError(t, err)

}
