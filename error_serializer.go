package loglayer

import "errors"

// UnwrappingErrorSerializer is an opt-in [ErrorSerializer] that walks the
// error chain and surfaces every wrapped cause as structured data.
//
// The default serializer (a flat `{"message": err.Error()}`) is the right
// choice when you only need the rendered string. Pick this one when you
// want to preserve the structure of `fmt.Errorf("...: %w", err)` chains
// or `errors.Join(...)` lists in the log output:
//
//	log := loglayer.New(loglayer.Config{
//	    Transport:       structured.New(structured.Config{}),
//	    ErrorSerializer: loglayer.UnwrappingErrorSerializer,
//	})
//	log.WithError(fmt.Errorf("op failed: %w", io.EOF)).Error("oops")
//	// {"err": {"message": "op failed: EOF", "causes": [{"message": "EOF"}]}}
//
// Behavior:
//
//   - The top-level message is `err.Error()` verbatim.
//   - For a single-chain error (`errors.Unwrap(err)` returns one), each
//     unwrap step appends one `{"message": ...}` object to `causes`.
//     The walk stops at the first nil unwrap.
//   - For an `errors.Join` value (`Unwrap() []error`), each member
//     becomes one `{"message": ...}` object in `causes`. Members are
//     not recursively walked, so nested Joined+wrapped errors flatten
//     to one level. If a Join member has its own chain you can recurse
//     by writing your own serializer that calls this one per member.
//   - `causes` is omitted when there are no wrapped errors below the
//     top frame, keeping the JSON shape identical to the default
//     serializer for unwrapped errors.
//
// Returns nil for a nil error (which the dispatch path treats as "no
// err key in the output", matching the default serializer's contract).
func UnwrappingErrorSerializer(err error) map[string]any {
	if err == nil {
		return nil
	}
	out := map[string]any{"message": err.Error()}
	causes := unwrapCauses(err)
	if len(causes) > 0 {
		out["causes"] = causes
	}
	return out
}

func unwrapCauses(err error) []map[string]any {
	// errors.Join: Unwrap returns the joined slice. Don't also walk
	// errors.Unwrap on the same value, which would yield nil for a
	// Join (it has no single chain).
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		members := multi.Unwrap()
		causes := make([]map[string]any, 0, len(members))
		for _, e := range members {
			if e == nil {
				continue
			}
			causes = append(causes, map[string]any{"message": e.Error()})
		}
		return causes
	}

	var causes []map[string]any
	for inner := errors.Unwrap(err); inner != nil; inner = errors.Unwrap(inner) {
		causes = append(causes, map[string]any{"message": inner.Error()})
	}
	return causes
}
