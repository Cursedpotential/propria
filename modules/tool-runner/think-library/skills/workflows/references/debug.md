# Workflow: debug

> **Integrated execution:** Treat the named thinking models below as stages/lenses inside this one workflow. Start each supporting lens at Micro unless the step needs more depth; escalate that same model to Focused/Full only when warranted. Feed its result into the next stage. Do not emit separate model reports or abandon the workflow. Keep Reflex active throughout.


> _Byline: Claude Code · Fable 5.1 · 2026-09-07._

## Purpose

Turn "why does it" into a root-cause note that keeps the disproven hypotheses visible, instead of
a single confident-sounding explanation that happens to be the first one that came to mind. The
value is in showing what was ruled out and how, not just naming a winner.

## When it fires

Trigger phrases: bug, regression, "why does it", "it works on my machine," intermittent/flaky
behavior, an error with an unclear cause.

## Roles

Single actor. No author/judge split.

## Steps

1. **Scientific method.** Load
   `../reason/references/thinking-scientific-method/SKILL.md`.
   Propose **at least two** competing hypotheses for the cause, each with a specific, falsifiable
   prediction, then test each prediction and record the actual result — not just "I think it's X."

2. **Five-whys-plus.** Load
   `../diagnose/references/thinking-five-whys-plus/SKILL.md`. Drill
   from the surviving hypothesis down to a root cause, applying the bias guards (stop at a cause
   that's actually actionable and evidenced, don't stop at "human error" or keep drilling past the
   point where you have evidence).

3. **Occam's razor.** Load
   `../reason/references/thinking-occams-razor/SKILL.md`. Among
   hypotheses that survived step 1's testing, prefer the one with fewer additional assumptions —
   state explicitly why the simpler surviving explanation beats any more elaborate one still on
   the table.

4. **Map-vs-territory check.** Load
   `../debias/references/thinking-map-territory/SKILL.md`. Check the
   mental model of the system implied by the root cause against what was actually observed (logs,
   traces, a repro) — not against how the system is supposed to work on paper.

5. **Fix + regression test.** State the fix and write (or fully describe, if this session can't
   execute it) a regression test that would have caught the bug before it shipped and will catch a
   recurrence.

## GATE

Invalid — say so explicitly — if:

- Only one hypothesis was ever considered (no real falsification attempt happened; "I'm sure it's
  X" without a competing hypothesis and a test is not the scientific method step, it's a guess with
  a citation).
- Disproven hypotheses are omitted from the output (they're required — they're what makes this
  different from an agent just asserting the answer).
- No regression test is named, and no reason is given for why one can't be written.
- The root cause given in step 2 isn't checked against real observed evidence in step 4 (a root
  cause that only ever lived in the model of the system, never checked against a log or a repro,
  fails this gate).

## OUTPUT TEMPLATE

```markdown
# Debug: <the symptom>

## Symptom
<what's actually observed — error message, behavior, frequency>

## Hypotheses Tested
| # | Hypothesis | Prediction if true | Test | Result | Verdict |
|---|---|---|---|---|---|
| 1 | <hypothesis> | <what we'd see> | <how tested> | <what happened> | survived / disproven |
| 2 | <hypothesis> | <what we'd see> | <how tested> | <what happened> | survived / disproven |

## Five Whys Chain
1. Why does <symptom>? Because <cause 1>.
2. Why <cause 1>? Because <cause 2>.
3. Why <cause 2>? Because <cause 3>.
(stop when the cause is actionable and evidenced, not merely "no more whys occur to me")

## Simplest Surviving Cause (Occam)
<the surviving hypothesis, and why it beats other still-plausible ones on assumption count>

## Map-vs-Territory Check
Model implied by the root cause: <what we'd expect to see in logs/traces if this is right>
Actually observed: <the real log/trace/repro evidence>
Match? <yes/no, and what that means>

## Fix
<the fix>

## Regression Test
<the test that would have caught this, or a stated reason none can be written>
```

## Worked example (synthetic)

**Symptom:** file-upload endpoint intermittently returns HTTP 500, roughly 1 in 200 requests,
only under concurrent load.

```markdown
# Debug: intermittent 500 on POST /uploads under load

## Symptom
~0.5% of concurrent upload requests return 500 with "ENOENT: no such file or directory" reading a
temp file; never reproduces with a single sequential request.

## Hypotheses Tested
| # | Hypothesis | Prediction if true | Test | Result | Verdict |
|---|---|---|---|---|---|
| 1 | Disk is intermittently full, write fails silently | Free-space graph shows dips near failures | Correlated failure timestamps against disk-usage metric | No correlation; disk stayed <40% used throughout | disproven |
| 2 | A background cleanup job deletes temp files before the upload handler finishes reading them | Failures cluster around the cleanup job's cron tick | Ran 500 concurrent uploads with the cleanup job's schedule logged alongside | 41 of 43 failures fell within 2s of a cleanup tick; 2 didn't | survived (mostly) |

## Five Whys Chain
1. Why does the upload sometimes 500? Because the temp file is missing when the handler reads it.
2. Why is the temp file missing? Because the cleanup job deleted it.
3. Why did the cleanup job delete an in-flight file? Because it selects "temp files older than 60s"
   by file mtime, and a slow client upload can leave a temp file sitting for >60s while still
   in-flight.
4. Why does mtime read as >60s for a file still being actively written? Because mtime updates on
   the *last write*, and a paused/throttled client connection can leave >60s between writes to the
   same temp file even though the upload is still active, not abandoned.

## Simplest Surviving Cause (Occam)
The cleanup job's staleness check uses mtime instead of tracking "upload in progress" state. This
explains 41/43 failures with one assumption (slow uploads exist); the alternative (a race in the
storage driver itself) would need a second, unevidenced assumption to explain the same data, so it
loses to the mtime explanation without further evidence for it.

## Map-vs-Territory Check
Model implied by the root cause: failed uploads should show a temp file whose *creation* time is
well within the cleanup window, but whose *last-write* gap exceeds 60s, deleted mid-upload.
Actually observed: pulled the cleanup job's deletion log for the failure window — every failed
upload's temp file was created 61-94s before deletion, consistent with a slow/throttled client, not
an abandoned upload.
Match? Yes — the observed deletion log matches the model. The remaining 2/43 failures (not
explained by this cause) are logged as a separate, still-open hypothesis, not folded into this fix.

## Fix
Track in-progress uploads by handle/lock rather than mtime; cleanup job skips any temp file with an
active lock regardless of mtime.

## Regression Test
Simulate a throttled client that pauses >60s mid-upload while the cleanup job runs on its normal
schedule; assert the upload completes successfully and the temp file is not deleted mid-write.
```
