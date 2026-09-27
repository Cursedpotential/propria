#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# Logged-out probes of the public *.int.mitechconsult.com routes, following redirects, from any
# machine off the ovh-app host (the owner's desktop is a real external vantage point). Read-only.
# Pass: final HTTP 200 on the Authentik flow, the page's api.base is https://auth.int.../, and no
# http:// URL to our own domain in the page (the 2026-09-23 mixed-content failure mode).
#   public_probe.sh                 # every public route
#   public_probe.sh homepage auth   # just these hosts
set -u
hosts="${*:-homepage auth workbench legal metabase filestash progress attu neo4j files n8n temporal contextforge portkey llmprobe opencode infisical databasement edit devbox fileflows librechat family-court}"
tmp="$(mktemp -d)"; ok=0; bad=0
for h in $hosts; do
  out=$(curl -sS -L --max-redirs 10 -m 25 -o "$tmp/body" -w '%{http_code} %{num_redirects} %{url_effective}' "https://$h.int.mitechconsult.com/" 2>&1)
  code=${out%% *}
  final=$(echo "$out" | awk '{print $3}' | cut -d'?' -f1)
  base=$(grep -o -E 'base: "[^"]*"' "$tmp/body" 2>/dev/null | head -1)
  insecure=$(grep -o 'http://[a-z0-9.-]*mitechconsult\.com' "$tmp/body" 2>/dev/null | wc -l | tr -d ' ')
  if [ "$code" = 200 ] && [ "$base" = 'base: "https://auth.int.mitechconsult.com/"' ] && [ "$insecure" = 0 ]; then
    ok=$((ok+1)); verdict=OK; else bad=$((bad+1)); verdict=FAIL; fi
  printf '%-4s %-13s code=%s redirects=%s final=%s %s http-to-us=%s\n' "$verdict" "$h" "$code" \
    "$(echo "$out" | awk '{print $2}')" "$final" "$base" "$insecure"
done
echo "SUMMARY ok=$ok fail=$bad at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
[ "$bad" = 0 ]
