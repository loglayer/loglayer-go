---
"transports/blank": patch
---

Smoke test of the monorel migration. No functional change; this changeset
exists only to exercise the full release pipeline (release-pr orchestrator →
release.yml → tag push → publish → deploy-docs workflow_call) end-to-end
on `transports/blank`, the lowest-blast-radius package in the repo.

After this lands, `transports/blank/v1.6.2` is the first monorel-driven tag
in this repo. Future releases follow the same path; this one verifies the
chain works.
