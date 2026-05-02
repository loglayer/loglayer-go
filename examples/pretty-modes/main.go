// Renders the same variety pack of log entries (every level, map and
// struct metadata, persistent fields, attached errors, MetadataOnly,
// ErrorOnly, Raw, WithContext) through each pretty.ViewMode in two
// configurations: the default (fields and map metadata flatten at root)
// and the keyed shape (FieldsKey="context", MetadataFieldName="metadata").
// Use it to eyeball the rendering options in your terminal.
//
// Run:
//
//	go run go.loglayer.dev/v2/examples/pretty-modes
//
// Or from a clone:
//
//	go run ./examples/pretty-modes
//
// Optional flags:
//
//	-mode inline|expanded|message-only             restrict to one ViewMode (default: all three)
//	-theme moonlight|sunlight|neon|nature|pastel   pick a color theme (default: moonlight)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go.loglayer.dev/transports/pretty/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport/transporttest"
)

func main() {
	modeFlag := flag.String("mode", "", `restrict to one ViewMode: "inline", "expanded", or "message-only"`)
	themeFlag := flag.String("theme", "moonlight", `theme: "moonlight", "sunlight", "neon", "nature", "pastel"`)
	flag.Parse()

	themes := map[string]*pretty.Theme{
		"moonlight": pretty.Moonlight(),
		"sunlight":  pretty.Sunlight(),
		"neon":      pretty.Neon(),
		"nature":    pretty.Nature(),
		"pastel":    pretty.Pastel(),
	}
	theme, ok := themes[*themeFlag]
	if !ok {
		log.Fatalf("unknown theme %q; valid: moonlight, sunlight, neon, nature, pastel", *themeFlag)
	}

	modes := []pretty.ViewMode{
		pretty.ViewModeInline,
		pretty.ViewModeExpanded,
		pretty.ViewModeMessageOnly,
	}
	if *modeFlag != "" {
		modes = []pretty.ViewMode{pretty.ViewMode(*modeFlag)}
	}

	for _, m := range modes {
		tr := pretty.New(pretty.Config{ViewMode: m, Theme: theme})
		fmt.Fprintf(os.Stderr, "\n\033[1;37m═══ ViewMode: %s ═══\033[0m\n", m)
		for _, v := range transporttest.LivetestVariants {
			fmt.Fprintf(os.Stderr, "\n\033[1;36m── %s config ──\033[0m\n\n", v.Name)
			cfg := v.Config
			cfg.Transport = tr
			cfg.DisableFatalExit = true
			transporttest.EmitLivetestSurface(loglayer.New(cfg), v.Name)
		}
	}
}
