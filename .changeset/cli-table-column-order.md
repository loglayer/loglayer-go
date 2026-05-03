---
"transports/cli": minor
---

New `Config.TableColumnOrder []string` knob pins the leading column order for
slice-of-map metadata renderings. Keys named here render in the listed order;
remaining keys sort lexicographically and follow. Pinned keys absent from
every row are silently skipped, so the same `TableColumnOrder` is safe to
reuse across call sites with different row shapes. Empty / nil falls back to
the previous fully-lexicographic behavior. Resolves [#66](https://github.com/loglayer/loglayer-go/issues/66).
