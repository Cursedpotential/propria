---
name: sequential-react-ship
description: "Run a development and deployment cycle as numbered sequential thoughts inside a ReAct (Thought-Action-Observation) loop, gated by reversibility, map-territory, and pre-mortem. Use when shipping, deploying, implementing a slice, running a dev/deploy cycle, HITL holds, BUILD_STATUS, or when asked to apply sequential thinking plus ReAct with the three thinking skills."
license: MIT
compatibility: "Works in Codex, Claude Code, and other agents that can load sibling skills or read bundled references."
metadata:
  author: Matthew Salem
  version: "1.0"
  composed_from:
    - thinking-reversibility
    - thinking-map-territory
    - thinking-pre-mortem
---

# Sequential ReAct Ship Cycle

A conductor skill. It does **not** replace the three thinking skills. It says **when** to load each one, then runs work as sequential thoughts inside a ReAct loop.

## When to Use This Skill

- Implementing or deploying a development slice
- Any ship / release / activation / rollback decision
- A long-running goal (for example a Swift MVP) that must keep moving without violating holds
- The user asks for sequential thinking, ReAct, or the reversibility + map-territory + pre-mortem trio

Do not use for pure research, writing, or one-shot Q&A with no code or deploy action.

## Companion skills (load, do not rewrite)

Load the live sibling if the host has it. Otherwise read the bundled copy.

| Checkpoint | Skill | Bundled copy | Job |
|---|---|---|---|
| Before choosing how to proceed | `thinking-reversibility` | [references/thinking-reversibility.md](references/thinking-reversibility.md) | Type 1 vs Type 2. Match process to door type. Prefer making Type 1 into Type 2 (pilot, abstraction, time-box, flag). |
| After every Observation, and before any status claim | `thinking-map-territory` | [references/thinking-map-territory.md](references/thinking-map-territory.md) | Maps (docs, ADRs, handoffs, metrics, green tests, mental models) are not territory (running code, live env, observed writes). Update the map or change navigation. Never insist the territory is wrong. |
| After a plan exists, before the first Act of a slice or any Type 1 door | `thinking-pre-mortem` | [references/thinking-pre-mortem.md](references/thinking-pre-mortem.md) | Failure already happened (past tense). Convert top reasons into tests, holds, or rollback steps. Worthless without plan updates. |

If a sibling skill is already loaded, follow **its** templates and checklists. This file only orchestrates.

---

## Sequential thinking (the spine)

Think in numbered thoughts. Do not skip. You may revise or branch.

```
Thought N / ~T:
  kind:    ORIENT | CLASSIFY | PREMORTEM | PLAN | REACT | REVISE | BRANCH | CLOSE
  status:  open | revised-by-M | branched-to-M | done
  claim:   one sentence
  evidence: map | territory | mixed | none
```

Rules:

1. Start with an estimate `T` (usually 6–12). Increase `T` if the slice is not closed.
2. One claim per thought. If you were wrong, emit `REVISE` that names the earlier thought. Do not silently overwrite it.
3. If two paths are live, `BRANCH` and say which branch is active.
4. Never mark `CLOSE` while a claim is still `map`-only and the work needed `territory`.
5. Show the thought stream to the user when a door type, hold, or BUILD_STATUS is at stake. Keep it short during routine Type 2 edits.

---

## ReAct (the motion)

Every implementation thought after CLASSIFY uses this triple:

```
Thought:  what you believe and why (one or two sentences)
Action:   the single next tool/command/edit (not a list of futures)
Observation: what actually came back (quote, exit code, file, count)
```

Rules:

- One Action per turn of the loop. Do not plan five Actions and skip Observation.
- Observation is territory. If you did not run or read it, you do not have an Observation — you have another map.
- After Observation, run the map-territory check in the next sequential thought before the next Action.
- If Observation falsifies the Thought, `REVISE` and re-CLASSIFY if the door type changed.
- Stop the inner loop when the slice gate is met or a Type 1 hold is hit. Do not idle: CLOSE this slice and open the next Type 2 slice.

---

## The cycle

```
ORIENT ──► CLASSIFY (reversibility) ──► PRE-MORTEM ──► ReAct loop
                                                        │
                                                        ▼
                                              map-territory after each Observation
                                                        │
                              Type 2 fail? ──► REVISE, retry or reverse
                              Type 1 hold? ──► HITL packet, next Type 2 slice
                              Gate met?    ──► CLOSE (territory evidence only)
```

### 1. ORIENT

List the maps you are about to trust (handoff, ADR, audit, compose file, `BUILD_STATUS`, owner message) and the territory you can actually inspect (repo, tests, live env, credentials). Name the slice in one sentence. Do not Act yet.

### 2. CLASSIFY — load `thinking-reversibility`

Produce, at minimum:

- Decision
- Type: `1` | `1.5` | `2`
- Why (technical / time / financial / reputation / dependency cost)
- Process: minutes / hours / days + who decides
- How to increase reversibility (pilot, abstraction, time-box, flag)
- Reversal plan

Then match behavior:

| Type | Do |
|---|---|
| 2 | Decide and enter ReAct. Do not write an ADR. |
| 1.5 | Short written rationale + peer/owner note. Pilot if possible. |
| 1 | Stop before Act. Write the owner/HITL packet. Continue a different Type 2 slice. |

Horizon defaults (override only with territory evidence):

- Type 2: local ingest port, lockfile pin, Workbench AI SDK, comments, contract tests, scratch DB, unused streaming generator wiring
- Type 1: live migrations `0026`–`0030`, Coolify/production writes, parked Surreal contact, corpus copy, Graphiti replacement, Semantica custody writes, deleting VIP trees

### 3. PRE-MORTEM — load `thinking-pre-mortem`

Required before the first Act of the slice and before any Type 1 door.

Past tense, 5–10 failure reasons, then top 3 with mitigation + owner + checkpoint. Fold mitigations into the plan (test, hold, or rollback). A pre-mortem with no plan change did not happen.

Horizon failure seeds (use only if they fit): empty `SBV_SERVICE_PASS`; Agno `ainsert` still the public contract; Chonkie not in the image; whole-file parser on the evidence lane; Workbench key missing; Surreal pane talking to the parked instance; `BUILD_STATUS=PASS` from config, not a run.

### 4. ReAct loop — map-territory after every Observation

After each Observation, load `thinking-map-territory` and answer:

1. Which map did I trust?
2. What did the territory show?
3. Freshness / drift?
4. What is still unmapped?
5. Calibrated confidence?

If map and territory disagree: update the map (debt register, handoff `BUILD_STATUS`, comment) **or** change navigation. Do not ship the map.

`BUILD_STATUS` is `UNKNOWN` unless the named checks were actually executed. Passing unit tests are a map of tested behavior, not of deployability.

### 5. CLOSE

Write, from territory:

- What was observed (commands, counts, files)
- Door type and whether it stayed Type 2
- Pre-mortem mitigations that landed
- Remaining holds
- Next Type 2 slice

Do not invent the next architecture wave.

---

## Output shape (keep this visible)

```markdown
## Slice
[one sentence]

## Sequential
- T1 ORIENT: ...
- T2 CLASSIFY: Type [1/1.5/2] because ...
- T3 PREMORTEM: it failed because ... → mitigations ...
- T4 REACT Thought/Action/Observation ...
- T5 MAP: trusted [map]; territory showed [fact]; status [aligned/drift]
- Tn CLOSE: ...

## Holds
[none | Type 1 packet path]

## BUILD_STATUS
[PASS | FAIL | UNKNOWN] — checks actually run: [...]
```

## Anti-patterns

- Re-implementing the three skills here instead of loading them
- Five planned Actions with no Observation
- Treating a green test suite or a handoff STATUS as a deploy
- Over-analyzing a Type 2 door (Bezos: this is how orgs stop inventing)
- Walking through a Type 1 door because the local path worked
- Closing a slice with only map evidence
- Idling on a hold instead of starting the next Type 2 slice
