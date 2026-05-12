---
"transports/newrelic": minor
---

Align encoder with TypeScript transport format: emit `timestamp`, `level`, `log`, and `attributes` instead of the previous flat `logtype`/`loglevel`/`message` shape. Adds attribute validation at encode time (max 255 attributes, 255-char names, 4,094-char string values).
