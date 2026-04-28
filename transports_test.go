package loglayer_test

import (
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

func TestMultipleTransports(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t1"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: []loglayer.Transport{t1, t2}})

	log.Info("broadcast")
	if lib1.Len() != 1 || lib2.Len() != 1 {
		t.Errorf("both transports should receive the log: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

func TestAddTransport(t *testing.T) {
	log, lib1 := setup(t)
	lib2 := &lltest.TestLoggingLibrary{}
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	log.AddTransport(t2)
	log.Info("both")
	if lib1.Len() != 1 || lib2.Len() != 1 {
		t.Errorf("both transports: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

func TestRemoveTransport(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t1"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: []loglayer.Transport{t1, t2}})

	removed := log.RemoveTransport("t2")
	if !removed {
		t.Error("RemoveTransport should return true for existing ID")
	}
	log.Info("only t1")
	if lib1.Len() != 1 || lib2.Len() != 0 {
		t.Errorf("after remove: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

func TestRemoveTransportMissing(t *testing.T) {
	log, _ := setup(t)
	if log.RemoveTransport("nonexistent") {
		t.Error("RemoveTransport should return false for missing ID")
	}
}

func TestSetTransports(t *testing.T) {
	log, oldLib := setup(t)
	newLib := &lltest.TestLoggingLibrary{}
	newTrans := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "new"}, Library: newLib})
	log.SetTransports(newTrans)
	log.Info("new transport only")
	if oldLib.Len() != 0 || newLib.Len() != 1 {
		t.Errorf("after replace: old=%d new=%d", oldLib.Len(), newLib.Len())
	}
}

func TestGetLoggerInstance(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	trans := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t"}, Library: lib})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: trans})
	instance := log.GetLoggerInstance("t")
	if instance == nil {
		t.Error("GetLoggerInstance should return the underlying library")
	}
}

// AddTransport with an ID that already exists must replace the previous
// transport, not duplicate it. Documented in transports.go but untested.
func TestAddTransport_SameIDReplaces(t *testing.T) {
	libOriginal := &lltest.TestLoggingLibrary{}
	libReplacement := &lltest.TestLoggingLibrary{}
	original := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "shared"}, Library: libOriginal})
	replacement := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "shared"}, Library: libReplacement})

	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: original})
	log.AddTransport(replacement)

	log.Info("after replace")
	if libOriginal.Len() != 0 {
		t.Errorf("original transport should be replaced, got %d entries", libOriginal.Len())
	}
	if libReplacement.Len() != 1 {
		t.Errorf("replacement transport should receive: got %d", libReplacement.Len())
	}
}

// closableTransport records Close calls. Used to verify that
// RemoveTransport / SetTransports / AddTransport-by-replace drain
// async-transport workers via io.Closer.
type closableTransport struct {
	*lltest.TestTransport
	closed *int
}

func (c *closableTransport) Close() error {
	*c.closed++
	return nil
}

func newClosable(id string, lib *lltest.TestLoggingLibrary, closed *int) *closableTransport {
	return &closableTransport{
		TestTransport: lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: id}, Library: lib}),
		closed:        closed,
	}
}

func TestRemoveTransport_ClosesRemoved(t *testing.T) {
	closed := 0
	keep := newClosable("keep", &lltest.TestLoggingLibrary{}, &closed)
	tr := newClosable("t", &lltest.TestLoggingLibrary{}, &closed)
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       []loglayer.Transport{keep, tr},
	})

	if !log.RemoveTransport("t") {
		t.Fatal("RemoveTransport should return true")
	}
	if closed != 1 {
		t.Errorf("removed transport should be closed once, got %d", closed)
	}
}

func TestSetTransports_ClosesEvicted(t *testing.T) {
	closed := 0
	old := newClosable("old", &lltest.TestLoggingLibrary{}, &closed)
	keep := newClosable("keep", &lltest.TestLoggingLibrary{}, &closed)
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       []loglayer.Transport{old, keep},
	})

	newTr := newClosable("new", &lltest.TestLoggingLibrary{}, &closed)
	log.SetTransports(keep, newTr)

	// "old" was evicted; "keep" carried over; "new" is fresh — only "old" closed.
	if closed != 1 {
		t.Errorf("only the evicted transport should be closed, got %d closes", closed)
	}
}

func TestAddTransport_SameIDClosesDisplaced(t *testing.T) {
	closed := 0
	original := newClosable("shared", &lltest.TestLoggingLibrary{}, &closed)
	replacement := newClosable("shared", &lltest.TestLoggingLibrary{}, &closed)

	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: original})
	log.AddTransport(replacement)
	if closed != 1 {
		t.Errorf("displaced transport should be closed, got %d closes", closed)
	}
}

// TestFatal_DisableFatalExitSkipsFlush pins the contract that
// DisableFatalExit skips both os.Exit and the pre-exit flush, so
// async transports remain operational for subsequent emissions
// (relevant for tests that exercise Fatal-via-mock).
func TestFatal_DisableFatalExitSkipsFlush(t *testing.T) {
	closed := 0
	tr := newClosable("t", &lltest.TestLoggingLibrary{}, &closed)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Fatal("goodbye")
	if closed != 0 {
		t.Errorf("DisableFatalExit should skip the flush, got %d closes", closed)
	}
}

// BaseConfig.Disabled at construction time skips the transport regardless
// of the global level state. We test the global Config.Disabled elsewhere
// but never the per-transport version.
func TestTransport_BaseConfigDisabled(t *testing.T) {
	libActive := &lltest.TestLoggingLibrary{}
	libDisabled := &lltest.TestLoggingLibrary{}
	active := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "on"},
		Library:    libActive,
	})
	disabled := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "off", Disabled: true},
		Library:    libDisabled,
	})

	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       []loglayer.Transport{active, disabled},
	})
	log.Info("only active should receive")
	if libActive.Len() != 1 {
		t.Errorf("active transport: got %d", libActive.Len())
	}
	if libDisabled.Len() != 0 {
		t.Errorf("disabled transport should drop entry: got %d", libDisabled.Len())
	}
}

// Child should inherit ALL parent transports, not just the first.
func TestChild_InheritsAllTransports(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t1"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	parent := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: []loglayer.Transport{t1, t2}})

	child := parent.Child()
	child.Info("from child")
	if lib1.Len() != 1 || lib2.Len() != 1 {
		t.Errorf("child should inherit both transports: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

// Parent transport mutations after Child() must not leak into the child.
// Mirror of TestPlugin_ChildPluginIsolation but for transports.
func TestChild_TransportIsolation_AddOnParent(t *testing.T) {
	libParent := &lltest.TestLoggingLibrary{}
	libExtra := &lltest.TestLoggingLibrary{}
	original := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "orig"}, Library: libParent})
	extra := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "extra"}, Library: libExtra})

	parent := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: original})
	child := parent.Child()

	parent.AddTransport(extra)
	child.Info("via child")
	if libExtra.Len() != 0 {
		t.Errorf("child should not see transports added to parent after Child(): got %d", libExtra.Len())
	}
}

func TestChild_TransportIsolation_RemoveOnParent(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t1"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	parent := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: []loglayer.Transport{t1, t2}})
	child := parent.Child()

	parent.RemoveTransport("t2")
	child.Info("via child")
	if lib2.Len() != 1 {
		t.Errorf("child should still have t2 after parent removed it: got %d", lib2.Len())
	}
}

func TestChild_TransportIsolation_SetOnParent(t *testing.T) {
	libOriginal := &lltest.TestLoggingLibrary{}
	libNew := &lltest.TestLoggingLibrary{}
	original := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "orig"}, Library: libOriginal})
	newOne := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "new"}, Library: libNew})

	parent := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: original})
	child := parent.Child()

	parent.SetTransports(newOne)
	child.Info("via child")
	if libOriginal.Len() != 1 {
		t.Errorf("child should retain its snapshot of original transports: got %d", libOriginal.Len())
	}
	if libNew.Len() != 0 {
		t.Errorf("child should not pick up parent's new transports: got %d", libNew.Len())
	}
}
