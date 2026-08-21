#!/usr/bin/env bash
# Pre-push / CI gate for vulnerabilities reachable from this repo's code.
#
# Policy: fail only on findings the repo can fix. govulncheck reports
# three classes:
#   - Standard library: fixed by upgrading the operating Go toolchain
#     (e.g. crypto/tls@go1.26 -> go1.26.6). The repo cannot fix these;
#     only the operator's `go` upgrade can. AGENTS.md documents this as
#     the operator's responsibility.
#   - Module / import (reachable): a dependency version we pin. This is
#     the repo's responsibility: bump the dependency (like the grpc
#     v1.79.3 -> v1.82.1 fix for GO-2026-6061 in transports/gcplogging).
#   - Module / import (unreachable): "doesn't appear to call" - not
#     reachable from code; accepted risk (same class as the advisory
#     SessionStart hook's noise floor).
#
# Exit codes:
#   0  nothing reachable-and-fixable found
#   1  govulncheck missing
#   2  a reachable NON-stdlib (dependency) vulnerability found
#
# Usage: bash scripts/govulncheck-gate.sh
# Env:  GOVULNCHECK_BIN  override the govulncheck binary path
set -uo pipefail

BIN="${GOVULNCHECK_BIN:-$(command -v govulncheck || true)}"
if [ -z "$BIN" ]; then
  echo "govulncheck not on PATH. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
  exit 1
fi

# Same module list CI uses (shipped modules + the livetest module).
MODULES=(. transports/otellog plugins/oteltrace plugins/datadogtrace/livetest \
  transports/blank transports/betterstack transports/charmlog transports/cli \
  transports/console transports/datadog transports/gcplogging transports/http \
  transports/logrus transports/lumberjack transports/newrelic \
  transports/phuslu transports/pretty transports/sentry transports/slog \
  transports/structured transports/testing transports/zap transports/zerolog \
  integrations/loghttp integrations/sloghandler plugins/fmtlog plugins/redact \
  plugins/sampling plugins/plugintest transports/central)

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

failures=0
for mod in "${MODULES[@]}"; do
  if ! (cd "$mod" && "$BIN" -scan=symbol ./...) >"$tmp" 2>&1; then
    # Parse reachable findings. A finding is fixable when its "Found in"
    # names a module other than the Go standard library (stdlib findings
    # read `Found in: <pkg>@go<version>`).
    fixable="$(awk '
      /^Vulnerability #/ { in_fixable = 0 }
      /Found in:/ {
        if ($0 !~ /@go[0-9]+\./) { in_fixable = 1 }
      }
      in_fixable && /^Vulnerability #/ { print $0 }
    ' "$tmp")"

    if [ -n "$fixable" ]; then
      echo "==> $mod (REACHABLE DEPENDENCY VULNERABILITY)"
      grep -E "Vulnerability #|Found in:|Fixed in:|More info:" "$tmp" | head -30
      failures=1
    else
      # Stdlib-only (or unreachable): advisory. Print a compact note so
      # the operator can see the toolchain upgrade path, but don't gate.
      stdlib="$(grep -cE '^Vulnerability #[0-9]+' "$tmp" || true)"
      if [ "$stdlib" -gt 0 ]; then
        echo "==> $mod (advisory: $stdlib stdlib/unreachable finding(s); upgrade Go toolchain to clear)"
      fi
    fi
  fi
done

if [ "$failures" -eq 1 ]; then
  echo
  echo "Reachable dependency vulnerabilities found. Bump the affected dependency"
  echo "in the module's go.mod (see Fixed in: lines above), or skip this push"
  echo "with --no-verify only if the finding is a false positive."
  exit 2
fi
exit 0
