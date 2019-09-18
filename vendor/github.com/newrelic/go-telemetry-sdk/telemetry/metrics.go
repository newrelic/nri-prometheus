package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/newrelic/go-telemetry-sdk/internal"
)

// Count is the metric type that counts the number of times an event occurred.
// This counter should be reset every time the data is reported, meaning the
// value reported represents the difference in count over the reporting time
// window.
//
// Example possible uses:
//
//  * the number of messages put on a topic
//  * the number of HTTP requests
//  * the number of errors thrown
//  * the number of support tickets answered
//
type Count struct {
	// Name is the name of this metric.
	Name string
	// Attributes is a map of attributes for this metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
	// Value is the value of this metric.
	Value float64
	// Timestamp is the start time of this metric's interval.  Timestamp may
	// be unset if it is set on the MetricBatch.
	Timestamp time.Time
	// Interval is the length of time for this metric.  Interval may be
	// unset if it is set on the MetricBatch.
	Interval time.Duration
}

// Metric is implemented by Count, Gauge, and Summary.
type Metric interface {
	writeJSON(buf *bytes.Buffer)
}

func writeTimestampInterval(w *internal.JSONFieldsWriter, timestamp time.Time, interval time.Duration) {
	if !timestamp.IsZero() {
		w.IntField("timestamp", timestamp.UnixNano()/(1000*1000))
	}
	if interval != 0 {
		w.IntField("interval.ms", interval.Nanoseconds()/(1000*1000))
	}
}

func (m Count) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	w.Buf.WriteByte('{')
	w.StringField("name", m.Name)
	w.StringField("type", "count")
	w.FloatField("value", m.Value)
	writeTimestampInterval(&w, m.Timestamp, m.Interval)
	if nil != m.Attributes {
		w.WriterField("attributes", internal.Attributes(m.Attributes))
	} else if nil != m.AttributesJSON {
		w.RawField("attributes", m.AttributesJSON)
	}
	w.Buf.WriteByte('}')
}

// Summary is the metric type used for reporting aggregated information about
// discrete events.   It provides the count, average, sum, min and max values
// over time.  All fields should be reset to 0 every reporting interval.
//
// Example possible uses:
//
//  * the duration and count of spans
//  * the duration and count of transactions
//  * the time each message spent in a queue
//
type Summary struct {
	// Name is the name of this metric.
	Name string
	// Attributes is a map of attributes for this metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
	// Count is the count of occurrences of this metric for this time period.
	Count float64
	// Sum is the sum of all occurrences of this metric for this time period.
	Sum float64
	// Min is the smallest value recorded of this metric for this time period.
	Min float64
	// Max is the largest value recorded of this metric for this time period.
	Max float64
	// Timestamp is the start time of this metric's interval.  Timestamp may
	// be unset if it is set on the MetricBatch.
	Timestamp time.Time
	// Interval is the length of time for this metric.  Interval may be
	// unset if it is set on the MetricBatch.
	Interval time.Duration
}

func (m Summary) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')

	w.StringField("name", m.Name)
	w.StringField("type", "summary")

	w.AddKey("value")
	buf.WriteByte('{')
	vw := internal.JSONFieldsWriter{Buf: buf}
	vw.FloatField("sum", m.Sum)
	vw.FloatField("count", m.Count)
	vw.FloatField("min", m.Min)
	vw.FloatField("max", m.Max)
	buf.WriteByte('}')

	writeTimestampInterval(&w, m.Timestamp, m.Interval)
	if nil != m.Attributes {
		w.WriterField("attributes", internal.Attributes(m.Attributes))
	} else if nil != m.AttributesJSON {
		w.RawField("attributes", m.AttributesJSON)
	}
	buf.WriteByte('}')
}

// Gauge is the metric type that records a value that can increase or decrease.
// It generally represents the value for something at a particular moment in
// time.  One typically records a Gauge value on a set interval.
//
// Example possible uses:
//
//  * the temperature in a room
//  * the amount of memory currently in use for a process
//  * the bytes per second flowing into Kafka at this exact moment in time
//  * the current speed of your car
//
type Gauge struct {
	// Name is the name of this metric.
	Name string
	// Attributes is a map of attributes for this metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
	// Value is the value of this metric.
	Value float64
	// Timestamp is the time at which this metric was gathered.
	Timestamp time.Time
}

func (m Gauge) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')
	w.StringField("name", m.Name)
	w.StringField("type", "gauge")
	w.FloatField("value", m.Value)
	writeTimestampInterval(&w, m.Timestamp, 0)
	if nil != m.Attributes {
		w.WriterField("attributes", internal.Attributes(m.Attributes))
	} else if nil != m.AttributesJSON {
		w.RawField("attributes", m.AttributesJSON)
	}
	buf.WriteByte('}')
}

// MetricBatch represents a single batch of metrics to report to New Relic.
//
// Timestamp/Interval are optional and can be used to represent the start and
// duration of the batch as a whole. Individual Count and Summary metrics may
// provide Timestamp/Interval fields which will take priority over the batch
// Timestamp/Interval. This is not the case for Gauge metrics which each require
// a Timestamp.
//
// Attributes are any attributes that should be applied to all metrics in this
// batch. Each metric type also accepts an Attributes field.
type MetricBatch struct {
	// Timestamp is the start time of all metrics in this MetricBatch.  This value
	// can be overridden by setting Timestamp on any particular metric.
	// Timestamp must be set here or on all metrics.
	Timestamp time.Time
	// Interval is the length of time for all metrics in this MetricBatch.  This
	// value can be overriden by setting Interval on any particular Count or
	// Summary metric.  Interval must be set to a non-zero value here or on
	// all Count and Summary metrics.
	Interval time.Duration
	// Attributes is a map of attributes to apply to all metrics in this MetricBatch.
	// They are included in addition to any attributes set on any particular
	// metric.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes to apply to all
	// metrics in this MetricBatch. It will only be sent if the Attributes field on
	// this MetricBatch is nil. These attributes are included in addition to any
	// attributes on any particular metric.
	AttributesJSON json.RawMessage
	// Metrics is the slice of metrics to send with this MetricBatch.
	Metrics []Metric
}

// AddMetric adds a Count, Gauge, or Summary metric to a MetricBatch.
func (batch *MetricBatch) AddMetric(metric Metric) {
	batch.Metrics = append(batch.Metrics, metric)
}

type metricsArray []Metric

func (ma metricsArray) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	for idx, m := range ma {
		if idx > 0 {
			buf.WriteByte(',')
		}
		m.writeJSON(buf)
	}
	buf.WriteByte(']')
}

type commonAttributes MetricBatch

func (c commonAttributes) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('{')
	w := internal.JSONFieldsWriter{Buf: buf}
	writeTimestampInterval(&w, c.Timestamp, c.Interval)
	if nil != c.Attributes {
		w.WriterField("attributes", internal.Attributes(c.Attributes))
	} else if nil != c.AttributesJSON {
		w.RawField("attributes", c.AttributesJSON)
	}
	buf.WriteByte('}')
}

func (batch *MetricBatch) writeJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	buf.WriteByte('{')
	w := internal.JSONFieldsWriter{Buf: buf}
	w.WriterField("common", commonAttributes(*batch))
	w.WriterField("metrics", metricsArray(batch.Metrics))
	buf.WriteByte('}')
	buf.WriteByte(']')
}

// split will split the MetricBatch into 2 equal parts, returning a slice of MetricBatches.
// If the number of metrics in the original is 0 or 1 then nil is returned.
func (batch *MetricBatch) split() []requestsBuilder {
	if len(batch.Metrics) < 2 {
		return nil
	}

	half := len(batch.Metrics) / 2
	mb1 := *batch
	mb1.Metrics = batch.Metrics[:half]
	mb2 := *batch
	mb2.Metrics = batch.Metrics[half:]

	return []requestsBuilder{
		requestsBuilder(&mb1),
		requestsBuilder(&mb2),
	}
}

const (
	defaultURL = "https://metric-api.newrelic.com/metric/v1"
)

// NewRequests creates new Requests from the MetricBatch. The requests can be
// sent with an http.Client.
//
// NewRequest returns the requests or an error if there was one.  Each Request
// has an UncompressedBody field that is useful in debugging or testing.
//
// Possible response codes to be expected when sending the request to New
// Relic:
//
//  202 for success
//  403 for an auth failure
//  404 for a bad path
//  405 for anything but POST
//  411 if the Content-Length header is not included
//  413 for a payload that is too large
//  400 for a generally invalid request
//  429 Too Many Requests
//
func (batch *MetricBatch) NewRequests(apiKey string, urlOverride string) ([]Request, error) {
	return newRequests(batch, apiKey, urlOverride, maxCompressedSizeBytes)
}

func (batch *MetricBatch) newRequest(apiKey, urlOverride string) (Request, error) {
	buf := &bytes.Buffer{}
	batch.writeJSON(buf)
	uncompressed := buf.Bytes()

	u := defaultURL
	if "" != urlOverride {
		u = urlOverride
	}
	compressed, err := internal.Compress(uncompressed)
	if nil != err {
		return Request{}, fmt.Errorf("error compressing metric data: %v", err)
	}
	req, err := http.NewRequest("POST", u, compressed)
	if nil != err {
		return Request{}, fmt.Errorf("error creating metric request: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Insert-Key", apiKey)
	req.Header.Add("Content-Encoding", "gzip")
	internal.AddUserAgentHeader(req.Header)
	return Request{
		Request:              req,
		UncompressedBody:     uncompressed,
		compressedBodyLength: compressed.Len(),
	}, nil
}
