---
name: thinking-debiasing
description: "Systematic checklist to identify and counteract cognitive biases in decision-making. Use before major decisions, when evaluating recommendations, or when stakes are high."
metadata:
  short-description: Systematic checklist to identify and counteract cognitive biases in decision-making. Use…
  think-kind: model
  think-family: debias
  think-registry-id: think:debias:thinking-debiasing
  think-depths: micro,focused,full
  think-reflex: available
---
# Debiasing

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Systematic checklist to identify and counteract cognitive biases in decision-making. Use before major decisions, when evaluating recommendations, or when stakes are high.

Ask only what is necessary:
- What assumption, bias, or framing could be distorting the read?
- What evidence cuts against the current interpretation?
- Would correcting that distortion materially change the conclusion or next action?

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

# Cognitive Debiasing

## Overview
Based on Daniel Kahneman, Dan Lovallo, and Olivier Sibony's research, this skill provides a systematic checklist to identify cognitive biases that distort decisions. Awareness of biases alone doesn't prevent them—structured checklists and processes do.

**Core Principle:** Your brain is systematically wrong in predictable ways. Use checklists to catch errors your intuition will miss.

## When to Use
- Before making or approving major decisions
- Evaluating recommendations from others
- When stakes are high and errors costly
- When you feel very confident (overconfidence is a bias)
- During investment, hiring, or strategic decisions
- When a decision "feels right" but you can't articulate why

## System 1 vs System 2

| System 1 (Fast) | System 2 (Slow) |
|-----------------|-----------------|
| Automatic, effortless | Deliberate, effortful |
| Emotional, intuitive | Analytical, logical |
| Pattern-matching | Rule-following |
| Prone to biases | Can catch biases |
| Default mode | Requires activation |

**Goal:** Activate System 2 for important decisions using structured processes.

## The 12-Point Decision Quality Checklist

Before approving any significant recommendation, evaluate:

### Self-Interest Biases
**1. Is there self-interest at play?**
- Does the recommender benefit from this decision?
- Would they recommend the same if incentives were different?
- Are there conflicts of interest?

**2. Is there emotional attachment (affect heuristic)?**
- Has the team fallen in love with the proposal?
- Are they dismissing concerns too quickly?
- Is criticism being taken personally?

### Group Dynamics
**3. Has dissenting opinion been suppressed (groupthink)?**
- Were alternative views genuinely explored?
- Is there pressure to conform?
- Has a devil's advocate been assigned?

**4. Is there appropriate diversity of opinion?**
- Did independent thinkers contribute?
- Were estimates made independently before discussion?
- Has anyone with a different perspective reviewed this?

### Pattern Recognition Errors
**5. Are we over-relying on a single analogy (saliency bias)?**
- Is there one "this is just like X" dominating thinking?
- Have we sought disconfirming analogies?
- Are we cherry-picking the comparison?

**6. Are we anchored on an initial number?**
- Where did the first estimate come from?
- Would a different starting point change the conclusion?
- Have we re-estimated from scratch?

### Confirmation Bias
**7. Were credible alternatives seriously considered?**
- Did we explore at least 2-3 real alternatives?
- Were alternatives given fair evaluation?
- Or were they strawmen to justify the preferred option?

**8. Are we seeking confirming evidence only?**
- What evidence would disprove this thesis?
- Have we actively looked for disconfirming data?
- Are we explaining away contradictory evidence?

### Planning Fallacies
**9. Is the base case realistic?**
- Is this more optimistic than similar past projects?
- What's the base rate of success for similar efforts?
- Have we adjusted for "this time is different" thinking?

**10. Is the worst case bad enough?**
- Does worst case assume only one thing goes wrong?
- What if multiple risks materialize simultaneously?
- Have we considered tail risks?

**11. Are we discounting sunk costs appropriately?**
- Would we make this decision if starting fresh?
- Are we continuing because we've "invested too much"?
- What would an outsider with no history decide?

### Halo Effects
**12. Are we assuming success transfers?**
- Are we trusting this team/approach because of past wins?
- Were past successes in similar contexts?
- Are we attributing success to skill when luck played a role?

## Quick Debiasing Techniques

### For Anchoring
- Generate estimate BEFORE seeing others' numbers
- Ask: "What if the true number is 2x or 0.5x?"
- Use multiple independent estimators

### For Confirmation Bias
- Assign someone to argue the opposite position
- Ask: "What would make us wrong?"
- Seek out critics, not supporters

### For Overconfidence
- Widen confidence intervals (usually too narrow)
- Use reference class forecasting (base rates)
- Ask: "How often have similar predictions been right?"

### For Sunk Cost
- Ask: "Would we start this project today knowing what we know?"
- Ignore past investment when evaluating future returns
- Consider opportunity cost of continuing

### For Groupthink
- Collect independent opinions before discussion
- Assign devil's advocate role
- Make it safe to dissent

## Decision Quality Audit Template

```markdown
# Decision Quality Audit: [Decision Name]

## Recommendation Summary
[Brief description]

## Bias Checklist

### Self-Interest & Emotion
- [ ] Self-interest checked: [Notes]
- [ ] Emotional attachment assessed: [Notes]

### Group Dynamics  
- [ ] Dissent encouraged: [Notes]
- [ ] Independent input gathered: [Notes]

### Pattern Recognition
- [ ] Multiple analogies considered: [Notes]
- [ ] Anchoring effects checked: [Notes]

### Confirmation Bias
- [ ] Alternatives genuinely evaluated: [Notes]
- [ ] Disconfirming evidence sought: [Notes]

### Planning Realism
- [ ] Base case reality-checked: [Notes]
- [ ] Worst case severe enough: [Notes]
- [ ] Sunk costs ignored: [Notes]

### Halo Effects
- [ ] Success transfer questioned: [Notes]

## Red Flags Identified
[List any concerns from checklist]

## Mitigations
[How will identified biases be addressed?]

## Decision
- [ ] Proceed as recommended
- [ ] Proceed with modifications
- [ ] Requires more analysis
- [ ] Reject recommendation
```

## Verification Checklist
- [ ] Systematically evaluated all 12 bias categories
- [ ] Identified at least 2 potential biases affecting this decision
- [ ] Applied specific debiasing technique to each identified bias
- [ ] Sought independent/outside perspective
- [ ] Documented reasoning for future reference
- [ ] Comfortable defending decision process (not just outcome)

## Key Questions
- "What would have to be true for this to be wrong?"
- "How confident am I, and what's driving that confidence?"
- "What would an outsider with fresh eyes conclude?"
- "Am I reasoning toward a conclusion I already want?"
- "What's the base rate of success for decisions like this?"
- "If I'm wrong, how will I know, and when?"

## Kahneman's Warning
"We can be blind to the obvious, and we are also blind to our blindness."

You cannot debias through willpower alone. Use checklists, processes, and outside perspectives to catch what your intuition misses.
