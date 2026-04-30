#!/usr/bin/env bash
# Run a check across every Go module in the repo. Used by CI; also
# runnable locally so devs can verify before pushing.
#
# Usage: scripts/foreach-module.sh <op>
#   op: tidy | vet | fmt | staticcheck | vuln | test | build
#
# The module list is the source of truth for "what counts as a module
# in this repo." Add new sub-modules here and CI / local tooling
# automatically picks them up.

set -euo pipefail

# All Go modules in the repo. Order is intentional: main first, then
# sub-modules. Don't reorder without a reason.
ALL_MODULES=(
  .
  transports/blank
  transports/central
  transports/charmlog
  transports/console
  transports/datadog
  transports/lumberjack
  transports/http
  transports/logrus
  transports/otellog
  transports/phuslu
  transports/pretty
  transports/slog
  transports/structured
  transports/testing
  transports/zap
  transports/zerolog
  plugins/datadogtrace
  plugins/datadogtrace/livetest
  plugins/fmtlog
  plugins/oteltrace
  plugins/plugintest
  plugins/redact
  plugins/sampling
  integrations/loghttp
  integrations/sloghandler
  examples/custom-plugin
  examples/datadog-shipping
  examples/http-server
  examples/multi-transport
  examples/otel-end-to-end
)

# Modules that ship as importable code (skip the example/livetest-only
# modules). Used by lint/vuln/staticcheck where the looser conventions
# of example code shouldn't gate the build.
SHIPPED_MODULES=(
  .
  transports/blank
  transports/central
  transports/charmlog
  transports/console
  transports/datadog
  transports/lumberjack
  transports/http
  transports/logrus
  transports/otellog
  transports/phuslu
  transports/pretty
  transports/slog
  transports/structured
  transports/testing
  transports/zap
  transports/zerolog
  plugins/datadogtrace
  plugins/fmtlog
  plugins/oteltrace
  plugins/plugintest
  plugins/redact
  plugins/sampling
  integrations/loghttp
  integrations/sloghandler
)

op="${1:-}"
if [ -z "$op" ]; then
  cat >&2 <<EOF
Usage: $0 <op>
  Ops:
    tidy         go mod tidy + diff check (all modules)
    vet          go vet ./...            (all modules)
    fmt          gofmt -l                (all modules)
    test         go test -race           (all modules with tests)
    build        go build ./...          (all modules)
    staticcheck  staticcheck ./...       (shipped modules only)
    vuln         govulncheck ./...       (shipped modules only)

The 'all modules' set is: ${ALL_MODULES[*]}
The 'shipped modules' set is: ${SHIPPED_MODULES[*]}
EOF
  exit 2
fi

case "$op" in
  tidy)
    # Run `go mod tidy` across every module first so a single invocation
    # cleans up *all* drift, then do one repo-wide diff check at the end.
    # The earlier per-module diff fail-fast pattern made this script a
    # poor pre-push hook: each run only ever found the first drifted
    # module, so multi-module drift took multiple iterations to surface.
    for mod in "${ALL_MODULES[@]}"; do
      echo "==> $mod (tidy)"
      (cd "$mod" && go mod tidy)
    done
    echo "==> diff check"
    if ! git diff --exit-code -- '*go.mod' '*go.sum'; then
      echo
      echo "go.mod / go.sum drift after tidy. Stage the changes above and commit." >&2
      exit 1
    fi
    ;;
  vet)
    for mod in "${ALL_MODULES[@]}"; do
      echo "==> $mod (vet)"
      (cd "$mod" && go vet ./...)
    done
    ;;
  fmt)
    failed=0
    for mod in "${ALL_MODULES[@]}"; do
      unformatted="$(cd "$mod" && gofmt -l .)"
      if [ -n "$unformatted" ]; then
        echo "Unformatted files in $mod:"
        echo "$unformatted"
        failed=1
      fi
    done
    if [ "$failed" -ne 0 ]; then
      echo
      echo "Run: gofmt -w <file>  (or 'goimports -w' if installed)"
      exit 1
    fi
    ;;
  test)
    # Example modules have no tests; skip to avoid the confusing
    # "[no test files]" output. Every other module has tests.
    for mod in \
      . \
      transports/blank \
      transports/central \
      transports/charmlog \
      transports/console \
      transports/datadog \
      transports/lumberjack \
      transports/http \
      transports/logrus \
      transports/otellog \
      transports/phuslu \
      transports/pretty \
      transports/slog \
      transports/structured \
      transports/testing \
      transports/zap \
      transports/zerolog \
      plugins/datadogtrace \
      plugins/datadogtrace/livetest \
      plugins/fmtlog \
      plugins/oteltrace \
      plugins/plugintest \
      plugins/redact \
      plugins/sampling \
      integrations/loghttp \
      integrations/sloghandler; do
      echo "==> $mod (test)"
      (cd "$mod" && go test -race -count=1 ./...)
    done
    ;;
  build)
    for mod in "${ALL_MODULES[@]}"; do
      echo "==> $mod (build)"
      (cd "$mod" && go build ./...)
    done
    ;;
  staticcheck)
    if ! command -v staticcheck >/dev/null 2>&1; then
      echo "staticcheck not found on PATH" >&2
      echo "Install: go install honnef.co/go/tools/cmd/staticcheck@latest" >&2
      exit 1
    fi
    for mod in "${SHIPPED_MODULES[@]}" plugins/datadogtrace/livetest; do
      echo "==> $mod (staticcheck)"
      (cd "$mod" && staticcheck ./...)
    done
    ;;
  vuln)
    if ! command -v govulncheck >/dev/null 2>&1; then
      echo "govulncheck not found on PATH" >&2
      echo "Install: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
      exit 1
    fi
    for mod in "${SHIPPED_MODULES[@]}" plugins/datadogtrace/livetest; do
      echo "==> $mod (vuln)"
      (cd "$mod" && govulncheck ./...)
    done
    ;;
  *)
    echo "unknown op: $op" >&2
    exit 2
    ;;
esac
