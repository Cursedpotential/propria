---
title: "As-Lived, Hindsight, and Foreshadowing — Owner-Approved Use Case"
status: owner-approved-use-case
approved_on: 2026-10-17
scope: "Propria review annotations and downstream perspective-correct analysis"
authority: "Owner's explicit acceptance of the preceding explanation and request to commit it as a source of truth"
implementation_status: "Requirements and intended behavior; not a claim of complete implementation or live verification"
source_issue: "https://github.com/Cursedpotential/propria/issues/12#issuecomment-6232094656"
---

# As-Lived, Hindsight, and Foreshadowing

## Authority and purpose

This is the durable, owner-approved explanation of the intended use case. It preserves the substance and examples of the explanation in the Workbench Rebuild Plan conversation, rather than replacing them with a new architecture or a generic temporal-database description.

The owner explicitly endorsed that explanation and requested this record:

> That might be the best explanation of like my intended use case for the whole hindsight. Uh as live walk thing. Can you commit? That right up. As some sort of a source of truth. Somewhere where it's relevant.

Read this alongside [PROJECT_CANON.md](PROJECT_CANON.md) and the existing source-availability and realization contracts. This document explains the intended experience and its required knowledge boundaries. It does not change the database stack, authorize deployments or migrations, or introduce additional approval screens for routine processing. Historical implementation references below are not evidence that the full experience works today.

_Byline: ChatGPT · 2026-10-17 · owner's approved explanation preserved as project documentation._

## 1. The intended experience

**The expanded card should handle observations and contributions—not just metadata:** potential graph connections, schedule mentions, events, timeline additions or corrections, and why something matters to the relationship.

**Foreshadowing needs to preserve what you know now without telling an as-lived agent what it could not have known then.**

The owner should be able to review the material with the benefit of hindsight, capture connections and significance, and retrieve those observations later. Separately, an agent walking through the material from a historical perspective must encounter only the information available to that perspective at that point. The two views serve different purposes; hiding a label on the screen is not enough to make them separate.

## 2. What you should be able to add while reviewing

The compact card can expose a few quick actions, with the details opening underneath:

| Action | What it lets you record |
|---|---|
| **Mark significant** | Identify something as important to a relationship, topic, or sequence of events, with an optional explanation. Distinguish significance apparent then from significance recognized afterward. |
| **Add or link an event** | Create an event from the selected passage, associate it with an existing event, or add another supporting source without creating a duplicate event. |
| **Capture a schedule mention** | Record a proposed arrangement, agreement, appointment, change, cancellation, or later report of what happened. Preserve the distinction between a plan and its outcome. |
| **Contribute to the timeline or graph** | Correct a date or sequence, add participants, connect related events, or propose a relationship such as “supports,” “contradicts,” “reschedules,” or “possibly contributes to.” |
| **Mark foreshadowing** | Flag an earlier message or event whose significance becomes apparent later. Add a note and, optionally, connect it to the later event or realization. |

These actions retain the selected message, passage, attachment, or unit as their source. A potential connection can remain potential; you should not have to declare it an established fact just to save it. The example relation labels describe useful actions, not a new mandatory global ontology.

**Marking foreshadowing should be quick:** turn it on and continue reading. The explanation and later-event association can be added afterward. You should also be able to edit the note without having to toggle the flag off and on.

This remains part of the compact editable card and expanding inspector in the selected-unit workspace. It does not displace the SBV-centered messaging reader, lose the selected message or attachments, or restart source discovery.

## 3. The crucial distinction: hide the hindsight, not the original

Suppose you are reviewing an old conversation and recognize:

> “This exchange becomes important later. At the time, I did not understand its significance.”

That creates two separate things:

**The original exchange:** what was actually said, with its original participants, dates, attachments, and context.

**Your retrospective annotation:** the foreshadowing flag, what you now understand, and any connection to a later event.

An as-lived agent should still encounter the original exchange when it reaches the point where that exchange was available to the selected perspective. **It must not receive your retrospective flag or explanation.**

Otherwise, either of two mistakes would spoil the walk:

- Removing the whole flagged message would deprive the agent of information that really was available then.
- Showing the message with “important later,” a future-event link, or a hindsight-based significance score would give the agent knowledge it had not reached yet.

**Even boosting that message’s search ranking because you marked it as foreshadowing would influence the walk with hindsight.** The separation has to cover selection and ordering, not merely hide the badge.

A generic “Relevant” label is not automatically safe for the historical perspective. Its basis may itself be retrospective. The same concern applies to labels such as “turning point,” later-resolved identities, and connections discovered only after later information became available.

## 4. Timeline editing needs the same distinction

The system needs to distinguish **when something happened or was planned, when the selected person could know about it, and when you entered the annotation or correction**. Preserve the original source wording and the basis for interpreted dates; these times are not interchangeable.

For example, in an illustrative sequence:

On the first day, a message says, “We are meeting next Friday.” The as-lived agent can know about that **plan immediately**, even though Friday is still in the future.

Several days later, another message cancels the meeting. The cancellation becomes available when that message becomes knowable—not when the original plan was made.

Months later, you mark the first exchange as a turning point. That interpretation belongs in hindsight. It must not rewrite the earlier timeline into “the beginning of the breakdown” for an agent that has only reached the first day.

The same applies to a graph connection or contact identification that depends on information learned later. **A later-discovered relationship should not silently become something the earlier perspective already knew.**

The example does not mean every future-dated event is hidden. The plan is already known; the later cancellation and outcome are not. Likewise, software import or annotation time is not a substitute for historical knowledge availability. Preserve the existing distinction between source availability, event time, and separate realizations.

A later source fact may become available as the walk advances. That does not automatically turn the owner's retrospective interpretation into contemporaneous knowledge. A suspicion that was possible then remains a suspicion, not the later-confirmed conclusion.

## 5. You see both; the walk receives only its perspective

In your review workspace, you should be able to filter for **Foreshadowing**, **Significant to this relationship**, **Timeline contributions**, or **Unresolved connections**, then open each item with its original context and your annotations.

For an as-lived walk, the separation must happen **before information reaches the agent**. It must cover the retrieved text, chunks, graph connections, summaries, timeline entries, and saved memory—not just the visible page. A source-only chunk should not have future-aware notes mixed into its embedding and then pretend to be clean because the notes are hidden.

The applicable horizon is therefore enforced before retrieval/ranking and context assembly. Future-aware annotations must not influence the eligible source set, ordering, graph traversal, resolved labels, summaries, cached responses, or memory delivered to the walk. The original, eligible source remains usable.

Likewise, an agent session that has already read the hindsight material cannot become an uninformed historical observer merely by changing a filter. The walk needs its own clean context, retaining only what it has learned up to that point. Normal continuity within that clean walk should be preserved; this is not a requirement to forget all earlier eligible observations on every step.

**None of this requires more approval screens for you.** The interface remains a quick flag and an expandable editor. The underlying system does the work of keeping the perspectives separate. The review page supplies annotations; the actual walk controls remain in the analysis workspace.

Routine discovery, extraction and embedding continue. The knowledge boundary determines which representation an analysis agent receives, not whether a source may be imported or embedded. No new duplicate authored as-lived/hindsight stores, evidence promotion rules, or Surreal analytical inclusion rules are established by this document.

## 6. What successful behavior must demonstrate

The visible checkbox alone is not acceptance. A focused example must show that:

1. The owner can mark, explain, edit, clear and filter foreshadowing, associate the source with an event or relationship, and reopen the saved result.
2. At an earlier walk point, the eligible original exchange and any already-known plan remain present.
3. The retrospective note, flag, future-event association, hindsight-based relevance and future-influenced ordering are absent from the actual agent inputs and its tool/memory responses.
4. Later source information becomes available only when it is knowable to that perspective; the earlier state is not rewritten with the later outcome.
5. Timeline corrections, contact/alias associations, chunks, summaries and graph connections obey the same distinction.
6. A clean walk is not resumed with a memory or conversation context that already contains the withheld hindsight.

Report these checks against the actual data paths. Do not substitute a build, a screenshot, a successful deployment, or a schema file for verified perspective-correct behavior.

## 7. Existing foundation and verification limits

The preceding explanation identified existing code at repository commit `c9c6604789683ccf10bfc065f87cf935093757f1`. These are integration references, not a live deployment attestation:

- [Context review UI](https://github.com/Cursedpotential/propria/blob/c9c6604789683ccf10bfc065f87cf935093757f1/modules/Probata/probata/modules/workbench/web/src/components/review/context-review.tsx): a per-message **Foreshadowing** checkbox, optional note, and separate hindsight-only write path already exist. The note is currently saved with a flag change; the intended editor allows independent note revision.
- [Context review service](https://github.com/Cursedpotential/propria/blob/c9c6604789683ccf10bfc065f87cf935093757f1/modules/Probata/probata/modules/workbench/api/app/service/context_review.py): non-hindsight reads remove the foreshadowing member on this endpoint. That is not proof that all search, graph, and agent-memory paths are isolated.
- [Context review SQL design](https://github.com/Cursedpotential/propria/blob/c9c6604789683ccf10bfc065f87cf935093757f1/modules/Probata/probata/scripts/2026-09-25-context-review-overlay.sql): a separate `context.record_foreshadowing_flag` and horizon-filtered views are declared. The file's historical header says **NOT APPLIED**; the file alone proves neither applied nor absent in the current database.

An important field-level distinction in that SQL is explicit: the flag's `knowledge_time` means **when the flag was recorded**, not when the underlying information became knowable. Preserve that meaning rather than repurposing the field for a different temporal purpose. Map the actual existing schema before proposing implementation changes.

## 8. Source record and maintenance

The basis is the owner's explanation and explicit approval in the Workbench Rebuild Plan conversation on 2026-10-17. The requirements were also recorded in [issue #12, foreshadowing refinement](https://github.com/Cursedpotential/propria/issues/12#issuecomment-6232094656), following the [compact-card and association refinement](https://github.com/Cursedpotential/propria/issues/12#issuecomment-6221896157).

This file is the stable prose reference for the intended as-lived/hindsight/foreshadowing use case. Indexes, handoffs and agent memories should link here rather than maintain divergent paraphrases. Preserve the original-versus-annotation distinction and the schedule example in future revisions. Record any owner-authorized change explicitly; do not silently broaden this explanation into an unrelated global restriction.

**In one sentence: keep what was knowable then intact, keep what you learned later available for hindsight, and never let the latter quietly influence the former.**
