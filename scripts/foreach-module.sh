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

# Internal entrypoint: run `go test` for one module and print its output
# atomically. Used by the parallelized `test` op so concurrent module
# output doesn't interleave. Not part of the public op set.
if [ "${1:-}" = "__test_one" ]; then
  shift
  mod="$1"
  out=$(cd "$mod" && go test -race -count=1 ./... 2>&1) && rc=0 || rc=$?
  if [ "$rc" -eq 0 ]; then
    printf '==> %s (test)\n%s\n' "$mod" "$out"
  else
    printf '==> %s (test) FAILED (rc=%d)\n%s\n' "$mod" "$rc" "$out" >&2
  fi
  exit "$rc"
fi

# All Go modules in the repo. Order is intentional: main first, then
# sub-modules. Don't reorder without a reason.
ALL_MODULES=(
  .
  transports/axiom
  transports/blank
  transports/betterstack
  transports/central
  transports/charmlog
  transports/cli
  transports/console
  transports/datadog
  transports/gcplogging
  transports/http
  transports/lumberjack
  transports/logrus
  transports/newrelic
  transports/otellog
  transports/phuslu
  transports/pretty
  transports/sentry
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
  examples/pretty-modes
)

# Modules that ship as importable code (skip the example/livetest-only
# modules). Used by lint/vuln/staticcheck where the looser conventions
# of example code shouldn't gate the build.
SHIPPED_MODULES=(
  .
  transports/axiom
  transports/blank
  transports/betterstack
  transports/central
  transports/charmlog
  transports/cli
  transports/console
  transports/datadog
  transports/gcplogging
  transports/http
  transports/lumberjack
  transports/logrus
  transports/newrelic
  transports/otellog
  transports/phuslu
  transports/pretty
  transports/sentry
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

# Core-only mode: restrict every op to the root module. Used during a
# major-version transition of the root (e.g. core moves to v3 while
# sub-modules are still on v2), when sub-modules cannot build against
# the unpublished root. Set CORE_ONLY=1 in CI for such PRs; unset once
# the sweep PR moves the sub-modules onto the new root version.
# TEST_MODULES is defined in the test branch (not here), so that branch
# applies the CORE_ONLY override itself.
if [ "${CORE_ONLY:-}" = "1" ]; then
  ALL_MODULES=(.)
  SHIPPED_MODULES=(.)
fi

op="${1:-}"
if [ -z "$op" ]; then
  cat >&2 <<USAGE_EOF
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
USAGE_EOF
  exit 2
fi

case "$op" in
  tidy)
    # Run `go mod tidy` across every module first so a single invocation
    # cleans up *all* drift, then do one repo-wide diff check at the end.
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
    TEST_MODULES=(
      .
      transports/axiom
      transports/blank
      transports/betterstack
      transports/central
      transports/charmlog
      transports/cli
      transports/console
      transports/datadog
      transports/gcplogging
      transports/http
      transports/lumberjack
      transports/logrus
      transports/newrelic
      transports/otellog
      transports/phuslu
      transports/pretty
      transports/sentry
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
    )
    # TEST_MODULES lives in this branch (not at the top with the other
    # lists), so the CORE_ONLY override must be applied here as well.
    if [ "${CORE_ONLY:-}" = "1" ]; then
      TEST_MODULES=(.)
    fi
    # Run modules concurrently. Each `go test ./...` already parallelizes
    # within a module; this layer parallelizes across modules so the
    # 27-module sequence stops being a 27x process-startup tax. Output
    # from each module is buffered by `__test_one` and printed atomically
    # so the lines from different modules don't interleave.
    #
    # Override with PARALLEL=1 to force serial (helpful when debugging a
    # specific module's output). getconf works on both Linux and macOS;
    # falls back to 4 if neither is available.
    PARALLEL="${PARALLEL:-$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)}"
    if [ "$PARALLEL" -le 1 ]; then
      for mod in "${TEST_MODULES[@]}"; do
        "$0" __test_one "$mod"
      done
    else
      printf '%s\n' "${TEST_MODULES[@]}" |
        xargs -n1 -P "$PARALLEL" -I{} "$0" __test_one {}
    fi
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
    # plugins/datadogtrace/livetest is a separate test module that
    # imports the v2-core-dependent plugin; skip it in core-only mode
    # for the same reason the shipped modules are collapsed.
    if [ "${CORE_ONLY:-}" = "1" ]; then
      MODS=("${SHIPPED_MODULES[@]}")
    else
      MODS=("${SHIPPED_MODULES[@]}" plugins/datadogtrace/livetest)
    fi
    for mod in "${MODS[@]}"; do
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
    if [ "${CORE_ONLY:-}" = "1" ]; then
      MODS=("${SHIPPED_MODULES[@]}")
    else
      MODS=("${SHIPPED_MODULES[@]}" plugins/datadogtrace/livetest)
    fi
    for mod in "${MODS[@]}"; do
      echo "==> $mod (vuln)"
      (cd "$mod" && govulncheck ./...)
    done
    ;;
  *)
    echo "unknown op: $op" >&2
    exit 2
    ;;
esac
