package telemetry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/go-telemetry-sdk/internal"
)

// Harvester aggregates and reports metrics and spans.
type Harvester struct {
	// These fields are not modified after Harvester creation.  They may be
	// safely accessed without locking.
	config               Config
	commonAttributesJSON json.RawMessage

	// lock protects the mutable fields below.
	lock        sync.Mutex
	lastHarvest time.Time
	rawMetrics  []Metric
	spans       []Span
}

const (
	defaultTimeout       = 5 * time.Second
	defaultHarvestPeriod = 5 * time.Second
	defaultMaxTries      = 3
	defaultRetryBackoff  = time.Second
)

// NewHarvester creates a new harvester.
func NewHarvester(options ...func(*Config)) *Harvester {
	cfg := Config{
		Client: &http.Client{
			Timeout: defaultTimeout,
		},
		HarvestPeriod: defaultHarvestPeriod,
		MaxTries:      defaultMaxTries,
		RetryBackoff:  defaultRetryBackoff,
	}
	for _, opt := range options {
		opt(&cfg)
	}

	h := &Harvester{
		config:      cfg,
		lastHarvest: time.Now(),
	}

	// Marshal the common attributes to JSON here to avoid doing it on every
	// harvest.  This also has the benefit that it avoids race conditions if
	// the consumer modifies the CommonAttributes map after calling
	// NewHarvester.
	if nil != h.config.CommonAttributes {
		attrs := vetAttributes(h.config.CommonAttributes, h.config.logError)
		attributesJSON, err := json.Marshal(attrs)
		if err != nil {
			h.config.logError(map[string]interface{}{
				"err":     err.Error(),
				"message": "error marshaling common attributes",
			})
		} else {
			h.commonAttributesJSON = attributesJSON
		}
		h.config.CommonAttributes = nil
	}

	spawnHarvester := h.needsHarvestThread()

	h.config.logDebug(map[string]interface{}{
		"event":                   "harvester created",
		"api-key":                 h.config.APIKey,
		"license-key":             h.config.LicenseKey,
		"harvest-period-seconds":  h.config.HarvestPeriod.Seconds(),
		"spawn-harvest-goroutine": spawnHarvester,
		"metrics-url-override":    h.config.MetricsURLOverride,
		"spans-url-override":      h.config.SpansURLOverride,
		"collect-metrics":         h.collectMetrics(),
		"collect-spans":           h.collectSpans(),
		"version":                 internal.Version,
	})

	if spawnHarvester {
		go h.harvest()
	}

	return h
}

func (h *Harvester) needsHarvestThread() bool {
	if 0 == h.config.HarvestPeriod {
		return false
	}
	if !h.collectMetrics() && !h.collectSpans() {
		return false
	}
	return true
}

func (h *Harvester) collectMetrics() bool {
	if nil == h {
		return false
	}
	if "" == h.config.APIKey {
		return false
	}
	return true
}

func (h *Harvester) collectSpans() bool {
	if nil == h {
		return false
	}
	if "" == h.config.LicenseKey {
		return false
	}
	return true
}

// RecordSpan records the given span.
func (h *Harvester) RecordSpan(s Span) {
	if !h.collectSpans() {
		return
	}

	// Marshal attributes immediately to avoid holding a reference to a map
	// that the consumer could change.
	if nil != s.Attributes {
		attributes := vetAttributes(s.Attributes, h.config.logError)
		var err error
		s.AttributesJSON, err = json.Marshal(attributes)
		if err != nil {
			h.config.logError(map[string]interface{}{
				"err":     err.Error(),
				"message": "error marshaling attributes",
				"span":    s.Name,
			})
		}
	}
	s.Attributes = nil

	h.lock.Lock()
	defer h.lock.Unlock()

	h.spans = append(h.spans, s)
}

// RecordMetric adds a fully formed metric.  This metric is not aggregated with
// any other metrics and is never dropped.  The Timestamp field must be
// specified on Gauge metrics.  The Timestamp/Interval fields on Count and
// Summary are optional and will be assumed to be the harvester batch times if
// unset.
func (h *Harvester) RecordMetric(m Metric) {
	if !h.collectMetrics() {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()

	h.rawMetrics = append(h.rawMetrics, m)
}

type response struct {
	statusCode int
	body       []byte
	err        error
	retryAfter string
}

func (r response) needsRetry(cfg *Config) (bool, time.Duration) {
	switch r.statusCode {
	case 202, 200:
		// success
		return false, 0
	case 400, 403, 404, 405, 411, 413:
		// errors that should not retry
		return false, 0
	case 429:
		// special retry backoff time
		if "" != r.retryAfter {
			// Honor Retry-After header value in seconds
			if d, err := time.ParseDuration(r.retryAfter + "s"); nil == err {
				if d > cfg.RetryBackoff {
					return true, d
				}
			}
		}
		return true, cfg.RetryBackoff
	default:
		// all other errors should retry
		return true, cfg.RetryBackoff
	}
}

func postData(req *http.Request, client *http.Client) response {
	resp, err := client.Do(req)
	if nil != err {
		return response{err: fmt.Errorf("error posting data: %v", err)}
	}
	defer resp.Body.Close()

	r := response{
		statusCode: resp.StatusCode,
		retryAfter: resp.Header.Get("Retry-After"),
	}

	// On success, metrics ingest returns 202, span ingest returns 200.
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		r.body, _ = ioutil.ReadAll(resp.Body)
	} else {
		r.err = fmt.Errorf("unexpected post response code: %d: %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return r
}

func (h *Harvester) swapOutMetrics(now time.Time) []Request {
	if !h.collectMetrics() {
		return nil
	}

	h.lock.Lock()
	lastHarvest := h.lastHarvest
	h.lastHarvest = now
	rawMetrics := h.rawMetrics
	h.rawMetrics = nil
	h.lock.Unlock()

	if 0 == len(rawMetrics) {
		return nil
	}

	batch := &MetricBatch{
		Timestamp:      lastHarvest,
		Interval:       now.Sub(lastHarvest),
		AttributesJSON: h.commonAttributesJSON,
		Metrics:        rawMetrics,
	}
	reqs, err := batch.NewRequests(h.config.APIKey, h.config.MetricsURLOverride)
	if nil != err {
		h.config.logError(map[string]interface{}{
			"err":     err.Error(),
			"message": "error creating requests for metrics",
		})
		return nil
	}
	return reqs
}

func (h *Harvester) swapOutSpans() []Request {
	if !h.collectSpans() {
		return nil
	}

	h.lock.Lock()
	sps := h.spans
	h.spans = nil
	h.lock.Unlock()

	if nil == sps {
		return nil
	}
	batch := &SpanBatch{Spans: sps}
	reqs, err := batch.NewRequests(h.config.LicenseKey, h.config.SpansURLOverride)
	if nil != err {
		h.config.logError(map[string]interface{}{
			"err":     err.Error(),
			"message": "error creating requests for spans",
		})
		return nil
	}
	return reqs
}

func harvestRequest(req Request, cfg *Config) {
	var tries int
	for {
		cfg.logDebug(map[string]interface{}{
			"event": "data post",
			"url":   req.Request.URL.String(),
			"data":  jsonString(req.UncompressedBody),
		})

		tries++
		resp := postData(req.Request, cfg.Client)

		if nil != resp.err {
			cfg.logError(map[string]interface{}{
				"err": resp.err.Error(),
			})
		} else {
			cfg.logDebug(map[string]interface{}{
				"event":  "data post response",
				"status": resp.statusCode,
				"body":   jsonOrString(resp.body),
			})
		}
		retry, backoff := resp.needsRetry(cfg)
		if !retry {
			return
		}
		if tries >= cfg.MaxTries {
			cfg.logError(map[string]interface{}{
				"event":     "data post retry limit reached",
				"max-tries": cfg.MaxTries,
				"message":   "dropping data",
			})
			return
		}
		time.Sleep(backoff)
	}
}

// HarvestNow synchronously harvests telemetry data
func (h *Harvester) HarvestNow() {
	if nil != h && nil != h.config.BeforeHarvestFunc {
		h.config.BeforeHarvestFunc(h)
	}
	for _, req := range h.swapOutMetrics(time.Now()) {
		harvestRequest(req, &h.config)
	}
	for _, req := range h.swapOutSpans() {
		harvestRequest(req, &h.config)
	}
}

// harvest concurrently harvests telemetry data
func (h *Harvester) harvest() {
	ticker := time.NewTicker(h.config.HarvestPeriod)
	for range ticker.C {
		go h.HarvestNow()
	}
}
