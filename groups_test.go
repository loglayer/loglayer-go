package loglayer_test

import (
	"errors"
	"os"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

// twoTransports builds two named TestLoggingLibrary-backed transports for
// routing tests.
func twoTransports(ids ...string) ([]loglayer.Transport, []*lltest.TestLoggingLibrary) {
	libs := make([]*lltest.TestLoggingLibrary, len(ids))
	out := make([]loglayer.Transport, len(ids))
	for i, id := range ids {
		libs[i] = &lltest.TestLoggingLibrary{}
		out[i] = lltest.New(lltest.Config{
			BaseConfig: transport.BaseConfig{ID: id},
			Library:    libs[i],
		})
	}
	return out, libs
}

func TestGroups_NoConfig_AllTransportsReceive(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
	})
	log.WithGroup("nonexistent").Info("hi")
	if libs[0].Len() != 1 || libs[1].Len() != 1 {
		t.Errorf("with no Groups config, both transports should receive: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_PerCallTagging_RoutesToGroupTransports(t *testing.T) {
	tr, libs := twoTransports("console", "datadog")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"database": {Transports: []string{"datadog"}},
		},
	})
	log.WithGroup("database").Error("connection lost")

	if libs[0].Len() != 0 {
		t.Errorf("console should not receive (not in group's transports): got %d", libs[0].Len())
	}
	if libs[1].Len() != 1 {
		t.Errorf("datadog should receive: got %d", libs[1].Len())
	}
}

func TestGroups_PersistentTagging_OnLogger(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"only-b": {Transports: []string{"b"}},
		},
	})
	dbLog := log.WithGroup("only-b")
	dbLog.Info("first")
	dbLog.Warn("second")

	if libs[0].Len() != 0 {
		t.Errorf("transport 'a' shouldn't receive: got %d", libs[0].Len())
	}
	if libs[1].Len() != 2 {
		t.Errorf("transport 'b' should receive both: got %d", libs[1].Len())
	}
}

func TestGroups_MultipleGroups_UnionOfTransports(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"g1": {Transports: []string{"a"}},
			"g2": {Transports: []string{"b"}},
		},
	})
	log.WithGroup("g1", "g2").Info("both")
	if libs[0].Len() != 1 || libs[1].Len() != 1 {
		t.Errorf("union of g1+g2 transports should receive: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_PerGroupLevelFilter(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"errors-only": {Transports: []string{"a"}, Level: loglayer.LogLevelError},
		},
	})
	log.WithGroup("errors-only").Info("dropped by group level")
	log.WithGroup("errors-only").Error("kept")
	if libs[0].Len() != 1 {
		t.Errorf("expected 1 line (error only): got %d", libs[0].Len())
	}
}

func TestGroups_DisabledGroupOnly_Drops(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"db": {Transports: []string{"b"}, Disabled: true},
		},
	})
	log.WithGroup("db").Info("disabled group only")
	if libs[0].Len() != 0 || libs[1].Len() != 0 {
		t.Errorf("disabled-only tag should drop (NOT fall back): a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_MixedDisabledAndEnabled_EnabledRoutes(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"off": {Transports: []string{"a"}, Disabled: true},
			"on":  {Transports: []string{"b"}},
		},
	})
	log.WithGroup("off", "on").Info("hi")
	if libs[0].Len() != 0 {
		t.Errorf("disabled group's transport should not receive: got %d", libs[0].Len())
	}
	if libs[1].Len() != 1 {
		t.Errorf("enabled group should still route: got %d", libs[1].Len())
	}
}

func TestGroups_UnknownGroupOnly_FallsBackToUngrouped(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"defined": {Transports: []string{"a"}},
		},
	})
	log.WithGroup("undefined").Info("hi")
	if libs[0].Len() != 1 {
		t.Errorf("entry tagged only with undefined group should fall back to ungrouped (default: all): got %d", libs[0].Len())
	}
}

func TestGroups_UngroupedToNone_DropsUntaggedEntries(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"defined": {Transports: []string{"a"}},
		},
		UngroupedRouting: loglayer.UngroupedRouting{Mode: loglayer.UngroupedToNone},
	})
	log.Info("untagged")
	if libs[0].Len() != 0 {
		t.Errorf("untagged entry should be dropped under UngroupedToNone: got %d", libs[0].Len())
	}
	log.WithGroup("defined").Info("tagged")
	if libs[0].Len() != 1 {
		t.Errorf("tagged entry should still route: got %d", libs[0].Len())
	}
}

func TestGroups_UngroupedToTransports(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"db": {Transports: []string{"b"}},
		},
		UngroupedRouting: loglayer.UngroupedRouting{
			Mode:       loglayer.UngroupedToTransports,
			Transports: []string{"a"},
		},
	})
	log.Info("untagged → only a")
	if libs[0].Len() != 1 || libs[1].Len() != 0 {
		t.Errorf("untagged should hit only allowlisted: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
	log.WithGroup("db").Info("tagged → only b")
	if libs[0].Len() != 1 || libs[1].Len() != 1 {
		t.Errorf("tagged should hit group transport: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_ActiveGroupsFilter(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"db":   {Transports: []string{"a"}},
			"auth": {Transports: []string{"b"}},
		},
		ActiveGroups: []string{"db"},
	})
	log.WithGroup("db").Info("via db")
	log.WithGroup("auth").Info("via auth (filtered out)")

	if libs[0].Len() != 1 {
		t.Errorf("db group should route: got %d", libs[0].Len())
	}
	if libs[1].Len() != 0 {
		t.Errorf("auth group should be filtered: got %d", libs[1].Len())
	}
}

func TestGroups_RuntimeMutators(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: tr})

	log.AddGroup("dynamic", loglayer.LogGroup{Transports: []string{"a"}, Level: loglayer.LogLevelWarn})

	log.WithGroup("dynamic").Info("dropped by group level")
	if libs[0].Len() != 0 {
		t.Errorf("Info under Warn-min group should drop: got %d", libs[0].Len())
	}

	log.SetGroupLevel("dynamic", loglayer.LogLevelDebug)
	log.WithGroup("dynamic").Info("kept after relaxing")
	if libs[0].Len() != 1 {
		t.Errorf("after SetGroupLevel(Debug): got %d", libs[0].Len())
	}

	log.DisableGroup("dynamic")
	log.WithGroup("dynamic").Info("disabled → drops, no fall-back")
	if libs[0].Len() != 1 {
		t.Errorf("disabled group should drop entry, not route: got %d", libs[0].Len())
	}

	log.EnableGroup("dynamic")
	log.WithGroup("dynamic").Info("re-enabled")
	if libs[0].Len() != 2 {
		t.Errorf("re-enabled group: got %d", libs[0].Len())
	}

	if !log.RemoveGroup("dynamic") {
		t.Error("RemoveGroup should return true for existing group")
	}
	if log.RemoveGroup("dynamic") {
		t.Error("RemoveGroup should return false for missing group")
	}
}

func TestGroups_SetActiveGroupsAndClear(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"g1": {Transports: []string{"a"}},
			"g2": {Transports: []string{"a"}},
		},
		UngroupedRouting: loglayer.UngroupedRouting{Mode: loglayer.UngroupedToNone},
	})

	log.SetActiveGroups("g1")
	log.WithGroup("g1").Info("g1 active")
	log.WithGroup("g2").Info("g2 inactive → falls to ungrouped → dropped (UngroupedToNone)")
	if libs[0].Len() != 1 {
		t.Errorf("only g1 should pass: got %d", libs[0].Len())
	}

	log.ClearActiveGroups()
	log.WithGroup("g2").Info("g2 active again")
	if libs[0].Len() != 2 {
		t.Errorf("after ClearActiveGroups, g2 should pass: got %d", libs[0].Len())
	}
}

func TestGroups_WithGroupChainsAdditively(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"g1": {Transports: []string{"a"}},
			"g2": {Transports: []string{"b"}},
		},
	})
	chained := log.WithGroup("g1").WithGroup("g2")
	chained.Info("hits both")
	if libs[0].Len() != 1 || libs[1].Len() != 1 {
		t.Errorf("chained WithGroup should accumulate: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_BuilderWithGroupMergesWithLoggerGroups(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"g1": {Transports: []string{"a"}},
			"g2": {Transports: []string{"b"}},
		},
	})
	persistent := log.WithGroup("g1")
	persistent.WithGroup("g2").Info("two routes")
	if libs[0].Len() != 1 || libs[1].Len() != 1 {
		t.Errorf("builder.WithGroup should merge with logger.WithGroup: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
	persistent.Info("only g1")
	if libs[0].Len() != 2 || libs[1].Len() != 1 {
		t.Errorf("plain emit should use only persistent groups: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_GetGroups_ReturnsCopy(t *testing.T) {
	tr, _ := twoTransports("a")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"g": {Transports: []string{"a"}, Level: loglayer.LogLevelInfo},
		},
	})

	// (1) Replacing entries in the returned map should not bleed back.
	got := log.GetGroups()
	got["g"] = loglayer.LogGroup{Transports: []string{"hacked"}}
	got["new"] = loglayer.LogGroup{Transports: []string{"injected"}}

	live := log.GetGroups()
	if len(live) != 1 {
		t.Errorf("map-level mutation should not bleed: %v", live)
	}
	if live["g"].Transports[0] != "a" {
		t.Errorf("transports should not be mutable via map replace: %v", live["g"])
	}

	// (2) Mutating the returned LogGroup's Transports slice in place
	// should also not bleed (load-bearing for cloneLogGroup).
	got2 := log.GetGroups()
	got2["g"].Transports[0] = "in-place-hack"

	live2 := log.GetGroups()
	if live2["g"].Transports[0] != "a" {
		t.Errorf("in-place Transports mutation bled through: %v", live2["g"])
	}
}

func TestGroups_RawHonorsEntryGroups(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"only-b": {Transports: []string{"b"}},
		},
	})
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw"},
		Groups:   []string{"only-b"},
	})
	if libs[0].Len() != 0 || libs[1].Len() != 1 {
		t.Errorf("Raw with Groups should route: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_ChildInheritsAssignedGroups(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"only-b": {Transports: []string{"b"}},
		},
	})
	tagged := log.WithGroup("only-b")
	child := tagged.Child()
	child.Info("inherits the group")
	if libs[0].Len() != 0 || libs[1].Len() != 1 {
		t.Errorf("child should inherit assigned groups: a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

func TestGroups_ChildGroupIsolation(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: tr})
	child := log.Child()
	child.AddGroup("only-a", loglayer.LogGroup{Transports: []string{"a"}})

	// Parent should not see the group: untagged emit on parent goes
	// through ungrouped (UngroupedToAll) → reaches a.
	log.Info("parent untagged")
	if libs[0].Len() != 1 {
		t.Errorf("parent should not see child's group config: got %d", libs[0].Len())
	}
	// And there's no parent group to tag, so this also goes via ungrouped.
	log.WithGroup("only-a").Info("parent tries to tag")
	// Parent has no Groups configured → no routing at all → all transports.
	if libs[0].Len() != 2 {
		t.Errorf("parent has no groups, should still emit: got %d", libs[0].Len())
	}
}

func TestGroups_PluginShouldSendStillRuns(t *testing.T) {
	tr, libs := twoTransports("a", "b")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"both": {Transports: []string{"a", "b"}},
		},
	})
	log.AddPlugin(loglayer.NewSendGate("drop-b", func(p loglayer.ShouldSendParams) bool {
		return p.TransportID != "b"
	}))
	log.WithGroup("both").Info("hi")
	if libs[0].Len() != 1 {
		t.Errorf("a should receive (group + ShouldSend allow): got %d", libs[0].Len())
	}
	if libs[1].Len() != 0 {
		t.Errorf("b should be vetoed by ShouldSend even though group allows: got %d", libs[1].Len())
	}
}

func TestGroups_Build_NoErrors(t *testing.T) {
	tr, _ := twoTransports("a")
	if _, err := loglayer.Build(loglayer.Config{
		Transports: tr,
		Groups:     map[string]loglayer.LogGroup{"g": {Transports: []string{"a"}}},
	}); err != nil {
		t.Errorf("Build with Groups should succeed: %v", err)
	}
}

// Setting UngroupedRouting.Transports without bumping Mode to
// UngroupedToTransports is a footgun: the transports list is silently
// ignored. Build should reject it loudly.
func TestGroups_UngroupedTransportsWithoutMode_Errors(t *testing.T) {
	tr, _ := twoTransports("a")
	_, err := loglayer.Build(loglayer.Config{
		Transports: tr,
		UngroupedRouting: loglayer.UngroupedRouting{
			// Mode left at zero value (UngroupedToAll).
			Transports: []string{"a"},
		},
	})
	if !errors.Is(err, loglayer.ErrUngroupedTransportsWithoutMode) {
		t.Errorf("Build should return ErrUngroupedTransportsWithoutMode; got %v", err)
	}
}

func TestActiveGroupsFromEnv(t *testing.T) {
	const envName = "TEST_LOGLAYER_GROUPS_ENV"

	t.Run("unset returns nil", func(t *testing.T) {
		os.Unsetenv(envName)
		if got := loglayer.ActiveGroupsFromEnv(envName); got != nil {
			t.Errorf("unset env: got %v, want nil", got)
		}
	})

	t.Run("empty returns nil", func(t *testing.T) {
		t.Setenv(envName, "")
		if got := loglayer.ActiveGroupsFromEnv(envName); got != nil {
			t.Errorf("empty env: got %v, want nil", got)
		}
	})

	t.Run("comma-separated parses with whitespace tolerance", func(t *testing.T) {
		t.Setenv(envName, " db , auth, payments ")
		got := loglayer.ActiveGroupsFromEnv(envName)
		want := []string{"db", "auth", "payments"}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("only commas returns nil", func(t *testing.T) {
		t.Setenv(envName, ",,,")
		if got := loglayer.ActiveGroupsFromEnv(envName); got != nil {
			t.Errorf("only commas: got %v, want nil", got)
		}
	})
}

// Smoke test that group routing doesn't break the WithError chain.
func TestGroups_WithErrorAndGroupsCompose(t *testing.T) {
	tr, libs := twoTransports("a")
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       tr,
		Groups: map[string]loglayer.LogGroup{
			"g": {Transports: []string{"a"}},
		},
	})
	log.WithError(errors.New("boom")).WithGroup("g").Error("explode")
	line := libs[0].PopLine()
	if line == nil {
		t.Fatal("expected line")
	}
	if line.Data["err"] == nil {
		t.Errorf("error should still be present: %v", line.Data)
	}
}
