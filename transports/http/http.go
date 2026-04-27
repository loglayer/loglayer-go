// Package httptransport sends log entries to an HTTP endpoint as JSON in
// configurable batches. The package directory is `transports/http` to mirror
// other transports; the package name is `httptransport` to avoid colliding
// with `net/http`.
//
// Mental model: log calls enqueue entries into a buffered channel; a
// background worker drains the channel into batches and POSTs them. Use Close
// to drain pending entries on shutdown.
package httptransport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

const (
	defaultBatchSize     = 100
	defaultBatchInterval = 5 * time.Second
	defaultBufferSize    = 1024
	defaultClientTimeout = 30 * time.Second
)

// Encoder serializes a batch of entries into the HTTP request body. The
// returned content type is set as the Content-Type header (overridable via
// Config.Headers).
type Encoder interface {
	Encode(entries []Entry) (body []byte, contentType string, err error)
}

// EncoderFunc adapts an ordinary function to the Encoder interface.
type EncoderFunc func(entries []Entry) ([]byte, string, error)

// Encode implements Encoder.
func (f EncoderFunc) Encode(entries []Entry) ([]byte, string, error) {
	return f(entries)
}

// Entry is the canonical log shape passed to the encoder. Constructed once
// per log call from loglayer.TransportParams.
type Entry struct {
	Level    loglayer.LogLevel
	Time     time.Time
	Messages []any
	// Data holds the assembled persistent fields and error map (may be nil).
	Data map[string]any
	// Metadata is the raw value the caller passed to WithMetadata. May be a
	// map, a struct, a scalar, or nil. Encoders decide how to render it.
	Metadata any
}

// Config holds HTTP transport configuration.
type Config struct {
	transport.BaseConfig

	// URL is the endpoint logs are POSTed to. Required.
	URL string

	// Method is the HTTP method. Defaults to "POST".
	Method string

	// Headers are added to every request. Content-Type is set automatically
	// from the Encoder unless overridden here.
	Headers map[string]string

	// Encoder serializes one or more entries into the request body. Defaults
	// to JSONArrayEncoder.
	Encoder Encoder

	// Client is the HTTP client used to send requests. Defaults to a client
	// with a 30-second timeout.
	Client *http.Client

	// BatchSize is the maximum number of entries per request. The worker
	// flushes whenever this is reached or BatchInterval elapses, whichever
	// comes first. Defaults to 100.
	BatchSize int

	// BatchInterval is the maximum wait before flushing a non-empty batch.
	// Defaults to 5 seconds.
	BatchInterval time.Duration

	// BufferSize is the size of the internal channel buffering entries before
	// the worker reads them. When full, new entries are dropped (oldest-out
	// is not implemented; the dispatch path stays non-blocking). Defaults to
	// 1024.
	BufferSize int

	// OnError is called when a batch fails to encode or send. The default
	// writes a one-line error to os.Stderr. Use this to plumb send errors
	// into a separate logger or metrics counter.
	OnError func(err error, entries []Entry)
}

// Transport implements loglayer.Transport with batched HTTP delivery.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	queue  chan Entry
	done   chan struct{}
	wg     sync.WaitGroup
	closed atomic.Bool
}

// New constructs an HTTP Transport and starts its background worker.
// Panics if cfg.URL is empty. Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs an HTTP Transport like New but returns ErrURLRequired
// instead of panicking when cfg.URL is empty. Use this when the URL is
// loaded at runtime (e.g. from an environment variable) and you want to
// handle the missing-config case explicitly.
func Build(cfg Config) (*Transport, error) {
	if cfg.URL == "" {
		return nil, ErrURLRequired
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.Encoder == nil {
		cfg.Encoder = JSONArrayEncoder
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: defaultClientTimeout}
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.BatchInterval <= 0 {
		cfg.BatchInterval = defaultBatchInterval
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = defaultBufferSize
	}
	if cfg.OnError == nil {
		cfg.OnError = defaultOnError
	}

	t := &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		queue:         make(chan Entry, cfg.BufferSize),
		done:          make(chan struct{}),
	}
	t.wg.Add(1)
	go t.worker()
	return t, nil
}

// GetLoggerInstance returns nil; the HTTP transport has no underlying logger.
func (t *Transport) GetLoggerInstance() any { return nil }

// SendToLogger enqueues the entry. Drops silently if the buffer is full or
// the transport is closed; both cases invoke OnError with a sentinel error so
// the caller can observe loss.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	if t.closed.Load() {
		t.cfg.OnError(ErrClosed, nil)
		return
	}
	entry := Entry{
		Level:    params.LogLevel,
		Time:     time.Now(),
		Messages: params.Messages,
		Metadata: params.Metadata,
	}
	if len(params.Data) > 0 {
		entry.Data = params.Data
	}
	select {
	case t.queue <- entry:
	default:
		t.cfg.OnError(ErrBufferFull, []Entry{entry})
	}
}

// Close drains the queue, flushes any pending entries, and stops the worker.
// Safe to call multiple times. After Close, SendToLogger drops entries.
func (t *Transport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(t.done)
	t.wg.Wait()
	return nil
}

func (t *Transport) worker() {
	defer t.wg.Done()

	batch := make([]Entry, 0, t.cfg.BatchSize)
	timer := time.NewTimer(t.cfg.BatchInterval)
	defer timer.Stop()

	for {
		select {
		case <-t.done:
			t.drainAndFlush(batch)
			return
		case e := <-t.queue:
			batch = append(batch, e)
			if len(batch) >= t.cfg.BatchSize {
				t.flush(batch)
				batch = batch[:0]
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(t.cfg.BatchInterval)
			}
		case <-timer.C:
			if len(batch) > 0 {
				t.flush(batch)
				batch = batch[:0]
			}
			timer.Reset(t.cfg.BatchInterval)
		}
	}
}

// drainAndFlush consumes any remaining queue entries (non-blocking) and
// flushes them in one or more batches. Called from the worker on shutdown.
func (t *Transport) drainAndFlush(batch []Entry) {
	for {
		select {
		case e := <-t.queue:
			batch = append(batch, e)
			if len(batch) >= t.cfg.BatchSize {
				t.flush(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				t.flush(batch)
			}
			return
		}
	}
}

func (t *Transport) flush(entries []Entry) {
	body, contentType, err := t.cfg.Encoder.Encode(entries)
	if err != nil {
		t.cfg.OnError(fmt.Errorf("loglayer/transports/http: encode: %w", err), entries)
		return
	}

	req, err := http.NewRequest(t.cfg.Method, t.cfg.URL, bytes.NewReader(body))
	if err != nil {
		t.cfg.OnError(fmt.Errorf("loglayer/transports/http: build request: %w", err), entries)
		return
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range t.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := t.cfg.Client.Do(req)
	if err != nil {
		t.cfg.OnError(fmt.Errorf("loglayer/transports/http: send: %w", err), entries)
		return
	}
	defer resp.Body.Close()
	// Drain the body so the underlying TCP connection can be reused (keep-alive).
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		t.cfg.OnError(&HTTPError{StatusCode: resp.StatusCode}, entries)
	}
}

func defaultOnError(err error, entries []Entry) {
	fmt.Fprintf(os.Stderr, "loglayer/transports/http: send failed (%d entries): %v\n", len(entries), err)
}

// JSONArrayEncoder serializes the batch as a JSON array of one object per
// entry: {level, time, msg, ...data, metadata?}. Map metadata merges at the
// root; non-map metadata lands under the "metadata" key.
var JSONArrayEncoder Encoder = EncoderFunc(jsonArrayEncode)

func jsonArrayEncode(entries []Entry) ([]byte, string, error) {
	objs := make([]map[string]any, len(entries))
	for i, e := range entries {
		obj := make(map[string]any, 3+len(e.Data))
		obj["level"] = e.Level.String()
		obj["time"] = e.Time.UTC().Format(time.RFC3339Nano)
		obj["msg"] = transport.JoinMessages(e.Messages)
		transport.MergeIntoMap(obj, e.Data, e.Metadata)
		objs[i] = obj
	}
	body, err := json.Marshal(objs)
	return body, "application/json", err
}
