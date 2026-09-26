#!/usr/bin/env bash
# Throwaway-instance proof for the Review overlay blocks: context review (+ the
# hindsight-only foreshadowing flag) and owner metadata corrections.
#
# Byline: Claude Code · Opus 5.5 · 2026-09-25
# Byline: Claude Code · Opus 5.5 · 2026-09-26 (second block: source metadata corrections)
#
# Runs ON a VPS that already has the platform image (ovh-files has
# probata-postgres:18-duckdb, the image the live platform database runs), in a
# container with no network that is stopped (and auto-removed) when the script
# exits. It never touches the live database.
#
#   0. check each block in the snapshot is byte-identical to its apply script's block;
#   1. bootstrap an empty platform database from the snapshot WITH both blocks
#      (the snapshot is the database, D-142 §3 / D-152), after creating the roles
#      its GRANTs name (a schema-only dump carries no CREATE ROLE);
#   2. apply each apply script twice (idempotency);
#   3. run each validation test (each ends in ROLLBACK).
#
# Usage (from the repository root, on the desktop):
#   tar -cf - sql/bootstrap/schema_snapshot_20260907.sql \
#       scripts/2026-09-25-context-review-overlay.sql scripts/2026-09-26-source-metadata-correction-overlay.sql \
#       sql/validation/2026-09-25-context-review-horizon-test.sql sql/validation/2026-09-26-source-metadata-correction-test.sql \
#       sql/validation/2026-09-25-context-review-proof.sh \
#     | ssh -i ~/.ssh/ovh root@100.91.190.107 'D=$(mktemp -d) && tar -xf - -C "$D" && bash "$D/sql/validation/2026-09-25-context-review-proof.sh" "$D"'
set -euo pipefail

ROOT="${1:?usage: proof.sh <directory holding the repository-relative files>}"
SNAPSHOT="$ROOT/sql/bootstrap/schema_snapshot_20260907.sql"
IMAGE="${PROOF_IMAGE:-probata-postgres:18-duckdb}"
NAME="overlay-proof-$(date +%s)"
# block name | apply script | validation test
BLOCKS=(
  "context-review-overlay|$ROOT/scripts/2026-09-25-context-review-overlay.sql|$ROOT/sql/validation/2026-09-25-context-review-horizon-test.sql"
  "source-metadata-correction-overlay|$ROOT/scripts/2026-09-26-source-metadata-correction-overlay.sql|$ROOT/sql/validation/2026-09-26-source-metadata-correction-test.sql"
)

block() { sed -n "/^-- >>> BEGIN $2 block/,/^-- <<< END $2 block <<</p" "$1"; }
for entry in "${BLOCKS[@]}"; do
  IFS='|' read -r name script _ <<<"$entry"
  if [ "$(block "$SNAPSHOT" "$name" | sha256sum)" != "$(block "$script" "$name" | sha256sum)" ]; then
    echo "FAIL: the snapshot block and the apply-script block differ ($name)" >&2
    exit 1
  fi
  echo "PASS 0: snapshot block == apply-script block for $name ($(block "$script" "$name" | wc -l) lines)"
done

PASSWORD="$(head -c 18 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')"
docker run -d --rm --name "$NAME" --network none \
  -e POSTGRES_USER=ai -e POSTGRES_PASSWORD="$PASSWORD" -e POSTGRES_DB=platform "$IMAGE" >/dev/null
trap 'docker stop "$NAME" >/dev/null 2>&1 || true' EXIT
for _ in $(seq 1 90); do
  if docker exec "$NAME" pg_isready -U ai -d platform >/dev/null 2>&1; then break; fi
  sleep 1
done
psql_in() { docker exec -i "$NAME" psql -X -U ai -d platform -v ON_ERROR_STOP=1 "$@"; }
# The image's own entrypoint restarts PostgreSQL once after initdb; wait for the final server.
for _ in $(seq 1 90); do
  if psql_in -Atc "SELECT 1" >/dev/null 2>&1; then break; fi
  sleep 1
done

ROLES="$(grep -oE '^(GRANT|REVOKE) .* (TO|FROM) [a-z_][a-z0-9_]*;$' "$SNAPSHOT" | grep -oE '[a-z_][a-z0-9_]*;$' | tr -d ';' ;
         grep -oE 'FOR ROLE [a-z_][a-z0-9_]*' "$SNAPSHOT" | awk '{print $3}')"
for role in $(printf '%s\n' "$ROLES" | sort -u); do
  psql_in -qc "DO \$\$ BEGIN CREATE ROLE $role NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END \$\$;"
done
echo "roles created: $(printf '%s\n' "$ROLES" | sort -u | wc -l)"

start=$(date +%s)
psql_in -q < "$SNAPSHOT" >/dev/null
echo "PASS 1: snapshot with both overlay blocks bootstrapped an empty database in $(( $(date +%s) - start ))s"

for entry in "${BLOCKS[@]}"; do
  IFS='|' read -r name script test <<<"$entry"
  echo "--- $name apply script, first run"
  psql_in < "$script"
  echo "--- $name apply script, second run (must be a no-op)"
  psql_in < "$script"
  echo "PASS 2: $name apply script ran twice without error"
  echo "--- $name validation test"
  psql_in < "$test" 2>&1
  echo "PASS 3: $name test completed (every assertion above printed PASS; the transaction rolled back)"
done
