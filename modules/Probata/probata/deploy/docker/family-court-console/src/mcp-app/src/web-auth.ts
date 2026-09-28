// Byline: Claude Code · Opus 5.5 · 2026-09-28
//
// Inbound authentication for the Family Law Toolkit web app (web.ts). A port of the Probata
// Workbench's trusted-proxy model (modules/Probata/probata/modules/workbench/api/app/runtime/
// auth.py), per the owner, 2026-09-28 04:49 EDT: every tsnet deployment works this way.
//
//   1. /api/health is the only open path.
//   2. Socket peer is the Tailscale Serve proxy for svc:family-court (exact single hosts in
//      TRUSTED_TAILSCALE_SERVE_PROXY_CIDRS): trust `Tailscale-User-Login`, which Serve strips
//      from the client and sets itself, if it is in TRUSTED_TAILSCALE_LOGINS; or an explicit
//      app-capability device grant (TAILSCALE_DEVICE_CAPABILITY). No login page on the tailnet.
//   3. Socket peer is the Traefik proxy for family-court.int (TRUSTED_AUTH_PROXY_CIDRS): require
//      Authentik's `X-authentik-uid` and `X-authentik-username`, which forward-auth sets.
//   4. Anything else, including identity headers from any other peer, is rejected (403).
// Only the socket peer address counts; X-Forwarded-For and X-Real-IP are never consulted.
// Empty or malformed configuration fails closed. /mcp keeps its own bearer (server.ts).

import { BlockList, isIP } from "node:net";

export interface Principal {
  principal: string;
  subject: string;
  via: "tailscale" | "tailscale-device" | "authentik";
}

export interface AuthConfig {
  serveProxies: BlockList | null;
  authProxies: BlockList | null;
  tailscaleLogins: Set<string>;
  deviceCapability: string;
}

const MAX_HEADER_VALUE_LEN = 256;
const MAX_CAPABILITIES_LEN = 8192;
const CTRL = /[\x00-\x1f\x7f]/;

/** Comma-separated CIDRs; any malformed entry (or, when `singleHost`, any range wider than one
 * address) makes the whole list null, which denies that path. */
export function parseCidrs(value: string | undefined, singleHost: boolean): BlockList | null {
  const parts = (value ?? "").split(",").map((p) => p.trim()).filter(Boolean);
  if (parts.length === 0) return null;
  const list = new BlockList();
  for (const part of parts) {
    const [addr, bitsRaw] = part.split("/");
    const family = isIP(addr);
    if (!family) return null;
    const max = family === 4 ? 32 : 128;
    const bits = bitsRaw === undefined ? max : Number(bitsRaw);
    if (!Number.isInteger(bits) || bits < 0 || bits > max) return null;
    if (singleHost && bits !== max) return null;
    list.addSubnet(addr, bits, family === 4 ? "ipv4" : "ipv6");
  }
  return list;
}

export function loadAuthConfig(env: NodeJS.ProcessEnv = process.env): AuthConfig {
  return {
    serveProxies: parseCidrs(env.TRUSTED_TAILSCALE_SERVE_PROXY_CIDRS, true),
    authProxies: parseCidrs(env.TRUSTED_AUTH_PROXY_CIDRS, false),
    tailscaleLogins: new Set((env.TRUSTED_TAILSCALE_LOGINS ?? "").split(",").map((s) => s.trim().toLowerCase()).filter(Boolean)),
    deviceCapability: (env.TAILSCALE_DEVICE_CAPABILITY ?? "").trim(),
  };
}

function peerIn(list: BlockList | null, peer: string | undefined): boolean {
  if (!list || !peer) return false;
  const addr = peer.startsWith("::ffff:") && isIP(peer.slice(7)) === 4 ? peer.slice(7) : peer;
  const family = isIP(addr);
  if (!family) return false;
  return list.check(addr, family === 4 ? "ipv4" : "ipv6");
}

function identity(value: string | string[] | undefined): string | null {
  if (typeof value !== "string") return null;
  const v = value.trim();
  if (!v || v.length > MAX_HEADER_VALUE_LEN || CTRL.test(v)) return null;
  return v;
}

/** The tag an explicit Serve app-capability grant names; never a client claim. */
function devicePrincipal(header: string | string[] | undefined, capability: string): string | null {
  if (!capability || typeof header !== "string" || header.length > MAX_CAPABILITIES_LEN) return null;
  try {
    const doc = JSON.parse(header) as Record<string, unknown>;
    const grants = doc?.[capability];
    if (!Array.isArray(grants)) return null;
    for (const grant of grants) {
      if (grant && typeof grant === "object" && (grant as { access?: unknown }).access === true) {
        const principal = (grant as { principal?: unknown }).principal;
        if (typeof principal === "string" && /^tag:[a-z0-9][a-z0-9-]{0,62}$/.test(principal)) return principal;
      }
    }
  } catch {
    return null;
  }
  return null;
}

/** Returns the authenticated principal, or null (the caller answers 403). */
export function authenticate(
  peer: string | undefined,
  headers: Record<string, string | string[] | undefined>,
  config: AuthConfig,
): Principal | null {
  if (peerIn(config.serveProxies, peer)) {
    const login = identity(headers["tailscale-user-login"]);
    if (login && config.tailscaleLogins.has(login.toLowerCase())) return { principal: login, subject: `tailscale:${login}`, via: "tailscale" };
    const device = devicePrincipal(headers["tailscale-app-capabilities"], config.deviceCapability);
    if (device) return { principal: device, subject: `tailscale-device:${device}`, via: "tailscale-device" };
    return null;
  }
  if (peerIn(config.authProxies, peer)) {
    const uid = identity(headers["x-authentik-uid"]);
    const username = identity(headers["x-authentik-username"]);
    if (uid && username) return { principal: username, subject: uid, via: "authentik" };
    return null;
  }
  return null;
}
