"""Probe every credential in ~/.secrets against its real service, before anything is migrated.

Owner, 2026-09-28: "You're gonna need to actually validate all of the ones in the dot secrets
directory... Don't add fucking stale shit."

Filtering by filename -- skipping .bak and STALE -- is a guess, not validation. A revoked key in a
normally named file looks identical to a live one, and the first pass proved it: the live
cloudflare.env holds a CLOUDFLARE_API_TOKEN the service rejects.

Each probe receives the WHOLE env dict of the file a key came from, because most credentials are
pairs -- R2 needs an id and a secret, Tailscale an OAuth client id and secret, Backblaze a key id
beside the application key. Probing values in isolation is why 678 entries had no probe on the
first pass.

VERDICTS
    live        the service accepted it
    dead        the service rejected it: 401, 403, or an explicit invalid-credential answer
    unreachable the service could not be contacted, so the credential is unproven either way
    untestable  no read-only probe exists for this kind, or it is not a credential at all

Only `live` is safe to migrate. `dead` is what makes a secrets directory untrustworthy.
`unreachable` is kept apart from `dead` on purpose: an unproven credential is not a revoked one,
and treating it as revoked would delete working keys.

Every probe is read-only -- list models, fetch the current user, read a version. Nothing is
created, modified or deleted, and no value is ever printed: the report carries names, lengths and
sha256 prefixes. Identical values are probed once however many files repeat them.

Usage:
    python tools/secrets-validate.py
    python tools/secrets-validate.py --kind nvidia
    python tools/secrets-validate.py --out report.json

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import base64
import concurrent.futures
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

HOME = pathlib.Path.home()
SECRETS_DIR = HOME / ".secrets"
RCLONE = pathlib.Path("C:/Users/matts/scoop/shims/rclone.exe")
TIMEOUT = 25

PLACEHOLDER = re.compile(
    r"^(changeme|change_me|replace(_me)?|x{3,}|your[_-]?\w+|<.*>|true|false|\d{1,5})$", re.I)
QUARANTINE = re.compile(r"^(_dead-\d|to_be_deleted/)")
NOT_A_SECRET = re.compile(
    r"^([A-Z0-9_]*_)?(URL|URI|HOST|PORT|FQDN|ENDPOINT|REGION|BUCKET|PROJECT|PROJECT_ID|"
    r"ACCOUNT_ID|EMAIL|USER|USERNAME|DIR|PATH|MODEL|VERSION|ENABLED|TIMEOUT|NAMESPACE|DB|"
    r"DATABASE|COLLECTION|INDEX|TZ|LANG|ENV|NAME|PREFIX|MODE|LEVEL|FORMAT)$")


def digest(value: str) -> str:
    return hashlib.sha256(value.encode()).hexdigest()[:12]


def get(url: str, headers: dict, method: str = "GET", data: bytes | None = None):
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            return response.status, response.read(400).decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read(300).decode("utf-8", "replace")
    except Exception as exc:  # noqa: BLE001 - reachability, not a verdict about the credential
        return 0, type(exc).__name__


def verdict_from(status: int, body: str) -> str:
    if status == 0:
        return "unreachable"
    if status in (401, 403):
        return "dead"
    if 200 <= status < 300 or status in (404, 429):
        return "live"          # authenticated; the path may not exist or we may be throttled
    if re.search(r"invalid.*(key|token|credential)|unauthor|expired", body, re.I):
        return "dead"
    if status in (400, 422):
        return "live"          # the credential passed; the request body did not
    return "unreachable"


# --- probes -----------------------------------------------------------------------------------

def _bearer(url, token, **kw):
    return verdict_from(*get(url, {"Authorization": f"Bearer {token}"}, **kw))


def p_nvidia(env, k):     return _bearer("https://integrate.api.nvidia.com/v1/models", env[k])
def p_openrouter(env, k): return _bearer("https://openrouter.ai/api/v1/key", env[k])
def p_openai(env, k):     return _bearer("https://api.openai.com/v1/models", env[k])
def p_groq(env, k):       return _bearer("https://api.groq.com/openai/v1/models", env[k])
def p_mistral(env, k):    return _bearer("https://api.mistral.ai/v1/models", env[k])
def p_together(env, k):   return _bearer("https://api.together.xyz/v1/models", env[k])
def p_cohere(env, k):     return _bearer("https://api.cohere.ai/v1/models", env[k])
def p_hf(env, k):         return _bearer("https://huggingface.co/api/whoami-v2", env[k])
def p_firecrawl(env, k):  return _bearer("https://api.firecrawl.dev/v1/team/credit-usage", env[k])
def p_langfuse(env, k):   return _bearer("https://cloud.langfuse.com/api/public/projects", env[k])


def p_deepgram(env, k):
    return verdict_from(*get("https://api.deepgram.com/v1/projects",
                             {"Authorization": f"Token {env[k]}"}))


def p_anthropic(env, k):
    return verdict_from(*get("https://api.anthropic.com/v1/models",
                             {"x-api-key": env[k], "anthropic-version": "2023-06-01"}))


def p_github(env, k):
    return verdict_from(*get("https://api.github.com/user",
                             {"Authorization": f"Bearer {env[k]}",
                              "Accept": "application/vnd.github+json"}))


def p_voyage(env, k):
    return verdict_from(*get(
        "https://api.voyageai.com/v1/embeddings",
        {"Authorization": f"Bearer {env[k]}", "Content-Type": "application/json"},
        "POST", json.dumps({"input": ["ping"], "model": "voyage-3"}).encode()))


def p_coolify(env, k):
    return verdict_from(*get("http://100.98.98.38:8000/api/v1/version",
                             {"Authorization": f"Bearer {env[k]}"}))


def p_tavily(env, k):
    return verdict_from(*get(
        "https://api.tavily.com/search", {"Content-Type": "application/json"}, "POST",
        json.dumps({"api_key": env[k], "query": "ping", "max_results": 1}).encode()))


def p_exa(env, k):
    return verdict_from(*get(
        "https://api.exa.ai/search", {"x-api-key": env[k], "Content-Type": "application/json"},
        "POST", json.dumps({"query": "ping", "numResults": 1}).encode()))


def p_perplexity(env, k):
    return verdict_from(*get(
        "https://api.perplexity.ai/chat/completions",
        {"Authorization": f"Bearer {env[k]}", "Content-Type": "application/json"}, "POST",
        json.dumps({"model": "sonar", "messages": [{"role": "user", "content": "hi"}],
                    "max_tokens": 1}).encode()))


def p_cloudflare(env, k):
    return verdict_from(*get("https://api.cloudflare.com/client/v4/user/tokens/verify",
                             {"Authorization": f"Bearer {env[k]}"}))


def p_gemini(env, k):
    """Google AI Studio keys travel in the query string, not a header."""
    return verdict_from(*get(
        "https://generativelanguage.googleapis.com/v1beta/models?key="
        + urllib.parse.quote(env[k]), {}))


def p_b2(env, k):
    """Backblaze: the key id is the partner of the application key."""
    key_id = next((env[n] for n in env
                   if re.search(r"B2.*(KEY_?ID|ACCOUNT_?ID|APPLICATION_?KEY_?ID)", n, re.I)
                   and env[n] != env[k]), None)
    if not key_id:
        return "untestable"
    basic = base64.b64encode(f"{key_id}:{env[k]}".encode()).decode()
    return verdict_from(*get("https://api.backblazeb2.com/b2api/v3/b2_authorize_account",
                             {"Authorization": "Basic " + basic}))


def p_tailscale(env, k):
    """OAuth client credentials; the id is the partner of the secret."""
    client_id = next((env[n] for n in env if re.search(r"TAILSCALE.*CLIENT_?ID", n, re.I)), None)
    if not client_id:
        return "untestable"
    body = urllib.parse.urlencode({"client_id": client_id, "client_secret": env[k]}).encode()
    return verdict_from(*get("https://api.tailscale.com/api/v2/oauth/token",
                             {"Content-Type": "application/x-www-form-urlencoded"}, "POST", body))


def _rclone(remote: str) -> str:
    if not RCLONE.is_file():
        return "untestable"
    done = subprocess.run([str(RCLONE), "lsd", f"{remote}:", "--max-depth", "1",
                           "--low-level-retries", "1", "--timeout", "30s"],
                          capture_output=True, text=True, timeout=90)
    if done.returncode == 0:
        return "live"
    err = done.stderr
    if re.search(r"didn.t find section|unknown remote", err, re.I):
        return "untestable"
    if re.search(r"401|403|SignatureDoesNotMatch|InvalidAccessKeyId|Unauthorized|invalid_grant",
                 err, re.I):
        return "dead"
    return "unreachable"


def p_r2(env, k):
    for remote in ("r2", "R2", "cloudflare-r2", "casebible-r2", "propria-r2"):
        state = _rclone(remote)
        if state != "untestable":
            return state
    return "untestable"


def p_surreal(env, k):
    url = next((env[n] for n in env if re.search(r"SURREAL.*(URL|HOST)", n, re.I)), None)
    user = next((env[n] for n in env if re.search(r"SURREAL.*USER", n, re.I)), None)
    if not (url and user):
        return "untestable"
    http = re.sub(r"^wss?://", "https://", url).replace("/rpc", "")
    basic = base64.b64encode(f"{user}:{env[k]}".encode()).decode()
    return verdict_from(*get(http + "/sql",
                             {"Authorization": "Basic " + basic, "Accept": "application/json",
                              "surreal-ns": "probata", "surreal-db": "docs"}, "POST", b"RETURN 1;"))


def p_neo4j(env, k):
    url = next((env[n] for n in env if re.search(r"NEO4J.*(URL|URI|HOST)", n, re.I)), None)
    user = next((env[n] for n in env if re.search(r"NEO4J.*USER", n, re.I)), "neo4j")
    if not url:
        return "untestable"
    http = re.sub(r"^(bolt|neo4j)(\+s)?://", "http://", url)
    basic = base64.b64encode(f"{user}:{env[k]}".encode()).decode()
    return verdict_from(*get(http.rstrip("/") + "/db/neo4j/tx/commit",
                             {"Authorization": "Basic " + basic, "Content-Type": "application/json"},
                             "POST", json.dumps({"statements": [{"statement": "RETURN 1"}]}).encode()))


PROBES = [
    (re.compile(r"NVIDIA|NIM_", re.I), p_nvidia),
    (re.compile(r"OPENROUTER", re.I), p_openrouter),
    (re.compile(r"OPENAI.*KEY", re.I), p_openai),
    (re.compile(r"ANTHROPIC", re.I), p_anthropic),
    (re.compile(r"GROQ", re.I), p_groq),
    (re.compile(r"MISTRAL", re.I), p_mistral),
    (re.compile(r"TOGETHER", re.I), p_together),
    (re.compile(r"PERPLEX", re.I), p_perplexity),
    (re.compile(r"COHERE", re.I), p_cohere),
    (re.compile(r"DEEPGRAM", re.I), p_deepgram),
    (re.compile(r"(HUGGING|^HF_).*(TOKEN|KEY)", re.I), p_hf),
    (re.compile(r"FIRECRAWL", re.I), p_firecrawl),
    (re.compile(r"LANGFUSE.*(SECRET|PUBLIC)?_?KEY", re.I), p_langfuse),
    (re.compile(r"GITHUB.*(TOKEN|PAT)|GH_TOKEN", re.I), p_github),
    (re.compile(r"VOYAGE", re.I), p_voyage),
    (re.compile(r"COOLIFY_API_TOKEN", re.I), p_coolify),
    (re.compile(r"TAVILY", re.I), p_tavily),
    (re.compile(r"^EXA_|EXA_API", re.I), p_exa),
    (re.compile(r"CLOUDFLARE.*(TOKEN|KEY)|^CF_API_TOKEN$", re.I), p_cloudflare),
    (re.compile(r"GEMINI.*KEY|GOOGLE.*(AI|GENERATIVE).*KEY", re.I), p_gemini),
    (re.compile(r"B2.*(APPLICATION_?KEY|APP_?KEY|SECRET)", re.I), p_b2),
    (re.compile(r"TAILSCALE.*(CLIENT_?SECRET|API_?KEY)", re.I), p_tailscale),
    (re.compile(r"R2.*(SECRET|ACCESS_?KEY)", re.I), p_r2),
    (re.compile(r"SURREAL.*PASS", re.I), p_surreal),
    (re.compile(r"NEO4J.*(PASS|AUTH)", re.I), p_neo4j),
]


# A probe family matching is not enough. GROQ_MODEL_PREF matched the GROQ family, was sent as a
# key, came back 401 and was reported dead -- it is a model name. A value is only worth probing if
# the name says it carries a credential.
CREDENTIAL_SHAPED = re.compile(
    r"(API_?KEY|_KEY$|^KEY$|TOKEN|SECRET|PASSWORD|PASSWD|^PASS$|_PASS$|CREDENTIAL|"
    r"PRIVATE_?KEY|ACCESS_?KEY|AUTH$|_AUTH$|DSN|SALT|SIGNING)", re.I)


def pick_probe(name: str):
    if not CREDENTIAL_SHAPED.search(name):
        return None
    for pattern, probe in PROBES:
        if pattern.search(name):
            return probe
    return None


def collect() -> list[tuple[str, str, str, dict]]:
    """(file, name, value, the whole env of that file) for every credential-looking pair."""
    found = []
    for path in sorted(p for p in SECRETS_DIR.rglob("*") if p.is_file()):
        relative = path.relative_to(SECRETS_DIR).as_posix()
        # Skip the quarantine this tool's sibling creates. Re-validating the copies it holds
        # reports its own dead credentials back as findings, which reads as if nothing was fixed.
        if QUARANTINE.match(relative) or path.suffix.lower() == ".json":
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue
        env: dict[str, str] = {}
        for line in text.splitlines():
            if line.lstrip().startswith("#"):
                continue
            m = re.match(r"^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$", line)
            if not m:
                continue
            value = m.group(2).strip()
            if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
                value = value[1:-1]
            if value and not PLACEHOLDER.match(value):
                env[m.group(1)] = value
        for name, value in env.items():
            found.append((relative, name, value, env))
    return found


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kind", default=None, help="only probe names matching this regex")
    parser.add_argument("--out",
                        default=str(pathlib.Path(__file__).parent / "secrets-validation.json"))
    args = parser.parse_args()

    entries = collect()
    print(f"  {len(entries)} key=value pairs across ~/.secrets")

    testable, skipped = [], []
    for relative, name, value, env in entries:
        if NOT_A_SECRET.match(name):
            skipped.append((relative, name, value, "not a credential"))
            continue
        probe = pick_probe(name)
        if probe is None:
            skipped.append((relative, name, value, "no read-only probe for this kind"))
            continue
        if args.kind and not re.search(args.kind, name, re.I):
            continue
        testable.append((relative, name, value, env, probe))

    by_value: dict[str, list] = {}
    for relative, name, value, env, probe in testable:
        by_value.setdefault(value, []).append((relative, name, env, probe))
    print(f"  {len(testable)} probeable entries, {len(by_value)} distinct values to call")

    results: dict[str, str] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        futures = {}
        for value, rows in by_value.items():
            _, name, env, probe = rows[0]
            futures[pool.submit(probe, env, name)] = value
        for future in concurrent.futures.as_completed(futures):
            value = futures[future]
            try:
                results[value] = future.result()
            except Exception:  # noqa: BLE001
                results[value] = "unreachable"

    tally: dict[str, int] = {}
    report = []
    for value, rows in by_value.items():
        state = results.get(value, "unreachable")
        tally[state] = tally.get(state, 0) + 1
        for relative, name, _, _ in rows:
            report.append({"file": relative, "name": name, "len": len(value),
                           "sha256": digest(value), "verdict": state})

    print("\n  verdicts, by distinct value:")
    for state in ("live", "dead", "unreachable", "untestable"):
        if tally.get(state):
            print(f"    {state:12} {tally[state]}")
    print(f"    {'no probe':12} {len(skipped)}  (unprobeable kind, or not a credential)")

    dead_files = sorted({r["file"] for r in report if r["verdict"] == "dead"})
    if dead_files:
        print(f"\n  DEAD credentials appear in {len(dead_files)} files:")
        for name in dead_files[:30]:
            keys = sorted({r["name"] for r in report if r["file"] == name and r["verdict"] == "dead"})
            print(f"    {name:52} {', '.join(keys)[:60]}")
        if len(dead_files) > 30:
            print(f"    … and {len(dead_files) - 30} more")

    out = pathlib.Path(args.out)
    out.write_text(json.dumps(
        {"probed": report,
         "unprobed": [{"file": f, "name": n, "len": len(v), "why": w} for f, n, v, w in skipped]},
        indent=1), encoding="utf-8")
    print(f"\n  report written to {out} (names, lengths and hashes only)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
