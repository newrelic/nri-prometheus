package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	maxCompressedSizeBytes = 1 << 20
)

// Request contains an http.Request and the UncompressedBody which is provided
// for logging.
type Request struct {
	Request          *http.Request
	UncompressedBody json.RawMessage

	compressedBodyLength int
}

type requestsBuilder interface {
	newRequest(key, urlOverride string) (Request, error)
	split() []requestsBuilder
}

var (
	errUnableToSplit = fmt.Errorf("unable to split large payload further")
)

func newRequests(batch requestsBuilder, key, urlOverride string, maxCompressedSize int) ([]Request, error) {
	req, err := batch.newRequest(key, urlOverride)
	if nil != err {
		return nil, err
	}

	if req.compressedBodyLength <= maxCompressedSize {
		return []Request{req}, nil
	}

	var reqs []Request
	batches := batch.split()
	if nil == batches {
		return nil, errUnableToSplit
	}

	for _, b := range batches {
		rs, err := newRequests(b, key, urlOverride, maxCompressedSize)
		if nil != err {
			return nil, err
		}
		reqs = append(reqs, rs...)
	}
	return reqs, nil
}
