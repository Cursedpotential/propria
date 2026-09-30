"""Bootstrap the self-hosted Infisical instance and record its credentials.

Infisical has run on ovh-files since 2026-09-26 but was never initialised: 0 projects, 0 secrets,
0 users, 0 identities, and `initialized: false`. Nothing consumed it. This creates the first admin
and the organization, which is the only step that cannot be automated afterwards.

WHY THIS ROUTE. `POST /api/v1/admin/signup` demands the end-to-end-encryption bootstrap fields a
browser normally derives -- protectedKey, its IV and tag, publicKey, encryptedPrivateKey and its
IV and tag, salt and an SRP verifier. Deriving those by hand is how you produce an account nobody
can log into. `POST /api/v1/admin/bootstrap` exists for exactly this case and takes three fields.

The password arrives on stdin, never as an argument, so it stays out of argv and out of this file.
Results are appended to ~/.secrets/infisical.env, which is not a git repository. Nothing secret is
printed; the script reports lengths and identifiers only.

Usage:
    printf '%s' '<password>' | python tools/infisical-bootstrap.py \
        --email you@example.com --organization Propria

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import json
import pathlib
import sys
import urllib.error
import urllib.request

BASE = "http://100.91.190.107:8880"
SECRETS = pathlib.Path.home() / ".secrets" / "infisical.env"


def call(path: str, method: str = "GET", body: dict | None = None, token: str | None = None):
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            raw = response.read()
            return response.status, (json.loads(raw) if raw[:1] in (b"{", b"[") else raw[:200])
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read()[:400]


def record(pairs: dict[str, str]) -> None:
    """Append to ~/.secrets/infisical.env, never overwriting what is already there."""
    existing = SECRETS.read_text(encoding="utf-8") if SECRETS.is_file() else ""
    lines = [l for l in (existing.splitlines() if existing else []) ]
    have = {l.split("=", 1)[0].strip() for l in lines if "=" in l and not l.strip().startswith("#")}
    added = []
    for key, value in pairs.items():
        if key in have:
            print(f"  {key} already recorded; left as it is")
            continue
        lines.append(f"{key}={value}")
        added.append(key)
    if added:
        SECRETS.write_text("\n".join(lines).rstrip("\n") + "\n", encoding="utf-8")
    print(f"  recorded in {SECRETS}: {', '.join(added) if added else 'nothing new'}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--email", required=True)
    parser.add_argument("--organization", default="Propria")
    args = parser.parse_args()

    password = sys.stdin.read().strip()
    if len(password) < 8:
        return int(bool(sys.stderr.write("  refusing: password came through empty or too short\n")))
    print(f"  password received on stdin: {len(password)} chars")

    status, config = call("/api/v1/admin/config")
    if status != 200:
        return int(bool(sys.stderr.write(f"  cannot read admin config: HTTP {status}\n")))
    if (config.get("config") or {}).get("initialized"):
        print("  instance already initialised; bootstrap is a one-time step, nothing to do")
        return 0

    status, out = call("/api/v1/admin/bootstrap", "POST", {
        "email": args.email, "password": password, "organization": args.organization})
    print(f"  POST /api/v1/admin/bootstrap -> HTTP {status}")
    if status not in (200, 201):
        sys.stderr.write("  " + str(out)[:350] + "\n")
        return 1

    user = (out.get("user") or {}) if isinstance(out, dict) else {}
    identity = (out.get("identity") or {}) if isinstance(out, dict) else {}
    org = (out.get("organization") or {}) if isinstance(out, dict) else {}
    token = (identity.get("credentials") or {}).get("token") or identity.get("token")

    print(f"  admin user   : {user.get('email') or user.get('username')}  id={user.get('id')}")
    print(f"  organization : {org.get('name')}  slug={org.get('slug')}  id={org.get('id')}")
    print(f"  identity     : {identity.get('name')}  id={identity.get('id')}  "
          f"token={'yes, ' + str(len(token)) + ' chars' if token else 'none returned'}")

    pairs = {"INFISICAL_URL": BASE, "INFISICAL_ADMIN_EMAIL": args.email,
             "INFISICAL_ADMIN_PASSWORD": password}
    if org.get("id"):
        pairs["INFISICAL_ORG_ID"] = org["id"]
    if token:
        pairs["INFISICAL_MACHINE_TOKEN"] = token
    record(pairs)
    return 0


if __name__ == "__main__":
    sys.exit(main())
