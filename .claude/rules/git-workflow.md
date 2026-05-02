# Git Workflow Rules

## Always rebase on the parent branch before opening (or pushing to) a PR

Before `gh pr create`, always:

```sh
git fetch origin <parent>
git rebase origin/<parent>
```

Where `<parent>` is the branch the PR will target (typically `main`).

If the local branch is stacked on commits that have already merged via squash-merge (so the local history doesn't match remote), the right form is:

```sh
git rebase --onto origin/<parent> <merge-base-of-just-this-PR's-work> <branch>
```

Then `git push --force-with-lease` (never `--force`).

## Why

GitHub's squash-merge takes the branch tip's tree and creates a commit with parent = current main. If a branch was based on stale main, files that were deleted on main but still exist on the branch get re-added by the squash-merge. monorel itself was bitten by this exact pattern: `disaresta-org/monorel#30` deleted a `.changeset/*.md` file as part of the v0.7.0 release, then `disaresta-org/monorel#31` (created from a stale local main) re-introduced the file via squash-merge, and the next release-pr cycle proposed to re-ship the same content as v0.8.0.

The rule exists to make that class of bug impossible by construction. If the local branch is rebased onto current `origin/main` before the PR opens, the squash diff can never contain a file the most-recent release-cut commit deleted (because the branch already incorporated that deletion).

## How to apply

- **Always** before `gh pr create`. Treat it as part of the PR-creation ritual.
- **Always** before `git push --force-with-lease` to a long-lived branch where main has advanced. For a branch you've just created off freshly-pulled main and pushed once, you don't need to re-rebase before every push; the rule is "rebase when main has advanced."
- Use `git rebase --onto origin/<parent>` form when the local branch's lower commits are squash-merge ancestors that won't replay cleanly. Replaying just the work-in-this-PR's commits keeps the rebase trivial.
- After the rebase succeeds, run the build / tests once before pushing. A rebase can produce silent semantic conflicts that compile but break behavior.

## What this rule does NOT cover

- Force-pushing to `main` or other shared protected branches. Don't.
- Resolving genuine merge conflicts during the rebase. Read both sides; never `git checkout --theirs/ours` blindly.
- Skipping the rule with `--no-verify`. The rule is the rule even when the hook isn't enforcing it.

## Always run a code review before committing the work that finishes a task

Before the commit that completes a feature, fix, or refactor (the kind of commit that ends up in a PR), invoke a code-review subagent against the diff. Treat it as part of the commit ritual, like running tests.

Concretely (skill names current as of this writing; substitute today's equivalent if they've been renamed):

```
# After implementing a task and confirming go build / go test pass:
/superpowers:requesting-code-review     # for substantive changes
# or
/simplify                                 # for cleanup-oriented passes
```

Then fix any Critical or Important findings before pushing. Minor findings are non-blocking but should be noted.

## Why

Code reviews catch things humans (including me) consistently miss. Concrete cases from `disaresta-org/monorel#35` (the doctor-feature PR):

- First review flagged a `Severity` shape inconsistency between two sibling packages, a misleading JSON-envelope-parity comment, and an option field whose name didn't match its constraint. I would have shipped all three without the review.
- Second review caught a stale field reference in the released changeset body (which would have ended up verbatim in the auto-generated CHANGELOG) and two em-dashes in GoDoc that violated the project's own style rule.
- A `/simplify` pass on the same PR found three duplicated patterns worth extracting (dedup-and-sort helper, hardcoded check name literal, severity-label ternary).

Skipping reviews "because the change is small" is the most common way regressions ship. The review subagents are cheap; the cost of an issue caught after merge (broken release pipeline, orphaned changelog text, `--no-verify` band-aids) is high.

## How to apply

- **Mandatory** before the final commit on a feature branch.
- **Mandatory** before `gh pr create`.
- **Optional but valuable** mid-implementation when stuck or after a non-trivial refactor.
- **Skip** for trivial commits: typo fixes, single-line config tweaks, dependency bumps with a lockfile change. The bar is "would a reviewer add value here?". If yes, run it.

The reviewer subagent gets fresh context (no session history). Brief it explicitly with what you built, what it should do, and the SHA range to look at; the precision of the brief is the precision of the review.

## Run a documentation review when the change touches docs

If the diff includes any of:

- Markdown under `docs/src/` (the VitePress site)
- A new or modified `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, or `.claude/rules/*.md`
- GoDoc on a newly exported symbol (function, type, constant, variable)
- A new or modified `whats-new.md` / `CHANGELOG.md` entry
- `.changeset/<name>.md` body text (since it ends up in the auto-generated changelog verbatim)

dispatch a separate documentation review with the framing **"act as a senior Go developer encountering this material for the first time"**. The reviewer should evaluate the changed docs along three axes:

1. **Accuracy.** Do the examples compile? Do the referenced types, functions, fields, and config keys actually exist with the names and signatures shown? Do the claims match the code's current behavior?
2. **Consistency.** Does the new prose match the existing docs' tone, structure, terminology, and section conventions? If similar concepts are documented elsewhere, are they framed the same way? Are headings parallel to the rest of the page?
3. **Clarity.** Could a senior Go developer (familiar with the language and ecosystem but new to this codebase) follow the explanation on first read? Is the lead-with-the-conclusion principle followed? Are there hidden assumptions a newcomer wouldn't have?

A standard `superpowers:requesting-code-review` will inspect docs alongside code, but tends to spend its budget on the implementation and treat docs as window-dressing. A separately-framed documentation pass catches things the code-focused reviewer skips: stale field names in examples, copy-paste from earlier docs that no longer apply, terminology drift, hand-wavy steps, missing prerequisites, and broken cross-links.

### Why

Docs are load-bearing for a tool like monorel. Mistakes in CLI examples don't fail tests; they break a user's first-run experience and erode trust. The doctor PR's own review history showed two doc-side findings caught only by separate passes:

- A `.changeset/<name>.md` body example that referenced a renamed `Options` field. Without the second review, the broken snippet would have shipped verbatim into the auto-generated CHANGELOG entry.
- Em-dashes in GoDoc that violated the project's own style rule. The code-quality reviewer doesn't grep for style; only the documentation-focused pass surfaces those.

### How to apply

- **Mandatory** when the diff's docs surface area is non-trivial (a new doc page, a new GoDoc-exported symbol, a new `whats-new.md` entry, a `.changeset/<name>.md` longer than a few lines).
- **Optional but valuable** when modifying existing docs: the reviewer's "first encounter" framing reliably surfaces stale assumptions that the original author has long since internalized.
- **Skip** for pure-code PRs that touch docs only mechanically (e.g. bumping a version number in a changelog stub, fixing a typo).
- Brief the reviewer with: the changed file paths, the framing ("senior Go developer, first read"), and the three axes (accuracy / consistency / clarity). Don't tell it the answer; the review is most useful when the reviewer surfaces what surprises a fresh reader.
