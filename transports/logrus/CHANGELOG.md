# Changelog

All notable changes to `go.loglayer.dev/transports/logrus` are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [SemVer](https://semver.org/spec/v2.0.0.html).
Tags use the prefixed form `transports/logrus/v<X.Y.Z>` so this module
versions independently of the framework core. Maintained
automatically by [monorel](https://monorel.disaresta.com) from
`.changeset/*.md` files.

## [2.0.0] - 2026-05-02

### Major Changes

- **Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

  See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.

## [1.6.1](https://github.com/loglayer/loglayer-go/compare/transports/logrus/v1.3.0...transports/logrus/v1.6.1) (2026-04-30)


### Miscellaneous Chores

* Bump every module touched by [#28](https://github.com/loglayer/loglayer-go/issues/28) to refresh pkg.go.dev ([#30](https://github.com/loglayer/loglayer-go/issues/30)) ([2ac74a7](https://github.com/loglayer/loglayer-go/commit/2ac74a7a58947f1d4c1c18ff5998b8042b6d1a59))

## [1.3.0](https://github.com/loglayer/loglayer-go/compare/transports/logrus/v1.2.0...transports/logrus/v1.3.0) (2026-04-30)


### Features

* Surface assembly Schema to transports and plugins, add Sentry transport ([#24](https://github.com/loglayer/loglayer-go/issues/24)) ([d35a0d5](https://github.com/loglayer/loglayer-go/commit/d35a0d5146e704d92f65fb208b17daaa4d151891))

## [1.2.0](https://github.com/loglayer/loglayer-go/compare/transports/logrus/v1.1.0...transports/logrus/v1.2.0) (2026-04-29)


### Features

* Cover Trace and Panic levels across transports and contract tests ([d511c7e](https://github.com/loglayer/loglayer-go/commit/d511c7eac484aadc3d3155876e497381b38f75e0))

## [1.1.0](https://github.com/loglayer/loglayer-go/compare/transports/logrus/v1.0.0...transports/logrus/v1.1.0) (2026-04-29)

No code changes in this module. v1.1.0 was a coordinated tag created alongside the main `go.loglayer.dev` v1.1.0 release; see the [main CHANGELOG](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the post-launch reshuffle that drove the bump.

## 1.0.0 (2026-04-29)


### ⚠ BREAKING CHANGES

* split pretty/http/datadog into modules; per-module README/CHANGELOG
* split heavy-dep wrapper transports into sub-modules
* drop Trace level, swap to goccy/go-json, rewrite structured transport

### Features

* Drop Trace level, swap to goccy/go-json, rewrite structured transport ([adb7114](https://github.com/loglayer/loglayer-go/commit/adb7114871931074b1e0c2b37de5466adf0e3ddc))
* HTTP/Datadog transports, dev tooling, structure cleanup ([feaacc9](https://github.com/loglayer/loglayer-go/commit/feaacc9cae99581c836edafe70bd5738fdbc2e06))
* Initial v0.1.0 implementation ([bca353c](https://github.com/loglayer/loglayer-go/commit/bca353cf965645d46360cfb3b6f5e43bb4e135d0))
* Optional auto-generated IDs for plugins and transports ([b6fc05a](https://github.com/loglayer/loglayer-go/commit/b6fc05a6c2d4a03359035c7f0aabbf9f77fca8d3))
* Split heavy-dep wrapper transports into sub-modules ([aef6053](https://github.com/loglayer/loglayer-go/commit/aef6053a1e80c646eb2c666fd0f893b98a66e883))
* Split pretty/http/datadog into modules; per-module README/CHANGELOG ([852d363](https://github.com/loglayer/loglayer-go/commit/852d363829eb622032bf5608010f24755bba2c7c))
* Trace + Panic levels, stdlib log bridge, sampling plugins, error chain serializer ([78485ec](https://github.com/loglayer/loglayer-go/commit/78485ec55305d03a48f72ed89f8475471528659c))

## [Unreleased] (target: v1.0.0)

Initial release as a separate Go module.

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main/transports/logrus
