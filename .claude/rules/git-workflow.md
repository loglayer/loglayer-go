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

GitHub's squash-merge takes the branch tip's tree and creates a commit with parent = current main. If a branch was based on stale main, files that were deleted on main but still exist on the branch get re-added by the squash-merge. monorel itself was bitten by this exact pattern: PR #30 deleted a `.changeset/*.md` file as part of the v0.7.0 release, then PR #31 (created from a stale local main) re-introduced the file via squash-merge, and the next release-pr cycle proposed to re-ship the same content as v0.8.0.

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

Before the commit that completes a feature, fix, or refactor (the kind of commit that ends up in a PR), invoke the `superpowers:requesting-code-review` skill OR the `/simplify` skill against the diff. Treat it as part of the commit ritual, like running tests.

Concretely:

```
# After implementing a task and confirming go build / go test pass:
/superpowers:requesting-code-review     # for substantive changes
# or
/simplify                                 # for cleanup-oriented passes
```

Then fix any Critical or Important findings before pushing. Minor findings are non-blocking but should be noted.

## Why

Code reviews catch things humans (including me) consistently miss:

- The `monorel doctor` PR went through a code review and the reviewer flagged `validate.Severity` vs `doctor.Severity` shape inconsistency, a misleading "match validate's envelope" JSON comment, and an `Options.ChangesetDir` field whose name didn't match its constraint. I would have shipped all three without the review.
- A second review on the same PR caught a stale `ChangesetDir` reference in the `.changeset/<name>.md` body (which would have ended up verbatim in the released CHANGELOG) and two em-dashes in GoDoc that violated the project's own style rule.
- A `/simplify` pass on the same PR found three duplicated patterns worth extracting (dedup-and-sort helper, hardcoded check name literal, severity-label ternary).

Skipping reviews "because the change is small" is the most common way regressions ship. The review subagents are cheap; the cost of an issue caught after merge (broken release pipeline, orphaned changelog text, `--no-verify` band-aids) is high.

## How to apply

- **Mandatory** before the final commit on a feature branch.
- **Mandatory** before `gh pr create`.
- **Optional but valuable** mid-implementation when stuck or after a non-trivial refactor.
- **Skip** for trivial commits: typo fixes, single-line config tweaks, dependency bumps with a lockfile change. The bar is "would a reviewer add value here?". If yes, run it.

The reviewer subagent gets fresh context (no session history). Brief it explicitly with what you built, what it should do, and the SHA range to look at; the precision of the brief is the precision of the review.
