---
name: mi-case-research
description: Structured Michigan case law research workflow with verification. Use when searching for case authority, verifying citations, or building legal arguments requiring specific Michigan precedent.
context: fork
agent: Explore
allowed-tools: WebSearch, WebFetch, Read, Grep
---

# Michigan Case Law Research Protocol

Execute this research workflow for any Michigan family law question. Follow ALL steps - do not skip verification.

## Step 1: Frame the Research Question

- State the specific legal issue
- Identify the applicable MCL/MCR provisions
- List key search terms (legal + factual)

## Step 2: Primary Authority Search

### Statutes & Rules
1. Search for current MCL text at legislature.mi.gov
2. Check for recent amendments (last 2 years)
3. Verify MCR provisions at courts.mi.gov

### Case Law (PUBLISHED ONLY)
Search in this order:
1. **Michigan Supreme Court (MSC)**: Google Scholar → filter Michigan → Supreme Court
2. **Michigan Court of Appeals (MCOA)**: Google Scholar → filter Michigan → Court of Appeals → Published only
3. **Cross-reference**: courts.mi.gov/opinions-orders

For each case found:
```
Case: [Name], [Mich/Mich App citation]; [NW2d citation] ([year])
Docket: [number]
Holding: [1-2 sentence summary]
Relevance: [how it applies to the research question]
Status: [Current / Modified by X / Overruled by Y]
```

### CRITICAL: Verify each citation
- Check that the case actually exists (search by docket number)
- Confirm it hasn't been overruled or significantly modified
- Mark any unverifiable citation as [UNVERIFIED - MANUAL CHECK REQUIRED]

## Step 3: Secondary Authority Search

Check these sources for supporting material:
- MJI Benchbooks (Family Division, Evidence, DV)
- FOCB Policy memoranda
- MDHHS/CPS protocols
- SCAO administrative orders and forms
- ICLE publications (if accessible)

## Step 4: Opposing Authority Search

**Always research the other side:**
- Search for cases reaching opposite conclusions
- Identify distinguishing facts
- Note any trend in recent decisions
- Flag if the area of law is unsettled

## Step 5: Compile Results

Return findings in this format:

```
## RESEARCH RESULTS: [Topic]

### Research Question
[Precise question]

### Primary Authority
#### Favorable
1. [Citation + holding + relevance]

#### Unfavorable/Distinguishable
1. [Citation + holding + why distinguishable]

### Secondary Authority
1. [Source + relevant section]

### Statutory Framework
- [MCL/MCR provisions with key language]

### Research Gaps
- [What couldn't be found or verified]

### Recommended Next Steps
- [Additional research or fact development needed]
```

## HARD RULES
- NEVER cite unpublished Michigan Court of Appeals opinions as binding authority
- NEVER fabricate case names, docket numbers, or holdings
- ALWAYS flag uncertainty: "I could not independently verify this citation"
- ALWAYS note the date of your search for currency purposes
- Michigan jurisdiction ONLY - refuse federal or other state research
