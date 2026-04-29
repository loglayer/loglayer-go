---
title: Use with AI / LLMs
description: LogLayer for Go ships LLM-friendly reference files following the llmstxt.org convention.
---

# Use with AI / LLMs

LogLayer for Go ships two reference files designed to be pasted into a chat with Claude, ChatGPT, Cursor, Copilot Chat, or any other LLM-backed coding assistant. They follow the [llmstxt.org](https://llmstxt.org) convention.

| File | Purpose | Size |
|------|---------|------|
| **[/llms.txt](https://go.loglayer.dev/llms.txt)** | Concise index: every page on the docs site with a one-line description and link. Use this when you want the model to know the surface area without burning context on full prose. | small |
| **[/llms-full.txt](https://go.loglayer.dev/llms-full.txt)** | Comprehensive single-file reference: API surface, types, transports, integrations, thread-safety contract, and key patterns inlined. Use this when you want the model to answer detailed questions without browsing. | medium |

Both files are kept in sync with the docs site on every release.

## How to use them

### With Claude / ChatGPT (web)

Paste the contents of `llms-full.txt` (or its URL) at the start of your conversation:

> Use the loglayer-go reference at https://go.loglayer.dev/llms-full.txt as authoritative.
> Then write me an HTTP handler that derives a per-request logger using `integrations/loghttp` and logs the request ID, method, and path.

### With Cursor / Continue / similar IDE assistants

Add the URL as a context source. Most IDE assistants accept a URL or local file as a "doc source": drop in `https://go.loglayer.dev/llms-full.txt` and the model will reference it when answering loglayer-related questions.

### With Claude Code / Aider / SDK-based tools

Fetch the file once and include it as a system message:

```sh
curl -sSL https://go.loglayer.dev/llms-full.txt > .ai/loglayer.txt
```

Then add `.ai/loglayer.txt` to your tool's context-files list (or wire it into a custom system prompt).

## Sample prompts

Once the reference is loaded, these prompts produce useful output:

- *"Show me how to set up loglayer with the structured transport in production and the pretty transport in dev, switched by an env var."*
- *"Write a Datadog setup that batches every 2 seconds and pipes send errors to a metrics counter."*
- *"What's the right pattern for a per-request logger in an HTTP handler?"*
- *"Explain when to use WithFields vs WithMetadata vs WithContext."*
- *"Write an integration test using `transports/testing` that verifies my handler logs the request ID."*

## Why two files?

`llms.txt` is for cases where context is precious: it's a sitemap that lets the model fetch only what it needs. `llms-full.txt` is for cases where context is cheap (long-context models, agentic tools): everything is inlined so the model doesn't need to browse.

If you're not sure, start with `llms-full.txt`. It's the lower-friction option.

## Other ways to feed loglayer to an LLM

- **Source code** at [github.com/loglayer/loglayer-go](https://github.com/loglayer/loglayer-go) is small enough to fit in most coding assistants' context.
- **pkg.go.dev** (`pkg.go.dev/go.loglayer.dev`) renders all GoDoc, including type signatures and doc comments. Useful for fact-checking the model's output.
- **This docs site** is itself indexed by most search-augmented assistants. Asking "from the loglayer.dev Go docs, ..." often works without any setup.
