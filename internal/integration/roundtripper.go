// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import "net/http"

// licenseKeyRoundTripper adds the infra license key to every request.
type licenseKeyRoundTripper struct {
	licenseKey string
	rt         http.RoundTripper
}

// RoundTrip wraps the `RoundTrip` method removing the "Api-Key"
// replacing it with "X-License-Key".
func (t licenseKeyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Del("Api-Key")
	req.Header.Add("X-License-Key", t.licenseKey)
	return t.rt.RoundTrip(req)
}

// newLicenseKeyRoundTripper wraps the given http.RoundTripper and inserts
// the appropriate headers for using the NewRelic licenseKey.
func newLicenseKeyRoundTripper(
	rt http.RoundTripper,
	licenseKey string,
) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return licenseKeyRoundTripper{
		licenseKey: licenseKey,
		rt:         rt,
	}
}
