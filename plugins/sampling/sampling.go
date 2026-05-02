// Package sampling provides plugins that drop log emissions to keep volume
// and cost under control. All samplers are SendGate plugins, so they make
// per-transport decisions before dispatch and never mutate the entry.
//
// Three strategies:
//
//   - [FixedRate]: independent random draw per emission. Best for "I want
//     about 1% of debug logs to make it through."
//   - [FixedRatePerLevel]: same shape, but the rate is per LogLevel.
//     Levels not in the map are kept unconditionally.
//   - [Burst]: keep the first N emissions per rolling window, drop the
//     rest. Best for hard caps like "no more than 100 logs/second."
//
// Compose by registering multiple instances; each runs as its own SendGate
// and an emission is kept only when every gate returns true.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package sampling

import (
	"math/rand/v2"
	"sync"
	"time"

	"go.loglayer.dev/v2"
)

// FixedRate keeps the given fraction of emissions and drops the rest.
//
//   - rate >= 1: keep every emission (the gate is a no-op).
//   - rate <= 0: drop every emission (effectively disables the logger
//     for this transport unless an earlier plugin overrides).
//   - 0 < rate < 1: independent Bernoulli draw per emission.
//
// Random selection uses [math/rand/v2], which is goroutine-safe and seeded
// by the runtime. For a deterministic sampler in tests, register your own
// SendGate that consults a controlled source.
func FixedRate(rate float64) loglayer.Plugin {
	switch {
	case rate >= 1:
		return loglayer.NewSendGate("sampling-fixed", func(loglayer.ShouldSendParams) bool { return true })
	case rate <= 0:
		return loglayer.NewSendGate("sampling-fixed", func(loglayer.ShouldSendParams) bool { return false })
	}
	return loglayer.NewSendGate("sampling-fixed", func(loglayer.ShouldSendParams) bool {
		return rand.Float64() < rate
	})
}

// FixedRatePerLevel applies a per-level rate. Levels not in the map are
// kept unconditionally so the common shape is "sample debug at 1%, keep
// everything else":
//
//	sampling.FixedRatePerLevel(map[loglayer.LogLevel]float64{
//	    loglayer.LogLevelTrace: 0.01,
//	    loglayer.LogLevelDebug: 0.01,
//	})
//
// rate semantics match [FixedRate]: >=1 keeps, <=0 drops, in between
// is a per-emission draw.
func FixedRatePerLevel(rates map[loglayer.LogLevel]float64) loglayer.Plugin {
	// Snapshot so callers can mutate the map afterward without affecting
	// the live sampler.
	frozen := make(map[loglayer.LogLevel]float64, len(rates))
	for k, v := range rates {
		frozen[k] = v
	}
	return loglayer.NewSendGate("sampling-per-level", func(p loglayer.ShouldSendParams) bool {
		rate, ok := frozen[p.LogLevel]
		if !ok {
			return true
		}
		switch {
		case rate >= 1:
			return true
		case rate <= 0:
			return false
		}
		return rand.Float64() < rate
	})
}

// Burst keeps the first n emissions in each rolling window of the given
// duration, dropping the rest until the window resets. Use it when you
// want a hard rate cap regardless of bursts:
//
//	sampling.Burst(100, time.Second)  // at most 100 logs per second
//
// The window is shared across all levels and transports; for per-level
// caps register multiple instances with distinct IDs (Burst's default
// ID would collide).
//
// Implementation note: the sampler holds an internal mutex, so under
// extreme contention it can serialize the dispatch path. The lock scope
// is tiny (a counter check and a time comparison); on real workloads
// this is negligible compared to the emission cost itself.
func Burst(n int, window time.Duration) loglayer.Plugin {
	if n <= 0 {
		return loglayer.NewSendGate("sampling-burst", func(loglayer.ShouldSendParams) bool { return false })
	}
	if window <= 0 {
		// A non-positive window means "no limit"; the gate is a no-op.
		return loglayer.NewSendGate("sampling-burst", func(loglayer.ShouldSendParams) bool { return true })
	}
	var (
		mu        sync.Mutex
		windowEnd time.Time
		count     int
	)
	return loglayer.NewSendGate("sampling-burst", func(loglayer.ShouldSendParams) bool {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if now.After(windowEnd) {
			windowEnd = now.Add(window)
			count = 0
		}
		if count >= n {
			return false
		}
		count++
		return true
	})
}
