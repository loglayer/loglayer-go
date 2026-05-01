#!/usr/bin/env node
// Lints commit messages with @conventional-commits/parser. Catches
// body-parse errors that type-prefix-only checks (and commitlint's
// default parser) miss — typo'd types, malformed scopes, and bodies
// the parser can't handle (e.g. backtick-wrapped Go signatures with
// parens). Pure git-history hygiene; releases are driven by
// .changeset/*.md files, not commit messages.
//
// Usage:
//   node scripts/lint-commit.mjs --range FROM..TO
//        Lints every commit in the git range.
//        Used by CI and pre-push.
//
//   node scripts/lint-commit.mjs --edit FILE
//        Lints a single message file (the .git/COMMIT_EDITMSG
//        path passed by lefthook to the commit-msg hook).

import { parser } from "@conventional-commits/parser";
import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";

const args = process.argv.slice(2);
const mode = args[0];
const value = args[1];

if ((mode !== "--range" && mode !== "--edit") || !value) {
  console.error("usage: lint-commit.mjs --range FROM..TO");
  console.error("       lint-commit.mjs --edit FILE");
  process.exit(2);
}

function checkOne(message, label) {
  // Commit-msg hooks see the raw editor content including comment lines
  // (`# Please enter the commit message...`). Strip them before parsing
  // so the parser doesn't trip on the leading `#`.
  const cleaned = message
    .split("\n")
    .filter((line) => !line.startsWith("#"))
    .join("\n")
    .trim();
  if (!cleaned) {
    return true;
  }
  try {
    parser(cleaned);
    return true;
  } catch (e) {
    console.error(`✗ ${label}: ${e.message}`);
    console.error("---");
    console.error(cleaned);
    console.error("---");
    return false;
  }
}

let failed = 0;

if (mode === "--edit") {
  const message = readFileSync(value, "utf8");
  if (!checkOne(message, value)) failed++;
} else {
  const range = value;
  const sep = "--END-OF-COMMIT--";
  const raw = execSync(`git log --format=%H%n%B%n${sep} ${range}`, {
    encoding: "utf8",
  });
  const commits = raw
    .split(`\n${sep}\n`)
    .map((s) => s.trim())
    .filter(Boolean);
  for (const commit of commits) {
    const nl = commit.indexOf("\n");
    const sha = nl === -1 ? commit : commit.slice(0, nl);
    const message = nl === -1 ? "" : commit.slice(nl + 1);
    if (!message.trim()) continue;
    if (!checkOne(message, sha.slice(0, 7))) failed++;
  }
}

if (failed > 0) {
  console.error(
    `\n${failed} commit(s) failed conventional-commits parse.\n` +
      `\n` +
      `Conventional-commits format is required for git-history hygiene.\n` +
      `(Releases are driven by .changeset/*.md files, not commit messages.)\n` +
      `\n` +
      `Common cause: backtick-wrapped function signatures in the commit body\n` +
      `containing parens, e.g.\n` +
      `\n` +
      `    \`fn(arg)\`  ← parser-unsafe inside backticks early in the body\n` +
      `    fn(arg)    ← OK\n`
  );
  process.exit(1);
}
console.error("✓ all commits parsed cleanly");
