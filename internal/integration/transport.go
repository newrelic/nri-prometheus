package integration

import "net/http"

// infraTransport adds the infra license key to every request.
type infraTransport struct {
	licenseKey string
	rt         http.RoundTripper
}

// RoundTrip wraps the `RoundTrip` method removing the "X-Insert-Key"
// replacing it with "X-License-Key".
func (t infraTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Del("X-Insert-Key")
	req.Header.Add("X-License-Key", t.licenseKey)
	return t.rt.RoundTrip(req)
}

func newInfraTransport(rt http.RoundTripper, licenseKey string) http.RoundTripper {

	if rt == nil {
		rt = http.DefaultTransport
	}
	return infraTransport{
		licenseKey: licenseKey,
		rt:         rt,
	}
}
