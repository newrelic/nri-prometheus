// Package scraper ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package scraper

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
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
