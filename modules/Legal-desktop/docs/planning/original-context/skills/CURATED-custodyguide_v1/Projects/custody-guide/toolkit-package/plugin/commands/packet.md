---
description: "Build a tailored parenting-time filing set from assembled case facts, or list what is still missing"
argument-hint: "[filing goal, or 'intake']"
allowed-tools: [Read, Grep, Glob, Bash, Write, Edit]
disable-model-invocation: false
---

Use the custody-packet skill. Read its bundled guardrails and verification ledger first, run the
intake gate against the facts supplied, and either list the missing items once or produce the
tailored motion set, proposed order, concurrence certificate, notice of hearing, service pack, and
hearing prep. Write real-case output only to the user's own filing workspace, never into the
bundled drafts. Run the banned-vocabulary, weekday, and deadline sweeps before hand-back. Do not
file, do not label anything filing-ready, and do not provide legal advice.
