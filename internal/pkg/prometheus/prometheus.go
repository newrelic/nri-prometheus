// Package prometheus ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package prometheus

import (
	"io"
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// MetricFamiliesByName is a map of Prometheus metrics family names and their
// representation.
type MetricFamiliesByName map[string]dto.MetricFamily

// HTTPDoer executes http requests. It is implemented by *http.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type countReadCloser struct {
	innerReadCloser io.ReadCloser
	count           int
}

func (rc *countReadCloser) Close() error {
	return rc.innerReadCloser.Close()
}

func (rc *countReadCloser) Read(p []byte) (n int, err error) {
	n, err = rc.innerReadCloser.Read(p)
	rc.count += n
	return
}

// ResetTotalScrapedPayload resets the integration totalScrapedPayload
// metric.
func ResetTotalScrapedPayload() {
	totalScrapedPayload.Set(0)
}

// Get scrapes the given URL and decodes the retrieved payload.
func Get(client HTTPDoer, url string) (MetricFamiliesByName, error) {
	mfs := MetricFamiliesByName{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return mfs, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return mfs, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	countedBody := &countReadCloser{innerReadCloser: resp.Body}
	d := expfmt.NewDecoder(countedBody, expfmt.FmtText)
	for {
		var mf dto.MetricFamily
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mfs[mf.GetName()] = mf
	}

	bodySize := float64(countedBody.count)
	targetSize.With(prom.Labels{"target": url}).Set(bodySize)
	totalScrapedPayload.Add(bodySize)
	return mfs, nil
}
