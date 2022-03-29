// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"

	promcli "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/newrelic/nri-prometheus/internal/pkg/prometheus"
)

// Fetcher provides fetching functionality to a set of Prometheus endpoints
type Fetcher interface {
	// Fetch fetches data from a set of Prometheus /metrics endpoints. It ignores failed endpoints.
	// It returns each data entry from a channel, assuming this function may run in background.
	Fetch(t []endpoints.Target) <-chan TargetMetrics
}

// TargetMetrics holds a pair of fetched metrics with the Target where they have been targetted from
type TargetMetrics struct {
	Metrics []Metric
	Target  endpoints.Target
}

// NewTLSConfig creates a TLS configuration. If a CA cert is provided it is
// read and used to validate the scrape target's certificate properly.
func NewTLSConfig(CAFile string, InsecureSkipVerify bool) (*tls.Config, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: InsecureSkipVerify}

	if len(CAFile) > 0 {
		caCertPool := x509.NewCertPool()
		caCert, err := ioutil.ReadFile(CAFile)
		if err != nil {
			return nil, fmt.Errorf("unable to use specified CA cert %s: %s", CAFile, err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}
	return tlsConfig, nil
}

// newRoundTripper creates a new roundtripper with the specified TLS
// configuration.
func newRoundTripper(CaFile string, InsecureSkipVerify bool) (http.RoundTripper, error) {
	tlsConfig, err := NewTLSConfig(CaFile, InsecureSkipVerify)
	if err != nil {
		return nil, err
	}
	return newDefaultRoundTripper(tlsConfig), nil
}

func newDefaultRoundTripper(tlsConfig *tls.Config) http.RoundTripper {
	var rt http.RoundTripper = &http.Transport{
		MaxIdleConns:        20000,
		MaxIdleConnsPerHost: 1000, // see https://github.com/golang/go/issues/13801
		DisableKeepAlives:   false,
		DisableCompression:  true,
		// 5 minutes is typically above the maximum sane scrape interval. So we can
		// use keepalive for all configurations.
		IdleConnTimeout: 5 * time.Minute,
		TLSClientConfig: tlsConfig,
	}
	return rt
}

// NewBearerAuthFileRoundTripper adds the bearer token read from the provided file to a request unless
// the authorization header has already been set. This file is read for every request.
func NewBearerAuthFileRoundTripper(bearerFile string, rt http.RoundTripper) http.RoundTripper {
	return &bearerAuthFileRoundTripper{bearerFile, rt}
}

type bearerAuthFileRoundTripper struct {
	bearerFile string
	rt         http.RoundTripper
}

func (rt *bearerAuthFileRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) == 0 {
		b, err := ioutil.ReadFile(rt.bearerFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read bearer token file %s: %s", rt.bearerFile, err)
		}
		bearerToken := strings.TrimSpace(string(b))

		req = cloneRequest(req)
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	return rt.rt.RoundTrip(req)
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// Shallow copy of the struct.
	r2 := new(http.Request)
	*r2 = *r
	// Deep copy of the Header.
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}

// NewFetcher returns the default Fetcher implementation
func NewFetcher(fetchDuration time.Duration, fetchTimeout time.Duration, workerThreads int, BearerTokenFile string, CaFile string, InsecureSkipVerify bool, queueLength int) Fetcher {
	roundTripper, _ := newRoundTripper(CaFile, InsecureSkipVerify)
	client := &http.Client{
		Transport: roundTripper,
		Timeout:   fetchTimeout,
	}

	// Some endpoints, namely kubelet/cadvisor, will also require an authenticated requests.
	bearerTokenRoundTripper := NewBearerAuthFileRoundTripper(BearerTokenFile, roundTripper)
	bearerTokenClient := &http.Client{
		Transport: bearerTokenRoundTripper,
		Timeout:   fetchTimeout,
	}

	return &prometheusFetcher{
		workerThreads: workerThreads,
		queueLength:   queueLength,
		httpClient:    client,
		bearerClient:  bearerTokenClient,
		duration:      fetchDuration,
		fetchTimeout:  fetchTimeout,
		getMetrics:    prometheus.Get,
		log:           logrus.WithField("component", "Fetcher"),
	}
}

type prometheusFetcher struct {
	workerThreads int
	queueLength   int
	duration      time.Duration
	fetchTimeout  time.Duration
	httpClient    prometheus.HTTPDoer
	bearerClient  prometheus.HTTPDoer
	// Provides IoC for better testability. Its usual value is 'prometheus.Get'.
	getMetrics func(httpClient prometheus.HTTPDoer, url string) (prometheus.MetricFamiliesByName, error)
	log        *logrus.Entry
}

// Fetch implementation runs the connections to many targets in parallel, limited by the maxTargetConnections constant,
// and submits TargetMetrics entries by the buffered channel, as long as they are retrieved
func (pf *prometheusFetcher) Fetch(targets []endpoints.Target) <-chan TargetMetrics {
	results := make(chan TargetMetrics, pf.queueLength)
	finishedTasks := sync.WaitGroup{}
	finishedTasks.Add(len(targets))
	prometheus.ResetTotalScrapedPayload()

	targetChan := make(chan endpoints.Target, len(targets))

	pf.log.WithFields(logrus.Fields{
		"component":    "fetcher",
		"thread_count": pf.workerThreads,
	}).Debug("Starting fetch worker threads...")

	for i := 0; i < pf.workerThreads; i++ {
		go pf.work(targetChan, &finishedTasks, results)
	}

	go func() {
		// Starts processing targets with some time of separation, to avoid buffering
		// too many metrics and generating peaks of memory that multiplies the heap and the
		// container working set
		nTargets := len(targets)
		if nTargets == 0 {
			pf.log.
				WithField("component", "fetcher").
				Info("Target list for fetching metrics is empty")
			return
		}

		// Slowly release the targets for this scrape cycle.
		// All targets should be added to the targetChan in half the duration of the scrape cycle,
		// giving slow targets time to finish before the new cycle starts.
		// E.g.
		// scrape cycle = 30 seconds, targets = 10
		// 30 / 2 / 10 = 1.5 seconds
		// After 15 seconds all targets are added to the queue, with 15 seconds left in the cycle
		ticker := time.NewTicker((pf.duration / 2) / time.Duration(nTargets))
		defer ticker.Stop()
		for _, target := range targets {
			targetChan <- target
			<-ticker.C
		}
	}()

	go func() {
		// The result channel needs to be closed so the rule processor knows when to stop
		// reading from it.
		finishedTasks.Wait()
		pf.log.WithField("component", "fetcher").Debug("Finished fetch process.")
		close(targetChan)
		close(results)
	}()
	return results
}

// work fetch the metrics of targets, pushing results to a channel and marking work as done.
func (pf *prometheusFetcher) work(targets <-chan endpoints.Target, wg *sync.WaitGroup, results chan<- TargetMetrics) {
	for target := range targets {
		if mfs, err := pf.fetch(target); err == nil {
			results <- TargetMetrics{
				Metrics: convertPromMetrics(pf.log, target.Name, mfs),
				Target:  target,
			}
		} else {
			pf.log.WithError(err).Warn("error while scraping target")
		}
		wg.Done()
	}
}

func (pf *prometheusFetcher) fetch(t endpoints.Target) (prometheus.MetricFamiliesByName, error) {
	pf.log.WithField("target", t.Name).Debug("fetching URL: ", t.URL)
	timer := promcli.NewTimer(promcli.ObserverFunc(fetchTargetDurationMetric.WithLabelValues(t.Name).Set))
	httpClient := pf.httpClient

	if isMutualTLSTarget(t) {
		rt, err := NewMutualTLSRoundTripper(t.TLSConfig)
		if err != nil {
			pf.log.WithError(err).Warnf("Error reading mTLS certs for %s (%s) ", t.Name, t.URL.String())
			fetchErrorsTotalMetric.WithLabelValues(t.Name).Set(1)
		}
		httpClient = &http.Client{
			Transport: rt,
			Timeout:   pf.fetchTimeout,
		}
	}

	// If this target needs the bearer token, we will use the authenticated client to make the request.
	if t.UseBearer {
		httpClient = pf.bearerClient
	}

	mfs, err := pf.getMetrics(httpClient, t.URL.String())
	timer.ObserveDuration()
	if err != nil {
		pf.log.WithError(err).Warnf("fetching Prometheus metrics: %s (%s)", t.URL.String(), t.Object.Name)
		fetchErrorsTotalMetric.WithLabelValues(t.Name).Set(1)
	}
	fetchesTotalMetric.WithLabelValues(t.Name).Set(1)
	return mfs, err
}

func isMutualTLSTarget(t endpoints.Target) bool {
	// If any of these is present it means we're looking at an mTLS-enabled target.
	// These targets need their own HTTP client because of very unique and different TLS
	// configuration.
	return t.TLSConfig != endpoints.TLSConfig{}
}

// NewMutualTLSRoundTripper creates a new roundtripper with the specified Mutual TLS
// configuration.
func NewMutualTLSRoundTripper(cfg endpoints.TLSConfig) (http.RoundTripper, error) {
	// Load our TLS key pair to use for authentication
	cert, err := tls.LoadX509KeyPair(cfg.CertFilePath, cfg.KeyFilePath)
	if err != nil {
		return nil, err
	}

	// Load our CA certificate
	clientCACert, err := ioutil.ReadFile(cfg.CaFilePath)
	if err != nil {
		return nil, err
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            clientCertPool,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	tlsConfig.BuildNameToCertificate()

	rt := newDefaultRoundTripper(tlsConfig)
	return rt, nil
}

type (
	metricValue interface{}
	metricType  string
)

//nolint:golint
const (
	metricType_COUNTER   metricType = "count"
	metricType_GAUGE     metricType = "gauge"
	metricType_SUMMARY   metricType = "summary"
	metricType_HISTOGRAM metricType = "histogram"
)

// Metric represents a Prometheus metric.
// https://prometheus.io/docs/concepts/data_model/
type Metric struct {
	name       string
	value      metricValue
	metricType metricType
	attributes labels.Set
}

var supportedMetricTypes = map[dto.MetricType]string{
	dto.MetricType_COUNTER:   "counter",
	dto.MetricType_GAUGE:     "gauge",
	dto.MetricType_HISTOGRAM: "histogram",
	dto.MetricType_SUMMARY:   "summary",
	dto.MetricType_UNTYPED:   "untyped",
}

func convertPromMetrics(log *logrus.Entry, targetName string, mfs prometheus.MetricFamiliesByName) []Metric {
	var metricsCap int
	for _, mf := range mfs {
		mtype, ok := supportedMetricTypes[mf.GetType()]
		if !ok {
			continue
		}
		metricsCap += len(mf.Metric)
		totalTimeseriesByTargetAndTypeMetric.WithLabelValues(mtype, targetName).Add(float64(len(mf.Metric)))
		totalTimeseriesByTypeMetric.WithLabelValues(mtype).Add(float64(len(mf.Metric)))
		totalTimeseriesByTargetMetric.WithLabelValues(targetName).Add(float64(len(mf.Metric)))
	}
	totalTimeseriesMetric.Add(float64(metricsCap))

	metrics := make([]Metric, 0, metricsCap)
	for mname, mf := range mfs {
		ntype := mf.GetType()
		mtype, ok := supportedMetricTypes[ntype]
		if !ok {
			continue
		}
		for _, m := range mf.GetMetric() {
			var value interface{}
			var nrType metricType
			switch ntype {
			case dto.MetricType_UNTYPED:
				value = m.GetUntyped().GetValue()
				nrType = metricType_GAUGE
			case dto.MetricType_COUNTER:
				value = m.GetCounter().GetValue()
				nrType = metricType_COUNTER
			case dto.MetricType_GAUGE:
				value = m.GetGauge().GetValue()
				nrType = metricType_GAUGE
			case dto.MetricType_SUMMARY:
				value = m.GetSummary()
				nrType = metricType_SUMMARY
			case dto.MetricType_HISTOGRAM:
				value = m.GetHistogram()
				nrType = metricType_HISTOGRAM
			default:
				if log.Level <= logrus.DebugLevel {
					log.WithField("target", targetName).Debugf("metric type not supported: %s", mtype)
				}
				continue
			}
			attrs := map[string]interface{}{}
			attrs["targetName"] = targetName
			for _, l := range m.GetLabel() {
				attrs[l.GetName()] = l.GetValue()
			}
			// nrMetricType and promMetricType attributes were created as a debugging tool, because some prometheus metric types weren't supported natively by NR.
			attrs["nrMetricType"] = string(nrType)
			attrs["promMetricType"] = mtype
			metrics = append(
				metrics,
				Metric{
					name:       mname,
					metricType: nrType,
					value:      value,
					attributes: attrs,
				},
			)
		}
	}
	return metrics
}

// MarshalJSON marshals a metric to json
func (m *Metric) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name       string      `json:"name"`
		Value      metricValue `json:"value"`
		Type       metricType  `json:"type"`
		Attributes labels.Set  `json:"attributes"`
	}{
		Name:       m.name,
		Value:      m.value,
		Type:       m.metricType,
		Attributes: m.attributes,
	})
}
