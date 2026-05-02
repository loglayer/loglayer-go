package loglayer_test

import (
	"fmt"
	"strings"
	"testing"

	"go.loglayer.dev/v2"
)

func TestWriter_BasicEmission(t *testing.T) {
	log, lib := setup(t)
	w := log.Writer(loglayer.LogLevelInfo)

	fmt.Fprintln(w, "hello world")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a captured line")
	}
	if line.Level != loglayer.LogLevelInfo {
		t.Errorf("level: got %v, want Info", line.Level)
	}
	if line.Messages[0] != "hello world" {
		t.Errorf("message: got %q, want \"hello world\"", line.Messages[0])
	}
}

func TestWriter_StripsTrailingNewlines(t *testing.T) {
	log, lib := setup(t)
	w := log.Writer(loglayer.LogLevelDebug)

	w.Write([]byte("with newline\n"))
	w.Write([]byte("without newline"))
	w.Write([]byte("multiple\n\n\n"))

	lines := lib.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].Messages[0] != "with newline" {
		t.Errorf("line[0]: %q", lines[0].Messages[0])
	}
	if lines[1].Messages[0] != "without newline" {
		t.Errorf("line[1]: %q", lines[1].Messages[0])
	}
	if lines[2].Messages[0] != "multiple" {
		t.Errorf("line[2]: %q", lines[2].Messages[0])
	}
}

func TestWriter_EmptyWriteSuppressed(t *testing.T) {
	log, lib := setup(t)
	w := log.Writer(loglayer.LogLevelInfo)

	w.Write([]byte(""))
	w.Write([]byte("\n"))
	w.Write([]byte("\n\n\n"))

	if lib.Len() != 0 {
		t.Errorf("empty / whitespace-only writes should produce no entries, got %d", lib.Len())
	}
}

func TestWriter_LevelHonored(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelWarn)

	w := log.Writer(loglayer.LogLevelInfo)
	fmt.Fprintln(w, "filtered")
	if lib.Len() != 0 {
		t.Errorf("Info-level write should be filtered when SetLevel(Warn), got %d", lib.Len())
	}

	w2 := log.Writer(loglayer.LogLevelError)
	fmt.Fprintln(w2, "kept")
	line := lib.PopLine()
	if line == nil || line.Level != loglayer.LogLevelError {
		t.Errorf("Error write should pass: got %v", line)
	}
}

// stdlib log uses the writer with its own formatting. With flags=0 and
// prefix="" (the defaults from NewLogLogger), what we see is just the
// message as the user passed it, no timestamp duplication.
func TestNewLogLogger_NoExtraPrefix(t *testing.T) {
	log, lib := setup(t)
	stdlog := log.NewLogLogger(loglayer.LogLevelInfo)

	stdlog.Print("from stdlib")

	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a line")
	}
	got := line.Messages[0].(string)
	if got != "from stdlib" {
		t.Errorf("message: got %q, want \"from stdlib\" (no extra prefix)", got)
	}
}

// stdlib log.Println calls Write with the trailing "\n" included. The
// writer should strip it.
func TestNewLogLogger_PrintlnTrim(t *testing.T) {
	log, lib := setup(t)
	stdlog := log.NewLogLogger(loglayer.LogLevelWarn)

	stdlog.Println("with trailing newline")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a line")
	}
	got := line.Messages[0].(string)
	if strings.HasSuffix(got, "\n") {
		t.Errorf("trailing newline should be stripped: got %q", got)
	}
	if got != "with trailing newline" {
		t.Errorf("message: got %q", got)
	}
}

// Persistent fields on the underlying logger flow through the writer.
// (The writer dispatches via Raw, which uses the logger's existing
// fields when RawLogEntry.Fields is nil.)
func TestWriter_PreservesPersistentFields(t *testing.T) {
	log, lib := setup(t)
	scoped := log.WithFields(loglayer.F{"requestId": "abc"})
	w := scoped.Writer(loglayer.LogLevelInfo)

	fmt.Fprintln(w, "served")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a line")
	}
	if line.Data["requestId"] != "abc" {
		t.Errorf("requestId not preserved: %v", line.Data)
	}
}

// The plugin pipeline runs for emissions through the writer.
func TestWriter_PluginPipelineRuns(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.NewMessageHook("upper", func(p loglayer.BeforeMessageOutParams) []any {
		out := make([]any, len(p.Messages))
		for i, m := range p.Messages {
			if s, ok := m.(string); ok {
				out[i] = strings.ToUpper(s)
			} else {
				out[i] = m
			}
		}
		return out
	}))

	w := log.Writer(loglayer.LogLevelInfo)
	fmt.Fprintln(w, "hello")
	line := lib.PopLine()
	if line == nil || line.Messages[0] != "HELLO" {
		t.Errorf("plugin should have uppercased the message: got %v", line)
	}
}
