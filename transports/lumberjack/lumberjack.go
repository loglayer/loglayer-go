// Package lumberjack provides a Transport that writes one JSON object
// per log entry to a rotating file. Rotation is delegated to
// lumberjack.v2: size-triggered rollover, configurable backup
// retention, optional gzip compression, and age-based cleanup.
//
// The render path is the structured transport's, so the on-disk format
// matches `transports/structured` exactly. Render-tuning fields
// (MessageField, DateField, LevelField, DateFn, LevelFn, MessageFn) pass
// through unchanged.
//
// The package name shadows the upstream `gopkg.in/natefinch/lumberjack.v2`
// (which is also `package lumberjack`); within this file the upstream is
// aliased to lj. Callers that import both can use a similar alias.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package lumberjack

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	lj "gopkg.in/natefinch/lumberjack.v2"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/structured"
)

// Config holds configuration options for Transport.
type Config struct {
	transport.BaseConfig

	// Filename is the file to write logs to. Required.
	//
	// Backup files are placed next to the active file with a timestamp
	// suffix (and optional .gz when Compress is true). Lumberjack creates
	// the parent directory tree if it does not already exist.
	Filename string

	// MaxSize is the maximum size in megabytes of the active log file
	// before it is rotated. Defaults to 100 (lumberjack's default).
	MaxSize int

	// MaxBackups is the maximum number of rotated files to retain. Older
	// files are deleted when the count is exceeded. 0 keeps every rotated
	// file (subject to MaxAge).
	MaxBackups int

	// MaxAge is the maximum number of days to retain rotated files based
	// on the timestamp encoded in their filename. 0 disables age-based
	// cleanup.
	MaxAge int

	// Compress, when true, gzip-compresses rotated files.
	Compress bool

	// LocalTime, when true, formats backup-filename timestamps in the
	// host's local time. Defaults to false (UTC).
	LocalTime bool

	// MessageField is the JSON key for the joined message text. Defaults
	// to "msg".
	MessageField string

	// DateField is the JSON key for the timestamp. Defaults to "time".
	DateField string

	// LevelField is the JSON key for the log level. Defaults to "level".
	LevelField string

	// DateFn overrides the default ISO-8601 timestamp.
	DateFn func() string

	// LevelFn overrides the default level string.
	LevelFn func(loglayer.LogLevel) string

	// MessageFn formats the message portion. Its return value is used
	// as the message text.
	MessageFn func(params loglayer.TransportParams) string

	// OnError is called when a write to the rotating file fails. The
	// default writes a one-line message to os.Stderr. Use this to plumb
	// write failures into a separate logger or metrics counter so a
	// failing file sink does not silence the application.
	OnError func(error)
}

// Transport writes one JSON object per log entry to a rotating file.
type Transport struct {
	transport.BaseTransport
	inner   *structured.Transport
	rotator *lj.Logger
	closed  atomic.Bool
}

// New constructs a file Transport.
//
// Panics with ErrFilenameRequired when cfg.Filename is empty. Use Build
// for an error-returning variant when the filename is loaded at runtime
// (e.g. from configuration or an environment variable).
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a file Transport like New but returns
// ErrFilenameRequired instead of panicking when cfg.Filename is empty.
func Build(cfg Config) (*Transport, error) {
	if cfg.Filename == "" {
		return nil, ErrFilenameRequired
	}
	if cfg.OnError == nil {
		cfg.OnError = defaultOnError
	}
	rot := &lj.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  cfg.LocalTime,
	}
	// errorReportingWriter wraps the rotator so lumberjack write errors
	// surface through cfg.OnError instead of being swallowed by
	// structured's `_, _ = writer.Write(...)`.
	inner := structured.New(structured.Config{
		MessageField: cfg.MessageField,
		DateField:    cfg.DateField,
		LevelField:   cfg.LevelField,
		DateFn:       cfg.DateFn,
		LevelFn:      cfg.LevelFn,
		MessageFn:    cfg.MessageFn,
		Writer:       &errorReportingWriter{w: rot, onError: cfg.OnError},
	})
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		inner:         inner,
		rotator:       rot,
	}, nil
}

// errorReportingWriter forwards writes to w and reports any error via
// onError. Without this wrapper, lumberjack write errors would be
// discarded by the structured renderer.
type errorReportingWriter struct {
	w       io.Writer
	onError func(error)
}

func (w *errorReportingWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if err != nil {
		w.onError(err)
	}
	return n, err
}

func defaultOnError(err error) {
	fmt.Fprintf(os.Stderr, "loglayer/transports/lumberjack: write failed: %v\n", err)
}

// GetLoggerInstance returns the underlying *lumberjack.Logger.
func (t *Transport) GetLoggerInstance() any { return t.rotator }

func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	t.inner.SendToLogger(params)
}

// Close releases the underlying file handle and disables the transport
// so post-Close log calls don't trigger lumberjack's lazy-reopen.
// Idempotent.
func (t *Transport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	t.SetEnabled(false)
	return t.rotator.Close()
}

// Rotate forces an immediate rotation of the current log file.
func (t *Transport) Rotate() error {
	return t.rotator.Rotate()
}
