package lumberjack_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go.loglayer.dev/transports/lumberjack/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
)

// readFileBuffer reads path into a *bytes.Buffer so we can reuse the
// existing transporttest.ParseJSONLine helper.
func readFileBuffer(t *testing.T, path string) *bytes.Buffer {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return bytes.NewBuffer(data)
}

func newLogger(t *testing.T, cfg lumberjack.Config) (*loglayer.LogLayer, *lumberjack.Transport, string) {
	t.Helper()
	if cfg.Filename == "" {
		cfg.Filename = filepath.Join(t.TempDir(), "app.log")
	}
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "file"
	}
	tr := lumberjack.New(cfg)
	t.Cleanup(func() { _ = tr.Close() })
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	return log, tr, cfg.Filename
}

func TestFile_New_PanicsWithoutFilename(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when Filename missing")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, lumberjack.ErrFilenameRequired) {
			t.Errorf("panic value: got %v, want ErrFilenameRequired", r)
		}
	}()
	_ = lumberjack.New(lumberjack.Config{})
}

func TestFile_Build_ReturnsErrFilenameRequired(t *testing.T) {
	_, err := lumberjack.Build(lumberjack.Config{})
	if !errors.Is(err, lumberjack.ErrFilenameRequired) {
		t.Errorf("Build with missing Filename: got %v, want ErrFilenameRequired", err)
	}
}

func TestFile_WritesJSONLine(t *testing.T) {
	log, tr, path := newLogger(t, lumberjack.Config{})
	log.Info("hello")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	obj := transporttest.ParseJSONLine(t, readFileBuffer(t, path))
	if obj["msg"] != "hello" {
		t.Errorf("msg: got %v", obj["msg"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
	if obj["time"] == nil {
		t.Error("expected time field")
	}
}

func TestFile_FieldsAndMetadataMergeAtRoot(t *testing.T) {
	log, tr, path := newLogger(t, lumberjack.Config{})
	log.WithFields(loglayer.Fields{"requestId": "abc"}).
		WithMetadata(loglayer.Metadata{"durationMs": 42}).
		Info("served")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	obj := transporttest.ParseJSONLine(t, readFileBuffer(t, path))
	if obj["requestId"] != "abc" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["durationMs"] != float64(42) {
		t.Errorf("durationMs: got %v", obj["durationMs"])
	}
}

func TestFile_RenameStandardFields(t *testing.T) {
	log, tr, path := newLogger(t, lumberjack.Config{
		MessageField: "message",
		DateField:    "timestamp",
		LevelField:   "severity",
	})
	log.Warn("renamed")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	obj := transporttest.ParseJSONLine(t, readFileBuffer(t, path))
	if obj["message"] != "renamed" {
		t.Errorf("message: got %v", obj["message"])
	}
	if obj["severity"] != "warn" {
		t.Errorf("severity: got %v", obj["severity"])
	}
	if obj["timestamp"] == nil {
		t.Error("expected timestamp field")
	}
	if _, dup := obj["msg"]; dup {
		t.Error("default 'msg' key should be absent when MessageField is overridden")
	}
}

func TestFile_LevelFiltering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	tr := lumberjack.New(lumberjack.Config{
		BaseConfig: transport.BaseConfig{ID: "file", Level: loglayer.LogLevelError},
		Filename:   path,
	})
	t.Cleanup(func() { _ = tr.Close() })
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Info("dropped")
	log.Warn("dropped")
	log.Error("kept")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Count(strings.TrimSpace(string(data)), "\n") + 1
	if lines != 1 {
		t.Fatalf("expected 1 line after level filter, got %d (content: %q)", lines, data)
	}
	obj := transporttest.ParseJSONLine(t, bytes.NewBuffer(data))
	if obj["msg"] != "kept" {
		t.Errorf("kept entry msg: got %v", obj["msg"])
	}
}

func TestFile_AppendsToExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte("preexisting line\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	log, tr, _ := newLogger(t, lumberjack.Config{Filename: path})
	log.Info("after")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(string(data), "preexisting line\n") {
		t.Errorf("expected file to start with seeded content, got: %q", data)
	}
	if !strings.Contains(string(data), `"msg":"after"`) {
		t.Errorf("expected appended JSON line, got: %q", data)
	}
}

func TestFile_CreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "app.log")
	log, tr, _ := newLogger(t, lumberjack.Config{Filename: path})
	log.Info("nested")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s, got: %v", path, err)
	}
}

// MaxSize is in megabytes; setting it to 1 plus a flood of large entries
// should trigger at least one rotation. Use a generous-but-bounded write
// loop so the test stays fast.
func TestFile_RotatesOnSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rotate.log")

	log, tr, _ := newLogger(t, lumberjack.Config{
		Filename: path,
		MaxSize:  1, // 1 MB
	})

	bigPayload := strings.Repeat("x", 10*1024) // 10 KiB per call
	for i := 0; i < 200; i++ {                 // ~2 MiB written → at least one rotation
		log.WithMetadata(loglayer.Metadata{"chunk": bigPayload}).Info("flood")
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) < 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected at least 2 files (active + 1 backup) after rotation, got %d: %v", len(entries), names)
	}
}

// Rotate forces a roll-over even when MaxSize hasn't been reached. The
// active filename is reused for new writes; the previous content lives
// in a timestamped backup next to it.
func TestFile_RotateForcesRollover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manual.log")

	log, tr, _ := newLogger(t, lumberjack.Config{Filename: path})
	log.Info("before")
	if err := tr.Rotate(); err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	log.Info("after")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 files (active + backup), got %d", len(entries))
	}

	active, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read active: %v", err)
	}
	if !strings.Contains(string(active), `"msg":"after"`) {
		t.Errorf("expected active file to contain post-rotate entry, got: %q", active)
	}
	if strings.Contains(string(active), `"msg":"before"`) {
		t.Errorf("expected pre-rotate entry to live in backup, but found in active: %q", active)
	}
}

func TestFile_CloseIsIdempotent(t *testing.T) {
	_, tr, _ := newLogger(t, lumberjack.Config{})
	if err := tr.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// After Close, further log calls must not reopen the file. lumberjack's
// natural behavior is lazy-reopen-on-write; the closed flag suppresses
// it so termination semantics stay clean.
func TestFile_PostCloseDropsEntries(t *testing.T) {
	log, tr, path := newLogger(t, lumberjack.Config{})
	log.Info("before close")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	beforeStat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	beforeSize := beforeStat.Size()

	log.Info("dropped post-close")

	afterStat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if afterStat.Size() != beforeSize {
		t.Errorf("expected file size unchanged after Close, got %d → %d", beforeSize, afterStat.Size())
	}
}

func TestFile_GetLoggerInstance(t *testing.T) {
	_, tr, _ := newLogger(t, lumberjack.Config{})
	got := tr.GetLoggerInstance()
	if got == nil {
		t.Fatal("GetLoggerInstance returned nil; expected *lumberjack.Logger")
	}
}

// blockWrites pre-creates parent and revokes its write permission so a
// subsequent OpenFile under it fails with EACCES. The cleanup restores
// permissions so t.TempDir's own cleanup can remove the tree.
func blockWrites(t *testing.T) string {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("test relies on UID-based permission denial; running as root bypasses DAC")
	}
	parent := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	return filepath.Join(parent, "app.log")
}

// A write failure (lumberjack can't open or rotate the underlying file)
// must surface via Config.OnError so an unobservable file sink can't
// silently swallow entries.
func TestFile_OnErrorFiresOnWriteFailure(t *testing.T) {
	target := blockWrites(t)

	var calls atomic.Int32
	var captured atomic.Pointer[error]
	tr := lumberjack.New(lumberjack.Config{
		Filename: target,
		OnError: func(err error) {
			calls.Add(1)
			captured.Store(&err)
		},
	})
	t.Cleanup(func() { _ = tr.Close() })
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Info("trigger write failure")

	if calls.Load() == 0 {
		t.Fatal("expected OnError to be called at least once")
	}
	got := captured.Load()
	if got == nil || *got == nil {
		t.Fatal("OnError captured a nil error")
	}
}

// Without a user OnError, a write failure must not panic. The default
// callback writes to stderr; the test verifies the call returns cleanly
// so a failing sink can't tear down the dispatcher. Stderr output for
// this one entry is expected and harmless in test output.
func TestFile_DefaultOnErrorDoesNotPanic(t *testing.T) {
	target := blockWrites(t)

	tr := lumberjack.New(lumberjack.Config{Filename: target})
	t.Cleanup(func() { _ = tr.Close() })
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("default OnError panicked: %v", r)
		}
	}()
	log.Info("trigger write failure")
}

// Concurrent writes must not corrupt the on-disk JSON: each line must
// parse as a complete object. lumberjack.Logger's internal mutex
// serializes writes; our wrapper adds nothing on the hot path, so this
// is really a compatibility check that we don't break that property.
func TestFile_ConcurrentWritesProduceWholeLines(t *testing.T) {
	log, tr, path := newLogger(t, lumberjack.Config{})

	var wg sync.WaitGroup
	const goroutines = 8
	const perGoroutine = 50
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				log.WithMetadata(loglayer.Metadata{"g": g, "i": i}).Info("concurrent")
			}
		}(g)
	}
	wg.Wait()
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if got, want := len(lines), goroutines*perGoroutine; got != want {
		t.Fatalf("expected %d lines, got %d", want, got)
	}
	for i, line := range lines {
		obj := transporttest.ParseJSONLine(t, bytes.NewBufferString(line))
		if obj["msg"] != "concurrent" {
			t.Errorf("line %d msg: got %v", i, obj["msg"])
		}
	}
}
