// Package idgen produces opaque, near-unique identifiers for plugins,
// transports, and any other LogLayer machinery that needs a stable handle
// when the caller didn't supply one. Every ID is `prefix + 12 hex chars`
// from crypto/rand, with a process-counter fallback if the OS RNG fails.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"sync/atomic"
)

// Prefixes used by LogLayer's built-in auto-ID call sites. Tests and tooling
// can match against them to recognize an auto-generated ID.
const (
	PluginPrefix    = "auto-plugin-"
	TransportPrefix = "auto-transport-"
)

var fallbackCounter atomic.Uint64

// Random returns prefix concatenated with 12 hex chars from crypto/rand.
func Random(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "fallback-" + strconv.FormatUint(fallbackCounter.Add(1), 16)
	}
	return prefix + hex.EncodeToString(b[:])
}
