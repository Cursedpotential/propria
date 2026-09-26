---
name: quality-checker
description: Run the 14-item legal research quality gate and decide pass/fail with remediation steps.
---

# Quality Checker

Use this skill at Step 8 before delivery.

## Pre-flight Checks (before running the checklist)

Before running the 13-item checklist, verify these prerequisites:

0. For legal opinion or formal memo deliverables, `python3 scripts/validate_legal_opinion.py <draft.md> --json` passes. If an audited markdown appendix has already been produced, run it with `--require-audit`.
1. All 8 mandatory output sections are present (Scope, Conclusion, Issue tree, Analysis, Counter-analysis, Practical implications, Bibliography, Verification guide)
2. `output/claim-registry.json` exists and `contradicted_count == 0`
3. Citation numbering follows grade-priority order (A sources get lowest numbers)
4. Temporal status tags are applied inline (not only in summary)
5. All `[Unverified]` tags are inline at specific findings
6. **Source laundering check:** No conclusion relies solely on a secondary source cited as if it were primary authority. Every key conclusion's supporting citation chain must trace back to a directly-fetched primary source.
7. **Similar-statute disambiguation:** claim registry에 `similar_statutes` 필드가 있는 `operative_language` 앵커가 있으면, Phase 3.3 교차검증이 완료되었고 모든 MISMATCH가 해소되었는지 확인.

If any pre-flight check fails, remediate before proceeding to the checklist.

## 13-Item Checklist

1. Every key conclusion has **Grade A or B primary-authority** source support — not merely secondary commentary, regardless of the secondary source's grade.
2. Legal hierarchy is not conflated.
3. Amendment/effective dates and currency are checked; temporal tags applied where relevant (`[Recently Amended]`, `[Pending Amendment]`, `[Not Yet In Force]`, `[Repealed]`).
4. Jurisdiction level is accurate.
5. Uncertain claims are clearly marked.
6. Pinpoints and verification guide are present.
7. No D-grade source is sole basis of any conclusion.
8. Every key conclusion has at least one counter-argument or risk scenario (per `references/counter-analysis-checklist.md`).
9. Practical Implications section is present and addresses client-actionable takeaways.
10. Executive summary / conclusion summary is consistent with the detailed analysis — no conclusion in the summary that is unsupported or contradicted in the body.
11. Every key conclusion is supported by at least one Grade A or Grade B source (not only C/D).
12. No `[Material Risk]` finding is omitted from the executive summary / conclusion summary.
13. **No source laundering detected:** No secondary source is cited as if it were primary authority. All secondary source citations use transparent attribution (e.g., "According to [Source]'s analysis..."). No conclusion relies on a paraphrased interpretation of a primary source without the primary source itself having been directly fetched and verified. Any `laundering_risk: true` flag from Step 4/5 has been resolved or the claim has been re-attributed.
14. **Quoted statutory text matches primary source verbatim:** 최종 output에 포함된 법문 직접 인용(인용부호 내 법문, 또는 operative text로 제시된 법령 문구)에 대해, 관할별 최소 1건을 Step 3에서 fetch한 1차 소스 텍스트와 word-level 대조. 인용 문구가 cited subsection에 존재하지 않으면 fail. claim registry에 status `Contradicted`인 `operative_language` 앵커가 있으면 이 항목은 자동 fail.

## Output Format

```json
{
  "quality_gate": "pass|fail",
  "failed_items": [1, 4],
  "remediation_plan": [
    "Re-enter Step 3 for issue X",
    "Patch section Y with new source Z"
  ]
}
```

## Remediation Policy

- Round 1: collect additional sources for failing items.
- Round 2: patch only failing sections.
- If still failing: deliver with `[Unverified]` tags.
