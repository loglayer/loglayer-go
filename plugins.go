package loglayer

// loadPlugins returns the current plugin snapshot. Hot path: called on every
// emission for hook lookup.
func (l *LogLayer) loadPlugins() *pluginSet {
	return l.plugins.Load()
}

// AddPlugin registers one or more plugins. Plugins whose ID() returns the
// empty string get an auto-generated identifier; supply your own ID if you
// plan to call RemovePlugin / GetPlugin or replace the plugin later. If a
// plugin's ID matches an already-registered plugin, the existing one is
// replaced.
//
// Safe to call from any goroutine: the plugin set is published atomically.
// Concurrent mutators on the same logger serialize via an internal mutex.
func (l *LogLayer) AddPlugin(plugins ...Plugin) *LogLayer {
	if len(plugins) == 0 {
		return l
	}
	l.pluginMu.Lock()
	defer l.pluginMu.Unlock()

	current := l.loadPlugins()
	newIDs := make(map[string]bool, len(plugins))
	for _, p := range plugins {
		if id := p.ID(); id != "" {
			newIDs[id] = true
		}
	}
	// Carry over existing plugins that aren't being replaced (replacement
	// matches on supplied non-empty IDs only; auto-IDs are unique).
	out := make([]Plugin, 0, len(current.entries)+len(plugins))
	for i, existing := range current.entries {
		if newIDs[existing.id] {
			continue
		}
		out = append(out, current.entries[i].plugin)
	}
	out = append(out, plugins...)
	l.plugins.Store(newPluginSet(out))
	return l
}

// RemovePlugin removes the plugin with the given ID. Returns true if a
// plugin was removed, false if no plugin with that ID was registered.
//
// Safe to call from any goroutine.
func (l *LogLayer) RemovePlugin(id string) bool {
	l.pluginMu.Lock()
	defer l.pluginMu.Unlock()

	current := l.loadPlugins()
	if _, ok := current.byID[id]; !ok {
		return false
	}
	out := make([]Plugin, 0, len(current.entries)-1)
	for _, e := range current.entries {
		if e.id == id {
			continue
		}
		out = append(out, e.plugin)
	}
	l.plugins.Store(newPluginSet(out))
	return true
}

// GetPlugin returns the registered plugin with the given ID, or (nil, false)
// if no plugin with that ID is registered.
func (l *LogLayer) GetPlugin(id string) (Plugin, bool) {
	set := l.loadPlugins()
	if i, ok := set.byID[id]; ok {
		return set.entries[i].plugin, true
	}
	return nil, false
}

// PluginCount returns the number of plugins currently registered.
func (l *LogLayer) PluginCount() int {
	return len(l.loadPlugins().entries)
}
