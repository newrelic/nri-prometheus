package telemetry

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

// Event is a unique set of data that happened at a specific point in time
type Event struct {
	// Required Fields:
	//
	// EventType is the name of the event
	EventType string
	// Timestamp is when this event happened.  If Timestamp is not set, it
	// will be assigned to time.Now() in Harvester.RecordEvent.
	Timestamp time.Time

	// Recommended Fields:
	//
	// Attributes is a map of user specified data on this event.  The map
	// values can be any of bool, number, or string.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
}

func (e *Event) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')

	w.StringField("eventType", e.EventType)
	w.IntField("timestamp", e.Timestamp.UnixNano()/(1000*1000))

	internal.AddAttributes(&w, e.Attributes)

	buf.WriteByte('}')
}

// eventBatch represents a single batch of events to report to New Relic.
type eventBatch struct {
	Events []Event
}

// split will split the eventBatch into 2 equally sized batches.
// If the number of events in the original is 0 or 1 then nil is returned.
func (batch *eventBatch) split() []requestsBuilder {
	if len(batch.Events) < 2 {
		return nil
	}

	half := len(batch.Events) / 2
	b1 := *batch
	b1.Events = batch.Events[:half]
	b2 := *batch
	b2.Events = batch.Events[half:]

	return []requestsBuilder{
		requestsBuilder(&b1),
		requestsBuilder(&b2),
	}
}

func (batch *eventBatch) writeJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	for idx, s := range batch.Events {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
	buf.WriteByte(']')
}

func (batch *eventBatch) makeBody() json.RawMessage {
	buf := &bytes.Buffer{}
	batch.writeJSON(buf)
	return buf.Bytes()
}
