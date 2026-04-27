package loglayer

import (
	"os"
	"strings"
)

// groupSet is an immutable snapshot of the group routing config.
// Mutators publish a new snapshot via atomic.Pointer; the dispatch path
// loads the current snapshot and reads from it without locks.
//
// activeSet nil means "no filter — all defined groups active." A populated
// activeSet restricts routing to its keys.
type groupSet struct {
	groups       map[string]LogGroup
	activeSet    map[string]bool
	ungrouped    UngroupedRouting
	ungroupedSet map[string]bool
	hasGroups    bool
}

func newGroupSet(groups map[string]LogGroup, active []string, ungrouped UngroupedRouting) *groupSet {
	s := &groupSet{
		groups:    copyGroupMap(groups),
		ungrouped: ungrouped,
		hasGroups: len(groups) > 0,
	}
	if len(active) > 0 {
		s.activeSet = make(map[string]bool, len(active))
		for _, g := range active {
			s.activeSet[g] = true
		}
	}
	if ungrouped.Mode == UngroupedToTransports && len(ungrouped.Transports) > 0 {
		s.ungroupedSet = make(map[string]bool, len(ungrouped.Transports))
		for _, t := range ungrouped.Transports {
			s.ungroupedSet[t] = true
		}
	}
	return s
}

// cloneLogGroup deep-copies a LogGroup so callers can't mutate the
// snapshot's Transports slice.
func cloneLogGroup(g LogGroup) LogGroup {
	return LogGroup{
		Transports: append([]string(nil), g.Transports...),
		Level:      g.Level,
		Disabled:   g.Disabled,
	}
}

func copyGroupMap(in map[string]LogGroup) map[string]LogGroup {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]LogGroup, len(in))
	for k, v := range in {
		out[k] = cloneLogGroup(v)
	}
	return out
}

// shouldRoute returns whether the transport with the given ID should
// receive an entry with the given level and group tags. Implements the
// routing precedence documented in groups.md:
//
//  1. If no groups are configured at all, every transport receives.
//  2. If the entry has no groups, apply UngroupedRouting.
//  3. For each tagged group: enabled? in active-filter? level passes?
//     transport listed? Any one match → route.
//  4. If the entry's groups are all undefined, fall back to ungrouped.
//  5. Otherwise no group passed for this transport → drop.
func (s *groupSet) shouldRoute(transportID string, level LogLevel, entryGroups []string) bool {
	if !s.hasGroups {
		return true
	}
	if len(entryGroups) == 0 {
		return s.applyUngrouped(transportID)
	}

	hasAnyDefined := false
	for _, name := range entryGroups {
		cfg, ok := s.groups[name]
		if !ok {
			continue
		}
		hasAnyDefined = true

		if cfg.Disabled {
			continue
		}
		if s.activeSet != nil && !s.activeSet[name] {
			continue
		}
		if cfg.Level != 0 && level < cfg.Level {
			continue
		}
		for _, tid := range cfg.Transports {
			if tid == transportID {
				return true
			}
		}
	}

	if !hasAnyDefined {
		return s.applyUngrouped(transportID)
	}
	return false
}

func (s *groupSet) applyUngrouped(transportID string) bool {
	switch s.ungrouped.Mode {
	case UngroupedToNone:
		return false
	case UngroupedToTransports:
		return s.ungroupedSet[transportID]
	default: // UngroupedToAll
		return true
	}
}

func (l *LogLayer) loadGroups() *groupSet {
	return l.groups.Load()
}

// publishGroupsLocked atomically swaps in a new groupSet derived from
// an existing snapshot, with the supplied groups + activeSet replacing
// the old. The ungrouped routing is untouched (no mutator changes it
// post-construction). Maps are taken by reference; callers must build
// fresh ones when they intend to mutate.
//
// Caller holds groupMu.
func (l *LogLayer) publishGroupsLocked(prev *groupSet, groups map[string]LogGroup, activeSet map[string]bool) {
	l.groups.Store(&groupSet{
		groups:       groups,
		activeSet:    activeSet,
		ungrouped:    prev.ungrouped,
		ungroupedSet: prev.ungroupedSet,
		hasGroups:    len(groups) > 0,
	})
}

// AddGroup registers (or replaces) a named group. If a group with the
// same name already exists it is replaced, matching the AddTransport /
// AddPlugin convention. Returns the receiver for chaining.
//
// Safe to call from any goroutine.
func (l *LogLayer) AddGroup(name string, group LogGroup) *LogLayer {
	l.groupMu.Lock()
	defer l.groupMu.Unlock()
	cur := l.loadGroups()
	groups := copyGroupMap(cur.groups)
	if groups == nil {
		groups = make(map[string]LogGroup, 1)
	}
	groups[name] = cloneLogGroup(group)
	l.publishGroupsLocked(cur, groups, cur.activeSet)
	return l
}

// RemoveGroup deletes a group by name. Returns true if the group was
// present.
//
// Safe to call from any goroutine.
func (l *LogLayer) RemoveGroup(name string) bool {
	l.groupMu.Lock()
	defer l.groupMu.Unlock()
	cur := l.loadGroups()
	if _, ok := cur.groups[name]; !ok {
		return false
	}
	groups := copyGroupMap(cur.groups)
	delete(groups, name)
	l.publishGroupsLocked(cur, groups, cur.activeSet)
	return true
}

// EnableGroup re-enables a previously disabled group. No-op when the
// group is not registered.
func (l *LogLayer) EnableGroup(name string) *LogLayer {
	return l.setGroupDisabled(name, false)
}

// DisableGroup suppresses a group without removing it. Entries tagged
// only with disabled groups are dropped (the explicit-off semantics;
// undefined-group tags fall back to UngroupedRouting). No-op when the
// group is not registered.
func (l *LogLayer) DisableGroup(name string) *LogLayer {
	return l.setGroupDisabled(name, true)
}

func (l *LogLayer) setGroupDisabled(name string, disabled bool) *LogLayer {
	l.groupMu.Lock()
	defer l.groupMu.Unlock()
	cur := l.loadGroups()
	g, ok := cur.groups[name]
	if !ok {
		return l
	}
	if g.Disabled == disabled {
		return l
	}
	groups := copyGroupMap(cur.groups)
	g.Disabled = disabled
	groups[name] = g
	l.publishGroupsLocked(cur, groups, cur.activeSet)
	return l
}

// SetGroupLevel updates the minimum level for a group. No-op when the
// group is not registered.
func (l *LogLayer) SetGroupLevel(name string, level LogLevel) *LogLayer {
	l.groupMu.Lock()
	defer l.groupMu.Unlock()
	cur := l.loadGroups()
	g, ok := cur.groups[name]
	if !ok {
		return l
	}
	if g.Level == level {
		return l
	}
	groups := copyGroupMap(cur.groups)
	g.Level = level
	groups[name] = g
	l.publishGroupsLocked(cur, groups, cur.activeSet)
	return l
}

// SetActiveGroups restricts routing to the named groups. Entries tagged
// with groups not in the list are dropped (or fall back to
// UngroupedRouting if every tagged group is excluded).
//
// At least one group name is required. Use ClearActiveGroups to remove
// the filter entirely; an empty filter is intentionally not expressible
// here because variadic-empty calls (e.g. SetActiveGroups(slice...)
// with an empty slice) would silently suppress all grouped routing.
func (l *LogLayer) SetActiveGroups(first string, more ...string) *LogLayer {
	l.groupMu.Lock()
	defer l.groupMu.Unlock()
	cur := l.loadGroups()
	activeSet := make(map[string]bool, 1+len(more))
	activeSet[first] = true
	for _, g := range more {
		activeSet[g] = true
	}
	l.publishGroupsLocked(cur, cur.groups, activeSet)
	return l
}

// ClearActiveGroups removes the active-groups filter, returning the
// logger to "all defined groups are active."
func (l *LogLayer) ClearActiveGroups() *LogLayer {
	l.groupMu.Lock()
	defer l.groupMu.Unlock()
	cur := l.loadGroups()
	l.publishGroupsLocked(cur, cur.groups, nil)
	return l
}

// GetGroups returns a snapshot of the current group configuration.
// Mutating the returned map (or its LogGroup entries) does not affect
// the logger's state.
func (l *LogLayer) GetGroups() map[string]LogGroup {
	return copyGroupMap(l.loadGroups().groups)
}

// WithGroup returns a child logger tagged with the given groups. Every
// log emitted from the returned logger is dispatched only to the
// transports allowed by these groups (per the routing rules in
// shouldRoute).
//
// The receiver is unchanged. Tags are additive across chained calls:
//
//	dbAuth := log.WithGroup("database").WithGroup("auth")
//	// dbAuth's entries route through both groups' transports.
//
// Returns a child even when groups is empty (so callers can chain
// safely).
func (l *LogLayer) WithGroup(groups ...string) *LogLayer {
	child := l.Child()
	merged := mergeGroups(l.assignedGroups, groups)
	// mergeGroups may return one of its inputs unchanged when the other
	// is empty. Detach from the caller's variadic backing so a later
	// mutation by the user can't affect this child's tags.
	if len(merged) > 0 && (len(l.assignedGroups) == 0 || len(groups) == 0) {
		merged = append([]string(nil), merged...)
	}
	child.assignedGroups = merged
	return child
}

// mergeGroups returns the deduplicated union of a and b, preserving the
// order of a's entries followed by any of b's that weren't already
// present.
//
// When one side is empty the other is returned unchanged. Group slices
// are treated as immutable downstream, so the defensive copy would be
// pure overhead on the dispatch hot path.
func mergeGroups(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, g := range a {
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	for _, g := range b {
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	return out
}

// ActiveGroupsFromEnv reads a comma-separated list of group names from
// the named environment variable and returns it as a slice suitable for
// Config.ActiveGroups or SetActiveGroups. Empty / unset returns nil.
//
// Whitespace around commas is trimmed; empty entries are skipped.
//
// Use it explicitly at startup (we don't read environment variables on
// your behalf):
//
//	loglayer.New(loglayer.Config{
//	    Transport:    ...,
//	    Groups:       ...,
//	    ActiveGroups: loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS"),
//	})
func ActiveGroupsFromEnv(name string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
