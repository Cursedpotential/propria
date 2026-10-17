# Workflow: evaluate

> **Integrated execution:** Treat the named thinking models below as stages/lenses inside this one workflow. Start each supporting lens at Micro unless the step needs more depth; escalate that same model to Focused/Full only when warranted. Feed its result into the next stage. Do not emit separate model reports or abandon the workflow. Keep Reflex active throughout.


> _Byline: Claude Code · Fable 5.1 · 2026-09-07._

## Purpose

The workflow for the owner's core complaint: agents grade their own work kindly, and the same
evaluation process gets re-typed by hand every time. `evaluate` is that process, frozen into fixed
steps with a template the model must fill and a gate it cannot talk its way past. It replaces an
ad hoc "looks good to me" with a verdict card that shows its work.

## When it fires

Trigger phrases: "evaluate this", "review my work", "is it done", "is this ready", "did I finish
X", any claim of completion before a PR, a commit, a handoff, or a report to the owner. If a
sentence in the conversation asserts something is fixed, working, or passing, that sentence is a
completion claim and belongs in this workflow's evidence inventory (step 2) whether or not the
word "evaluate" was used.

## Roles

State the roles explicitly before step 1, in the output:

- **Author** — whoever produced the work under review (an agent, a subagent, a prior session, or
  the current session in an earlier turn).
- **Judge** — this workflow, run now.

If Author and Judge are the same session in the same run (self-review), that is **strict mode**,
automatically, with no opt-out: no self-scored PASS without at least one artifact produced by a
tool call in *this* run (see STRICT MODE below). If a genuinely separate agent or a human produced
the work, ordinary mode applies, but the evidence-inventory gate (step 2) is unchanged either way.

## Steps

1. **Separate the roles.** Write the one-line Author/Judge statement above. Decide now whether
   strict mode is in force — do not decide this after seeing whether the work looks good.

2. **Evidence inventory.** List every claim of completion made about the work, one row each, with
   the artifact that proves it: command output, file content, a test run, a live probe, a diff.
   Apply Hitchens's razor to this step —
   `../reason/references/hitchens-razor/SKILL.md` — what is asserted
   without evidence gets dismissed without evidence: mark that row `UNVERIFIED` and it cannot count
   toward PASS regardless of how plausible it sounds.

3. **Red team.** Load `../decide/references/thinking-red-team/SKILL.md`
   and attack the work as an adversary. Produce **at least three concrete attacks**, each tied to a
   specific file, line, or scenario — not "there might be edge cases" but "in `parser.go:118`, a
   zero-length attachment array causes `items[0]` to panic" or "scenario: two workers dequeue the
   same job ID within the same 50ms window because the row lock is acquired after the read, not
   before." Fewer than three, or attacks with no file/line/scenario anchor, fail the gate.

4. **Steel-man the opposite.** Load
   `../reason/references/thinking-steel-manning/SKILL.md` and write
   the strongest honest case that this work should be **rejected** — not a strawman, the version a
   competent skeptic would actually make, as if you believed it while writing it.

5. **Pre-mortem.** Load
   `../decide/references/thinking-pre-mortem/SKILL.md`. It is one
   month later, this shipped, and it failed. Name the three most likely causes, ranked.

6. **Falsification.** Load
   `../reason/references/poppers-falsifiability/SKILL.md`. State one
   test that would prove the work wrong if it is in fact wrong. If it is runnable in this session,
   run it and report the real output. If it is not runnable here (needs a live service, a human,
   production data), state exactly why not — "not runnable: requires the VPS Coolify deploy" is a
   valid answer; silence is not.

7. **Calibrated verdict.** Load
   `../forecast/references/calibration/SKILL.md` and
   `../forecast/references/thinking-in-bets/SKILL.md`. Render
   **PASS / PARTIAL / FAIL** with a **confidence range** (e.g. "70–85% this holds under load"), not
   a bare percentage and not a bare label. Name the single fact that would change the verdict.

## GATE

The card is **INVALID** — say so explicitly, do not silently soften it — if any of these hold:

- Any of the seven sections is empty or missing.
- Step 2 (evidence inventory) has zero artifacts for a claim that was scored as verified.
- Step 3 (red team) names fewer than three attacks, or any attack lacks a file/line/scenario anchor.
- Step 6 (falsification) has neither a run result nor a stated reason it couldn't be run.
- Step 7 (verdict) is a point estimate ("PASS, 90%") instead of a range.
- **Strict mode only:** the verdict is PASS and zero artifacts in the evidence inventory were
  produced by a tool call *in this run* (pre-existing files the author already wrote, with no new
  read/test/probe performed now, do not satisfy this — go re-check them).

An invalid card is the correct output when the inputs don't support more. Do not upgrade it to
look finished.

## STRICT MODE

Strict mode is **on by default** whenever Author and Judge are the same run, and should stay on
for anything that will be relied on downstream (a PR, a deploy, a claim to the owner). In strict
mode, `PASS` may never be emitted unless at least one artifact in the evidence inventory came from
a tool call executed during this same evaluate run — not a memory of having run something earlier,
not the author's own prior report. If nothing can be freshly checked, the ceiling is `PARTIAL`.

## OUTPUT TEMPLATE

```markdown
# Evaluate: <what is being evaluated>

## Roles
- Author: <who produced the work>
- Judge: <this run>
- Strict mode: <on/off> — <why>

## 1. Evidence Inventory
| Claim | Artifact | Fresh in this run? | Status |
|---|---|---|---|
| <claim> | <command/file/test/probe> | yes/no | VERIFIED / UNVERIFIED |

## 2. Red Team (>= 3 attacks)
1. [<file:line or scenario>] <attack>
2. [<file:line or scenario>] <attack>
3. [<file:line or scenario>] <attack>

## 3. Steel-Man the Opposite
<the strongest honest case for rejecting this work>

## 4. Pre-Mortem (3 causes, ranked)
1. <most likely cause of future failure>
2. <second>
3. <third>

## 5. Falsification Test
- Test: <what would prove this wrong>
- Result: <actual output> OR Not run: <stated reason>

## 6. Verdict
**PASS / PARTIAL / FAIL** — confidence: <NN>–<MM>%
What would change this verdict: <the single fact>

## Gate Check
- [ ] All six analysis sections populated
- [ ] Every claim scored VERIFIED has a fresh artifact
- [ ] >= 3 red-team attacks, each with a file/line/scenario anchor
- [ ] Falsification test run, or a stated reason it could not be
- [ ] Verdict is a confidence range, not a point
- [ ] (strict mode) PASS backed by >= 1 tool-produced artifact from this run
```

## Worked example (synthetic)

**Claim under review:** "Fixed the flaky checkout test — added a retry so it passes now."

```markdown
# Evaluate: PR #482 — "fix flaky checkout test with retry"

## Roles
- Author: prior session (subagent `checkout-fix`)
- Judge: this run
- Strict mode: on — this PR is about to merge to main

## 1. Evidence Inventory
| Claim | Artifact | Fresh in this run? | Status |
|---|---|---|---|
| "test now passes" | ran `pytest tests/test_checkout.py -q` 20x in a loop | yes | VERIFIED (20/20 pass) |
| "root cause was network flakiness" | none provided — no log, no trace | no | UNVERIFIED |

## 2. Red Team (>= 3 attacks)
1. [tests/test_checkout.py:41] The retry wraps the whole test body, including the assertion — a
   retry can mask a real race condition in the code under test, not just a flaky harness.
2. [checkout.py:88] No change to the underlying `save_order` call the test exercises; if the real
   bug is a double-submit race, this PR hides it under CI's lighter load instead of fixing it.
3. [scenario] Under production traffic (higher concurrency than CI), the same race could still fire
   and now has no test coverage flagging it, because the flaky test now silently retries past it.

## 3. Steel-Man the Opposite
This PR should be rejected: a retry loop around a flaky test is a well-known anti-pattern for
exactly the reason above — it converts a signal of a real concurrency bug into green CI, and the
actual defect (if one exists in `save_order`) ships unfixed and undetected.

## 4. Pre-Mortem (3 causes, ranked)
1. The underlying race in `save_order` fires in production; the retry hid it from CI forever.
2. The retry itself introduces a duplicate order on the rare case both attempts partially commit.
3. A future dev sees the retry, assumes it's "just flaky infra," and copies the pattern elsewhere.

## 5. Falsification Test
- Test: run the original (non-retried) test 50x under artificially added latency on the DB call to
  see if it fails at a rate consistent with "test infra flakiness" vs. "real race."
- Result: ran it — 6/50 failures, all showing two `INSERT`s for one order ID in the DB log. This is
  a real race, not test flakiness.

## 6. Verdict
**FAIL** — confidence: 80–90% this is masking a real bug, not fixing a flaky test.
What would change this verdict: a fix to `save_order`'s locking (not just the test) that makes the
50x-under-latency run come back 50/50 with zero duplicate-ID inserts.

## Gate Check
- [x] All six analysis sections populated
- [x] Every claim scored VERIFIED has a fresh artifact
- [x] >= 3 red-team attacks, each with a file/line/scenario anchor
- [x] Falsification test run
- [x] Verdict is a confidence range
- [x] (strict mode) verdict is FAIL, not PASS, so the artifact requirement is moot here
```
