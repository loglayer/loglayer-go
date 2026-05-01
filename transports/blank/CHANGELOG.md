# Changelog

## [1.6.2] - 2026-05-01

### Patch Changes

- Smoke test of the monorel migration. No functional change; this changeset
  exists only to exercise the full release pipeline (release-pr orchestrator →
  release.yml → tag push → publish → deploy-docs workflow_call) end-to-end
  on `transports/blank`, the lowest-blast-radius package in the repo.

  After this lands, `transports/blank/v1.6.2` is the first monorel-driven tag
  in this repo. Future releases follow the same path; this one verifies the
  chain works.

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/transports/blank/v1.1.0...transports/blank/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/blank/v1.0.0...transports/blank/v1.1.0) (2026-04-29)

No code changes in this module. v1.1.0 was a coordinated tag created alongside the main `go.loglayer.dev` v1.1.0 release; see the [main CHANGELOG](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the post-launch reshuffle that drove the bump.
