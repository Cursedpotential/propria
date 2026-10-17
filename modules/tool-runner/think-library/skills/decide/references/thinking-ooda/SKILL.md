---
name: thinking-ooda
description: "Rapid decision-making loop for dynamic situations. Use for incident response, competitive scenarios, time-sensitive decisions, and situations requiring quick adaptation."
metadata:
  short-description: Rapid decision-making loop for dynamic situations. Use for incident response, competitive…
  think-kind: model
  think-family: decide
  think-registry-id: think:decide:thinking-ooda
  think-depths: micro,focused,full
  think-reflex: 'true'
---
# Ooda

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Rapid decision-making loop for dynamic situations. Use for incident response, competitive scenarios, time-sensitive decisions, and situations requiring quick adaptation.

Ask only what is necessary:
- What has actually been observed now, including tool results or changed state?
- What does that evidence mean for the current goal and assumptions?
- What is the smallest useful next action, and what feedback should it generate?
- After that action, what observation would make us continue, adapt, or stop?

**Micro exit:** if the lens does not reveal a material issue, return the useful conclusion immediately. Do not manufacture a report.

**Escalate to focused** when the pass reveals material uncertainty, a cross-component effect, conflicting evidence, meaningful downside, or an unresolved issue that could change the next action.

### Focused branch

Apply the relevant steps from the Full Process to the specific issue that caused escalation. Follow the model faithfully, but do not execute unrelated sections merely for completeness and do not produce the formal full deliverable unless it is needed.

**Focused exit:** return the findings, changed assumptions, and any constraints to the invoking process.

**Escalate to full** when the user explicitly requests the complete framework/deliverable, the focused pass remains materially uncertain, the stakes or irreversibility justify it, or the complete process would materially change the decision.

### Full branch

Run the complete process below. Use its full checks/templates/deliverable where present. When this skill was invoked from a parent process, return the result to that parent rather than abandoning or restarting the parent workflow.

> **Branch invariant:** nested use is a branch, not a replacement workflow. New findings may revise the parent plan, but they do not erase unaffected requirements or prior decisions.

---

## Full Process — canonical model

# OODA Loop

## Overview
The OODA Loop (Observe, Orient, Decide, Act), developed by military strategist Colonel John Boyd, is a framework for rapid decision-making in dynamic, competitive, or time-sensitive situations. The key insight: speed through the loop creates advantage. In competitive scenarios, operating faster than your opponent disrupts their decision-making.

**Core Principle:** Agility beats perfection. Cycle through OODA faster than the situation changes (or faster than your opponent).

## When to Use
- Incident response and outages
- Competitive market situations
- Time-sensitive decisions
- Rapidly changing requirements
- Crisis management
- Debugging under pressure
- Any situation requiring quick adaptation

Decision flow:
```
Situation changing rapidly? → yes → Need quick decisions? → yes → APPLY OODA LOOP
                                                         ↘ no → Standard analysis
                          ↘ no → Deliberate analysis may be better
```

## The Four Phases

### 1. OBSERVE
**Gather information rapidly**

What to observe:
- Current state of the system/situation
- Changes since last observation
- External factors and environment
- Feedback from previous actions
- Competitor/opponent movements

```
Incident Example:
- Error rates: Spiking 10x normal
- Affected services: API gateway, user service
- Timeline: Started 5 minutes ago
- Recent changes: Deploy 15 min ago
- User reports: "Can't log in"
```

Observation principles:
- Cast wide net initially, narrow as pattern emerges
- Don't filter prematurely—gather raw data
- Include lagging AND leading indicators
- Time-bound: Don't observe forever

### 2. ORIENT
**Make sense of observations**

Orientation factors (Boyd's framework):
- **Cultural traditions**: How does our org typically respond?
- **Genetic heritage**: Our built-in biases and tendencies
- **Previous experience**: What have we seen before?
- **New information**: What's different this time?
- **Analysis/Synthesis**: Combining all of the above

```
Incident Example:
- Pattern matches: Similar to DB connection pool exhaustion last month
- But different: No DB metrics anomaly this time
- Recent deploy touched: Auth service rate limiting
- Hypothesis: Rate limit config too aggressive
```

Orient is the CRITICAL phase:
- This is where mental models apply
- Misorientation leads to wrong decisions
- Update orientation as new info arrives
- Challenge your initial framing

### 3. DECIDE
**Select course of action**

Decision characteristics:
- Based on current orientation
- Acknowledges uncertainty
- Identifies what to observe next
- Has implicit/explicit hypothesis

```
Incident Example:
Decision: Roll back auth service deploy
Hypothesis: This will restore normal error rates
Observation plan: Watch error rates for 2 minutes post-rollback
Fallback: If no improvement, investigate DB connections
```

Decision speed vs. quality tradeoff:
- 70% confidence now beats 90% confidence too late
- Reversible decisions: Bias toward action
- Irreversible decisions: Gather more info first
- "Good enough" decision executed fast > perfect decision too slow

### 4. ACT
**Execute the decision**

Action principles:
- Execute decisively
- Immediately return to OBSERVE
- Don't wait for complete results
- Create new observations through action

```
Incident Example:
Action: kubectl rollback deployment/auth-service
Immediate observe: Error rates, response times
Time limit: 2 minutes to see effect
```

The loop restarts:
- Actions create new situation
- New situation requires new observation
- Cycle continues until stable state

## OODA Loop Speed

### Tempo Advantage
Operating inside opponent's loop:
```
You:     O → O → D → A → O → O → D → A → O ...
Opponent:     O → O → O →  ...  → D → A (too late)
```

When you complete loops faster:
- Your actions change situation before they decide
- Their orientation becomes outdated
- They react to old information
- You maintain initiative

### Speed Multipliers
| Factor | Effect |
|--------|--------|
| Pre-planned responses | Skip D phase for known scenarios |
| Distributed authority | Parallel loops at different levels |
| Clear mental models | Faster O (orientation) |
| Training/practice | Faster execution (A) |
| Good observability | Faster O (observation) |

### Speed Killers
| Factor | Effect |
|--------|--------|
| Waiting for certainty | Loop stalls at O or D |
| Hierarchical approval | Adds latency to D |
| Information overload | O phase never completes |
| Analysis paralysis | Loop stalls at Orient |
| Perfect solution seeking | D phase never completes |

## Application Patterns

### Incident Response
```
OBSERVE: Metrics, logs, alerts, user reports
ORIENT:  Match pattern, form hypothesis, assess blast radius
DECIDE:  Mitigation action (rollback, scale, disable)
ACT:     Execute mitigation, immediately observe results
LOOP:    Continue until stable
```

### Competitive Response
```
OBSERVE: Competitor announcement, market reaction, customer feedback
ORIENT:  Assess threat level, identify our advantages, gaps
DECIDE:  Response strategy (match, differentiate, ignore)
ACT:     Execute response, observe market reaction
LOOP:    Adjust based on effectiveness
```

### Debugging Under Pressure
```
OBSERVE: Error messages, stack traces, recent changes
ORIENT:  Form hypothesis about cause
DECIDE:  Test most likely hypothesis first
ACT:     Add logging, try fix, or eliminate possibility
LOOP:    Update hypothesis based on results
```

## OODA for Teams

### Parallel Loops
Different team members can run loops simultaneously:
```
SRE:     Infrastructure OODA (scaling, failover)
Dev:     Code OODA (debugging, fixes)
Support: Communication OODA (users, stakeholders)
Lead:    Strategy OODA (coordination, escalation)
```

### Shared Orientation
Teams need synchronized mental models:
- Runbooks create shared orientation
- Incident channels share observations
- Clear roles enable parallel action
- Post-incident updates orientation for next time

## Verification Checklist
- [ ] Observing actual current state, not assumptions
- [ ] Orientation considers multiple hypotheses
- [ ] Decision is actionable and time-bound
- [ ] Action creates observable feedback
- [ ] Loop is actually cycling (not stuck in one phase)
- [ ] Speed is appropriate to situation urgency

## Common Failure Modes

| Failure | Symptom | Fix |
|---------|---------|-----|
| Observation overload | Can't process all data | Filter to key indicators |
| Orientation lock | Stuck on one hypothesis | Force alternative framing |
| Decision paralysis | Waiting for certainty | Set decision deadline |
| Action without observation | Blind execution | Mandate observe after act |
| Single loop | Not cycling | Time-box each phase |

## Key Questions
- "What do I observe RIGHT NOW?" (not 5 minutes ago)
- "What does this mean? What pattern does it match?"
- "What's my best action given current understanding?"
- "How will I know if my action worked?"
- "Am I cycling fast enough?"

## Boyd's Insight
"He who can handle the quickest rate of change survives."

The goal isn't just making decisions—it's making decisions faster than the situation evolves, faster than competitors adapt, faster than problems compound. Speed creates options; delay eliminates them.
