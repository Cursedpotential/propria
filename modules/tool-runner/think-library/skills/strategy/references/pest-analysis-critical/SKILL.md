---
name: pest-analysis-critical
description: "Critically evaluate PEST synthesis tables for biases, gaps, and oversights. Use when user mentions critical review, bias assessment, gap analysis, or after completing PEST synthesis."
metadata:
  short-description: Critically evaluate PEST synthesis tables for biases, gaps, and oversights. Use when user…
  think-kind: model
  think-family: strategy
  think-registry-id: think:strategy:pest-analysis-critical
  think-depths: micro,focused,full
  think-reflex: available
---
# Pest Analysis Critical

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Critically evaluate PEST synthesis tables for biases, gaps, and oversights. Use when user mentions critical review, bias assessment, gap analysis, or after completing PEST synthesis.

Ask only what is necessary:
- What objective or outcome is actually being pursued?
- What constraint, tradeoff, competitor, user need, or contradiction shapes the choice?
- Do the proposed actions form one coherent strategy rather than a collection of activities?

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

# PEST Analysis - Critical Assessment

Critically evaluate PEST synthesis tables to identify biases, gaps, and oversights, ensuring robustness and reliability of strategic recommendations.

---

## When to Use This Skill

Use this skill when:

- User requests critical evaluation of PEST synthesis
- Assessment of potential biases or gaps in analysis is needed
- User mentions "critical review," "bias assessment," or "blind spots"
- After completing PEST synthesis table (pest-analysis-synthesis skill)
- Senior-level strategic review with rigor is required

---

## Workflow

### Step 1: Critically Reflect on Synthesized Output

Carefully review the previously created synthesis table to identify:

- Areas where assumptions or biases might exist
- Underrepresented emerging trends
- Factors disputed by industry insiders
- Connections between PEST categories creating compound effects
- Key uncertainties or alternative scenarios not considered
- Potential cognitive biases or blind spots

### Step 2: Generate Critical Assessment Table

Structure critical findings into the following table format:

| **Number** | **Trend or Insight Evaluated** | **Critical Question** | **Assessment / Potential Bias or Gap** | **Recommended Action for User** |
| --- | --- | --- | --- | --- |

---

## Critical Questions to Consider

When evaluating each trend, ask:

1. **Emerging Trends**: What significant emerging trends might be underrepresented because they haven't yet appeared in published sources?

2. **Industry Perspective**: Which factors would be most disputed by industry insiders, and why might their perspective differ from public information?

3. **Compound Effects**: What connections between different PEST categories might create compound effects that aren't captured in the individual analyses?

4. **Uncertainties & Scenarios**: Are there key uncertainties or alternative scenarios that haven't been adequately considered?

5. **Cognitive Biases**: What potential blind spots or cognitive biases could have influenced the original synthesis?

---

## Quality Standards

- **Nuanced Evaluation**: Provide insightful, rigorous assessments beyond surface-level critique
- **Actionable Recommendations**: Each identified issue must have clear, concrete user action
- **Critical Tone**: Maintain appropriate critical and reflective consulting approach
- **Systematic Coverage**: Evaluate across all PEST categories and impact levels

---

## Example Output

| **Number** | **Trend or Insight Evaluated** | **Critical Question** | **Assessment / Potential Bias or Gap** | **Recommended Action for User** |
| --- | --- | --- | --- | --- |
| P.1 | Green incentives in California | Are emerging policy shifts adequately represented? | Current synthesis relies on published incentives; emerging policy trends might be overlooked. | Monitor ongoing policy discussions and pending legislation. |
| T.1 | Poor infrastructure in the Midwest | Would industry insiders agree on infrastructure status? | Public information may lag behind current industry actions to improve infrastructure. | Consult industry experts or stakeholders in infrastructure planning. |
| E.1 | Inflation driving commodity costs | Are alternative economic scenarios considered? | Analysis assumes continued inflation; deflationary scenarios or central bank actions not explored. | Run scenario planning with varying inflation trajectories. |
| S.1 | Growing demand for sustainable products | What compound effects with other factors? | Demand growth may be amplified by regulatory changes (P) or undermined by cost pressures (E). | Analyze intersection effects between Social and Political factors. |

---

## Task Templates

### Template A: Full Critical Assessment

```
1. Review complete synthesis table from prior analysis
2. Apply all 5 critical questions to each trend
3. Identify biases, gaps, and oversights
4. Generate critical assessment table with recommendations
5. Prioritize findings by severity of potential impact
6. Present collaborative invitation for review
```

### Template B: Focused Bias Review

```
1. Select specific cognitive bias framework (confirmation bias, availability heuristic, etc.)
2. Apply framework to synthesis table
3. Identify specific instances of each bias type
4. Provide mitigation strategies for each bias
5. Present bias-specific recommendations
```

### Template C: Gap Analysis Priority

```
1. Identify trends with High Impact but uncertain data quality
2. Flag trends where industry perspective may differ
3. Highlight compound effects not captured
4. Prioritize gaps by potential strategic risk
5. Create action plan for highest-priority gaps
```

---

## Common Cognitive Biases to Watch For

| Bias | How It Manifests in PEST Analysis | Mitigation |
| --- | --- | --- |
| **Confirmation Bias** | Emphasizing data that supports preexisting beliefs | Actively seek disconfirming evidence |
| **Availability Heuristic** | Overweighting recent or highly visible events | Include systematic historical data |
| **Anchoring** | Initial information heavily influencing conclusions | Challenge initial assumptions explicitly |
| **Recency Bias** | Overweighting recent trends at expense of long-term patterns | Balance short-term with long-term indicators |
| **Groupthink** | Avoiding contrarian views | Explicitly consider alternative scenarios |

---

## Closing Collaboration Statement

After completing the critical assessment table, invite the user:

*"I've critically assessed the synthesized insights from your PEST analysis. Please review the assessment above and let me know if further exploration or adjustments are needed to enhance the reliability and strategic clarity of the analysis."*

---

## Post-Change Checklist

After modifying THIS skill:

1. [ ] YAML frontmatter valid (name lowercase-hyphen, description has triggers)
2. [ ] Description includes WHEN to use (trigger keywords)
3. [ ] Task templates cover all common scenarios
4. [ ] Table format example demonstrates proper structure
5. [ ] Critical questions list is comprehensive
6. [ ] Cognitive biases reference is included
7. [ ] Cross-reference to other pest-analysis skills is clear
