---
name: pest-analysis-synthesis
description: "Create PEST synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions synthesis table, strategic recommendations, opportunity threat analysis, or after completing PEST research."
metadata:
  short-description: Create PEST synthesis tables with strategic implications, impact and likelihood ratings.…
  think-kind: model
  think-family: strategy
  think-registry-id: think:strategy:pest-analysis-synthesis
  think-depths: micro,focused,full
  think-reflex: available
---
# Pest Analysis Synthesis

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Create PEST synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions synthesis table, strategic recommendations, opportunity threat analysis, or a…

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

# PEST Analysis - Synthesis Table

Transform PEST research findings into a comprehensive synthesis table with strategic implications, impact assessment, and actionable recommendations.

---

## When to Use This Skill

Use this skill when:

- User requests a PEST synthesis table or strategic matrix
- PEST research results need to be translated into business decisions
- User mentions "synthesis," "strategic implications," or "opportunity/threat"
- After completing PEST research (pest-analysis-execute skill)
- Senior-level strategic analysis is required (McKinsey-level output)

---

## Workflow

### Step 1: Review Provided Information

Carefully read through:

- PEST analysis insights, trends, opportunities, and threats
- Additional context from uploaded files or documents
- Original core strategic question
- Specific industry or market details

### Step 2: Create Synthesis Table

Structure findings into the following table format:

| **Number** | **Trend** | **PEST Category** | **Opportunity / Threat** | **Impact (High/Medium/Low)** | **Likelihood (High/Medium/Low)** | **Strategic Implication** |
| --- | --- | --- | --- | --- | --- | --- |

**Column Specifications:**

| Column | Format | Description |
| --- | --- | --- |
| **Number** | P.1, E.1, S.1, T.1 | Unique identifier by PEST category |
| **Trend** | Concise | Clear description of the identified trend |
| **PEST Category** | Political/Economic/Social/Technological | Factor category |
| **Opportunity / Threat** | Opportunity or Threat | Nature of the trend |
| **Impact** | High/Medium/Low | Potential business impact |
| **Likelihood** | High/Medium/Low | Probability of occurrence/continuation |
| **Strategic Implication** | Actionable recommendation | Concrete business guidance |

---

## Quality Standards

- **Clarity and Precision**: Each trend must be specific and well-defined
- **Actionable Insights**: Strategic implications must be concrete, not generic
- **Senior-Level Quality**: McKinsey-level clarity and practicality expected
- **Prioritization**: Clearly rank implications based on combined Impact × Likelihood

---

## Example Output

| **Number** | **Trend** | **PEST Category** | **Opportunity / Threat** | **Impact** | **Likelihood** | **Strategic Implication** |
| --- | --- | --- | --- | --- | --- | --- |
| P.1 | Green incentives in California | Political | Opportunity | High | High | Prioritize entry in California, form strategic alliances with cities. |
| T.1 | Poor infrastructure in the Midwest | Technological | Threat | Medium | High | Delay Midwest entry; explore infrastructure partnerships or innovations. |
| E.1 | Inflation driving commodity costs up | Economic | Threat | High | Medium | Lock in long-term supplier contracts now before prices rise further. |
| S.1 | Growing demand for sustainable products | Social | Opportunity | High | High | Accelerate product line expansion with eco-friendly features. |

---

## Task Templates

### Template A: Full Synthesis Table

```
1. Review all PEST research findings and uploaded documents
2. Extract all trends from each PEST factor
3. Assign impact and likelihood ratings based on available data
4. Determine if each trend is opportunity or threat
5. Craft specific, actionable strategic implications
6. Present complete synthesis table
7. Provide collaborative invitation for review
```

### Template B: Priority-Focused Synthesis

```
1. Filter trends for High Impact + High Likelihood combinations
2. Present priority table focusing on critical items
3. Provide additional appendix for all other trends
4. Highlight top 3-5 strategic priorities
```

### Template C: Category-Focused Synthesis

```
1. Identify which PEST factor(s) user wants to emphasize
2. Prioritize trends within selected category
3. Provide brief summary of other categories
4. Focus strategic implications on primary category
```

---

## Closing Collaboration Statement

After completing the table, invite the user:

*"I've synthesized your PEST analysis insights into the strategic recommendations table above. Let's collaborate: Please review and let me know if any adjustments, refinements, or additions are needed to ensure it perfectly meets your strategic goals."*

---

## Post-Change Checklist

After modifying THIS skill:

1. [ ] YAML frontmatter valid (name lowercase-hyphen, description has triggers)
2. [ ] Description includes WHEN to use (trigger keywords)
3. [ ] Task templates cover all common scenarios
4. [ ] Table format example demonstrates proper structure
5. [ ] Quality standards reflect senior-level expectations
6. [ ] Cross-reference to pest-analysis and pest-analysis-execute skills is clear
