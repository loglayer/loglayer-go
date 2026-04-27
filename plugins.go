package loglayer

// loadPlugins returns the current plugin snapshot. Hot path: called on every
// emission for hook lookup.
func (l *LogLayer) loadPlugins() *pluginSet {
	return l.plugins.Load()
}

// AddPlugin registers a plugin. If a plugin with the same ID already exists,
// it is replaced (matching the AddTransport convention). Plugin.ID is
// required.
//
// Safe to call from any goroutine: the plugin set is published atomically.
// Concurrent mutators on the same logger serialize via an internal mutex.
func (l *LogLayer) AddPlugin(p Plugin) *LogLayer {
	if p.ID == "" {
		panic("loglayer: Plugin.ID is required")
	}
	l.pluginMu.Lock()
	defer l.pluginMu.Unlock()

	current := l.loadPlugins().all
	out := make([]Plugin, 0, len(current)+1)
	for _, existing := range current {
		if existing.ID != p.ID {
			out = append(out, existing)
		}
	}
	out = append(out, p)
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

	current := l.loadPlugins().all
	out := make([]Plugin, 0, len(current))
	removed := false
	for _, p := range current {
		if p.ID == id {
			removed = true
			continue
		}
		out = append(out, p)
	}
	if removed {
		l.plugins.Store(newPluginSet(out))
	}
	return removed
}

// GetPlugin returns a copy of the registered plugin with the given ID, or
// (Plugin{}, false) if no plugin with that ID is registered. The returned
// value is a copy; mutating it does not affect the logger's state.
func (l *LogLayer) GetPlugin(id string) (Plugin, bool) {
	set := l.loadPlugins()
	if i, ok := set.byID[id]; ok {
		return set.all[i], true
	}
	return Plugin{}, false
}

// PluginCount returns the number of plugins currently registered.
func (l *LogLayer) PluginCount() int {
	return len(l.loadPlugins().all)
}
