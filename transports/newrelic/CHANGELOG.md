# Changelog

## [1.1.0] - 2026-05-12

### Minor Changes

- Align encoder with TypeScript transport format: emit `timestamp`, `level`, `log`, and `attributes` instead of the previous flat `logtype`/`loglevel`/`message` shape. Adds attribute validation at encode time (max 255 attributes, 255-char names, 4,094-char string values).

## [1.0.0] - 2026-05-11

### Major Changes

- Initial release.

