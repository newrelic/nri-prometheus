// Package scraper ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package scraper

import (
	"fmt"
	"strings"
	"testing"

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
