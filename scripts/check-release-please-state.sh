#!/usr/bin/env bash
# Validate the consistency of .release-please-config.json and
# .release-please-manifest.json against the runbook in AGENTS.md
# (see "Release-please gotchas").
#
# Two invariants:
#
#   1. A package whose manifest version is "0.0.0" (never released)
#      must have "release-as" set on its config block. Without this,
#      release-please's full-history scan can leak old `Release-As:`
#      footers into the package's initial release.
#
#   2. A package whose manifest version is non-zero (already released)
#      must NOT have "release-as" set. `release-as` is a one-shot
#      override; if it's left in place after the first tag, every
#      subsequent release of the package is forced back to that
#      version.
#
# This script also reports any package in one file but not the other.
#
# Hard-fails if jq isn't on PATH; install with the system package
# manager (apt: jq, brew: jq, etc.).

set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "check-release-please-state: jq not on PATH" >&2
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
config="$repo_root/.release-please-config.json"
manifest="$repo_root/.release-please-manifest.json"

if [ ! -f "$config" ]; then
  echo "check-release-please-state: $config not found" >&2
  exit 1
fi
if [ ! -f "$manifest" ]; then
  echo "check-release-please-state: $manifest not found" >&2
  exit 1
fi

errors=0
err() {
  echo "  ✗ $1" >&2
  errors=$((errors + 1))
}

# Single jq pass: emit one tab-separated row per package across the
# union of both files, with membership flags + manifest version +
# release-as. Avoids 50+ jq forks for ~25 packages.
rows="$(jq -r -n \
  --slurpfile c "$config" \
  --slurpfile m "$manifest" '
  ($c[0].packages // {}) as $cfg |
  ($m[0] // {}) as $man |
  ([$cfg | keys[]] + [$man | keys[]] | unique)[] as $pkg |
  [
    $pkg,
    (if $cfg[$pkg] then "1" else "0" end),
    (if $man[$pkg] != null then "1" else "0" end),
    ($man[$pkg] // ""),
    ($cfg[$pkg]["release-as"] // "")
  ] | @tsv
  ')"

while IFS=$'\t' read -r pkg in_config in_manifest version release_as; do
  if [ "$in_config" = "0" ]; then
    err "$pkg in .release-please-manifest.json but missing from .release-please-config.json"
    continue
  fi
  if [ "$in_manifest" = "0" ]; then
    err "$pkg in .release-please-config.json but missing from .release-please-manifest.json"
    continue
  fi
  if [ "$version" = "0.0.0" ] && [ -z "$release_as" ]; then
    err "$pkg: manifest is 0.0.0 (never released) but config has no \"release-as\". Add \"release-as\": \"1.0.0\" to force the initial version. See AGENTS.md → Release-please gotchas."
  fi
  if [ "$version" != "0.0.0" ] && [ -n "$release_as" ]; then
    err "$pkg: manifest is $version but config still has \"release-as\": \"$release_as\". Remove the leftover \"release-as\" so subsequent releases use conventional commits. See AGENTS.md → Release-please gotchas."
  fi
done <<<"$rows"

if [ "$errors" -gt 0 ]; then
  echo >&2
  echo "check-release-please-state: $errors invariant violation(s)" >&2
  exit 1
fi
