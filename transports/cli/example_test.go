package cli_test

import (
	"os"

	"github.com/fatih/color"

	clitr "go.loglayer.dev/transports/cli/v2"
	"go.loglayer.dev/v2"
)

// The package-level Example shows the canonical wiring: a CLI app's
// status messages emit unadorned to stdout, warnings and errors get
// short prefixes and route to stderr.
//
// Color is forced off so the example output is byte-stable; a real
// CLI would leave [cli.Config.Color] at its zero value (ColorAuto)
// to get TTY-detected color.
func Example() {
	t := clitr.New(clitr.Config{
		Stdout: os.Stdout,
		Stderr: os.Stdout, // merged for demo determinism
		Color:  clitr.ColorNever,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})

	log.Info("Applied 1 release(s) at f5f6a9a:")
	log.Warn("running on stale credentials")
	log.Error("connection refused")
	// Output:
	// Applied 1 release(s) at f5f6a9a:
	// warning: running on stale credentials
	// error: connection refused
}

// ExampleNew shows the minimal constructor: zero-value Config gives
// stdout / stderr routing, TTY-detected color, no level label on
// info, and short prefixes on warn / error.
func ExampleNew() {
	t := clitr.New(clitr.Config{
		Stdout: os.Stdout,
		Color:  clitr.ColorNever, // deterministic example output
	})
	log := loglayer.New(loglayer.Config{Transport: t})
	log.Info("hello")
	// Output: hello
}

// ExampleConfig_ShowFields demonstrates the verbose-mode toggle:
// fields and metadata get appended in logfmt style for callers that
// wire `-vv` or `--debug` to ShowFields.
func ExampleConfig_ShowFields() {
	t := clitr.New(clitr.Config{
		Stdout:     os.Stdout,
		Color:      clitr.ColorNever,
		ShowFields: true,
	})
	log := loglayer.New(loglayer.Config{Transport: t})

	log.WithFields(loglayer.Fields{"requestID": "abc-123"}).
		WithMetadata(loglayer.Metadata{"latencyMs": 42}).
		Info("handled")
	// Output: handled latencyMs=42 requestID=abc-123
}

// Example_table demonstrates the table rendering mode: when
// metadata is a slice of maps, the transport renders an aligned
// table after the message. Same call site emits a proper JSON
// array when paired with the structured transport.
func Example_table() {
	t := clitr.New(clitr.Config{
		Stdout: os.Stdout,
		Color:  clitr.ColorNever,
	})
	log := loglayer.New(loglayer.Config{Transport: t})

	log.WithMetadata([]loglayer.Metadata{
		{"package": "transports/foo", "from": "v1.5.0", "to": "v1.6.0"},
		{"package": "transports/bar", "from": "v0.2.0", "to": "v1.0.0"},
	}).Info("Plan:")
	// Output:
	// Plan:
	// FROM    PACKAGE         TO
	// v1.5.0  transports/foo  v1.6.0
	// v0.2.0  transports/bar  v1.0.0
}

// ExampleConfig_DisableLevelPrefix turns off every per-level prefix
// in one shot. Use when the host CLI already renders its own
// urgency markers (icon column, status indicator) and the
// transport's prefixes would be redundant.
func ExampleConfig_DisableLevelPrefix() {
	t := clitr.New(clitr.Config{
		Stdout:             os.Stdout,
		Stderr:             os.Stdout,
		Color:              clitr.ColorNever,
		DisableLevelPrefix: true,
	})
	log := loglayer.New(loglayer.Config{Transport: t})

	log.Info("a")
	log.Warn("b")
	log.Error("c")
	// Output:
	// a
	// b
	// c
}

// ExampleConfig_LevelColor demonstrates per-level color overrides.
// Each entry can be a custom *color.Color (from fatih/color) or nil
// to suppress color for that level while keeping other defaults.
//
// This Example doesn't have a deterministic `// Output:` directive
// because ANSI escape sequences vary by terminal capability and
// fatih/color's resolution. The shape is what matters: pass a
// LevelColor map alongside Color: ColorAlways and the level's
// rendering uses the provided *color.Color.
func ExampleConfig_LevelColor() {
	t := clitr.New(clitr.Config{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Color:  clitr.ColorAlways,
		LevelColor: map[loglayer.LogLevel]*color.Color{
			loglayer.LogLevelWarn: color.New(color.FgCyan), // cyan instead of yellow
			loglayer.LogLevelInfo: nil,                     // unchanged from default
		},
	})
	log := loglayer.New(loglayer.Config{Transport: t})
	log.Warn("rebranded warning") // renders cyan
	_ = log
}

// ExampleConfig_LevelPrefix shows how to override the default
// per-level prefix. Useful for localizing the `warning:` / `error:`
// labels or matching a project's existing CLI brand.
func ExampleConfig_LevelPrefix() {
	t := clitr.New(clitr.Config{
		Stdout: os.Stdout,
		Stderr: os.Stdout,
		Color:  clitr.ColorNever,
		LevelPrefix: map[loglayer.LogLevel]string{
			loglayer.LogLevelWarn:  "[warn]  ",
			loglayer.LogLevelError: "[error] ",
		},
	})
	log := loglayer.New(loglayer.Config{Transport: t})

	log.Warn("about to retry")
	log.Error("retry failed")
	// Output:
	// [warn]  about to retry
	// [error] retry failed
}

// ExampleNew_multiline shows the cli transport rendering authored
// multi-line content. Color is forced off so the rendered output is
// byte-stable.
func ExampleNew_multiline() {
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: os.Stdout,
			Color:  clitr.ColorNever,
		}),
	})
	log.Info(loglayer.Multiline("Configuration:", "  port: 8080", "  host: ::1"))
	// Output:
	// Configuration:
	//   port: 8080
	//   host: ::1
}
