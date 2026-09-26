---
description: "Verify currentness and traceability of cited sources"
argument-hint: "[source IDs or topic]"
allowed-tools: [Read, Grep, Glob, Bash]
disable-model-invocation: false
---

Use the toolkit skill. Read the ledger/audit limits and, if appropriate, run only bundled read-only verification scripts through `${CLAUDE_PLUGIN_ROOT}`. Report blocked/unreadable text as unverified rather than inferring a rule.
