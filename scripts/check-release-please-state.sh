#!/usr/bin/env bash
# Validate the consistency of .release-please-config.json and
# .release-please-manifest.json against the runbook in AGENTS.md
# (see "Release-please gotchas").
#
# Two invariants:
#
#   1. manifest "0.0.0" (never released) requires "release-as" on the
#      config block. Without it, release-please's full-history scan
#      can leak old `Release-As:` footers into the initial release.
#
#   2. "release-as" equal to the manifest version is stale: the
#      one-shot override has been applied and now pins every future
#      release at the same version. Pending overrides (release-as
#      higher than manifest) are legitimate mid-life uses and not
#      flagged.
#
#   3. "release-as" lower than the manifest version is dead config:
#      release-please will silently ignore a downgrade, so the config
#      contributes nothing. Most likely a stale override that wasn't
#      cleaned up after a normal release advanced past it.
#
# Also reports any package in one file but not the other.
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
  if [ -n "$release_as" ] && [ "$version" = "$release_as" ]; then
    err "$pkg: manifest is $version and config has \"release-as\": \"$release_as\". Remove the leftover \"release-as\" so subsequent releases use conventional commits. See AGENTS.md → Release-please gotchas."
  fi
  if [ -n "$release_as" ] && [ "$version" != "$release_as" ]; then
    lower="$(printf '%s\n%s\n' "$version" "$release_as" | sort -V | head -1)"
    if [ "$lower" = "$release_as" ]; then
      err "$pkg: manifest is $version but config has \"release-as\": \"$release_as\" (lower than manifest, dead config). release-please ignores downgrades; remove the stale override. See AGENTS.md → Release-please gotchas."
    fi
  fi
done <<<"$rows"

if [ "$errors" -gt 0 ]; then
  echo >&2
  echo "check-release-please-state: $errors invariant violation(s)" >&2
  exit 1
fi
