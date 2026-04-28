package sampling_test

import (
	"sync"
	"testing"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/plugintest"
	"go.loglayer.dev/plugins/sampling"
)

func TestFixedRate_KeepAll(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.FixedRate(1.0))
	for i := 0; i < 100; i++ {
		log.Info("hi")
	}
	if got := lib.Len(); got != 100 {
		t.Errorf("rate=1.0 should keep every emission: got %d, want 100", got)
	}
}

func TestFixedRate_DropAll(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.FixedRate(0))
	for i := 0; i < 100; i++ {
		log.Info("hi")
	}
	if got := lib.Len(); got != 0 {
		t.Errorf("rate=0 should drop every emission: got %d, want 0", got)
	}
}

// Statistical: with rate=0.1 and 10000 draws, the kept count should be
// near 1000. Wide tolerance to avoid flakes (5σ for a binomial would be
// ~150; we use ±400 for safety).
func TestFixedRate_ApproximateRate(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.FixedRate(0.1))
	const n = 10000
	for i := 0; i < n; i++ {
		log.Info("hi")
	}
	got := lib.Len()
	if got < 600 || got > 1400 {
		t.Errorf("rate=0.1 of %d should yield ~1000 kept: got %d", n, got)
	}
}

func TestFixedRatePerLevel_OnlyDebugSampled(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.FixedRatePerLevel(map[loglayer.LogLevel]float64{
		loglayer.LogLevelDebug: 0,
	}))

	log.Debug("dropped")
	log.Info("kept")
	log.Warn("kept")
	log.Error("kept")

	if got := lib.Len(); got != 3 {
		t.Errorf("expected 3 lines (debug dropped), got %d", got)
	}
	for _, line := range lib.Lines() {
		if line.Level == loglayer.LogLevelDebug {
			t.Errorf("debug should not appear: %v", line)
		}
	}
}

// Levels not in the map are kept unconditionally.
func TestFixedRatePerLevel_UnmappedLevelsKept(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.FixedRatePerLevel(map[loglayer.LogLevel]float64{
		// Only Trace is rate-limited; others (Info, Warn, Error) unmapped.
		loglayer.LogLevelTrace: 0,
	}))

	log.Trace("dropped")
	log.Debug("kept")
	log.Info("kept")
	log.Warn("kept")

	if got := lib.Len(); got != 3 {
		t.Errorf("expected 3 lines (only trace dropped), got %d", got)
	}
}

// Caller's mutation of the rates map after construction must not affect
// the live sampler — the plugin took a snapshot.
func TestFixedRatePerLevel_RatesSnapshot(t *testing.T) {
	t.Parallel()
	rates := map[loglayer.LogLevel]float64{loglayer.LogLevelInfo: 0}
	log, lib := plugintest.Install(t, sampling.FixedRatePerLevel(rates))

	rates[loglayer.LogLevelInfo] = 1.0 // caller mutates after install
	log.Info("still dropped")

	if lib.Len() != 0 {
		t.Errorf("post-install map mutation must not affect the sampler: got %d", lib.Len())
	}
}

func TestBurst_KeepsFirstN(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.Burst(3, time.Hour))

	for i := 0; i < 10; i++ {
		log.Info("emit")
	}
	if got := lib.Len(); got != 3 {
		t.Errorf("Burst(3, 1h) should keep first 3, got %d", got)
	}
}

func TestBurst_ResetsAfterWindow(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.Burst(2, 50*time.Millisecond))

	log.Info("kept-1")
	log.Info("kept-2")
	log.Info("dropped")

	time.Sleep(80 * time.Millisecond)

	log.Info("kept-3-after-window")
	log.Info("kept-4-after-window")
	log.Info("dropped-again")

	if got := lib.Len(); got != 4 {
		t.Errorf("Burst should reset after window: got %d, want 4", got)
	}
}

func TestBurst_ZeroNDropsEverything(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.Burst(0, time.Hour))
	log.Info("dropped")
	if lib.Len() != 0 {
		t.Errorf("Burst(0, ...) should drop everything: got %d", lib.Len())
	}
}

func TestBurst_NonPositiveWindowKeepsAll(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.Burst(5, 0))
	for i := 0; i < 100; i++ {
		log.Info("emit")
	}
	if got := lib.Len(); got != 100 {
		t.Errorf("Burst(_, 0) should keep everything (no time limit): got %d", got)
	}
}

func TestBurst_ConcurrentEmission(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.Burst(50, time.Hour))

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 32; j++ {
				log.Info("emit")
			}
		}()
	}
	wg.Wait()

	// 16 * 32 = 512 attempts; cap is 50.
	if got := lib.Len(); got != 50 {
		t.Errorf("Burst(50) under concurrency should yield 50 kept, got %d", got)
	}
}

// Multiple sampling plugins compose: emission must pass every gate.
func TestSampling_Composition(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, sampling.FixedRate(1.0))
	log.AddPlugin(sampling.Burst(3, time.Hour))

	for i := 0; i < 10; i++ {
		log.Info("emit")
	}
	if got := lib.Len(); got != 3 {
		t.Errorf("Burst should bound the rate even when FixedRate=1: got %d", got)
	}
}
