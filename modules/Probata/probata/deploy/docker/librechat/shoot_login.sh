#!/usr/bin/env bash
# Byline: Claude Code · Fable 5.1 · 2026-09-26 (form-text check fixed the same night)
#
# Headless-Chrome proof shots of LibreChat's signed-out pages (login, register). Runs INSIDE the
# Probata devbox on ovh-files as kasm-user, never on the owner's desktop. Nothing here signs in.
#
#   usage: shoot_login.sh <out-dir> [base-url]      (base-url defaults to the tailnet name)
#   e.g.   ssh ovh-files 'docker exec -i -u kasm-user <devbox> bash -s -- /tmp/lc-shots' < shoot_login.sh
#
# Writes librechat-login.png and librechat-register.png into <out-dir>, and prints each page's
# <title> and the form texts found in the rendered DOM ("Welcome back", "Create your account",
# "Email address", "Continue", "Sign up"), so "the page rendered" is read from the page instead of
# eyeballed. v0.8.7's email field is not type="email"; the first run died on that grep (pipefail).
set -euo pipefail

out="${1:?usage: shoot_login.sh <out-dir> [base-url]}"
base="${2:-https://librechat.tilapia-skilift.ts.net}"
chrome="${CHROME_BIN:-/opt/google/chrome/chrome}"
mkdir -p "$out"
profile="$(mktemp -d)"
common=(--headless=new --no-first-run --no-default-browser-check --disable-gpu --hide-scrollbars
  --mute-audio "--user-data-dir=$profile" --window-size=1400,900 --virtual-time-budget=20000)

for page in login register; do
  "$chrome" "${common[@]}" "--screenshot=$out/librechat-$page.png" "$base/$page" 2>/dev/null
  dom="$("$chrome" "${common[@]}" --dump-dom "$base/$page" 2>/dev/null)"
  title="$(printf '%s' "$dom" | grep -o '<title>[^<]*</title>' | head -1 || true)"
  texts="$(printf '%s' "$dom" | grep -oE 'Welcome back|Create your account|Email address|Continue|Sign up' | sort | uniq -c | tr -s ' ' | paste -sd ';' - || true)"
  echo "$page: ${title:-<no title>} form_texts=[${texts:-none}] png_bytes=$(stat -c %s "$out/librechat-$page.png")"
done
