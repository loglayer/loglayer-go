#!/usr/bin/env bash
# Advisory govulncheck run intended for Claude Code (or other coding
# agent) hooks. Not a gate. Emits a compact summary on stdout if there
# are findings; emits nothing when there's nothing to report.
#
# Why advisory:
# - Stdlib vulns require the operator to upgrade their Go install;
#   the repo can't fix them.
# - Some dep vulns are in code we never call (govulncheck's reachability
#   analysis is conservative).
# - Some findings are documented as accepted risk.
#
# So the agent should *see* findings to reason about them, not be
# blocked by them. CI runs the same scan with stricter thresholds; that
# remains the gating channel.

set -uo pipefail

if ! command -v govulncheck >/dev/null 2>&1; then
  # Don't try to install during a hook. Surface the gap instead.
  cat <<'EOF'
[govulncheck] not installed; skipping advisory scan.
Install with: go install golang.org/x/vuln/cmd/govulncheck@latest
EOF
  exit 0
fi

ALL_MODULES=(. transports/otellog plugins/oteltrace plugins/datadogtrace/livetest)

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

any_finding=0
for mod in "${ALL_MODULES[@]}"; do
  if (cd "$mod" && govulncheck ./...) >"$tmp" 2>&1; then
    continue
  fi
  # Non-zero exit means findings exist or the scan errored. Surface
  # both cases under the same module heading.
  if [ "$any_finding" -eq 0 ]; then
    echo "[govulncheck] advisory findings (run \`scripts/foreach-module.sh vuln\` for full output):"
    echo
    any_finding=1
  fi
  echo "==> $mod"
  # Compact summary: count of vulns and a one-line summary per ID.
  # Pull vulnerability IDs and where they're "Found in".
  awk '
    /^Vulnerability #/ { id = ""; mod = ""; }
    /^Vulnerability #/ {
      # Strip the "Vulnerability #N: " prefix.
      sub(/^Vulnerability #[0-9]+: /, "")
      id = $0
    }
    /Found in:/ {
      sub(/^[[:space:]]*Found in: /, "")
      mod = $0
    }
    /Fixed in:/ {
      sub(/^[[:space:]]*Fixed in: /, "")
      if (id != "" && mod != "") {
        printf "  %-18s %s -> %s\n", id, mod, $0
        id = ""; mod = ""
      }
    }
  ' "$tmp"
  echo
done

if [ "$any_finding" -eq 0 ]; then
  # Stay silent when there are no findings; the agent doesn't need a
  # "scan clean" notification cluttering its session.
  exit 0
fi

cat <<'EOF'
Note: stdlib vulns (Standard library@goX.Y) are fixed by upgrading the
operator's Go install, not by code changes here. Dep vulns may be in
unreachable code paths. Use judgment — some findings warrant a fix,
others are accepted risk.
EOF
