# Benchmarking Rules

## Where benchmarks live

`bench_test.go` at the repo root. It contains four groups, each prefixed for easy filtering:

- `BenchmarkDirect_*`: underlying library used without LogLayer (baseline).
- `BenchmarkWrapped_*`: LogLayer wrapping the same library via the wrapper transport.
- `BenchmarkRender_*`: LogLayer with a self-contained renderer transport.
- `BenchmarkLoglayer_*`: LogLayer with a no-op transport, no I/O. Pure overhead.

Run the full suite:

```sh
go test -bench=. -benchmem -run=^$ -benchtime=1s .
```

Filter to one library:

```sh
go test -bench='Zerolog' -benchmem -run=^$ .
```

## Use `discardWriter`, never `io.Discard`

`charmbracelet/log` detects `io.Discard` and skips its formatting pipeline. Using `io.Discard` understates its real cost by ~300x and silently invalidates the comparison. Every benchmark in this repo writes to the package-local `discardWriter` (a no-op `io.Writer` that returns the byte count without writing anywhere) so every library exercises its full write path.

If you add a new wrapper transport, plumb its writer to `discard` (the package-level var), not `io.Discard`.

## What to measure

Each library/transport scenario has three flavors at minimum:

- **Simple message**: `log.Info("user logged in")`. The dominant log shape.
- **Map metadata**: `log.WithMetadata(loglayer.Metadata{"id":42, "name":"Alice", "email":"..."}).Info(...)`. Three fields is the standard.
- **Struct metadata**: `log.WithMetadata(benchTestUser).Info(...)`. Tests the pass-through path.

Use the shared `runSimple`, `runMap`, `runStruct` helpers in `bench_test.go` so all transports measure the same payload.

For the core (`BenchmarkLoglayer_*`) also measure `WithFields` and `WithError` since these exercise different code paths.

## Verifying perf changes

When making a change you think improves perf:

1. Capture baseline numbers before the change (run with `-count=10`).
2. Apply the change.
3. Capture new numbers (`-count=10`).
4. Compare with `benchstat` (don't eyeball; bench noise is real):

   ```sh
   go install golang.org/x/perf/cmd/benchstat@latest
   go test -bench=. -benchmem -run=^$ -count=10 . > old.txt
   # ... apply change ...
   go test -bench=. -benchmem -run=^$ -count=10 . > new.txt
   benchstat old.txt new.txt
   ```

5. If a change makes some benchmarks faster and others slower, **explicitly justify the trade-off** in the PR or revert. Default action is revert.

## Don't fight the compiler

Before adding sync.Pool, manual escape-analysis tricks, or other allocation-elimination strategies, check what the Go compiler is already doing:

```sh
go build -gcflags='-m' . 2>&1 | grep -E "moved to heap|escapes to heap|can inline"
```

Recent Go versions (1.22+) inline aggressively and stack-allocate small structs that don't escape. Before believing your "alloc savings" is real:

- Measure baseline allocs with `-benchmem`.
- Make the change.
- Measure new allocs.
- If allocs/op didn't decrease, the change is overhead with no benefit. Revert.

The performance attempt log in `AGENTS.md` records what's been tried and rejected. Read it before reinventing.

## Adding a new benchmark

When adding a new transport or scenario:

1. Add a `BenchmarkDirect_<Library>_*` if there's a baseline to compare against.
2. Add a `BenchmarkWrapped_<Library>_*` for the wrapper transport.
3. Use the shared `runSimple` / `runMap` / `runStruct` helpers; don't re-roll the loop.
4. Re-run the full suite and update `docs/src/benchmarks.md` with the new numbers and a row in the appropriate table.
5. Note the hardware + Go version in your commit if absolute numbers shift meaningfully.

## What not to benchmark

- **Don't benchmark concurrency throughput.** The point of these benches is per-call cost. Concurrent throughput is dominated by the writer (real I/O), not LogLayer.
- **Don't benchmark with real file/network I/O.** That's the underlying writer's cost, not ours. Use `discardWriter`.
- **Don't add micro-benchmarks for trivial helpers** (e.g. `joinMessages`). They're a maintenance burden and noise-prone.
