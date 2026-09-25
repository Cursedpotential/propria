#!/usr/bin/env bash
# Throwaway-instance proof for the context review overlay block.
#
# Byline: Claude Code · Opus 5.5 · 2026-09-25
#
# Runs ON a VPS that already has the platform image (ovh-files has
# probata-postgres:18-duckdb, the image the live platform database runs), in a
# container with no network that is stopped (and auto-removed) when the script
# exits. It never touches the live database.
#
#   1. bootstrap an empty platform database from the snapshot WITH the overlay block
#      (the snapshot is the database, D-142 §3 / D-152), after creating the roles
#      its GRANTs name (a schema-only dump carries no CREATE ROLE);
#   2. check the block in the snapshot is byte-identical to the apply script's block;
#   3. apply scripts/2026-09-25-context-review-overlay.sql twice (idempotency);
#   4. run sql/validation/2026-09-25-context-review-horizon-test.sql (ends in ROLLBACK).
#
# Usage (from the repository root, on the desktop):
#   tar -cf - sql/bootstrap/schema_snapshot_20260907.sql scripts/2026-09-25-context-review-overlay.sql \
#       sql/validation/2026-09-25-context-review-horizon-test.sql sql/validation/2026-09-25-context-review-proof.sh \
#     | ssh -i ~/.ssh/ovh root@100.91.190.107 'D=$(mktemp -d) && tar -xf - -C "$D" && bash "$D/sql/validation/2026-09-25-context-review-proof.sh" "$D"'
set -euo pipefail

ROOT="${1:?usage: proof.sh <directory holding the repository-relative files>}"
SNAPSHOT="$ROOT/sql/bootstrap/schema_snapshot_20260907.sql"
OVERLAY="$ROOT/scripts/2026-09-25-context-review-overlay.sql"
TEST="$ROOT/sql/validation/2026-09-25-context-review-horizon-test.sql"
IMAGE="${PROOF_IMAGE:-probata-postgres:18-duckdb}"
NAME="ctxreview-proof-$(date +%s)"

block() { sed -n '/^-- >>> BEGIN context-review-overlay block/,/^-- <<< END context-review-overlay block <<</p' "$1"; }
if [ "$(block "$SNAPSHOT" | sha256sum)" != "$(block "$OVERLAY" | sha256sum)" ]; then
  echo "FAIL: the snapshot block and the apply-script block differ" >&2
  exit 1
fi
echo "PASS 0: snapshot block == apply-script block ($(block "$OVERLAY" | wc -l) lines)"

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
echo "PASS 1: snapshot with the overlay block bootstrapped an empty database in $(( $(date +%s) - start ))s"

echo "--- overlay script, first run"
psql_in < "$OVERLAY"
echo "--- overlay script, second run (must be a no-op)"
psql_in < "$OVERLAY"
echo "PASS 2: apply script ran twice without error"

echo "--- horizon test"
psql_in < "$TEST" 2>&1
echo "PASS 3: horizon test completed (every assertion above printed PASS; the transaction rolled back)"
