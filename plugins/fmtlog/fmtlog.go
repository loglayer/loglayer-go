// Package fmtlog adds Sprintf-style log messages as a LogLayer plugin.
// Register it once and every call site that passes a format string
// followed by arguments gets fmt.Sprintf semantics:
//
//	log.AddPlugin(fmtlog.New())
//
//	log.Info("user %d signed in", userID)
//	log.WithMetadata(loglayer.Metadata{"reqId": id}).
//	    Error("request %s failed: %v", id, err)
//
// Without the plugin, multi-argument calls are space-joined
// (`fmt.Sprintf("%v", arg)` per element). Registering [New] opts the
// logger into format-string semantics: any call where the first
// message is a string and there are extra arguments is rewritten to
// fmt.Sprintf(messages[0], messages[1:]...) before downstream
// MessageHooks run.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package fmtlog

import (
	"fmt"

	"go.loglayer.dev/v2"
)

// New returns a plugin that resolves multi-argument log messages via
// fmt.Sprintf. The plugin is a single MessageHook: zero hot-path cost
// when a call has only one message; one Sprintf when there are extras.
func New() loglayer.Plugin {
	return loglayer.NewMessageHook("fmtlog", apply)
}

func apply(p loglayer.BeforeMessageOutParams) []any {
	if len(p.Messages) < 2 {
		return p.Messages
	}
	format, ok := p.Messages[0].(string)
	if !ok {
		return p.Messages
	}
	return []any{fmt.Sprintf(format, p.Messages[1:]...)}
}
