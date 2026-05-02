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

GitHub's squash-merge takes the branch tip's tree and creates a commit with parent = current main. If a branch was based on stale main, files that were deleted on main but still exist on the branch get re-added by the squash-merge. The monorel project was bitten by this exact pattern: a release commit deleted a `.changeset/*.md` file, then a follow-up PR created from a stale local main re-introduced the file via squash-merge, and the next release-pr cycle proposed to re-ship the same content under a new version.

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
