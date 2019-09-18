package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/newrelic/go-telemetry-sdk/internal"
)

// Span is a distributed tracing span.
type Span struct {
	// GUID is a unique identifier for this span.
	GUID string
	// TraceID is a unique identifier shared by all spans within a single
	// trace.
	TraceID string
	// Name is the name of this span.
	Name string
	// ParentID is the span id of the previous caller of this span.  This
	// can be empty if this is the first span.
	ParentID string
	// Timestamp is when this span started.
	Timestamp time.Time
	// Duration is the duration of this span.  This field will be reported
	// in milliseconds.
	Duration time.Duration
	// EntityName is the name of the service that created this span.
	EntityName string
	// Attributes is a map of user specified tags on this span.  The map
	// values can be any of bool, number, or string.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
}

func (s *Span) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')
	w.StringField("guid", s.GUID)
	w.StringField("traceId", s.TraceID)
	w.StringField("name", s.Name)
	if "" != s.ParentID {
		w.StringField("parentId", s.ParentID)
	}
	w.IntField("timestamp", s.Timestamp.UnixNano()/(1000*1000))
	w.FloatField("durationMs", s.Duration.Seconds()*1000.0)
	w.StringField("entityName", s.EntityName)

	if nil != s.Attributes {
		w.WriterField("tags", internal.Attributes(s.Attributes))
	} else if nil != s.AttributesJSON {
		w.RawField("tags", s.AttributesJSON)
	}
	buf.WriteByte('}')
}

// SpanBatch represents a single batch of spans to report to New Relic.
type SpanBatch struct {
	Spans []Span
}

// AddSpan appends a span to the SpanBatch.
func (batch *SpanBatch) AddSpan(s Span) {
	batch.Spans = append(batch.Spans, s)
}

// split will split the SpanBatch into 2 equally sized batches.
// If the number of spans in the original is 0 or 1 then nil is returned.
func (batch *SpanBatch) split() []requestsBuilder {
	if len(batch.Spans) < 2 {
		return nil
	}

	half := len(batch.Spans) / 2
	b1 := *batch
	b1.Spans = batch.Spans[:half]
	b2 := *batch
	b2.Spans = batch.Spans[half:]

	return []requestsBuilder{
		requestsBuilder(&b1),
		requestsBuilder(&b2),
	}
}

func (batch *SpanBatch) writeJSON(buf *bytes.Buffer) {
	buf.WriteByte('{')
	buf.WriteString(`"spans":`)
	buf.WriteByte('[')
	for idx, s := range batch.Spans {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
	buf.WriteByte(']')
	buf.WriteByte('}')
}

const (
	defaultSpanURL = "https://collector.newrelic.com/agent_listener/invoke_raw_method"
)

func getSpansURL(licenseKey, urlOverride string) string {
	u := defaultSpanURL
	if "" != urlOverride {
		u = urlOverride
	}
	return u + "?method=external_span_data&protocol_version=1&license_key=" + licenseKey
}

// NewRequests creates new requests from the SpanBatch. The request can be
// sent with an http.Client.
//
// NewRequest returns requests or an error if there was one.  Each Request
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
func (batch *SpanBatch) NewRequests(licenseKey, urlOverride string) ([]Request, error) {
	return newRequests(batch, licenseKey, urlOverride, maxCompressedSizeBytes)
}

func (batch *SpanBatch) newRequest(licenseKey, urlOverride string) (Request, error) {
	buf := &bytes.Buffer{}
	batch.writeJSON(buf)
	uncompressed := buf.Bytes()
	compressed, err := internal.Compress(uncompressed)
	if nil != err {
		return Request{}, fmt.Errorf("error compressing span data: %v", err)
	}
	req, err := http.NewRequest("POST", getSpansURL(licenseKey, urlOverride), compressed)
	if nil != err {
		return Request{}, fmt.Errorf("error creating span request: %v", err)
	}

	req.Header.Add("Content-Encoding", "gzip")
	internal.AddUserAgentHeader(req.Header)
	return Request{
		Request:              req,
		UncompressedBody:     uncompressed,
		compressedBodyLength: compressed.Len(),
	}, nil
}
