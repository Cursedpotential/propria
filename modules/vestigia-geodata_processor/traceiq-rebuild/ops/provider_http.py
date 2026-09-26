# Byline: Claude Code · Opus 5 · 2026-07-25
# Single request builder shared by provider.py and provider_test.py.
#
# This module exists because the builder was duplicated once and the copies drifted: a
# User-Agent fix landed in one and not the other, so the same provider passed a direct test
# and failed the probe. One definition, imported by both.
import urllib.request

# Cloudflare-fronted providers (Radar) reject urllib's DEFAULT User-Agent with
# HTTP 403 "error code: 1010" — that is a client-signature ban, NOT an auth failure. Every
# key looks dead without an explicit UA, which is very likely why these keys were once
# assumed expired. We identify ourselves honestly rather than spoofing a browser.
USER_AGENT = "traceiq-ops/0.1 (+forensic timeline tool; contact: owner)"


def build_request(base, tpl, key, args):
    """Build an HTTP request from a provider call template.

    {key} is substituted into BOTH the URL and any header value, so each provider's auth
    style stays DATA (ADR-0014) rather than code:
      Radar    -> headers {"Authorization": "{key}"}   (raw key, no Bearer prefix)
      HERE     -> url    "...&apiKey={key}"
      Geoapify -> url    "...&apiKey={key}"
      Google   -> url    "...&key={key}"
    The credential value is never printed or logged.
    """
    url = (base or "") + tpl["url"].format(key=key or "", **args)
    headers = {k: str(v).format(key=key or "", **args)
               for k, v in (tpl.get("headers") or {}).items()}
    headers.setdefault("User-Agent", USER_AGENT)
    return urllib.request.Request(url, method=tpl.get("method", "GET"), headers=headers)
