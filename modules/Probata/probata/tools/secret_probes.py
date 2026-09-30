"""Read-only probes that decide whether a credential is still accepted by its service.

One registry, imported by every caller, because a second copy drifts. Today two callers use it:

    secrets-validate.py     reads ~/.secrets      the one-time migration audit
    infisical-validate.py   reads Infisical       the durable one, runs anywhere

Nothing here knows where a secret came from. A probe receives the credential's name and the
whole group it belongs to -- a file's env, or a folder in Infisical -- because most credentials
are pairs: R2 needs an id beside its secret, Tailscale an OAuth client id, Backblaze a key id.
Probing a value alone is why an earlier pass could only test 19 of 1,083 entries.

VERDICTS
    live        the service accepted it
    dead        401, 403, or an explicit invalid-credential answer
    unreachable the service could not be contacted; the credential is unproven, NOT revoked
    untestable  no probe for this kind, or the partner value is missing

`unreachable` is deliberately not `dead`. Quarantining on a network blip would remove working
credentials.

Every probe is read-only: list models, fetch the current user, read a version. No value is
returned or logged by anything here.

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import base64
import json
import pathlib
import re
import subprocess
import urllib.error
import urllib.parse
import urllib.request

TIMEOUT = 25
RCLONE = pathlib.Path("C:/Users/matts/scoop/shims/rclone.exe")

PLACEHOLDER = re.compile(
    r"^(changeme|change_me|replace(_me)?|x{3,}|your[_-]?\w+|<.*>|true|false|\d{1,5})$", re.I)
NOT_A_SECRET = re.compile(
    r"^([A-Z0-9_]*_)?(URL|URI|HOST|PORT|FQDN|ENDPOINT|REGION|BUCKET|PROJECT|PROJECT_ID|"
    r"ACCOUNT_ID|EMAIL|USER|USERNAME|DIR|PATH|MODEL|VERSION|ENABLED|TIMEOUT|NAMESPACE|DB|"
    r"DATABASE|COLLECTION|INDEX|TZ|LANG|ENV|NAME|PREFIX|MODE|LEVEL|FORMAT)$")
# A family match is not enough. GROQ_MODEL_PREF matched the Groq family, was sent as a key,
# returned 401 and was almost quarantined as a revoked credential. It is a model name.
CREDENTIAL_SHAPED = re.compile(
    r"(API_?KEY|_KEY$|^KEY$|TOKEN|SECRET|PASSWORD|PASSWD|^PASS$|_PASS$|CREDENTIAL|"
    r"PRIVATE_?KEY|ACCESS_?KEY|AUTH$|_AUTH$|DSN|SALT|SIGNING)", re.I)


def http(url: str, headers: dict, method: str = "GET", data: bytes | None = None):
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            return response.status, response.read(400).decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read(300).decode("utf-8", "replace")
    except Exception as exc:  # noqa: BLE001 - reachability, not a verdict about the credential
        return 0, type(exc).__name__


def verdict(status: int, body: str = "") -> str:
    if status == 0:
        return "unreachable"
    if status in (401, 403):
        return "dead"
    if 200 <= status < 300 or status in (404, 429):
        return "live"          # authenticated; path may not exist, or we are throttled
    if re.search(r"invalid.*(key|token|credential)|unauthor|expired", body, re.I):
        return "dead"
    if status in (400, 422):
        return "live"          # the credential passed; the request body did not
    return "unreachable"


def _bearer(url, token):
    return verdict(*http(url, {"Authorization": f"Bearer {token}"}))


def p_nvidia(g, k):     return _bearer("https://integrate.api.nvidia.com/v1/models", g[k])
def p_openrouter(g, k): return _bearer("https://openrouter.ai/api/v1/key", g[k])
def p_openai(g, k):     return _bearer("https://api.openai.com/v1/models", g[k])
def p_groq(g, k):       return _bearer("https://api.groq.com/openai/v1/models", g[k])
def p_mistral(g, k):    return _bearer("https://api.mistral.ai/v1/models", g[k])
def p_together(g, k):   return _bearer("https://api.together.xyz/v1/models", g[k])
def p_cohere(g, k):     return _bearer("https://api.cohere.ai/v1/models", g[k])
def p_hf(g, k):         return _bearer("https://huggingface.co/api/whoami-v2", g[k])
def p_firecrawl(g, k):  return _bearer("https://api.firecrawl.dev/v1/team/credit-usage", g[k])
def p_langfuse(g, k):   return _bearer("https://cloud.langfuse.com/api/public/projects", g[k])


def p_deepgram(g, k):
    return verdict(*http("https://api.deepgram.com/v1/projects", {"Authorization": f"Token {g[k]}"}))


def p_anthropic(g, k):
    return verdict(*http("https://api.anthropic.com/v1/models",
                         {"x-api-key": g[k], "anthropic-version": "2023-06-01"}))


def p_github(g, k):
    return verdict(*http("https://api.github.com/user",
                         {"Authorization": f"Bearer {g[k]}", "Accept": "application/vnd.github+json"}))


def p_voyage(g, k):
    return verdict(*http("https://api.voyageai.com/v1/embeddings",
                         {"Authorization": f"Bearer {g[k]}", "Content-Type": "application/json"},
                         "POST", json.dumps({"input": ["ping"], "model": "voyage-3"}).encode()))


def p_coolify(g, k):
    return verdict(*http("http://100.98.98.38:8000/api/v1/version",
                         {"Authorization": f"Bearer {g[k]}"}))


def p_tavily(g, k):
    return verdict(*http("https://api.tavily.com/search", {"Content-Type": "application/json"},
                         "POST", json.dumps({"api_key": g[k], "query": "ping",
                                             "max_results": 1}).encode()))


def p_exa(g, k):
    return verdict(*http("https://api.exa.ai/search",
                         {"x-api-key": g[k], "Content-Type": "application/json"}, "POST",
                         json.dumps({"query": "ping", "numResults": 1}).encode()))


def p_perplexity(g, k):
    return verdict(*http("https://api.perplexity.ai/chat/completions",
                         {"Authorization": f"Bearer {g[k]}", "Content-Type": "application/json"},
                         "POST", json.dumps({"model": "sonar",
                                             "messages": [{"role": "user", "content": "hi"}],
                                             "max_tokens": 1}).encode()))


def p_cloudflare(g, k):
    return verdict(*http("https://api.cloudflare.com/client/v4/user/tokens/verify",
                         {"Authorization": f"Bearer {g[k]}"}))


def p_gemini(g, k):
    """Google AI Studio keys travel in the query string, not a header."""
    return verdict(*http("https://generativelanguage.googleapis.com/v1beta/models?key="
                         + urllib.parse.quote(g[k]), {}))


def p_b2(g, k):
    key_id = next((g[n] for n in g
                   if re.search(r"B2.*(KEY_?ID|ACCOUNT_?ID|APPLICATION_?KEY_?ID)", n, re.I)
                   and g[n] != g[k]), None)
    if not key_id:
        return "untestable"
    basic = base64.b64encode(f"{key_id}:{g[k]}".encode()).decode()
    return verdict(*http("https://api.backblazeb2.com/b2api/v3/b2_authorize_account",
                         {"Authorization": "Basic " + basic}))


def p_tailscale(g, k):
    client_id = next((g[n] for n in g if re.search(r"TAILSCALE.*CLIENT_?ID", n, re.I)), None)
    if not client_id:
        return "untestable"
    body = urllib.parse.urlencode({"client_id": client_id, "client_secret": g[k]}).encode()
    return verdict(*http("https://api.tailscale.com/api/v2/oauth/token",
                         {"Content-Type": "application/x-www-form-urlencoded"}, "POST", body))


def p_r2(g, k):
    """R2 is S3-compatible. Signing a request by hand here would duplicate rclone's config."""
    if not RCLONE.is_file():
        return "untestable"
    for remote in ("r2", "R2", "cloudflare-r2", "casebible-r2", "propria-r2"):
        done = subprocess.run([str(RCLONE), "lsd", f"{remote}:", "--max-depth", "1",
                               "--low-level-retries", "1", "--timeout", "30s"],
                              capture_output=True, text=True, timeout=90)
        if done.returncode == 0:
            return "live"
        if re.search(r"didn.t find section|unknown remote", done.stderr, re.I):
            continue
        if re.search(r"401|403|SignatureDoesNotMatch|InvalidAccessKeyId|Unauthorized",
                     done.stderr, re.I):
            return "dead"
    return "untestable"


def p_surreal(g, k):
    url = next((g[n] for n in g if re.search(r"SURREAL.*(URL|HOST)", n, re.I)), None)
    user = next((g[n] for n in g if re.search(r"SURREAL.*USER", n, re.I)), None)
    if not (url and user):
        return "untestable"
    endpoint = re.sub(r"^wss?://", "https://", url).replace("/rpc", "")
    basic = base64.b64encode(f"{user}:{g[k]}".encode()).decode()
    return verdict(*http(endpoint + "/sql",
                         {"Authorization": "Basic " + basic, "Accept": "application/json",
                          "surreal-ns": "probata", "surreal-db": "docs"}, "POST", b"RETURN 1;"))


def p_neo4j(g, k):
    url = next((g[n] for n in g if re.search(r"NEO4J.*(URL|URI|HOST)", n, re.I)), None)
    user = next((g[n] for n in g if re.search(r"NEO4J.*USER", n, re.I)), "neo4j")
    if not url:
        return "untestable"
    endpoint = re.sub(r"^(bolt|neo4j)(\+s)?://", "http://", url).rstrip("/")
    basic = base64.b64encode(f"{user}:{g[k]}".encode()).decode()
    return verdict(*http(endpoint + "/db/neo4j/tx/commit",
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
    (re.compile(r"LANGFUSE", re.I), p_langfuse),
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

FAMILIES = len(PROBES)


def pick(name: str):
    """The probe for this credential name, or None when nothing can test it."""
    if NOT_A_SECRET.match(name) or not CREDENTIAL_SHAPED.search(name):
        return None
    for pattern, probe in PROBES:
        if pattern.search(name):
            return probe
    return None


def check(group: dict[str, str], name: str) -> str:
    """Verdict for one credential, given the group it belongs to."""
    value = group.get(name, "")
    if not value or PLACEHOLDER.match(value):
        return "untestable"
    probe = pick(name)
    if probe is None:
        return "untestable"
    try:
        return probe(group, name)
    except Exception:  # noqa: BLE001 - a probe crash is not a verdict about the credential
        return "unreachable"
