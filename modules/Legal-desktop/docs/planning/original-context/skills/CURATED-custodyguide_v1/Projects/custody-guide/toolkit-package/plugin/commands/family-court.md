---
description: "Route a Genesee family-court task to the toolkit"
argument-hint: "[issue or goal]"
allowed-tools: [Read, Grep, Glob, Bash]
disable-model-invocation: false
---

Start the `toolkit` skill workflow. Confirm jurisdiction, safety, currentness, and the requested outcome before reading the preserved baseline at `${CLAUDE_PLUGIN_ROOT}/skills/toolkit/content/toolkit/`. Use `$ARGUMENTS` as the task framing.
