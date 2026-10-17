---
name: pest-analysis
description: "Generate PEST analysis plans with strategic questions and Perplexity AI prompts. Use when user mentions PEST analysis, strategic planning, market analysis, external factors, environmental scanning, or needs research questions for political economic social technological factors."
metadata:
  short-description: Generate PEST analysis plans with strategic questions and Perplexity AI prompts. Use when…
  think-kind: model
  think-family: strategy
  think-registry-id: think:strategy:pest-analysis
  think-depths: micro,focused,full
  think-reflex: available
---
# Pest Analysis

## Progressive-depth execution

This model supports **micro → focused → full** execution. A direct/manual invocation does not force full mode unless the user asks for the full framework or the problem requires it.

### Micro / Reflex branch

Use this branch when called from the persistent Reflex loop, when the question is routine, or when this model is only a supporting lens inside another process. Keep the pass short and return control to the caller.

**Model lens:** Generate PEST analysis plans with strategic questions and Perplexity AI prompts. Use when user mentions PEST analysis, strategic planning, market analysis, external factors, environmental s…

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

# PEST Analysis

Generates structured PEST analysis plans with strategic questions and optimized Perplexity AI prompts for strategic research.

---

## When to Use This Skill

Use this skill when:

- User requests PEST analysis or environmental scanning
- Strategic planning requires understanding external factors
- Market entry or expansion decisions need factor assessment
- Risk analysis involves political, economic, social, or technological dimensions
- User mentions "market analysis," "competitive landscape," or "external forces"

---

## Workflow

### Step 1: Clarify User Input

Before generating the analysis plan, ask the user the following:

- **Core Strategic Question**: Clearly state the primary strategic question you're aiming to answer.
- **Context & Industry**: Provide context or specific industry details relevant to your strategic decision.
- **Geographical Scope**: Specify the geographical regions or markets relevant to your question.
- **Specific Concerns**: Mention any particular factors (political, economic, social, technological) you believe may strongly influence your scenario.

### Step 2: Create the PEST Analysis Plan

Based on the user input, provide a structured **PEST Analysis Plan** table with the following columns:

| Column Header | Description |
| --- | --- |
| **Number** | Give each row a number. The number consists of a letter for the PEST factor and a running number, e.g. P.3 for the third line in Political factors or E.2 for the second row of Economic factors. |
| **PEST Factor** | Clearly label as Political, Economic, Social, or Technological. |
| **Strategic Questions** | List highly relevant strategic questions tailored to the core question provided. |
| **Advanced Prompt for Perplexity AI** | Formulate a detailed, actionable research prompt tailored for Perplexity AI to conduct effective research, specifying content type, source quality, and targeted search operators. |

### Example Row

| **Number** | **PEST Factor** | **Strategic Questions** | **Advanced Prompt for Perplexity AI** |
| --- | --- | --- | --- |
| P.1 | Political | Which U.S. states offer incentives for electric mobility? | Identify and list U.S. states providing incentives for electric vehicles (EVs), such as rebates, tax credits, or HOV lane access. Focus on retrieving high-quality sources, particularly PDF documents from reputable government (.gov) sites or recognized institutions like the International Council on Clean Transportation (theicct.org). Use search keywords like "electric vehicle incentives by state PDF," "state EV incentives report PDF," or "electric mobility incentives USA PDF" combined with the advanced search operator filetype:pdf. |

---

## Instructions for the LLM

- Clearly align each strategic question and corresponding Perplexity AI prompt to the user's core question and context.
- Provide specific search strategies including keywords, authoritative domains, and advanced operators like `filetype:pdf`, `site:.gov`, or `site:.edu`.
- Ensure the prompt to Perplexity AI directs towards authoritative, detailed, and actionable insights suitable for strategic analysis.

---

## Task Templates

### Template A: Generate PEST Plan from Scratch

```
1. Ask Step 1 clarification questions (Core Question, Context, Geography, Specific Concerns)
2. Wait for user responses
3. Generate 3-4 strategic questions per PEST factor (12-16 total)
4. Create table with Number, PEST Factor, Strategic Questions, and Perplexity AI prompts
5. Ensure each Perplexity prompt includes: specific keywords, source requirements, and search operators
```

### Template B: Expand Existing PEST Plan

```
1. Review existing PEST analysis plan
2. Identify which factor(s) need expansion
3. Ask user for clarification on focus areas
4. Generate additional strategic questions and Perplexity prompts
5. Present updated table with new entries highlighted
```

### Template C: Refine Perplexity Prompts

```
1. Extract existing strategic questions from current analysis
2. Review current Perplexity prompts for quality
3. Identify areas for improvement (source quality, specificity, operators)
4. Rewrite prompts with enhanced search strategies
5. Present comparison: original vs. improved prompts
```

---

## Post-Change Checklist

After modifying THIS skill:

1. [ ] YAML frontmatter valid (name lowercase-hyphen, description has triggers)
2. [ ] Description includes WHEN to use (trigger keywords)
3. [ ] Task templates cover all common scenarios
4. [ ] Example row demonstrates proper formatting
5. [ ] Perplexity prompt instructions remain clear and actionable
6. [ ] All markdown links use relative paths (if resources added)
