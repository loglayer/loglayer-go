# Axiom Transport Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a loglayer transport that ships structured logs to Axiom using the axiom-go SDK

**Architecture:** The transport takes a pre-configured `*axiom.Client` as input (like gcplogging) and uses its ingest API directly. No internal buffering - the Axiom client handles batching/compression internally.

**Tech Stack:**
- Main module: go.loglayer.dev/v2
- Dependency: github.com/axiomhq/axiom-go@v0.32.0
- Follows gcplogging pattern (caller-supplied logger/client)

---

## File Structure

```
transports/axiom/
├── axiom.go        # Main transport implementation (new)
├── errors.go       # Custom error types (new)
├── axiom_test.go   # Unit tests + contract tests (new)
└── go.mod          # Module definition (new)
```

---

### Task 1: Create the transport module structure

**Files:**
- Create: `transports/axiom/go.mod`
- Create: `transports/axiom/axiom.go`
- Create: `transports/axiom/errors.go`

- [ ] **Step 1: Create go.mod**

```go
module go.loglayer.dev/transports/axiom/v2

go 1.25.0

replace go.loglayer.dev => ../..

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	github.com/axiomhq/axiom-go v0.32.0
)
```

- [ ] **Step 2: Create errors.go**

```go
package axiom

import "errors"

// ErrClientRequired is returned by Build (and panicked by New) when
// Config.Client is nil. The user supplies the Axiom client; the transport
// never constructs one itself, so a nil client can't be defaulted.
var ErrClientRequired = errors.New("loglayer/transports/axiom: Config.Client is required")

// ErrDatasetNameRequired is returned by Build (and panicked by New) when
// Config.DatasetName is empty. Axiom requires a dataset name for ingestion.
var ErrDatasetNameRequired = errors.New("loglayer/transports/axiom: Config.DatasetName is required")
```

- [ ] **Step 3: Create axiom.go with Config and Transport types**

```go
// Package axiom provides a LogLayer transport backed by a caller-supplied
// *axiom.Client from github.com/axiomhq/axiom-go.
//
// The user owns client lifecycle (authentication, configuration); this
// transport assembles log entries and dispatches them via Client.Ingest.
//
// # Payload shape
//
// Each log entry is ingested as a JSON object with:
//
//   - the joined message text under Config.MessageField (default "msg");
//   - all persistent fields and the serialized error from params.Data
//     merged at root;
//   - map metadata flattened at root, or any other metadata nested
//     under params.Schema.MetadataFieldName (or "metadata" by default).
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package axiom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/axiom-go/axiom/ingest"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// Config holds configuration options for the Axiom transport.
type Config struct {
	transport.BaseConfig

	// Client is the underlying *axiom.Client from github.com/axiomhq/axiom-go.
	// Required. Construct via axiom.NewClient() with appropriate authentication
	// options (SetAPIToken, SetPersonalTokenConfig, etc.).
	Client *axiom.Client

	// DatasetName is the Axiom dataset to ingest logs into. Required.
	DatasetName string

	// MessageField is the key under which the joined message text is placed
	// in the ingested JSON object. Defaults to "msg".
	MessageField string

	// OnError is called when Ingest returns an error. The default writes
	// a one-line message to os.Stderr.
	OnError func(error)
}

// Transport ships log entries to Axiom via Client.Ingest.
type Transport struct {
	transport.BaseTransport
	cfg Config
}

// New creates an Axiom Transport from the given Config. Panics if required
// fields are invalid. Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build creates a Transport like New but returns errors instead of panicking.
func Build(cfg Config) (*Transport, error) {
	if cfg.Client == nil {
		return nil, ErrClientRequired
	}
	if cfg.DatasetName == "" {
		return nil, ErrDatasetNameRequired
	}
	if cfg.MessageField == "" {
		cfg.MessageField = "msg"
	}

	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}, nil
}

// GetLoggerInstance returns the underlying *axiom.Client.
func (t *Transport) GetLoggerInstance() any { return t.cfg.Client }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}

	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)

	entry := make(map[string]any, transport.FieldEstimate(params)+1)
	entry[t.cfg.MessageField] = transport.JoinMessages(params.Messages)
	transport.MergeIntoMap(entry, params.Data, params.Metadata, params.Schema.MetadataFieldName)

	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	buf, err := json.Marshal(entry)
	if err != nil {
		t.reportError(fmt.Errorf("failed to marshal entry: %w", err))
		return
	}

	r := io.NopCloser(bytes.NewReader(buf))

	_, ingestErr := t.cfg.Client.Ingest(ctx, t.cfg.DatasetName, r, axiom.NDJSON, axiom.NoEncoding, t.cfg.IngestOptions...)
	if ingestErr != nil {
		t.reportError(ingestErr)
	}
}

func (t *Transport) reportError(err error) {
	if t.cfg.OnError != nil {
		t.cfg.OnError(err)
		return
	}
	fmt.Fprintf(os.Stderr, "loglayer/transports/axiom: %v\n", err)
}
```

- [ ] **Step 4: Add axiom to scripts/foreach-module.sh**

Update `ALL_MODULES`, `SHIPPED_MODULES`, and `TEST_MODULES` arrays to include `transports/axiom`.

- [ ] **Step 5: Add axiom to go.work**

Add `./transports/axiom` to the use block in `go.work`.

---

### Task 2: Run tidy and verify build

**Files:** Modified by previous task
**Commands:** Run from repo root

- [ ] **Step 1: Run foreach-module.sh tidy**

```bash
bash scripts/foreach-module.sh tidy
```

Expected: All go.mod/go.sum files updated, no git diff output after staging.

- [ ] **Step 2: Run foreach-module.sh build**

```bash
bash scripts/foreach-module.sh build
```

Expected: No errors.

---

### Task 3: Write contract tests

**Files:**
- Create: `transports/axiom/axiom_test.go`

- [ ] **Step 1: Create axiom_test.go with factory and level mapping**

```go
package axiom

import (
	"bytes"
	"context"
	"testing"

	"github.com/axiomhq/axiom-go/axiom"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
)

// mockClient is a minimal *axiom.Client replacement for testing.
// It captures the last ingested dataset and data for assertion.
type mockClient struct {
	lastDataset string
	lastData    []byte
}

func (m *mockClient) Ingest(ctx context.Context, id string, r io.Reader, typ axiom.ContentType, enc axiom.ContentEncoder, opts ...axiom.Option) (*axiom.Response, error) {
	data, _ := io.ReadAll(r)
	m.lastDataset = id
	m.lastData = data
	return &axiom.Response{}, nil
}

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	client := &mockClient{}
	tr, err := Build(Config{
		BaseConfig:  transport.BaseConfig{ID: "axiom", Level: opts.Level},
		Client:      client,
		DatasetName: "test-logs",
	})
	if err != nil {
		panic(err)
	}
	return transporttest.NewLogger(tr, opts), &bytes.Buffer{}
}

func TestAxiomContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "axiom",
		Factory: factory,
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
```

- [ ] **Step 2: Add missing import for io**

Fix the import in the test file to include `io`.

- [ ] **Step 3: Fix Build call to handle error**

Update the factory to properly handle the Build error.

---

### Task 4: Run contract tests

**Commands:** From repo root

- [ ] **Step 1: Run foreach-module.sh test for axiom module**

```bash
cd transports/axiom && go test -race ./...
```

Expected: Contract tests pass.

---

### Task 5: Add unit tests for Build validation

**Files:** Modify `transports/axiom/axiom_test.go`

- [ ] **Step 1: Add TestBuild_NilClientReturnsError**

```go
func TestBuild_NilClientReturnsError(t *testing.T) {
	_, err := Build(Config{})
	if !errors.Is(err, ErrClientRequired) {
		t.Errorf("got %v, want ErrClientRequired", err)
	}
}

func TestBuild_EmptyDatasetNameReturnsError(t *testing.T) {
	client := &mockClient{}
	_, err := Build(Config{Client: client})
	if !errors.Is(err, ErrDatasetNameRequired) {
		t.Errorf("got %v, want ErrDatasetNameRequired", err)
	}
}

func TestNew_NilClientPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, ErrClientRequired) {
			t.Errorf("panic value: got %v, want ErrClientRequired", r)
		}
	}()
	New(Config{})
}

func TestNew_EmptyDatasetNamePanics(t *testing.T) {
	client := &mockClient{}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, ErrDatasetNameRequired) {
			t.Errorf("panic value: got %v, want ErrDatasetNameRequired", r)
		}
	}()
	New(Config{Client: client})
}
```

- [ ] **Step 2: Add missing import for errors**

Update imports to include `errors`.

---

### Task 6: Run all tests

**Commands:** From repo root

- [ ] **Step 1: Verify tests pass**

```bash
cd transports/axiom && go test -race ./...
```

Expected: All tests pass.

---

### Task 7: Add example documentation

**Files:**
- Create: `transports/axiom/example_test.go`

- [ ] **Step 1: Create example_test.go**

```go
package axiom_test

import (
	"context"
	"fmt"

	"github.com/axiomhq/axiom-go/axiom"

	"go.loglayer.dev/transports/axiom/v2"
	"go.loglayer.dev/v2"
)

// ExampleNew demonstrates creating an Axiom transport with a client.
func ExampleNew() {
	ctx := context.Background()
	client, err := axiom.NewClient(
		axiom.SetAPIToken("your-api-token"),
		axiom.SetOrganizationID("your-org-id"),
	)
	if err != nil {
		panic(err)
	}

	t := axiom.New(axiom.Config{
		BaseConfig:  transport.BaseConfig{ID: "axiom"},
		Client:      client,
		DatasetName: "my-logs",
	})

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})

	log.Info("Hello from Axiom!")
}

// ExampleWithCustomMessageField shows how to customize the message key.
func ExampleConfig_MessageField() {
	client, _ := axiom.NewClient(axiom.SetAPIToken("token"))
	t := axiom.New(axiom.Config{
		BaseConfig:   transport.BaseConfig{ID: "axiom"},
		Client:       client,
		DatasetName:  "my-logs",
		MessageField: "message", // Use "message" instead of default "msg"
	})

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})

	log.Info("This will use 'message' as the key")
}
```

- [ ] **Step 2: Fix import in example**

Update imports to include `transport`.

---

### Task 8: Verify staticcheck passes

**Commands:** From repo root

- [ ] **Step 1: Run foreach-module.sh staticcheck**

```bash
bash scripts/foreach-module.sh staticcheck
```

Expected: No errors for axiom module.

---

## Summary

After completing all tasks:

1. Transport created at `transports/axiom/`
2. Module registered in `go.work` and `scripts/foreach-module.sh`
3. Contract tests pass for standard transport behavior
4. Unit tests validate Build/New validation logic
5. Staticcheck passes with no warnings
6. Example code demonstrates usage

---

**Plan complete and saved to `docs/superpowers/plans/YYYY-MM-DD-axiom-transport.md`.**
