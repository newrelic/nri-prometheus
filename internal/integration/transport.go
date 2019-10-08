package integration

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

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

// newInfraTransport clones the given emitter, sets TLS proxy support
// and returns a wrapper over the cloned Transport that inserts the
// appropriate headers for using the NewRelic licenseKey.
func newInfraTransport(
	rt http.RoundTripper,
	licenseKey string,
	tlsConfig *tls.Config,
	proxyURL *url.URL,
) http.RoundTripper {

	if rt == nil {
		rt = http.DefaultTransport
	}

	t, ok := rt.(*http.Transport)
	if !ok {
		return infraTransport{
			licenseKey: licenseKey,
			rt:         rt,
		}
	}

	t = t.Clone()
	if proxyURL != nil {
		t.Proxy = http.ProxyURL(proxyURL)
	}
	t.TLSClientConfig = tlsConfig
	return infraTransport{
		licenseKey: licenseKey,
		rt:         t,
	}
}
