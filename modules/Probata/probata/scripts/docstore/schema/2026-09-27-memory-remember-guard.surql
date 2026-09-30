-- Memory write path: a working duplicate guard and a working supersession.
-- Byline: Claude Code · Opus 5.5 · 2026-09-27 — owner, 10:01 EDT: "I want it fixed".
-- Target: SurrealDB probata_memory/memory (surreal-case, ovh-files :8471).
--
-- Two defects, both measured live on 2026-09-27 before this change:
--   1. The vector leg of fn::remember took the 5 nearest active rows with no distance cutoff, so
--      every write into a scope holding at least one embedded row was refused as a "near-duplicate".
--      An unrelated owner rule scored 0.216-0.297 cosine distance against its 5 neighbours; two
--      paraphrases of stored rules scored 0.061, 0.090, 0.113 and 0.184 (a heavy rewording); a new,
--      unrelated claim's nearest row scored 0.216-0.325. Only rows within 0.20 now count as conflicts;
--      the cutoff sits in that gap and is a calibration on 7 measured points, not a law.
--   2. fn::remember called fn::supersede_memory with 3 arguments (id, payload, reason) and that
--      function took 2, so every supersede failed with "Incorrect arguments ... expects 2 arguments".
--      supersede_memory now takes an optional reason, records it in decision_log, refuses an unknown
--      or inactive target, and accepts the target id as a string ("memory:<id>") from JSON callers.
-- observed_at is accepted as an ISO-8601 string; UNIQUE(scope, claim) is unchanged, so a supersession
-- has to change the claim text.

BEGIN TRANSACTION;

DEFINE FUNCTION OVERWRITE fn::remember($payload: object) {
  LET $scope = $payload.scope;
  LET $claim = $payload.claim;

  LET $bm25_conflicts = (
    SELECT id, claim, confidence, status FROM memory
    WHERE scope = $scope AND status = "active" AND claim @@ $claim
    LIMIT 5
  );

  -- The distance is computed explicitly: vector::distance::knn() read through a subquery is NONE,
  -- and NONE <= 0.20 is true, which would readmit every neighbour.
  LET $neighbours = IF $payload.embedding != NONE {
    (SELECT id, claim, confidence, status,
            1 - vector::similarity::cosine(embedding, $payload.embedding) AS dist
     FROM memory
     WHERE scope = $scope AND status = "active"
       AND embedding <|5,40|> $payload.embedding)
  } ELSE {
    []
  };
  LET $vec_conflicts = $neighbours[WHERE dist != NONE AND dist <= 0.20];

  LET $conflicts = array::concat($bm25_conflicts, $vec_conflicts);

  IF array::len($conflicts) > 0 AND $payload.force != true AND $payload.supersede = NONE {
    RETURN {
      written: NONE,
      conflicts: $conflicts,
      note: "Similar active memory exists in this scope (same words, or cosine distance <= 0.20). Pass force:true to keep both, or supersede:<id> to replace one."
    };
  };

  IF $payload.supersede != NONE {
    RETURN fn::supersede_memory(
      type::record($payload.supersede),
      $payload,
      ($payload.reason OR "fn::remember: explicit supersede")
    );
  };

  LET $new = (CREATE ONLY memory SET
    kind        = $payload.kind,
    claim       = $payload.claim,
    detail      = $payload.detail,
    evidence    = $payload.evidence,
    scope       = $payload.scope,
    agent       = $payload.agent,
    confidence  = ($payload.confidence OR 0.6),
    observed_at = IF $payload.observed_at != NONE { <datetime> $payload.observed_at } ELSE { time::now() },
    embedding   = $payload.embedding
  );
  RETURN { written: $new.id, conflicts: [] };
} COMMENT "Guarded memory write: refuses silent duplication (BM25 match, or vector cosine distance <= 0.20), unless force:true or supersede:<id>." PERMISSIONS FULL;

DEFINE FUNCTION OVERWRITE fn::supersede_memory($old: record<memory>, $new_payload: object, $reason: option<string>) {
  LET $current = (SELECT status FROM ONLY $old);
  IF $current = NONE { THROW "supersede target " + <string> $old + " does not exist" };
  IF $current.status != "active" { THROW "supersede target " + <string> $old + " is " + $current.status + ", not active" };
  LET $new = (CREATE ONLY memory SET
    kind        = $new_payload.kind,
    claim       = $new_payload.claim,
    detail      = $new_payload.detail,
    evidence    = $new_payload.evidence,
    scope       = $new_payload.scope,
    agent       = $new_payload.agent,
    confidence  = ($new_payload.confidence OR 0.6),
    observed_at = IF $new_payload.observed_at != NONE { <datetime> $new_payload.observed_at } ELSE { time::now() },
    embedding   = $new_payload.embedding
  );
  RELATE ($new.id)->supersedes->$old SET at = time::now();
  UPDATE $old SET status = "superseded";
  CREATE decision_log SET
    subject     = $old,
    actor       = ($auth.id OR "system"),
    action      = "memory_superseded",
    from_status = "active",
    to_status   = "superseded",
    reason      = ($reason OR "superseded by " + <string> $new.id);
  RETURN { written: $new.id, superseded: $old, conflicts: [] };
} COMMENT "Correct a memory without ever destroying it: new row, RELATE edge, old row flipped to superseded, reason in decision_log. Target must exist and be active." PERMISSIONS FULL;

COMMIT TRANSACTION;
