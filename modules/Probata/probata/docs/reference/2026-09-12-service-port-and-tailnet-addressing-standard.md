# Service port families and Tailscale-only addressing

> Byline: Codex · GPT-5 · 2026-09-12
>
> Status: owner-directed addressing standard; source adoption started, live
> fleet migration staged and incomplete.

## Decision

Every service gets a stable two-digit product code, `NN`. The leading two
digits identify the surface class; the last two digits identify the same
product across its database, API and portal surfaces.

| Surface class | Backend family | Shape | Example for product `72` |
|---|---:|---|---:|
| PostgreSQL | `5400–5499` | `54NN` | `5472` |
| SurrealDB | `8400–8499` | `84NN` | `8472` |
| machine HTTP/API/MCP | `8000–8099` | `80NN` | `8072` |
| human portal/UI | `9000–9099` | `90NN` | `9072` |

Additional protocol families must be registered before use; they must preserve
the same `NN` suffix. Never allocate the next apparently free port without
checking the registry and live listeners.

## Initial product codes

| `NN` | Product/service | Current bindings | Canonical shape |
|---:|---|---|---|
| `71` | Case Bible / Probata graph store | SurrealDB `8471` | DB `8471`; API `8071`; portal `9071`; PG `5471` if needed |
| `72` | Docstore | SurrealDB `8472`; HTTP worker `8474` | DB `8472`; API `8072`; portal `9072`; PG `5472` if needed |
| `73` | Intake filesystem index | SurrealDB `8473` | DB `8473`; API `8073`; portal `9073`; PG `5473` if needed |

Codes are identities, not ordinals to recycle. Retired services leave a
reservation or an explicit release record.

## Client addressing

Applications, agents, desktop tools and operator documentation use Tailscale
Service DNS names, never raw `100.x` addresses or backend ports:

- `wss://surreal-docs.tilapia-skilift.ts.net`
- `https://docstore-api.tilapia-skilift.ts.net`
- `wss://surreal-intake.tilapia-skilift.ts.net`
- the corresponding named PostgreSQL service on standard client port `5432`
  once that TCP Service is registered and verified.

Backend ports remain only in host-local Docker/Coolify publication, Tailscale
Service proxy configuration and explicit diagnostic receipts. Loopback
healthchecks may use their container-local port. Historical documents keep the
address that was true at the time and must be labeled historical rather than
silently rewritten.

Tailscale Services provide stable MagicDNS/TailVIP identity independent of the
hosting node. HTTPS/WSS surfaces are presented on `443`; raw TCP services such
as PostgreSQL can be registered at their standard client port and forwarded to
the classified backend port. Access grants remain separate from service
advertisement and auto-approval.

## Collision and migration gate

For each service:

1. Query Docstore for current ownership/port decisions.
2. Read live Docker bindings, listeners and Tailscale Service registration.
3. Reserve `NN` and its required class ports in this registry.
4. Register/verify the Tailscale Service name and client access grant.
5. Prove the new backend listener without removing the old route.
6. Repoint only that named Service, then test through its DNS name.
7. Update active client configuration to the DNS name and independently verify.
8. Retire the old backend binding in a later controlled deployment.

No bulk port rewrite, `tailscale serve reset`, raw-IP fallback, or simultaneous
unverified cutover is permitted. Coolify-owned services change through their
reviewed source/deployment path; agents do not mutate containers around it.

## Verified 2026-09-12

Read-only host inspection found `8471=surreal-case`,
`8472=surreal-docs`, `8473=surreal-intake`, and
`8474=docstore-worker`. `svc:docstore-api` currently proxies to `8474`.
The Docstore control environment and the existing worker container both
successfully authenticated and executed `RETURN 1` through
`wss://surreal-docs.tilapia-skilift.ts.net`; credential values were not printed.

The Docstore worker source default now uses that WSS hostname. Its compose source
temporarily publishes both canonical `8072` and legacy `8474`; that is the safe
first half of the two-step Coolify/Tailscale cutover. It is not yet deployed.
After deployment proves `8072`, repoint only `svc:docstore-api`, verify the DNS
route, and remove `8474` in a later deployment. The repository's
current `main` is dirty and diverged from `origin/main`; no broad merge, push or
automatic deployment was attempted.

**Status 2026-09-26 (Claude Code · Opus 5.5): the cutover is complete, by a different route.**
- Docstore 0.8.1 (2026-09-20) replaced the worker app. The owner deleted that app on 2026-09-25, and `8474` had nothing listening.
- The Coolify service `propria-docstore-0-8-1` now publishes the worker API on `100.91.190.107:8072` (image `propria-docstore:0.8.1-r2` onward).
- `svc:docstore-api` proxies to `8072`, and `https://docstore-api.tilapia-skilift.ts.net/health` answers 200.
- `8474` is retired in `deploy/service-port-registry.json`.
- Receipt: `docs/pending-review/2026-09-26-docstore-0.8.1-r2/`.

`deploy/service-port-registry.json` is the machine-readable allocation source.
`scripts/validate_service_ports.py` enforces unique two-digit product codes,
class-prefix/product-suffix canonical ports, unique Tailscale Service identities,
and DNS-only client endpoints without network access or mutation. Its canonical
registry proof passed for three products/four endpoints; the complete Docstore
control suite passed 232 tests with two explicitly gated live revision tests
skipped.

## Fleet debt

The rest of the VPS predates this standard. Current APIs and portals occupy
many unrelated ports (`3001`, `4096`, `5244`, `5678`, `8030/8031`, `8090`,
`8233`, `8384`, `8880`, and others). They require an explicit product-code
inventory and one-service-at-a-time migration. This document does not declare
those ports corrected merely because the standard now exists.
