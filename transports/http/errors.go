package httptransport

import (
	"errors"
	"fmt"
)

// ErrBufferFull is reported via OnError when SendToLogger drops an entry
// because the internal buffer is full.
var ErrBufferFull = errors.New("loglayer/transports/http: buffer full, entry dropped")

// ErrClosed is reported via OnError when SendToLogger is called after Close.
var ErrClosed = errors.New("loglayer/transports/http: transport closed, entry dropped")

// HTTPError is reported via OnError when the server responds with a status
// >= 400. The original entries are still passed to OnError so callers can
// implement retry/dead-letter behavior.
type HTTPError struct {
	StatusCode int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("loglayer/transports/http: server returned status %d", e.StatusCode)
}
