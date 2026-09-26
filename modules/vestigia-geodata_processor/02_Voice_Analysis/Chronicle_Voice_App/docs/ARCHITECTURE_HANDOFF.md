
# Chronicle (App 1): Architecture Handoff

**To:** The Independent Analyst / System Integrator  
**From:** Chronicle Dev Team  
**Subject:** App 1 (User Recall) Data Structure & Provenance

---

## 1. Scope of This Application

This application (**Chronicle**) is responsible for **User Ingest**. It captures:
1.  **Voice Sessions:** Real-time memory recall.
2.  **Manual Entry:** User corrections and text input.
3.  **Entity Definitions:** Who the key players are (from the user's perspective).

It does **not** process video files (Sentinel) or mine chat logs (Anamnesis).

---

## 2. The Data Contract (Provenance)

To support your future "Merger" system, we have enforced strict data tagging.

### The `origin` Field
Every record (Event, Entity, Relationship, Fact) in our database includes an `origin` column.
*   **Value:** `'Chronicle'`
*   **Meaning:** This data came directly from the subject (The User).

### Conflict vs. Duplicate
When you build the Merger:
*   **Duplicate:** If `origin: 'Chronicle'` says "Dinner at 7 PM" and `origin: 'Sentinel'` says "Dinner at 7 PM", they confirm each other.
*   **Conflict:** If `origin: 'Chronicle'` says "I was alone" and `origin: 'Sentinel'` shows "A second person in the room", this is a Flagged Conflict.

**Our data structure supports this resolution by keeping our specific perspective isolated via the `origin` tag.**

---

## 3. Current Data Model (Types)

See `types.ts` for the exact schema. Key fields we populate:

*   **`TimelineEvent`**: The core memory.
    *   `description`: The user's narrative.
    *   `courtExplanation`: How the user explains it legally.
*   **`Entity`**: People involved.
    *   `impactOnCase`: User's subjective rating ('high', 'medium', 'low').
*   **`ContextPhase`**: The "Era" the user believes they were in (e.g., "The Love Bombing Phase").

---

## 4. Integration Note

We currently write to a Supabase table `chronicle_cases`. 
We understand this table may eventually become a "Child Table" or a specific partition in your final architecture. 
We have designed the frontend to be agnostic to this change, provided the JSON schema remains consistent.
