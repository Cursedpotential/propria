---
name: pest-analysis-execute
description: "Execute PEST analysis by processing research results from Perplexity AI prompts. Create synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions PEST results, analysis execution, research synthesis, or after generating PEST analysis plans."
metadata:
  short-description: Execute PEST analysis by processing research results from Perplexity AI prompts. Create s…
  think-kind: model
  think-family: strategy
  think-registry-id: think:strategy:pest-analysis-execute
  think-depths: micro,focused,full
  think-reflex: available
---
# Pest Analysis Execute

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Execute PEST analysis by processing research results from Perplexity AI prompts. Create synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions P…

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

# PEST Analysis - Execute

Process and synthesize research results from PEST analysis prompts to generate strategic insights, synthesis tables, and recommendations.

---

## When to Use This Skill

Use this skill when:

- User has PEST analysis research results to process
- After generating PEST analysis plans (pest-analysis skill)
- User requests a PEST synthesis table or strategic matrix
- User mentions "analyze results," "synthesize findings," "process research," or "synthesis table"
- Research outputs need to be consolidated into actionable insights
- User has completed Perplexity AI searches and needs analysis
- Senior-level strategic analysis is required (McKinsey-level output)

---

## Workflow

### Step 1: Gather Research Inputs

Collect the following from the user:

- **PEST Analysis Plan**: The original plan with strategic questions
- **Research Results**: Perplexity AI outputs for each question
- **Core Question**: Remind of the original strategic question
- **Format Preference**: How user wants results presented (summary, table, matrix)

### Step 2: Review Provided Information

Carefully read through:

- PEST analysis insights, trends, opportunities, and threats
- Additional context from uploaded files or documents
- Original core strategic question
- Specific industry or market details

### Step 3: Process Each Factor

For each PEST factor (Political, Economic, Social, Technological):

1. **Review findings** - Extract key insights from research
2. **Identify patterns** - Note trends, contradictions, gaps
3. **Assess relevance** - Rate impact on core strategic question (High/Medium/Low)
4. **Synthesize** - Consolidate into 2-3 key points per factor

### Step 4: Create Synthesis Table

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

### Step 5: Generate Strategic Output

Create structured output including:

| Section | Content |
| --- | --- |
| **Executive Summary** | 3-5 bullet points highlighting critical findings |
| **Factor-by-Factor Analysis** | Detailed breakdown of each PEST factor |
| **Synthesis Table** | Complete table with all trends and implications |
| **Strategic Implications** | What findings mean for the core question |
| **Risk Assessment** | Identified threats and opportunities |
| **Actionable Recommendations** | 3-5 concrete next steps |

---

## Quality Standards

- **Clarity and Precision**: Each trend must be specific and well-defined
- **Actionable Insights**: Strategic implications must be concrete, not generic
- **Senior-Level Quality**: McKinsey-level clarity and practicality expected
- **Prioritization**: Clearly rank implications based on combined Impact × Likelihood

---

## Example Output

```
# PEST Analysis Results: [Core Question]

## Executive Summary
- Key insight 1
- Key insight 2
- Key insight 3

## Synthesis Table

| **Number** | **Trend** | **PEST Category** | **Opportunity / Threat** | **Impact** | **Likelihood** | **Strategic Implication** |
| --- | --- | --- | --- | --- | --- | --- |
| P.1 | Green incentives in California | Political | Opportunity | High | High | Prioritize entry in California, form strategic alliances with cities. |
| T.1 | Poor infrastructure in the Midwest | Technological | Threat | Medium | High | Delay Midwest entry; explore infrastructure partnerships or innovations. |
| E.1 | Inflation driving commodity costs up | Economic | Threat | High | Medium | Lock in long-term supplier contracts now before prices rise further. |
| S.1 | Growing demand for sustainable products | Social | Opportunity | High | High | Accelerate product line expansion with eco-friendly features. |

## Factor-by-Factor Analysis

### Political Analysis
**Impact**: High
- Finding 1 with source
- Finding 2 with source
- Strategic implication

### Economic Analysis
[Similar structure...]

## Strategic Implications
1. Implication 1
2. Implication 2

## Recommendations
1. Actionable step 1
2. Actionable step 2
```

---

## Task Templates

### Template A: Full PEST Synthesis

```
1. Receive PEST plan and all research results
2. Process each factor systematically (P, E, S, T)
3. Generate executive summary
4. Create synthesis table with impact/likelihood ratings
5. Create factor-by-factor analysis with impact ratings
6. Synthesize strategic implications
7. Identify risks and opportunities
8. Provide actionable recommendations
9. Present collaborative invitation for review
```

### Template B: Partial PEST Synthesis

```
1. Identify which PEST factors have research results
2. Process available factors only
3. Note gaps in research coverage
4. Generate analysis based on available data
5. Suggest areas requiring additional research
```

### Template C: Priority-Focused Synthesis

```
1. Filter trends for High Impact + High Likelihood combinations
2. Present priority table focusing on critical items
3. Provide additional appendix for all other trends
4. Highlight top 3-5 strategic priorities
```

### Template D: Quick Insight Extraction

```
1. Scan all research results
2. Extract top 3 most critical findings across all factors
3. Explain why each finding matters to core question
4. Provide immediate action recommendation
```

---

## Closing Collaboration Statement

After completing the synthesis table, invite the user:

*"I've synthesized your PEST analysis insights into the strategic recommendations table above. Let's collaborate: Please review and let me know if any adjustments, refinements, or additions are needed to ensure it perfectly meets your strategic goals."*

---

## Post-Change Checklist

After modifying THIS skill:

1. [ ] YAML frontmatter valid (name lowercase-hyphen, description has triggers)
2. [ ] Description includes WHEN to use (trigger keywords)
3. [ ] Task templates cover all common scenarios
4. [ ] Example output format demonstrates structure
5. [ ] Synthesis table format is included and documented
6. [ ] Cross-reference to pest-analysis and pest-analysis-critical skills is clear
