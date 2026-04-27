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

func TestWithFreshTransports(t *testing.T) {
	log, oldLib := setup(t)
	newLib := &lltest.TestLoggingLibrary{}
	newTrans := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "new"}, Library: newLib})
	log.WithFreshTransports(newTrans)
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
