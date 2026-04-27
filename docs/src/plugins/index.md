---
title: Plugins
description: Hook into the LogLayer pipeline to transform metadata, fields, data, messages, log level, or per-transport dispatch.
---

# Plugins

Plugins extend LogLayer's emission pipeline. They run on every `*loglayer.LogLayer` they're registered on and apply to every emission until removed.

## Available plugins

<!--@include: ./_partials/plugin-list.md-->
