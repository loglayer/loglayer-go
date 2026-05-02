package loglayer_test

// Tests that concurrent log emission is safe. Run with -race to catch any
// missing synchronization in the emission path.
//
// Contract: every method on *LogLayer is safe to call from any goroutine,
// including concurrently with emission. Returns-new methods (WithFields,
// WithoutFields, Child, WithPrefix) build a new logger; level/transport/mute
// mutators are atomic.

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/internal/lltest"
	"go.loglayer.dev/v2/transport"
)

func TestConcurrentEmission_SimpleMessage(t *testing.T) {
	log, lib := setup(t)

	const goroutines = 32
	const perGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				log.Info("hello")
			}
		}()
	}
	wg.Wait()

	want := goroutines * perGoroutine
	if got := lib.Len(); got != want {
		t.Errorf("expected %d captured lines, got %d", want, got)
	}
}

func TestConcurrentEmission_WithMetadataAndError(t *testing.T) {
	log, lib := setup(t)

	const goroutines = 16
	const perGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		gid := g
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				log.WithMetadata(loglayer.Metadata{
					"goroutine": gid,
					"iter":      i,
				}).Info("loop")
				log.WithError(benchErr("boom")).Error("failed")
			}
		}()
	}
	wg.Wait()

	want := goroutines * perGoroutine * 2
	if got := lib.Len(); got != want {
		t.Errorf("expected %d captured lines, got %d", want, got)
	}
}

func TestConcurrentEmission_ChildLoggersIndependent(t *testing.T) {
	parent, parentLib := setup(t)
	parent = parent.WithFields(loglayer.Fields{"shared": "value"})

	const goroutines = 8
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			child := parent.Child()
			for i := 0; i < perGoroutine; i++ {
				child.Info("from child")
			}
		}()
	}
	wg.Wait()

	want := goroutines * perGoroutine
	if got := parentLib.Len(); got != want {
		t.Errorf("expected %d captured lines, got %d", want, got)
	}
	last := parentLib.GetLastLine()
	if last == nil || last.Data["shared"] != "value" {
		t.Errorf("child should inherit parent fields: got %+v", last)
	}
}

// TestConcurrentRuntimeLevelToggle verifies that toggling the level on a live
// logger from one goroutine does not race with emission from another. Models
// the operator pattern: SIGUSR1 / admin endpoint flipping debug logging while
// production traffic continues.
func TestConcurrentRuntimeLevelToggle(t *testing.T) {
	log, _ := setup(t)

	const emitters = 16
	const togglers = 4
	const iters = 200

	var wg sync.WaitGroup
	var stop atomic.Bool

	wg.Add(emitters)
	for g := 0; g < emitters; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				log.Info("traffic")
				log.Debug("verbose")
				log.Warn("warning")
				if stop.Load() {
					return
				}
			}
		}()
	}

	wg.Add(togglers)
	for g := 0; g < togglers; g++ {
		gid := g
		go func() {
			defer wg.Done()
			for i := 0; i < iters/2; i++ {
				switch (gid + i) % 4 {
				case 0:
					log.SetLevel(loglayer.LogLevelDebug)
				case 1:
					log.SetLevel(loglayer.LogLevelWarn)
				case 2:
					log.DisableLogging()
				case 3:
					log.EnableLogging()
				}
			}
		}()
	}

	wg.Wait()
	stop.Store(true)
}

// TestConcurrentTransportSwap verifies that AddTransport / RemoveTransport
// can run while emission is in flight. Models the hot-reload pattern where a
// config-watcher swaps transports during live traffic.
func TestConcurrentTransportSwap(t *testing.T) {
	libBase := &lltest.TestLoggingLibrary{}
	base := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "base"}, Library: libBase})
	log := loglayer.New(loglayer.Config{Transport: base, DisableFatalExit: true})

	libExtra := &lltest.TestLoggingLibrary{}
	extra := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "extra"}, Library: libExtra})

	const emitters = 16
	const swappers = 4
	const iters = 200

	var wg sync.WaitGroup
	var stop atomic.Bool

	wg.Add(emitters)
	for g := 0; g < emitters; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				log.Info("traffic")
				if stop.Load() {
					return
				}
			}
		}()
	}

	wg.Add(swappers)
	for g := 0; g < swappers; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters/4; i++ {
				log.AddTransport(extra)
				log.RemoveTransport("extra")
			}
		}()
	}

	wg.Wait()
	stop.Store(true)
}

// TestConcurrentPluginMutation runs AddPlugin/RemovePlugin concurrently
// with emission. The plugin set is published as an immutable snapshot
// via atomic.Pointer, so the dispatch path always sees a consistent
// snapshot, but only -race + a stress test like this proves it.
func TestConcurrentPluginMutation(t *testing.T) {
	libBase := &lltest.TestLoggingLibrary{}
	base := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "base"}, Library: libBase})
	log := loglayer.New(loglayer.Config{Transport: base, DisableFatalExit: true})

	tagger := loglayer.NewDataHook("tag", func(p loglayer.BeforeDataOutParams) loglayer.Data {
		return loglayer.Data{"tagged": true}
	})

	const emitters = 16
	const mutators = 4
	const iters = 200

	var wg sync.WaitGroup
	var stop atomic.Bool

	wg.Add(emitters)
	for g := 0; g < emitters; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				// Mix WithFields/WithMetadata so the dispatch path
				// exercises OnFieldsCalled, OnMetadataCalled, and
				// OnBeforeDataOut while plugins are added/removed.
				log.WithFields(loglayer.Fields{"k": "v"}).
					WithMetadata(loglayer.Metadata{"m": 1}).
					Info("traffic")
				if stop.Load() {
					return
				}
			}
		}()
	}

	wg.Add(mutators)
	for g := 0; g < mutators; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters/4; i++ {
				log.AddPlugin(tagger)
				log.RemovePlugin("tag")
			}
		}()
	}

	wg.Wait()
	stop.Store(true)
}
