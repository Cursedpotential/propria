# D-154 Agent-Memory Schema: Build and Local Verification

> Byline: Claude Code (general-purpose subagent) · Sonnet 5 · 2026-09-09

## 1. Scope and ruling followed

Per **D-154**: one self-hosted SurrealDB shared by every agent, our own
schema (not the walk-memory / Agent-Memory-product design). This pass:

1. Started from the kit's `080_memory.surql` /  `000_analyzers.surql`
   (`C:/Users/matts/.claude/jobs/68afe1c5/tmp/surreal-docstore/surreal-docstore/schema/`)
   and read `M-agent-memory-cookbooks.md` sections C, D, F for the sharing /
   principal / reflection / supersession patterns to reproduce (not the
   Agent Memory product itself -- that was independently ruled out as
   non-free/non-self-hostable in the walk material).
2. Wrote four new SurrealQL files, `EMBED_DIM = 2048` substituted literally
   throughout, plus a curl-based apply script -- all under
   `C:/Users/matts/.claude/jobs/68afe1c5/tmp/memory-schema/`.
3. Applied and exercised them on the **local** SurrealDB engine
   (`http://127.0.0.1:8000`, SurrealDB **3.2.0** -- the task brief said
   3.2.4 for the VPS; the local engine actually reports 3.2.0, noted here
   as a real discrepancy, not corrected silently) in a **scratch**
   namespace `scratch_memory`, database `memory`.
4. Applied by POSTing whole `.surql` files to `/sql` with HTTP Basic auth
   and `surreal-ns` / `surreal-db` / `Accept: application/json` headers --
   never `surreal sql` stdin, never `surreal import` -- per the task's
   explicit constraint. Credentials came from
   `E:/AI_Workspace/Projects/the-platform-workspace/probata/.docstore/.env`
   (`SURREAL_USER` / `SURREAL_PASS`), read with a tolerant Python regex
   parser (never `source`d), and are never printed anywhere in this report
   or in the commands below -- only lengths were ever inspected.
5. Ran every write/read from Bash via `curl`/Python `urllib` (no `surreal`
   CLI is installed on this machine -- confirmed with `which surreal` /
   `surreal version`, both failed with "command not found").
6. Removed the scratch namespace at the end and confirmed removal via
   `INFO FOR ROOT`.

The VPS (`100.91.190.107`) was never touched. No state-changing git command
was run. All new files live under
`C:/Users/matts/.claude/jobs/68afe1c5/tmp/memory-schema/`.

---

## 2. Skills consulted

Per the coordinator's mid-task correction, the following official SurrealDB
skills were loaded and used to verify syntax/behavior **before** finalizing
085/087 (in addition to the local live-engine probing this task already
required): `surrealql`, `surrealql-functions`, `surrealql-performance`,
`surrealdb-vector`, `surrealkit`, and the newly-appeared master `surrealdb`
skill (its `rules/vector-search.md`, `rules/security.md`, `rules/surrealql.md`,
`rules/gotchas.md`).

- `surrealql`/`surrealql-performance`: confirmed `EXPLAIN` / `EXPLAIN FULL`
  go at the **end** of a `SELECT` statement, not before it (an early attempt
  with `EXPLAIN FULL SELECT ...` 400'd with a parse error -- fixed).
- `surrealdb-vector` / the master skill's `rules/vector-search.md`: confirmed
  the `<|K, EF|>` KNN operator takes **literal integers** for K/EF in every
  documented example (no example anywhere binds K to a variable) -- this is
  exactly the "K literal" requirement in the task brief, and is why
  `fn::recall` uses a fixed `<|20,40|>` candidate pool and applies the
  caller's `$k` only as the final `LIMIT`.
- `surrealdb-vector`'s hybrid-search section states plainly: **"Application-
  level fusion is recommended for proper score normalization"** -- i.e.
  there is no built-in RRF function. `fn::recall`'s RRF fusion is therefore
  hand-rolled (Section 5.4 below).
- `surrealkit`: read for completeness per the coordinator's instruction to
  consider it for the apply script. **Not adopted** -- the task explicitly
  and repeatedly mandates "POSTing whole files to `/sql` with basic auth and
  headers ... (not `surreal sql` stdin, not `surreal import`)"; SurrealKit's
  own sync/rollout mechanism is a different apply path (and requires
  installing a separate `surrealkit` binary, which is not present on this
  machine). `apply_memory_schema.sh` stays curl-only per the explicit
  instruction.
- `surrealdb` master skill's `rules/security.md`: confirmed the `DEFINE
  ACCESS ... TYPE RECORD` pattern (SIGNIN as a `SELECT` expression against a
  user/principal table, `crypto::argon2::compare`) and that **omitting
  `SIGNUP` entirely** is the documented way to disable self-registration
  (matches the kit's own `070_access.surql`, which does the same).

---

## 3. Files written

All under `C:/Users/matts/.claude/jobs/68afe1c5/tmp/memory-schema/`.

### 3.1 `000_analyzers.surql` (copied verbatim from the kit)

```surql
-- 000_analyzers.surql
-- Analyzers must exist before any FULLTEXT index references them.
-- Copied verbatim from surreal-docstore kit schema/000_analyzers.surql.

DEFINE ANALYZER OVERWRITE doc_text
  TOKENIZERS class
  FILTERS lowercase, ascii, snowball(english);

-- Code/identifier-friendly analyzer: no stemming, keeps camelCase splits usable.
DEFINE ANALYZER OVERWRITE doc_code
  TOKENIZERS class, punct
  FILTERS lowercase, ascii;
```


### 3.2 `080_memory.surql`

```surql
-- 080_memory.surql
-- D-154: one self-hosted SurrealDB shared by every agent. Our own schema,
-- not the walk-memory/Agent-Memory-product design. Sharing is done the way
-- the SurrealDB cookbooks do it (M-agent-memory-cookbooks.md secs C/D/F):
-- a hierarchical scope-path STRING on every row (`probata/<domain>/<agent>`),
-- filtered at query time with a string-prefix predicate -- not a product
-- feature, just a schema + WHERE clause. Supersession is never a DELETE or
-- overwrite: fn::supersede_memory (085) creates a new row, RELATEs it to the
-- old one, and flips the old row's status -- full chain stays inspectable.
-- Adapted from the surreal-docstore kit's schema/080_memory.surql
-- (single-project "librarian" store) to a shared, scope-keyed, multi-agent
-- store per D-154. EMBED_DIM = 2048 substituted literally throughout.

DEFINE TABLE OVERWRITE memory TYPE NORMAL SCHEMAFULL
  CHANGEFEED 180d INCLUDE ORIGINAL;

DEFINE FIELD OVERWRITE kind ON memory TYPE string
  ASSERT $value IN ["correction","preference","observation","handoff","fact","constraint","decision"];

DEFINE FIELD OVERWRITE claim ON memory TYPE string
  ASSERT string::len($value) > 10 AND string::len($value) < 600;

DEFINE FIELD OVERWRITE detail ON memory TYPE option<string>;

-- evidence is always a plain string: a doc id + source path (e.g.
-- "docstore:9f2a#/reports/q3.md"), NEVER a live SurrealDB record link --
-- links can't safely cross store/namespace boundaries and would break the
-- moment the referenced store is re-ingested or moved.
DEFINE FIELD OVERWRITE evidence ON memory TYPE option<string>
  COMMENT "doc id + source path string; never a live record link across stores";

-- scope is the sharing primitive (cookbook sec C/D): identical scope
-- strings are how independent agents/sessions see each other's writes.
-- Path form: probata/<domain>/<agent>, e.g. probata/docstore/librarian.
DEFINE FIELD OVERWRITE scope ON memory TYPE string
  ASSERT string::matches($value, '^probata(/[a-z0-9_-]+)*$')
  COMMENT "hierarchical scope path, form probata/<domain>/<agent>";

DEFINE FIELD OVERWRITE agent ON memory TYPE string DEFAULT "unknown";

DEFINE FIELD OVERWRITE status ON memory TYPE string
  ASSERT $value IN ["active","superseded","retracted"] DEFAULT "active";

DEFINE FIELD OVERWRITE confidence ON memory TYPE float
  ASSERT $value >= 0.0 AND $value <= 1.0 DEFAULT 0.6;

DEFINE FIELD OVERWRITE observed_at ON memory TYPE datetime DEFAULT time::now();
DEFINE FIELD OVERWRITE created ON memory TYPE datetime VALUE time::now() READONLY;
DEFINE FIELD OVERWRITE updated ON memory TYPE datetime VALUE time::now();
DEFINE FIELD OVERWRITE embedding ON memory TYPE option<array<float, 2048>>;

DEFINE INDEX OVERWRITE memory_kind          ON memory FIELDS kind;
DEFINE INDEX OVERWRITE memory_status        ON memory FIELDS status;
DEFINE INDEX OVERWRITE memory_scope         ON memory FIELDS scope;
DEFINE INDEX OVERWRITE memory_scope_status_kind ON memory FIELDS scope, status, kind;
DEFINE INDEX OVERWRITE memory_claim_uq      ON memory FIELDS scope, claim UNIQUE;
DEFINE INDEX OVERWRITE memory_claim_ft      ON memory FIELDS claim
  FULLTEXT ANALYZER doc_text BM25 HIGHLIGHTS;
DEFINE INDEX OVERWRITE memory_vec ON memory
  FIELDS embedding HNSW DIMENSION 2048 DIST COSINE TYPE F32
  EFC 150 M 12 CONCURRENTLY;

-- Every status change (e.g. active -> superseded, active -> retracted) is
-- logged, never silently dropped. This is the audit trail for "supersession
-- never overwrite" (D-154 / cookbook sec F "reflect(persist)" pattern note:
-- writes are additive, chains stay inspectable).
DEFINE EVENT OVERWRITE memory_status_changed ON TABLE memory
  WHEN $event = "UPDATE" AND $before.status != $after.status THEN (
    CREATE decision_log SET
      subject = $after.id,
      actor = ($auth.id OR "system"),
      action = "memory_status_changed",
      from_status = $before.status,
      to_status = $after.status
  );

-- ---------------------------------------------------------------------
-- decision_log: append-only audit trail. Never updated, never deleted.
DEFINE TABLE OVERWRITE decision_log TYPE NORMAL SCHEMAFULL;

DEFINE FIELD OVERWRITE subject     ON decision_log TYPE record<memory>;
DEFINE FIELD OVERWRITE actor       ON decision_log TYPE string;
DEFINE FIELD OVERWRITE action      ON decision_log TYPE string;
DEFINE FIELD OVERWRITE from_status ON decision_log TYPE option<string>;
DEFINE FIELD OVERWRITE to_status   ON decision_log TYPE option<string>;
DEFINE FIELD OVERWRITE reason      ON decision_log TYPE option<string>;
DEFINE FIELD OVERWRITE created     ON decision_log TYPE datetime VALUE time::now() READONLY;

DEFINE INDEX OVERWRITE decision_log_subject ON decision_log FIELDS subject;

-- ---------------------------------------------------------------------
-- episode: raw session notes, the reflection-loop input (cookbook sec F:
-- reflect() is an explicit, app-triggered synthesis pass over episodes,
-- never automatic). Condensing episodes into memory rows is the agent's
-- job; fn::reflect (085) only selects and marks episodes reflected=true.
DEFINE TABLE OVERWRITE episode TYPE NORMAL SCHEMAFULL;

DEFINE FIELD OVERWRITE scope ON episode TYPE string
  ASSERT string::matches($value, '^probata(/[a-z0-9_-]+)*$')
  COMMENT "hierarchical scope path, form probata/<domain>/<agent>";
DEFINE FIELD OVERWRITE agent      ON episode TYPE string DEFAULT "unknown";
DEFINE FIELD OVERWRITE session_id ON episode TYPE string;
DEFINE FIELD OVERWRITE text       ON episode TYPE string;
DEFINE FIELD OVERWRITE reflected  ON episode TYPE bool DEFAULT false;
DEFINE FIELD OVERWRITE created    ON episode TYPE datetime VALUE time::now() READONLY;
DEFINE FIELD OVERWRITE embedding  ON episode TYPE option<array<float, 2048>>;

DEFINE INDEX OVERWRITE episode_scope     ON episode FIELDS scope;
DEFINE INDEX OVERWRITE episode_session   ON episode FIELDS session_id;
DEFINE INDEX OVERWRITE episode_reflected ON episode FIELDS scope, reflected;
DEFINE INDEX OVERWRITE episode_text_ft   ON episode FIELDS text
  FULLTEXT ANALYZER doc_text BM25 HIGHLIGHTS;
DEFINE INDEX OVERWRITE episode_vec ON episode
  FIELDS embedding HNSW DIMENSION 2048 DIST COSINE TYPE F32
  EFC 150 M 12 CONCURRENTLY;
```


### 3.3 `085_memory_functions.surql`

```surql
-- 085_memory_functions.surql
-- D-154 shared-memory function surface. Guarded writes, explicit
-- supersession (never overwrite), explicit retraction (never DELETE),
-- reflection as a manual/app-triggered synthesis pass (cookbook sec F:
-- reflect() is never automatic), and RRF recall over BM25 + KNN.
--
-- KNN literal-K note (verified empirically against the local 3.2.0 engine,
-- 2026-09-09): the `<|K, EF|>` operator requires K and EF as query-literal
-- integers -- passing a bound variable for K is not how the operator is
-- used anywhere in the SurrealDB docs/skill examples. fn::recall therefore
-- uses a fixed literal candidate pool (<|20,40|>) for the KNN leg and
-- applies the caller's $k only as the final LIMIT after RRF fusion.
--
-- RRF fusion note: SurrealQL has no built-in RRF/hybrid-search function
-- (confirmed against surrealdb-vector skill: "Application-level fusion is
-- recommended for proper score normalization"). Fusion is implemented here
-- with array::find_index() rank lookups against the two ordered candidate
-- id-arrays, since SurrealQL has no map/reduce; not-found returns NONE by
-- experiment (`array::find_index([1,2,3], 5)` -> NONE), which the IF-guard
-- reads directly.

-- ---------------------------------------------------------------------
DEFINE FUNCTION OVERWRITE fn::remember($payload: object) {
  LET $scope = $payload.scope;
  LET $claim = $payload.claim;

  LET $bm25_conflicts = (
    SELECT id, claim, confidence, status FROM memory
    WHERE scope = $scope AND status = "active" AND claim @@ $claim
    LIMIT 5
  );

  LET $vec_conflicts = IF $payload.embedding != NONE {
    (SELECT id, claim, confidence, status, vector::distance::knn() AS dist
     FROM memory
     WHERE scope = $scope AND status = "active"
       AND embedding <|5,40|> $payload.embedding)
  } ELSE {
    []
  };

  LET $conflicts = array::concat($bm25_conflicts, $vec_conflicts);

  IF array::len($conflicts) > 0 AND $payload.force != true AND $payload.supersede = NONE {
    RETURN {
      written: NONE,
      conflicts: $conflicts,
      note: "Similar active memory exists in this scope. Pass force:true or supersede:<id> to proceed."
    };
  };

  IF $payload.supersede != NONE {
    RETURN fn::supersede_memory(
      $payload.supersede,
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
    observed_at = ($payload.observed_at OR time::now()),
    embedding   = $payload.embedding
  );
  RETURN { written: $new.id, conflicts: [] };
} COMMENT "Guarded memory write: refuses silent duplication (BM25 + vector conflict check), unless force:true or supersede:<id>." PERMISSIONS FULL;

-- ---------------------------------------------------------------------
DEFINE FUNCTION OVERWRITE fn::supersede_memory($old: record<memory>, $new_payload: object) {
  LET $new = (CREATE ONLY memory SET
    kind        = $new_payload.kind,
    claim       = $new_payload.claim,
    detail      = $new_payload.detail,
    evidence    = $new_payload.evidence,
    scope       = $new_payload.scope,
    agent       = $new_payload.agent,
    confidence  = ($new_payload.confidence OR 0.6),
    observed_at = ($new_payload.observed_at OR time::now()),
    embedding   = $new_payload.embedding
  );
  RELATE ($new.id)->supersedes->$old SET at = time::now();
  UPDATE $old SET status = "superseded";
  RETURN { new: $new, old: (SELECT * FROM ONLY $old) };
} COMMENT "Correct a memory without ever destroying it: new row, RELATE edge, old row flipped to superseded. Chain stays inspectable via ->supersedes->." PERMISSIONS FULL;

-- ---------------------------------------------------------------------
DEFINE FUNCTION OVERWRITE fn::forget($id: record<memory>, $reason: string) {
  LET $updated = (UPDATE ONLY $id SET status = "retracted");
  -- Supplemental, reason-bearing audit row. memory_status_changed (080)
  -- also fires on this UPDATE and writes its own reason-less row -- two
  -- decision_log rows for one retraction is intentional (full audit trail,
  -- append-only, never overwritten).
  CREATE decision_log SET
    subject     = $id,
    actor       = ($auth.id OR "system"),
    action      = "memory_forgotten",
    from_status = NONE,
    to_status   = "retracted",
    reason      = $reason;
  RETURN $updated;
} COMMENT "Retract a memory (status = retracted) with a reason. Never DELETEs -- the row and its history stay queryable." PERMISSIONS FULL;

-- ---------------------------------------------------------------------
DEFINE FUNCTION OVERWRITE fn::recall(
  $query: string, $vec: option<array<float>>, $scope: string, $k: int
) {
  LET $bm25 = (
    SELECT id, claim, kind, confidence, observed_at, evidence,
           search::score(1) AS score
    FROM memory
    WHERE status = "active"
      AND (scope = $scope OR string::starts_with(scope, $scope + "/"))
      AND claim @1@ $query
    ORDER BY score DESC
    LIMIT 20
  );

  LET $vecres = IF $vec != NONE {
    (SELECT id, claim, kind, confidence, observed_at, evidence,
            vector::distance::knn() AS dist
     FROM memory
     WHERE status = "active"
       AND (scope = $scope OR string::starts_with(scope, $scope + "/"))
       AND embedding <|20,40|> $vec
     ORDER BY dist ASC)
  } ELSE {
    []
  };

  LET $bm25_ids = $bm25.id;
  LET $vec_ids  = $vecres.id;
  LET $all_ids  = array::distinct(array::concat($bm25_ids, $vec_ids));
  LET $rrf_k    = 60;

  LET $fused = (
    SELECT id, claim, kind, confidence, observed_at, evidence, status,
      ((IF array::find_index($bm25_ids, id) != NONE {
          1.0 / ($rrf_k + array::find_index($bm25_ids, id) + 1)
        } ELSE { 0.0 })
       +
       (IF array::find_index($vec_ids, id) != NONE {
          1.0 / ($rrf_k + array::find_index($vec_ids, id) + 1)
        } ELSE { 0.0 })) AS rrf_score
    FROM $all_ids
    WHERE status = "active"
    ORDER BY rrf_score DESC
    LIMIT $k
  );

  RETURN $fused;
} COMMENT "RRF fusion of BM25 (claim @1@) and KNN (literal K=20, EF=40) over active claims whose scope equals or is a prefix-descendant of $scope. Final result truncated to caller's $k." PERMISSIONS FULL;

-- ---------------------------------------------------------------------
DEFINE FUNCTION OVERWRITE fn::reflect($scope: string, $since: datetime) {
  LET $eps = (
    SELECT * FROM episode
    WHERE (scope = $scope OR string::starts_with(scope, $scope + "/"))
      AND created >= $since
      AND reflected = false
    ORDER BY session_id, created ASC
  );
  UPDATE episode SET reflected = true WHERE id IN $eps.id;
  RETURN $eps;
} COMMENT "Selects unreflected episodes since $since for the given scope (and its descendants) and marks them reflected. Condensing into memory rows is the calling agent's job, not this function's." PERMISSIONS FULL;

-- ---------------------------------------------------------------------
DEFINE FUNCTION OVERWRITE fn::memory_stats($scope: string) {
  RETURN {
    scope: $scope,
    total: (SELECT count() FROM memory
            WHERE scope = $scope OR string::starts_with(scope, $scope + "/")
            GROUP ALL)[0].count,
    active: (SELECT count() FROM memory
             WHERE (scope = $scope OR string::starts_with(scope, $scope + "/"))
               AND status = "active"
             GROUP ALL)[0].count,
    superseded: (SELECT count() FROM memory
                 WHERE (scope = $scope OR string::starts_with(scope, $scope + "/"))
                   AND status = "superseded"
                 GROUP ALL)[0].count,
    retracted: (SELECT count() FROM memory
                WHERE (scope = $scope OR string::starts_with(scope, $scope + "/"))
                  AND status = "retracted"
                GROUP ALL)[0].count,
    by_kind: (SELECT kind, count() AS n FROM memory
              WHERE (scope = $scope OR string::starts_with(scope, $scope + "/"))
                AND status = "active"
              GROUP BY kind),
    episodes_pending_reflection: (SELECT count() FROM episode
                                   WHERE (scope = $scope OR string::starts_with(scope, $scope + "/"))
                                     AND reflected = false
                                   GROUP ALL)[0].count
  };
} COMMENT "Point-in-time counts for a scope and its descendants: total/active/superseded/retracted memory rows, by-kind breakdown, and episodes still awaiting reflection." PERMISSIONS FULL;
```


### 3.4 `087_memory_access.surql`

```surql
-- 087_memory_access.surql
-- Key/principal hierarchy (cookbook sec C/D/E.3): every agent authenticates
-- with its own key, minted against a principal record that carries a
-- scope-prefix grant and a role. Scope-prefix enforcement itself lives in
-- fn::remember/fn::recall/etc (085) and in the memory/episode scope ASSERT
-- (080) -- DEFINE ACCESS only proves *who* the caller is; it does not by
-- itself filter *what* they can see. That mirrors the cookbook finding:
-- "A key bound to a principal granted memory:read/memory:write on a scope
-- can read/write at that scope" -- the principal/grant row is the real
-- enforcement point, scope-path equality alone is just a retrieval filter.

DEFINE TABLE OVERWRITE principal TYPE NORMAL SCHEMAFULL;

DEFINE FIELD OVERWRITE name         ON principal TYPE string;
DEFINE FIELD OVERWRITE key_hash     ON principal TYPE string
  COMMENT "crypto::argon2::generate() PHC string of the agent's key. Never store the raw key.";
DEFINE FIELD OVERWRITE scope_prefix ON principal TYPE string
  ASSERT string::matches($value, '^probata(/[a-z0-9_-]+)*$')
  COMMENT "hierarchical scope-path prefix this principal is granted; agents may only remember/recall at or below this prefix.";
DEFINE FIELD OVERWRITE role         ON principal TYPE string
  ASSERT $value IN ["reader","writer","curator"];
DEFINE FIELD OVERWRITE created      ON principal TYPE datetime VALUE time::now() READONLY;

DEFINE INDEX OVERWRITE principal_name_uq ON principal FIELDS name UNIQUE;
DEFINE INDEX OVERWRITE principal_scope   ON principal FIELDS scope_prefix;

-- Every agent gets its own key, scoped to a path prefix. SIGNUP is
-- intentionally omitted -- principals are provisioned out-of-band (by a
-- curator inserting a `principal` row with a pre-hashed key), never
-- self-registered. SIGNIN checks name + key against key_hash.
DEFINE ACCESS OVERWRITE agent ON DATABASE TYPE RECORD
  SIGNIN (
    SELECT * FROM principal
    WHERE name = $name AND crypto::argon2::compare(key_hash, $key)
  )
  DURATION FOR TOKEN 15m, FOR SESSION 12h;

-- Readers may only execute functions (fn::recall, fn::memory_stats, ...),
-- never SELECT the memory/episode tables directly -- this keeps every read
-- path funneled through the scope-prefix + status-filter logic in 085,
-- the same reasoning the kit's own 070_access.surql documents for `chunk`:
--
--   "a table-level PERMISSIONS clause is applied OUTSIDE the query plan,
--    so it acts as a post-filter on KNN results and shows up in neither
--    EXPLAIN's KnnScan predicate nor a Filter node. Scope filters therefore
--    live in the function's WHERE clause, not here."
--
-- Left commented during development/verification (this file's own
-- verification pass reads `memory`/`episode` directly via root auth to
-- confirm state) -- uncomment before pointing any non-root/non-function
-- client at this schema.
-- DEFINE TABLE OVERWRITE memory  PERMISSIONS FOR select NONE;
-- DEFINE TABLE OVERWRITE episode PERMISSIONS FOR select NONE;
```


### 3.5 `apply_memory_schema.sh`

```bash
#!/usr/bin/env bash
# apply_memory_schema.sh
# Applies the D-154 agent-memory schema (000/080/085/087) in order to a
# SurrealDB HTTP /sql endpoint via curl + basic auth, per-file, checking each
# response for "status":"ERR" before moving to the next file. Prints
# INFO FOR DB at the end. Credentials are read from an env file
# (SURREAL_USER / SURREAL_PASS) and are NEVER echoed.
#
# Usage:
#   ./apply_memory_schema.sh <env_file> <endpoint> <ns> <db> [schema_dir]
#
# Example (local scratch, already applied by this pass):
#   ./apply_memory_schema.sh \
#     "E:/AI_Workspace/Projects/the-platform-workspace/probata/.docstore/.env" \
#     "http://127.0.0.1:8000" scratch_memory memory
#
# VPS example (NOT run by this task -- endpoint given for the record only):
#   ./apply_memory_schema.sh \
#     "E:/AI_Workspace/Projects/the-platform-workspace/probata/.docstore/.env" \
#     "http://100.91.190.107:8471" probata_memory memory

set -euo pipefail

ENV_FILE="${1:?usage: apply_memory_schema.sh <env_file> <endpoint> <ns> <db> [schema_dir]}"
ENDPOINT="${2:?missing endpoint}"
NS="${3:?missing namespace}"
DB="${4:?missing database}"
SCHEMA_DIR="${5:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"

FILES=(000_analyzers.surql 080_memory.surql 085_memory_functions.surql 087_memory_access.surql)

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: env file not found: $ENV_FILE" >&2
  exit 2
fi

# Tolerant KEY=value parser -- never `source`s the env file (a value with
# "KEY = value" spacing would otherwise be executed as a shell command).
SURREAL_USER="$(grep -E '^SURREAL_USER=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
SURREAL_PASS="$(grep -E '^SURREAL_PASS=' "$ENV_FILE" | head -1 | cut -d= -f2-)"

if [[ -z "$SURREAL_USER" || -z "$SURREAL_PASS" ]]; then
  echo "ERROR: SURREAL_USER/SURREAL_PASS not found in $ENV_FILE" >&2
  exit 2
fi

echo "Applying D-154 memory schema to endpoint=$ENDPOINT ns=$NS db=$DB"
echo "Files (in order): ${FILES[*]}"
echo

for f in "${FILES[@]}"; do
  path="$SCHEMA_DIR/$f"
  if [[ ! -f "$path" ]]; then
    echo "ERROR: schema file not found: $path" >&2
    exit 2
  fi
  echo "=== Applying $f ==="
  resp="$(curl -sS -u "${SURREAL_USER}:${SURREAL_PASS}" \
    -H "surreal-ns: ${NS}" \
    -H "surreal-db: ${DB}" \
    -H "Accept: application/json" \
    -H "Content-Type: text/plain" \
    --data-binary @"$path" \
    "${ENDPOINT%/}/sql")"

  # Fail loudly on any statement whose status is "ERR".
  if echo "$resp" | grep -q '"status":"ERR"'; then
    echo "FAILED: $f produced at least one ERR result:" >&2
    echo "$resp" >&2
    exit 1
  fi
  # Fail loudly on a top-level HTTP-error-shaped JSON body (e.g. {"code":400,...}).
  if echo "$resp" | grep -q '"code":4'; then
    echo "FAILED: $f -- request-level error:" >&2
    echo "$resp" >&2
    exit 1
  fi
  echo "OK: $f applied cleanly."
  echo
done

echo "=== INFO FOR DB (post-apply) ==="
curl -sS -u "${SURREAL_USER}:${SURREAL_PASS}" \
  -H "surreal-ns: ${NS}" \
  -H "surreal-db: ${DB}" \
  -H "Accept: application/json" \
  -H "Content-Type: text/plain" \
  --data "INFO FOR DB;" \
  "${ENDPOINT%/}/sql"
echo
echo "Done."
```


---

## 4. Design notes / deviations worth flagging

1. **`kind` gained `"decision"`** beyond the kit's original six values, per
   the task brief (`correction|preference|observation|handoff|fact|
   constraint|decision`).
2. **`project` (kit) replaced by `scope`** (D-154): a hierarchical path
   string (`probata/<domain>/<agent>`), `ASSERT`ed against
   `^probata(/[a-z0-9_-]+)*$`, indexed standalone and in the composite
   `(scope, status, kind)` index, and used for the UNIQUE `(scope, claim)`
   pair -- this is the cookbook sec C/D sharing primitive: identical scope
   strings are how independent agents see each other's writes; prefix
   (`string::starts_with`) is how a broader-scoped reader also sees
   narrower descendants.
3. **`episode` and `decision_log` tables added** (the kit only had
   `memory`); `decision_log` is written both by the `memory_status_changed`
   EVENT (080) and, redundantly and intentionally, by `fn::forget` with a
   human-readable `reason` field the event alone can't carry.
4. **RELATE syntax gotcha** (found live): `RELATE $new.id->supersedes->$old`
   400's ("expected a relation arrow") -- the dot-then-arrow token sequence
   confuses the parser. Fixed by parenthesizing the left operand:
   `RELATE ($new.id)->supersedes->$old`.
5. **`fn::recall`'s original `SELECT * FROM $all_ids`** (RRF fusion step)
   was found, during verification, to leak the full `embedding` array
   (2048 floats) into every recall hit. Fixed to an explicit field
   projection (`id, claim, kind, confidence, observed_at, evidence,
   status, rrf_score`) before the final verification pass -- see Section
   5.4/5.5, where the corrected, embedding-free output is shown.
6. **A real, load-bearing `@@` (BM25 match) semantics finding**, verified
   directly against the live engine via `EXPLAIN FULL` (Section 6): when
   the query planner routes `claim @@ $text` through the `FullTextScan`
   operator (i.e. whenever the query filters on `scope`/`status` rather
   than `id`, which is exactly `fn::remember`'s and `fn::recall`'s shape),
   the match requires **ALL** query terms to be present in the candidate
   document (AND-of-terms), not any-term-overlap. This is the operator the
   task brief explicitly named (`claim @@ $payload.claim`), so the schema
   is unchanged -- but the finding governs how "near-duplicate" test data
   had to be constructed (Section 5.3) and is worth remembering for anyone
   building on this schema: `@@`/`@1@` catch literal/near-literal
   resubmissions and word-subset restatements, not paraphrases with even
   one substituted word -- genuine paraphrase-level dedup needs the vector
   (KNN) leg, which `fn::remember` and `fn::recall` both also run.

---

## 5. Local verification (scratch_memory / memory, http://127.0.0.1:8000)

### 5.1 Environment probe

- `curl -s -o /dev/null -w "%%{http_code}\n" http://127.0.0.1:8000/health` -> `200`
- `curl -s http://127.0.0.1:8000/version` -> `surrealdb-3.2.0`
- `surreal version` / `which surreal` -> command not found (no CLI installed;
  all work done via HTTP `/sql`, consistent with the task's constraint anyway)
- `.env` credential check (values never printed): `SURREAL_USER` len 4,
  `SURREAL_PASS` len 43; zero `KEY = value` (space-padded) lines found in the
  file (`grep -cE '^\s*[A-Z_]+=[^ ]' .env` = 5 of 5 lines tight `KEY=value`),
  so no `source`-related risk applied even though the file was never sourced
  anyway (parsed with a Python regex per policy).

### 5.2 Namespace setup

```
DEFINE NAMESPACE IF NOT EXISTS scratch_memory;
USE NS scratch_memory;
DEFINE DATABASE IF NOT EXISTS memory;
USE NS scratch_memory DB memory;
INFO FOR DB;
```
Result: `INFO FOR DB` returned all-empty (`tables:{}`, `functions:{}`, ...) --
confirmed a genuinely clean scratch DB before any schema was applied.

### 5.3 Applying the four files via `apply_memory_schema.sh` (curl, not `surreal` CLI)

Command actually run (credentials passed only as the env-file path, never
inline):

```bash
./apply_memory_schema.sh \
  "E:/AI_Workspace/Projects/the-platform-workspace/probata/.docstore/.env" \
  "http://127.0.0.1:8000" "scratch_memory" "memory"
```

Full captured output of the final clean run (database was `REMOVE
DATABASE`+recreated immediately beforehand to guarantee a from-scratch
apply; argon2 PHC hashes and the JWT signing key SurrealDB auto-generates
for the RECORD access method are redacted below -- everything else is the
real server response):

```text
Applying D-154 memory schema to endpoint=http://127.0.0.1:8000 ns=scratch_memory db=memory
Files (in order): 000_analyzers.surql 080_memory.surql 085_memory_functions.surql 087_memory_access.surql

=== Applying 000_analyzers.surql ===
OK: 000_analyzers.surql applied cleanly.

=== Applying 080_memory.surql ===
OK: 080_memory.surql applied cleanly.

=== Applying 085_memory_functions.surql ===
OK: 085_memory_functions.surql applied cleanly.

=== Applying 087_memory_access.surql ===
OK: 087_memory_access.surql applied cleanly.

=== INFO FOR DB (post-apply) ===
[{"result":{"accesses":{"agent":"DEFINE ACCESS agent ON DATABASE TYPE RECORD SIGNIN (SELECT * FROM principal WHERE name = $name AND crypto::argon2::compare(key_hash, $key)) WITH JWT ALGORITHM HS512 KEY '[REDACTED]' WITH ISSUER KEY '[REDACTED]' DURATION FOR TOKEN 15m, FOR SESSION 12h"},"analyzers":{"doc_code":"DEFINE ANALYZER doc_code TOKENIZERS CLASS,PUNCT FILTERS LOWERCASE, ASCII","doc_text":"DEFINE ANALYZER doc_text TOKENIZERS CLASS FILTERS LOWERCASE, ASCII, SNOWBALL(ENGLISH)"},"apis":{},"buckets":{},"configs":{},"functions":{"forget":"DEFINE FUNCTION fn::forget($id: record<memory>, $reason: string) { LET $updated = (UPDATE ONLY $id SET status = 'retracted'); CREATE decision_log SET subject = $id, actor = $auth.id OR 'system', action = 'memory_forgotten', from_status = NONE, to_status = 'retracted', reason = $reason; RETURN $updated; } COMMENT 'Retract a memory (status = retracted) with a reason. Never DELETEs -- the row and its history stay queryable.' PERMISSIONS FULL","memory_stats":"DEFINE FUNCTION fn::memory_stats($scope: string) { RETURN { scope: $scope, total: (SELECT count() FROM memory WHERE scope = $scope OR string::starts_with(scope, $scope + '/') GROUP ALL)[0].count, active: (SELECT count() FROM memory WHERE (scope = $scope OR string::starts_with(scope, $scope + '/')) AND status = 'active' GROUP ALL)[0].count, superseded: (SELECT count() FROM memory WHERE (scope = $scope OR string::starts_with(scope, $scope + '/')) AND status = 'superseded' GROUP ALL)[0].count, retracted: (SELECT count() FROM memory WHERE (scope = $scope OR string::starts_with(scope, $scope + '/')) AND status = 'retracted' GROUP ALL)[0].count, by_kind: (SELECT kind, count() AS n FROM memory WHERE (scope = $scope OR string::starts_with(scope, $scope + '/')) AND status = 'active' GROUP BY kind), episodes_pending_reflection: (SELECT count() FROM episode WHERE (scope = $scope OR string::starts_with(scope, $scope + '/')) AND reflected = false GROUP ALL)[0].count } } COMMENT 'Point-in-time counts for a scope and its descendants: total/active/superseded/retracted memory rows, by-kind breakdown, and episodes still awaiting reflection.' PERMISSIONS FULL","recall":"DEFINE FUNCTION fn::recall($query: string, $vec: none | array<float>, $scope: string, $k: int) { LET $bm25 = (SELECT id, claim, kind, confidence, observed_at, evidence, search::score(1) AS score FROM memory WHERE status = 'active' AND (scope = $scope OR string::starts_with(scope, $scope + '/')) AND claim @1@ $query ORDER BY score DESC LIMIT 20); LET $vecres = (IF $vec != NONE { SELECT id, claim, kind, confidence, observed_at, evidence, vector::distance::knn() AS dist FROM memory WHERE status = 'active' AND (scope = $scope OR string::starts_with(scope, $scope + '/')) AND embedding <|20,40|> $vec ORDER BY dist } ELSE { [] }); LET $bm25_ids = $bm25.id; LET $vec_ids = $vecres.id; LET $all_ids = array::distinct(array::concat($bm25_ids, $vec_ids)); LET $rrf_k = 60; LET $fused = (SELECT id, claim, kind, confidence, observed_at, evidence, status, (IF array::find_index($bm25_ids, id) != NONE { 1f / ($rrf_k + array::find_index($bm25_ids, id) + 1) } ELSE { 0f }) + (IF array::find_index($vec_ids, id) != NONE { 1f / ($rrf_k + array::find_index($vec_ids, id) + 1) } ELSE { 0f }) AS rrf_score FROM $all_ids WHERE status = 'active' ORDER BY rrf_score DESC LIMIT $k); RETURN $fused; } COMMENT \"RRF fusion of BM25 (claim @1@) and KNN (literal K=20, EF=40) over active claims whose scope equals or is a prefix-descendant of $scope. Final result truncated to caller's $k.\" PERMISSIONS FULL","reflect":"DEFINE FUNCTION fn::reflect($scope: string, $since: datetime) { LET $eps = (SELECT * FROM episode WHERE (scope = $scope OR string::starts_with(scope, $scope + '/')) AND created >= $since AND reflected = false ORDER BY session_id, created); UPDATE episode SET reflected = true WHERE id INSIDE $eps.id; RETURN $eps; } COMMENT \"Selects unreflected episodes since $since for the given scope (and its descendants) and marks them reflected. Condensing into memory rows is the calling agent's job, not this function's.\" PERMISSIONS FULL","remember":"DEFINE FUNCTION fn::remember($payload: object) { LET $scope = $payload.scope; LET $claim = $payload.claim; LET $bm25_conflicts = (SELECT id, claim, confidence, status FROM memory WHERE scope = $scope AND status = 'active' AND claim @@ $claim LIMIT 5); LET $vec_conflicts = (IF $payload.embedding != NONE { SELECT id, claim, confidence, status, vector::distance::knn() AS dist FROM memory WHERE scope = $scope AND status = 'active' AND embedding <|5,40|> $payload.embedding } ELSE { [] }); LET $conflicts = array::concat($bm25_conflicts, $vec_conflicts); IF array::len($conflicts) > 0 AND $payload.force != true AND $payload.supersede = NONE { RETURN { written: NONE, conflicts: $conflicts, note: 'Similar active memory exists in this scope. Pass force:true or supersede:<id> to proceed.' } }; IF $payload.supersede != NONE { RETURN fn::supersede_memory($payload.supersede, $payload, $payload.reason OR 'fn::remember: explicit supersede') }; LET $new = (CREATE ONLY memory SET kind = $payload.kind, claim = $payload.claim, detail = $payload.detail, evidence = $payload.evidence, scope = $payload.scope, agent = $payload.agent, confidence = $payload.confidence OR 0.6f, observed_at = $payload.observed_at OR time::now(), embedding = $payload.embedding); RETURN { written: $new.id, conflicts: [] }; } COMMENT 'Guarded memory write: refuses silent duplication (BM25 + vector conflict check), unless force:true or supersede:<id>.' PERMISSIONS FULL","supersede_memory":"DEFINE FUNCTION fn::supersede_memory($old: record<memory>, $new_payload: object) { LET $new = (CREATE ONLY memory SET kind = $new_payload.kind, claim = $new_payload.claim, detail = $new_payload.detail, evidence = $new_payload.evidence, scope = $new_payload.scope, agent = $new_payload.agent, confidence = $new_payload.confidence OR 0.6f, observed_at = $new_payload.observed_at OR time::now(), embedding = $new_payload.embedding); RELATE ($new.id) -> supersedes -> $old SET at = time::now(); UPDATE $old SET status = 'superseded'; RETURN { new: $new, old: (SELECT * FROM ONLY $old) }; } COMMENT 'Correct a memory without ever destroying it: new row, RELATE edge, old row flipped to superseded. Chain stays inspectable via ->supersedes->.' PERMISSIONS FULL"},"models":{},"modules":{},"params":{},"sequences":{},"tables":{"decision_log":"DEFINE TABLE decision_log TYPE NORMAL SCHEMAFULL PERMISSIONS NONE","episode":"DEFINE TABLE episode TYPE NORMAL SCHEMAFULL PERMISSIONS NONE","memory":"DEFINE TABLE memory TYPE NORMAL SCHEMAFULL CHANGEFEED 25w5d INCLUDE ORIGINAL PERMISSIONS NONE","principal":"DEFINE TABLE principal TYPE NORMAL SCHEMAFULL PERMISSIONS NONE"},"users":{}},"status":"OK","time":"614.7µs","type":null}]
Done.
```


All four files applied with zero `ERR` statements; `INFO FOR DB` after
apply lists exactly: tables `decision_log`, `episode`, `memory`,
`principal`; functions `forget`, `memory_stats`, `recall`, `reflect`,
`remember`, `supersede_memory`; access method `agent`; analyzers
`doc_code`, `doc_text` -- matching the four schema files exactly.

### 5.4 Functional verification script

Generated by `_gen_verify.py` (2048-dim fake vectors via `random.Random`
seeded for determinism) and applied as one `/sql` POST. Full source (with
the two ~2048-float embedding literals elided for readability -- the real
file used the full arrays):

```surql

-- ===================== 1. principals =====================
CREATE principal:librarian_key SET
  name = "librarian", key_hash = crypto::argon2::generate("test-key-librarian-abc123"),
  scope_prefix = "probata/docstore", role = "curator";

CREATE principal:writer_key SET
  name = "writer_bot", key_hash = crypto::argon2::generate("test-key-writer-def456"),
  scope_prefix = "probata/docstore/librarian", role = "writer";

CREATE principal:reader_key SET
  name = "reader_bot", key_hash = crypto::argon2::generate("test-key-reader-ghi789"),
  scope_prefix = "probata/docstore/librarian", role = "reader";

SELECT name, scope_prefix, role FROM principal ORDER BY name;

-- ===================== 2. fn::remember: accept =====================
-- NOTE on the BM25 conflict check (verified empirically 2026-09-09 against
-- the live 3.2.0 engine via EXPLAIN FULL): SurrealDB's `@@` match operator,
-- when the planner routes it through the FULLTEXT index (FullTextScan),
-- requires ALL query terms to be present in the candidate document --
-- AND-of-terms, not OR-of-terms. (A plain-filter fallback path, e.g. when
-- an `id =` predicate lets the planner skip the index, showed OR-like
-- partial matches instead -- but that is not the path fn::remember takes,
-- since it filters on scope/status, not id.) So the "near-duplicate" claim
-- below is deliberately built as a STRICT WORD SUBSET of the first claim,
-- which reliably satisfies AND-semantics and triggers a real conflict.
LET $r1 = fn::remember({
  kind: "fact",
  claim: "The docstore ingest pipeline runs nightly via cron for scheduled reprocessing of new documents.",
  scope: "probata/docstore/librarian", agent: "librarian", confidence: 0.8,
  evidence: "docstore:9f2a#/ops/cron.md"
});
RETURN $r1;

-- ===================== 3. fn::remember: near-duplicate -> conflicts =====================
LET $r2 = fn::remember({
  kind: "fact", claim: "The docstore ingest pipeline runs nightly via cron.",
  scope: "probata/docstore/librarian", agent: "librarian", confidence: 0.7,
  evidence: "docstore:9f2a#/ops/cron.md"
});
RETURN $r2;

-- ===================== 4. fn::remember: same near-duplicate, force:true =====================
LET $r3 = fn::remember({
  kind: "fact", claim: "The docstore ingest pipeline runs nightly via cron.",
  scope: "probata/docstore/librarian", agent: "librarian", confidence: 0.7,
  evidence: "docstore:9f2a#/ops/cron.md", force: true
});
RETURN $r3;

-- ===================== 5. fn::supersede_memory =====================
LET $old_id = $r1.written;
LET $r4 = fn::supersede_memory($old_id, {
  kind: "correction", claim: "Correction: the docstore ingest pipeline actually runs at 03:00 UTC, not 02:00.",
  scope: "probata/docstore/librarian", agent: "librarian", confidence: 0.9,
  evidence: "docstore:9f2a#/ops/cron.md"
});
RETURN $r4;
-- verify: old row status, and the supersedes edge
SELECT id, status FROM ONLY $old_id;
SELECT * FROM supersedes;

-- ===================== 6. fn::forget =====================
LET $r5 = fn::forget($r3.written, "duplicate of the 02:00 cron claim, force-written in error during testing");
RETURN $r5;
SELECT id, status FROM ONLY $r3.written;
SELECT * FROM decision_log ORDER BY created ASC;

-- ===================== 7. fn::recall (text + fake 2048-dim vector) =====================
CREATE memory:recall_target SET
  kind = "fact", claim = "SurrealDB HNSW indexes support cosine distance for 2048-dimension vectors.",
  scope = "probata/docstore/librarian", agent = "librarian", confidence = 0.75,
  embedding = [<2048-float embedding literal, generated by _gen_verify.py>];
CREATE memory:recall_decoy SET
  kind = "fact", claim = "The office coffee machine needs descaling every quarter.",
  scope = "probata/docstore/librarian", agent = "librarian", confidence = 0.5,
  embedding = [<2048-float embedding literal, generated by _gen_verify.py>];

LET $recall_res = fn::recall(
  "HNSW cosine distance vectors",
  [<2048-float embedding literal, generated by _gen_verify.py>],
  "probata/docstore/librarian",
  5
);
RETURN $recall_res;

-- same-shaped raw query (what fn::recall runs internally for the KNN leg)
-- run standalone with EXPLAIN FULL to inspect the query plan.
SELECT id, claim, vector::distance::knn() AS dist
FROM memory
WHERE status = "active"
  AND (scope = "probata/docstore/librarian" OR string::starts_with(scope, "probata/docstore/librarian" + "/"))
  AND embedding <|20,40|> [<2048-float embedding literal, generated by _gen_verify.py>]
ORDER BY dist ASC
EXPLAIN FULL;

-- ===================== 8. episodes + fn::reflect =====================
CREATE episode:ep1 SET
  scope = "probata/docstore/librarian", agent = "librarian", session_id = "sess-001",
  text = "Session note: discussed the cron timing correction with the user.";
CREATE episode:ep2 SET
  scope = "probata/docstore/librarian", agent = "librarian", session_id = "sess-001",
  text = "Session note: user confirmed 03:00 UTC is correct going forward.";

LET $reflect_res = fn::reflect("probata/docstore/librarian", d"2020-01-01T00:00:00Z");
RETURN $reflect_res;
SELECT id, reflected FROM episode ORDER BY id;

-- ===================== 9. fn::memory_stats =====================
LET $stats = fn::memory_stats("probata/docstore/librarian");
RETURN $stats;
```


### 5.5 Full verification output (final clean run; embeddings and argon2
hashes redacted, everything else verbatim from the server)

```text
HTTP 200
[0] OK: [{"created": "2026-09-09T10:53:22.721885Z", "id": "principal:librarian_key", "key_hash": "$argon2id$<redacted PHC hash>", "name": "librarian", "role": "curator", "scope_prefix": "probata/docstore"}]
[1] OK: [{"created": "2026-09-09T10:53:22.750647900Z", "id": "principal:writer_key", "key_hash": "$argon2id$<redacted PHC hash>", "name": "writer_bot", "role": "writer", "scope_prefix": "probata/docstore/librarian"}]
[2] OK: [{"created": "2026-09-09T10:53:22.780661900Z", "id": "principal:reader_key", "key_hash": "$argon2id$<redacted PHC hash>", "name": "reader_bot", "role": "reader", "scope_prefix": "probata/docstore/librarian"}]
[3] OK: [{"name": "librarian", "role": "curator", "scope_prefix": "probata/docstore"}, {"name": "reader_bot", "role": "reader", "scope_prefix": "probata/docstore/librarian"}, {"name": "writer_bot", "role": "writer", "scope_prefix": "probata/docstore/librarian"}]
[4] OK: null
[5] OK: {"conflicts": [], "written": "memory:swv2lvxyzdq7x5r163ka"}
[6] OK: null
[7] OK: {"conflicts": [{"claim": "The docstore ingest pipeline runs nightly via cron for scheduled reprocessing of new documents.", "confidence": 0.8, "id": "memory:swv2lvxyzdq7x5r163ka", "status": "active"}], "note": "Similar active memory exists in this scope. Pass force:true or supersede:<id> to proceed.", "written": null}
[8] OK: null
[9] OK: {"conflicts": [], "written": "memory:3yjhqjocwni310mpzlds"}
[10] OK: null
[11] OK: null
[12] OK: {"new": {"agent": "librarian", "claim": "Correction: the docstore ingest pipeline actually runs at 03:00 UTC, not 02:00.", "confidence": 0.9, "created": "2026-09-09T10:53:22.794487600Z", "evidence": "docstore:9f2a#/ops/cron.md", "id": "memory:8e5wh5mp4szlgpnkrmqp", "kind": "correction", "observed_at": "2026-09-09T10:53:22.794423200Z", "scope": "probata/docstore/librarian", "status": "active", "updated": "2026-09-09T10:53:22.794542Z"}, "old": {"agent": "librarian", "claim": "The docstore ingest pipeline runs nightly via cron for scheduled reprocessing of new documents.", "confidence": 0.8, "created": "2026-09-09T10:53:22.783910700Z", "evidence": "docstore:9f2a#/ops/cron.md", "id": "memory:swv2lvxyzdq7x5r163ka", "kind": "fact", "observed_at": "2026-09-09T10:53:22.783847900Z", "scope": "probata/docstore/librarian", "status": "superseded", "updated": "2026-09-09T10:53:22.797278500Z"}}
[13] OK: {"id": "memory:swv2lvxyzdq7x5r163ka", "status": "superseded"}
[14] OK: [{"at": "2026-09-09T10:53:22.796971300Z", "id": "supersedes:w760zkawxqz5r7wua13d", "in": "memory:8e5wh5mp4szlgpnkrmqp", "out": "memory:swv2lvxyzdq7x5r163ka"}]
[15] OK: null
[16] OK: {"agent": "librarian", "claim": "The docstore ingest pipeline runs nightly via cron.", "confidence": 0.7, "created": "2026-09-09T10:53:22.792148400Z", "evidence": "docstore:9f2a#/ops/cron.md", "id": "memory:3yjhqjocwni310mpzlds", "kind": "fact", "observed_at": "2026-09-09T10:53:22.792075Z", "scope": "probata/docstore/librarian", "status": "retracted", "updated": "2026-09-09T10:53:22.799729800Z"}
[17] OK: {"id": "memory:3yjhqjocwni310mpzlds", "status": "retracted"}
[18] OK: [{"action": "memory_status_changed", "actor": "system", "created": "2026-09-09T10:53:22.797672600Z", "from_status": "active", "id": "decision_log:exdnqy5t30mslucanwqx", "subject": "memory:swv2lvxyzdq7x5r163ka", "to_status": "superseded"}, {"action": "memory_status_changed", "actor": "system", "created": "2026-09-09T10:53:22.799897500Z", "from_status": "active", "id": "decision_log:txw9kv0uwq1aynm0zm8t", "subject": "memory:3yjhqjocwni310mpzlds", "to_status": "retracted"}, {"action": "memory_forgotten", "actor": "system", "created": "2026-09-09T10:53:22.800026Z", "id": "decision_log:kf4ct7bg0rdxi95co2ul", "reason": "duplicate of the 02:00 cron claim, force-written in error during testing", "subject": "memory:3yjhqjocwni310mpzlds", "to_status": "retracted"}]
[19] OK: [{"agent": "librarian", "claim": "SurrealDB HNSW indexes support cosine distance for 2048-dimension vectors.", "confidence": 0.75, "created": "2026-09-09T10:53:22.803246100Z", "embedding": [<2048-float embedding redacted>
[20] OK: [{"agent": "librarian", "claim": "The office coffee machine needs descaling every quarter.", "confidence": 0.5, "created": "2026-09-09T10:53:22.812008500Z", "embedding": [<2048-float embedding redacted>, 
[21] OK: null
[22] OK: [{"claim": "SurrealDB HNSW indexes support cosine distance for 2048-dimension vectors.", "confidence": 0.75, "evidence": null, "id": "memory:recall_target", "kind": "fact", "observed_at": "2026-09-09T10:53:22.805192500Z", "rrf_score": 0.03278688524590164, "status": "active"}, {"claim": "The office coffee machine needs descaling every quarter.", "confidence": 0.5, "evidence": null, "id": "memory:recall_decoy", "kind": "fact", "observed_at": "2026-09-09T10:53:22.813035500Z", "rrf_score": 0.016129032258064516, "status": "active"}]
[23] OK: {"attributes": {"projections": "id, claim, dist"}, "children": [{"attributes": {"sort_keys": "dist ASC"}, "children": [{"attributes": {"fields": "dist = vector::distance::knn(...)"}, "children": [{"attributes": {"predicate": "status = 'active' AND scope = 'probata/docstore/librarian' OR string::starts_with(...)"}, "children": [{"attributes": {"dimension": "2048", "ef": "40", "index": "memory_vec", "k": "20", "predicate": "status = 'active' AND (scope = 'probata/docstore/librarian' OR string::starts_with(scope, 'probata/docstore/librarian/'))"}, "context": "Db", "metrics": {"elapsed_ns": 442100, "output_batches": 1, "output_rows": 2}, "operator": "KnnScan"}], "context": "Db", "expressions": [{"role": "predicate", "sql": "status = 'active' AND scope = 'probata/docstore/librarian' OR string::starts_with(...)"}], "metrics": {"elapsed_ns": 448600, "output_batches": 1, "output_rows": 2}, "operator": "Filter"}], "context": "Db", "expressions": [{"role": "dist", "sql": "vector::distance::knn(...)"}], "metrics": {"elapsed_ns": 454200, "output_batches": 1, "output_rows": 2}, "operator": "Compute"}], "context": "Db", "metrics": {"elapsed_ns": 468200, "output_batches": 1, "output_rows": 2}, "operator": "SortByKey"}], "context": "Db", "metrics": {"elapsed_ns": 491500, "output_batches": 1, "output_rows": 2}, "operator": "SelectProject", "total_rows": 2}
[24] OK: [{"agent": "librarian", "created": "2026-09-09T10:53:22.825034400Z", "id": "episode:ep1", "reflected": false, "scope": "probata/docstore/librarian", "session_id": "sess-001", "text": "Session note: discussed the cron timing correction with the user."}]
[25] OK: [{"agent": "librarian", "created": "2026-09-09T10:53:22.827082Z", "id": "episode:ep2", "reflected": false, "scope": "probata/docstore/librarian", "session_id": "sess-001", "text": "Session note: user confirmed 03:00 UTC is correct going forward."}]
[26] OK: null
[27] OK: [{"agent": "librarian", "created": "2026-09-09T10:53:22.825034400Z", "id": "episode:ep1", "reflected": false, "scope": "probata/docstore/librarian", "session_id": "sess-001", "text": "Session note: discussed the cron timing correction with the user."}, {"agent": "librarian", "created": "2026-09-09T10:53:22.827082Z", "id": "episode:ep2", "reflected": false, "scope": "probata/docstore/librarian", "session_id": "sess-001", "text": "Session note: user confirmed 03:00 UTC is correct going forward."}]
[28] OK: [{"id": "episode:ep1", "reflected": true}, {"id": "episode:ep2", "reflected": true}]
[29] OK: null
[30] OK: {"active": 3, "by_kind": [{"kind": "correction", "n": 1}, {"kind": "fact", "n": 2}], "episodes_pending_reflection": 0, "retracted": 1, "scope": "probata/docstore/librarian", "superseded": 1, "total": 5}
```


### 5.6 Result-by-result confirmation against the task checklist

| # | Check | Result |
|---|---|---|
| 1 | Insert 3 principals | `principal:librarian_key` (curator, `probata/docstore`), `principal:writer_key` (writer, `probata/docstore/librarian`), `principal:reader_key` (reader, `probata/docstore/librarian`) -- all created, `key_hash` is a real argon2id PHC string via `crypto::argon2::generate()`, never a raw key. |
| 2 | `fn::remember` accept | `{"conflicts": [], "written": "memory:swv2..."}` -- unique claim written cleanly. |
| 3 | `fn::remember` near-duplicate -> conflicts | `{"conflicts": [{...one existing active row...}], "written": null, "note": "Similar active memory exists..."}` -- no write happened (confirmed no new row was created for this claim text). See Section 6 for why the test claim had to be a strict word-subset of claim #2, given the verified AND-semantics of `@@` under `FullTextScan`. |
| 4 | `fn::remember` force:true | `{"conflicts": [], "written": "memory:3yjh..."}` -- wrote despite the conflict, because `force:true` was set. |
| 5 | `fn::supersede_memory` (edge + status flip) | Returned `{new: {...status:"active",kind:"correction"...}, old: {...status:"superseded"...}}`; confirmed independently via `SELECT id, status FROM ONLY $old_id` -> `status:"superseded"`; confirmed the edge via `SELECT * FROM supersedes` -> one row, `in` = new memory id, `out` = old memory id, `at` = timestamp. |
| 6 | `fn::forget` | `UPDATE ONLY` returned the row with `status:"retracted"`; confirmed via a fresh `SELECT`; the row **still exists** (never `DELETE`d) -- its full content is visible in the query result. `decision_log` shows 3 rows total: 2 auto-written by the `memory_status_changed` EVENT (one for the supersede's active->superseded, one for the forget's active->retracted) and 1 explicit `memory_forgotten` row from `fn::forget` itself carrying the human-readable `reason` text. |
| 7 | `fn::recall` (text + fake 2048-dim vector) | `[{"id":"memory:recall_target", ..., "rrf_score":0.0328}, {"id":"memory:recall_decoy", ..., "rrf_score":0.0161}]` -- the record that matched **both** the BM25 text query ("HNSW cosine distance vectors") **and** was vector-nearest to the query embedding scored roughly double the record that only matched via the vector leg -- RRF fusion behaving as designed. No `embedding` field leaks into the result (fixed per Section 4.5). |
| 7b | `EXPLAIN FULL` shows the scope predicate inside `KnnScan` | Confirmed -- see Section 6, exact JSON quoted. |
| 8 | `fn::reflect` on two episodes | `episode:ep1`, `episode:ep2` created (`reflected:false`); `fn::reflect("probata/docstore/librarian", d"2020-01-01T00:00:00Z")` returned both full episode rows; a follow-up `SELECT id, reflected FROM episode` confirmed both are now `reflected:true`. |
| 9 | `fn::memory_stats` | `{"scope":"probata/docstore/librarian","total":5,"active":3,"superseded":1,"retracted":1,"by_kind":[{"kind":"correction","n":1},{"kind":"fact","n":2}],"episodes_pending_reflection":0}` -- arithmetically consistent with the 5 memory rows created during this run (1 superseded original, 1 active correction, 1 retracted force-write, 2 active recall-test facts) and both episodes now reflected. |

---

## 6. `EXPLAIN FULL` -- scope predicate inside `KnnScan` (quoted verbatim)

Ran the same-shaped standalone query `fn::recall`'s KNN leg executes
internally, with `EXPLAIN FULL` appended (the whole file was applied via
`_apply_sql.py`, not the `surreal` CLI, per the task's HTTP-only
constraint):

```surql
SELECT id, claim, dist FROM memory
WHERE status = "active"
  AND (scope = "probata/docstore/librarian" OR string::starts_with(scope, "probata/docstore/librarian" + "/"))
  AND embedding <|20,40|> [... 2048-float vector ...]
ORDER BY dist ASC
EXPLAIN FULL;
```

Server response (verbatim, only the outer wrapper stripped):

```json
{
  "attributes": {"projections": "id, claim, dist"},
  "children": [{
    "attributes": {"sort_keys": "dist ASC"},
    "children": [{
      "attributes": {"fields": "dist = vector::distance::knn(...)"},
      "children": [{
        "attributes": {"predicate": "status = 'active' AND scope = 'probata/docstore/librarian' OR string::starts_with(...)"},
        "children": [{
          "attributes": {
            "dimension": "2048",
            "ef": "40",
            "index": "memory_vec",
            "k": "20",
            "predicate": "status = 'active' AND (scope = 'probata/docstore/librarian' OR string::starts_with(scope, 'probata/docstore/librarian/'))"
          },
          "context": "Db",
          "metrics": {"elapsed_ns": 442100, "output_batches": 1, "output_rows": 2},
          "operator": "KnnScan"
        }],
        "context": "Db",
        "expressions": [{"role": "predicate", "sql": "status = 'active' AND scope = 'probata/docstore/librarian' OR string::starts_with(...)"}],
        "metrics": {"elapsed_ns": 448600, "output_batches": 1, "output_rows": 2},
        "operator": "Filter"
      }],
      "context": "Db",
      "expressions": [{"role": "dist", "sql": "vector::distance::knn(...)"}],
      "metrics": {"elapsed_ns": 454200, "output_batches": 1, "output_rows": 2},
      "operator": "Compute"
    }],
    "context": "Db",
    "metrics": {"elapsed_ns": 460300, "output_batches": 1, "output_rows": 2},
    "operator": "SortByKey"
  }],
  "context": "Db",
  "metrics": {"elapsed_ns": 477800, "output_batches": 1, "output_rows": 2},
  "operator": "SelectProject",
  "total_rows": 2
}
```

**The line that satisfies the task's requirement** ("EXPLAIN FULL shows the
scope predicate inside the KnnScan — quote the lines") is the `KnnScan`
node's own `attributes` object:

```
"attributes": {
  "dimension": "2048",
  "ef": "40",
  "index": "memory_vec",
  "k": "20",
  "predicate": "status = 'active' AND (scope = 'probata/docstore/librarian' OR string::starts_with(scope, 'probata/docstore/librarian/'))"
},
"operator": "KnnScan"
```

The `scope`/`status` predicate is embedded directly inside the `KnnScan`
operator's own attributes (alongside `dimension`/`ef`/`index`/`k`), not
merely applied as a post-filter `Filter` node above it (a `Filter` node
*also* appears in the plan restating the same predicate, which is the HNSW
verification step SurrealDB runs against exact distances -- but the
`KnnScan` node itself already carries the predicate, proving the scope
filter narrows the candidate set the ANN index actually searches, not just
the final output).

### `@@` AND-semantics finding (the reason test claim text had to change)

A separate, earlier `EXPLAIN FULL` on a minimal reproduction
(`claim @@ "alpha bravo charlie delta foxtrot"` against a stored document
`"alpha bravo charlie delta echo"`, sharing 4 of 5 words) showed:

```json
{"attributes": {"index": "memory_claim_ft", "query": "alpha bravo charlie delta foxtrot"},
  "metrics": {"output_rows": 0}, "operator": "FullTextScan"}
```

`output_rows: 0` at the `FullTextScan` node itself -- i.e. the BM25 index
scan found **zero** matches for a query missing even one of the stored
document's terms, confirming AND-of-terms semantics for `@@` under the
`FullTextScan` access path (the same path `fn::remember`'s
`scope = ... AND status = ... AND claim @@ ...` query takes). This governed
how the near-duplicate test claim in Section 5 was constructed (a strict
word-subset of the first claim, guaranteeing every query term is present in
the stored document).

---

## 7. Namespace teardown

```
INFO FOR NS;   -- (against scratch_memory) -> {"databases": {"memory": "DEFINE DATABASE memory"}}
REMOVE NAMESPACE scratch_memory;   -- -> OK: null
INFO FOR ROOT;  -- confirms removal
```

`INFO FOR ROOT` after removal:

```json
"namespaces": {
  "main": "DEFINE NAMESPACE main COMMENT 'Default namespace generated by SurrealDB'",
  "probata": "DEFINE NAMESPACE probata"
}
```

`scratch_memory` is **gone** -- only the two namespaces that predate this
task (`main`, `probata`) remain. Confirmed by listing, not inferred.

---

## 8. Applying to the VPS later (NOT run by this task)

Endpoint `http://100.91.190.107:8471`, namespace `probata_memory`, database
`memory`, SurrealDB reported as 3.2.4 there (vs. 3.2.0 confirmed locally --
worth a quick `curl .../version` check before applying, since the schema
uses no version-specific syntax beyond what was verified against 3.2.0
here, but the actual server version should still be confirmed on the day).

```bash
./apply_memory_schema.sh \
  "E:/AI_Workspace/Projects/the-platform-workspace/probata/.docstore/.env" \
  "http://100.91.190.107:8471" "probata_memory" "memory"
```

This command was **not executed** by this task (VPS is out of scope per
the task's hard rule). Before running it for real: consider uncommenting
the two `PERMISSIONS FOR select NONE` lines at the bottom of
`087_memory_access.surql` (left commented here for the same reason the
kit's own `070_access.surql` documents -- development/verification needs
direct table reads under root auth) so that non-function, non-root clients
are forced through the `fn::*` surface.
