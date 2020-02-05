package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	req.Header.Add("Api-Key", licenseKey)

	rt := new(mockedRoundTripper)
	rt.On("RoundTrip", req).Return().Run(func(args mock.Arguments) {
		req := args.Get(0).(*http.Request)
		assert.Equal(t, licenseKey, req.Header.Get("X-License-Key"))
		assert.Equal(t, "", req.Header.Get("Api-Key"))
	})
	tr := newLicenseKeyRoundTripper(rt, licenseKey)

	_, _ = tr.RoundTrip(req)
	rt.AssertExpectations(t)
}
