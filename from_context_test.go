package loglayer_test

import (
	"context"
	"testing"

	"go.loglayer.dev/v2"
)

func TestNewContextAndFromContext_RoundTrip(t *testing.T) {
	log, _ := setup(t)
	ctx := loglayer.NewContext(context.Background(), log)

	got := loglayer.FromContext(ctx)
	if got != log {
		t.Errorf("FromContext returned a different logger: want %p, got %p", log, got)
	}
}

func TestFromContext_NilWhenNotAttached(t *testing.T) {
	if got := loglayer.FromContext(context.Background()); got != nil {
		t.Errorf("expected nil from bare context, got %v", got)
	}
}

func TestFromContext_NilContextIsNil(t *testing.T) {
	var nilCtx context.Context // typed nil; exercises the guard
	if got := loglayer.FromContext(nilCtx); got != nil {
		t.Errorf("expected nil from nil context, got %v", got)
	}
}

func TestNewContext_NilLoggerReturnsParent(t *testing.T) {
	parent := context.Background()
	ctx := loglayer.NewContext(parent, nil)
	if ctx != parent {
		t.Error("NewContext with nil logger should return parent unchanged")
	}
}

func TestMustFromContext_PanicsIfMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when no logger is attached")
		}
	}()
	loglayer.MustFromContext(context.Background())
}

func TestMustFromContext_ReturnsAttached(t *testing.T) {
	log, _ := setup(t)
	ctx := loglayer.NewContext(context.Background(), log)
	if got := loglayer.MustFromContext(ctx); got != log {
		t.Errorf("MustFromContext returned a different logger: want %p, got %p", log, got)
	}
}

func TestNewContext_ChildLoggerIsIndependent(t *testing.T) {
	parent, parentLib := setup(t)
	parent = parent.WithFields(loglayer.Fields{"app": "main"})

	child := parent.Child()
	child = child.WithFields(loglayer.Fields{"requestId": "abc"})

	// Attach the child to ctx; FromContext should return the child specifically,
	// not the parent.
	ctx := loglayer.NewContext(context.Background(), child)
	got := loglayer.FromContext(ctx)
	if got != child {
		t.Fatalf("FromContext returned wrong logger: want %p, got %p", child, got)
	}

	got.Info("from handler")
	line := parentLib.PopLine()
	if line == nil {
		t.Fatal("expected line via parent transport")
	}
	if line.Data["requestId"] != "abc" || line.Data["app"] != "main" {
		t.Errorf("child should carry both contexts: got %v", line.Data)
	}
}
