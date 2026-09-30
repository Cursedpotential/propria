-- Memory duplicate guard: cosine distance alone cannot tell a reworded duplicate from an unrelated claim.
-- Byline: Claude Code · Opus 5.5 · 2026-09-28. Supersedes the vector rule of 2026-09-27-memory-remember-guard.surql
-- (cosine <= 0.20), which refused an unrelated claim at 04:44 EDT as a near-duplicate of four others.
-- Target: SurrealDB probata_memory/memory (surreal-case, ovh-files :8471).
--
-- Measured 2026-09-28 on the live store (16 active embedded rows, 120 pairs, nvidia/nemotron-3-embed-1b):
--   unrelated pairs:     cosine distance min 0.174, 5th percentile 0.216; word Jaccard max 0.26
--   reworded duplicates: cosine distance 0.061, 0.113, 0.184;             word Jaccard 0.44, 0.54, 0.56
-- Distance ranges overlap (0.174 unrelated vs 0.184 duplicate); word overlap does not. A row now conflicts if
--   cosine distance <= 0.10 (same meaning in any words; well under the unrelated floor), or
--   cosine distance <= 0.20 AND word Jaccard >= 0.35.
-- Words: lowercase, split on anything but [a-z0-9], keep words longer than 2 characters, de-duplicated.
-- BM25 (all query words present) still counts on its own. Calibration on few points; revisit as the store grows.

BEGIN TRANSACTION;

DEFINE FUNCTION OVERWRITE fn::claim_words($text: string) {
  RETURN array::distinct(
    string::words(string::replace(string::lowercase($text), <regex> "[^a-z0-9]+", " "))
      .filter(|$w| string::len($w) > 2)
  );
} COMMENT "Word set used by the memory duplicate guard: lowercase, split on non-alphanumerics, words over 2 characters." PERMISSIONS FULL;

DEFINE FUNCTION OVERWRITE fn::remember($payload: object) {
  LET $scope = $payload.scope;
  LET $claim = $payload.claim;
  LET $words = fn::claim_words($claim);

  LET $bm25_conflicts = (
    SELECT id, claim, confidence, status FROM memory
    WHERE scope = $scope AND status = "active" AND claim @@ $claim
    LIMIT 5
  );

  -- The distance is computed explicitly: vector::distance::knn() read through a subquery is NONE,
  -- and NONE <= x is true, which would readmit every neighbour.
  LET $neighbours = IF $payload.embedding != NONE {
    (SELECT id, claim, confidence, status,
            1 - vector::similarity::cosine(embedding, $payload.embedding) AS dist
     FROM memory
     WHERE scope = $scope AND status = "active"
       AND embedding <|5,40|> $payload.embedding)
  } ELSE {
    []
  };
  LET $scored = $neighbours.map(|$n| object::extend($n, {
    overlap: <float> array::len(array::intersect($words, fn::claim_words($n.claim)))
             / math::max([1, array::len(array::union($words, fn::claim_words($n.claim)))])
  }));
  LET $vec_conflicts = $scored[WHERE dist != NONE AND (dist <= 0.10 OR (dist <= 0.20 AND overlap >= 0.35))];

  LET $conflicts = array::concat($bm25_conflicts, $vec_conflicts);

  IF array::len($conflicts) > 0 AND $payload.force != true AND $payload.supersede = NONE {
    RETURN {
      written: NONE,
      conflicts: $conflicts,
      note: "Similar active memory exists in this scope (same words; cosine distance <= 0.10; or distance <= 0.20 with word overlap >= 0.35). Pass force:true to keep both, or supersede:<id> to replace one."
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
} COMMENT "Guarded memory write: refuses silent duplication (BM25 match; cosine <= 0.10; or cosine <= 0.20 with word Jaccard >= 0.35), unless force:true or supersede:<id>." PERMISSIONS FULL;

COMMIT TRANSACTION;
