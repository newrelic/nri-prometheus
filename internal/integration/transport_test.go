package integration

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockedRoundTripper struct {
	mock.Mock
}

func (m *mockedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	m.Called(req)
	return &http.Response{}, nil
}

func TestRoundTripHeaderDecoration(t *testing.T) {
	licenseKey := "myLicenseKey"
	req := &http.Request{Header: make(http.Header)}
	req.Header.Add("X-Insert-Key", "insertKey")

	rt := new(mockedRoundTripper)
	rt.On("RoundTrip", req).Return().Run(func(args mock.Arguments) {
		req := args.Get(0).(*http.Request)
		assert.Equal(t, licenseKey, req.Header.Get("X-License-Key"))
		assert.Equal(t, "", req.Header.Get("X-Insert-Key"))
	})
	tr := newInfraTransport(rt, licenseKey, nil, nil)

	_, _ = tr.RoundTrip(req)
	rt.AssertExpectations(t)
}

func TestSetTransportTLSConfig(t *testing.T) {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	rt := newInfraTransport(nil, "", tlsConfig, nil).(infraTransport).rt
	tr, ok := rt.(*http.Transport)
	assert.True(t, ok)
	assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)
}

func TestSetTransportProxy(t *testing.T) {
	proxyStr := "http://myproxy:444"
	proxyURL, err := url.Parse(proxyStr)
	require.NoError(t, err)
	rt := newInfraTransport(nil, "", nil, proxyURL).(infraTransport).rt
	tr, ok := rt.(*http.Transport)
	assert.True(t, ok)
	actualProxyURL, err := tr.Proxy(&http.Request{})
	require.NoError(t, err)
	assert.Equal(t, proxyURL, actualProxyURL)
}
