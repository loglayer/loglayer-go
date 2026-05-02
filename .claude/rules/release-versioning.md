# Release Versioning Rules

## Initial release of a new package / module: ship at v1.0.0 by default

When the first release of a new package, sub-module, or integration is being cut, the changeset MUST use `:major` so the version is `v1.0.0`, NOT `:minor` (which produces `v0.1.0`).

```sh
# Correct: stamps the package as SemVer-committed from day one.
monorel add --package "<path>:major" --message "Initial release."

# Wrong (default judgment): signals pre-1.0 instability without an explicit reason.
monorel add --package "<path>:minor" --message "Initial release."
```

## Why

`v0.x.x` versions carry a specific signal in the Go ecosystem: the API may change in breaking ways without a major bump (per [SemVer §4](https://semver.org/#spec-item-4)). Consumers consequently treat `v0.x.x` as "use at your own risk." Defaulting to v0.x.x without a real reason offloads stability uncertainty onto every consumer of the module.

For most new packages in this monorepo, by the time a PR opens:
- The API has been designed deliberately (often through brainstorming + plan + code review).
- There are tests covering the public surface.
- A real consumer is teed up (or already migrated).
- The breaking-change risk is no higher than it would be for any future version.

In that environment, `v0.1.0` is just a hedge against doubt. The honest version is `v1.0.0`. If a breaking change ever comes, it ships as `v2.0.0` and the import path bumps to `<path>/v2/` (Go module convention); that's the SemVer cost of being wrong. It's not free, but it's also not catastrophic, and it's much smaller than the cost of leaving every package on `v0.x.x` perpetually because nobody wants to commit.

## When pre-1.0 IS the right call

Use `:minor` (→ `v0.1.0`) only when the package's API genuinely hasn't been validated yet, AND that lack of validation is the reason to ship pre-1.0. Concrete signals:

- The package is exposing an interface for third-party plugins to implement, and we haven't yet seen any third-party plugin try to implement it. (We may discover the seam needs to change after a real implementer hits it.)
- A dependency the package wraps is itself pre-1.0, and the wrapping shape is likely to change as the dep stabilizes.
- Internal experimentation: the package is being shipped in advance of a redesign that's already planned.

If you opt into `:minor`, **state the reason in the changeset body** so future maintainers (and the next reviewer) know what soak signal would unblock the v1.0.0 promotion. Example:

```markdown
---
"transports/foo": minor
---

Initial release. Pre-1.0 (`v0.1.0`) because the Encoder seam may change once
we've seen the second concrete implementation. Promote to v1.0.0 when at
least one external transport implements the interface successfully.
```

## How to apply

- **Mandatory** for every initial-release changeset on a new package, sub-module, or integration. Default to `:major`.
- **Mandatory** to state a soak signal in the changeset body when opting into `:minor`. "Initial release." alone is not sufficient justification for pre-1.0.
- **Skip** for packages explicitly designed for staged rollout where pre-1.0 is the documented plan (rare).

## Recovering from an accidental v0.1.0

If a package shipped at `v0.1.0` and the API is actually stable, the recovery is a follow-up `:major` changeset on the same package:

```sh
monorel add --package "<path>:major" --message "Promote to v1.0.0; API is stable."
```

When the release PR merges, monorel cuts `<path>/v1.0.0` directly from `v0.1.0` (skipping `v0.2.0` ... `v0.x.x`). The promotion is cheap when caught early.
