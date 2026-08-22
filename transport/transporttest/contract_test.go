package transporttest_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.loglayer.dev/v3"
	"go.loglayer.dev/v3/transport"
	"go.loglayer.dev/v3/transport/transporttest"
)

// fakeTransport renders each entry as a single JSON line in the manner of a
// typical wrapper: message under "msg", level under "level", fields and
// error in Data merged at the root, and metadata assembled per
// params.Schema.MetadataFieldName (nesting key from the schema when set).
// Its small shape lets the contract suite run inside the core module
// without pulling in any wrapper dependency.
type fakeTransport struct {
	transport.BaseTransport
	buf *bytes.Buffer
}

// GetLoggerInstance returns nil; the fake has no underlying library.
func (t *fakeTransport) GetLoggerInstance() any { return nil }

func (t *fakeTransport) SendToLogger(params loglayer.TransportParams) {
	// Filter before assembling so a dropped entry writes nothing (the
	// contract's LevelFiltering case asserts an empty buffer).
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	// Fold the prefix into Messages[0] for the rendered output;
	// transports own this rendering choice.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)

	obj := map[string]any{
		"msg":   transport.JoinMessages(params.Messages),
		"level": params.LogLevel.String(),
	}
	for k, v := range params.Data {
		obj[k] = v
	}
	if params.Metadata != nil {
		if key := params.Schema.MetadataFieldName; key != "" {
			obj[key] = params.Metadata
		} else if md, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range md {
				obj[k] = v
			}
		} else {
			obj["metadata"] = params.Metadata
		}
	}
	line, _ := json.Marshal(obj)
	t.buf.Write(line)
	t.buf.WriteByte('\n')
}

// fakeFactory mirrors how real wrapper factories translate FactoryOpts.Level
// onto their transport: the fake's BaseConfig must carry it, or
// ShouldProcess defaults to accepting every level and the contract's
// LevelFiltering case (FactoryOpts{Level: LogLevelError}) fails.
func fakeFactory() transporttest.Factory {
	return func(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
		buf := &bytes.Buffer{}
		tr := &fakeTransport{BaseTransport: transport.NewBaseTransport(transport.BaseConfig{Level: opts.Level}), buf: buf}
		return transporttest.NewLogger(tr, opts), buf
	}
}

func TestRunContract_FakeTransport(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "fake",
		Factory: fakeFactory(),
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				loglayer.LogLevelTrace: "trace",
				loglayer.LogLevelDebug: "debug",
				loglayer.LogLevelInfo:  "info",
				loglayer.LogLevelWarn:  "warn",
				loglayer.LogLevelError: "error",
				loglayer.LogLevelFatal: "fatal",
				loglayer.LogLevelPanic: "panic",
			},
		},
	})
}

// TestRunContract_FlattenOptOutIsDistinct proves the FlattenMetadataOptOut
// case is sensitive to its opt-in: the same fake transport, run with the
// default (zero-value) FactoryOpts, must nest map metadata under
// "metadata" rather than merge it at the root. The opt-out case itself
// asserts the root-merge side, so this pair pins the whole default flip.
// Level needs no explicit opt here: BaseConfig.Level zero maps to
// LogLevelTrace in NewBaseTransport, which accepts every level, and this
// test only emits Info.
func TestRunContract_FlattenOptOutIsDistinct(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := &fakeTransport{BaseTransport: transport.NewBaseTransport(transport.BaseConfig{}), buf: buf}
	log := transporttest.NewLogger(tr, transporttest.FactoryOpts{}) // zero value: FlattenMetadata false
	log.WithMetadata(loglayer.Metadata{"requestId": "xyz"}).Info("req")

	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj); err != nil {
		t.Fatalf("output is not valid JSON: %v: got %q", err, buf.String())
	}
	md, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata nested under default key, got %v", obj)
	}
	if md["requestId"] != "xyz" {
		t.Errorf("nested requestId: got %v", md["requestId"])
	}
	if _, atRoot := obj["requestId"]; atRoot {
		t.Errorf("requestId should be nested under \"metadata\", not at root")
	}
	if !strings.Contains(buf.String(), "metadata") {
		t.Errorf("expected \"metadata\" key in output, got %q", buf.String())
	}
}
