---
name: pest-analysis-strategy
description: "Transform PEST analysis into actionable strategy: export to CSV, save to Obsidian, create SWOT, define strategic actions, prioritize, and build work plan. Use after completing PEST critical assessment."
metadata:
  short-description: 'Transform PEST analysis into actionable strategy: export to CSV, save to Obsidian, create…'
  think-kind: model
  think-family: strategy
  think-registry-id: think:strategy:pest-analysis-strategy
  think-depths: micro,focused,full
  think-reflex: available
---
# Pest Analysis Strategy

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Transform PEST analysis into actionable strategy: export to CSV, save to Obsidian, create SWOT, define strategic actions, prioritize, and build work plan. Use after completing PEST critical…

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

# PEST Analysis - Strategy & Action

Transform PEST analysis findings into actionable strategic plans: export data, document in Obsidian, create SWOT, define prioritized actions, and build execution work plan.

---

## When to Use This Skill

Use this skill when:

- PEST synthesis and critical assessment are complete
- User needs to export findings to CSV format
- User wants to save analysis in Obsidian (using Obsidian-flavored markdown)
- Moving from analysis to actionable strategy
- User mentions "SWOT," "strategic actions," "prioritization," or "work plan"
- Ready to translate insights into execution

---

## Workflow

### Phase 1: Export and Document

#### Step 1: Export as CSV

Generate CSV files for:

- `pest-synthesis.csv` - PEST Synthesis table
- `pest-critical-assessment.csv` - Critical Assessment with enhancements

#### Step 2: Save to Obsidian

Create Obsidian notes with proper structure:

```
PEST Analysis/
├── 00-PEST-Analysis.md (main page)
├── 01-PEST-Plan.md
├── 02-PEST-Synthesis.md
├── 03-Critical-Assessment.md
├── 04-SWOT-Analysis.md
├── 05-Strategic-Actions.md
└── 06-Work-Plan.md
```

Each note should include:
- YAML frontmatter with tags, dates, project info
- Proper wikilinks to other notes
- Obsidian-flavored markdown syntax
- Callout blocks for key insights

### Phase 2: Enhance Critical Assessment

Add two columns to Critical Assessment table:

| **Action Item** | **Status** |
| --- | --- |
| Follow recommendation / Ignore / Custom action | Not Started / In Progress / Complete / Blocked |

For each critical assessment row, decide the action and set status.

### Phase 3: Create SWOT Analysis

Collect internal factors:

**Strengths (Internal)**
- Technology advantages
- Market position
- Resources and capabilities
- Competitive differentiators

**Weaknesses (Internal)**
- Knowledge/data gaps
- Resource constraints
- Operational limitations
- Competitive disadvantages

**Opportunities (External from PEST Synthesis)**
- P.1, E.1, S.1, T.1 marked as Opportunities

**Threats (External from PEST Synthesis)**
- P.1, E.1, S.1, T.1 marked as Threats

Create SWOT table:
```
| Strengths | Weaknesses |
|-----------|------------|
| [List strengths] | [List weaknesses] |
| **Opportunities** | **Threats** |
| [List opportunities from PEST] | [List threats from PEST] |
```

### Phase 4: Define Strategic Actions

Connect internal factors with external trends:

| Combination | Question | Example |
|-------------|----------|---------|
| **Strengths + Opportunities** | How to use strengths to leverage opportunities? | Position superior technology as market differentiator |
| **Strengths + Threats** | How to use strengths to mitigate threats? | Use expertise to develop compliant systems exceeding regulations |
| **Weaknesses + Opportunities** | Which weaknesses must improve to leverage opportunities? | Partner for data on user behavior patterns |
| **Threats + Weaknesses** | Which weaknesses must improve to avoid threat exposure? | Acquire historical data for seasonal planning |

Generate 3-5 strategic actions per combination.

### Phase 5: Prioritize Actions

Create prioritization table:

| **Strategic Action** | **Benefit (1-10)** | **Effort (1-10)** | **Benefit/Effort Ratio** | **Priority** |
|---------------------|-------------------|-------------------|-------------------------|--------------|

**Scoring Guidelines:**
- **Benefit**: Revenue impact, market advantage, risk reduction (10 = transformational)
- **Effort**: Time, cost, resource requirements (10 = highly demanding)
- **Ratio**: Benefit divided by Effort (higher = better ROI)

**Priority Tiers:**
- Tier 1: Ratio > 1.5 or Benefit ≥ 8
- Tier 2: Ratio 1.0-1.5
- Tier 3: Ratio < 1.0

### Phase 6: Build Work Plan

For Tier 1 and top Tier 2 actions:

| **Action** | **Owner** | **Dependencies** | **Effort Estimate** | **Start Date** | **End Date** |
|------------|-----------|-----------------|---------------------|---------------|-------------|

**Work Plan Elements:**
1. Estimate effort for each work package (hours/days/weeks)
2. Identify dependencies (other actions, resources, approvals)
3. Match effort with team capacity
4. Build project timeline
5. Assign responsibilities

---

## Obsidian Note Templates

### Main Page (00-PEST-Analysis.md)

```markdown
---
tags: [PEST, Strategy, ProjectName]
created: YYYY-MM-DD
project: [Project Name]
status: Active
---

# [Project Name] PEST Analysis

## Overview

[[01-PEST-Plan]] - Research questions and Perplexity prompts
[[02-PEST-Synthesis]] - Strategic synthesis with impact assessment
[[03-Critical-Assessment]] - Critical review with action items
[[04-SWOT-Analysis]] - Internal/external factor analysis
[[05-Strategic-Actions]] - Prioritized strategic initiatives
[[06-Work-Plan]] - Execution timeline and assignments

## Summary

> [!summary]
> Brief executive summary of key findings and next steps

## Status Timeline

- [x] Phase 1: Research Planning
- [x] Phase 2: Analysis & Synthesis
- [x] Phase 3: Critical Review
- [ ] Phase 4: Strategy Development
- [ ] Phase 5: Execution Planning
```

---

## Task Templates

### Template A: Full Strategy Pipeline

```
1. Export synthesis and critical assessment to CSV
2. Create Obsidian vault structure with main page
3. Add Action Item and Status columns to critical assessment
4. Guide SWOT creation with internal factors
5. Generate strategic actions for all 4 SWOT combinations
6. Prioritize actions using benefit/effort matrix
7. Create work plan for prioritized actions
8. Document everything in Obsidian
```

### Template B: Export Only

```
1. Extract synthesis table
2. Extract critical assessment table
3. Convert to CSV format with proper headers
4. Save files with timestamps
5. Provide summary of exported data
```

### Template C: SWOT Focus

```
1. Review PEST synthesis for opportunities and threats
2. Collect internal strengths and weaknesses from user
3. Create SWOT matrix
4. Generate strategic actions for each quadrant
5. Prioritize based on strategic fit
```

---

## Quality Standards

- **CSV Export**: Proper escaping, UTF-8 encoding, clear headers
- **Obsidian Format**: Valid YAML frontmatter, proper wikilinks, callout syntax
- **Action Items**: Concrete, specific, time-bound when possible
- **Prioritization**: Benefit and effort scores justified, ratio calculated
- **Work Plan**: Dependencies identified, capacity considered, realistic timelines

---

## Example Output Structure

```
Files Created:
├── pest-synthesis-YYYY-MM-DD.csv
├── pest-critical-assessment-YYYY-MM-DD.csv
└── Obsidian Notes/
    └── PEST Analysis/
        ├── 00-PEST-Analysis.md
        ├── 02-PEST-Synthesis.md
        ├── 03-Critical-Assessment.md
        ├── 04-SWOT-Analysis.md
        ├── 05-Strategic-Actions.md
        └── 06-Work-Plan.md
```

---

## Post-Change Checklist

After modifying THIS skill:

1. [ ] YAML frontmatter valid (name lowercase-hyphen, description has triggers)
2. [ ] Description includes WHEN to use (trigger keywords)
3. [ ] Task templates cover all common scenarios
4. [ ] Obsidian note templates demonstrate proper format
5. [ ] CSV export format is specified
6. [ ] SWOT and prioritization frameworks are clear
7. [ ] Cross-reference to other pest-analysis skills is clear
8. [ ] Examples show complete workflow
