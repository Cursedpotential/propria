// Byline: Claude Code · Opus 5.5 · 2026-09-28
// The toolkit web app's trusted-proxy authentication (src/web-auth.ts, built to dist/web-auth.js).
import assert from "node:assert/strict";
import { test } from "node:test";
import { authenticate, loadAuthConfig, parseCidrs } from "../dist/web-auth.js";

const SERVE = "100.91.190.107";
const TRAEFIK = "100.72.169.40";
const config = loadAuthConfig({
  TRUSTED_TAILSCALE_SERVE_PROXY_CIDRS: `${SERVE}/32`,
  TRUSTED_AUTH_PROXY_CIDRS: `${TRAEFIK}/32`,
  TRUSTED_TAILSCALE_LOGINS: "Owner@Example.com",
  TAILSCALE_DEVICE_CAPABILITY: "mitechconsult.com/cap/family-court",
});

test("tailnet login from the Serve proxy is admitted when allowlisted (case-insensitive)", () => {
  const who = authenticate(SERVE, { "tailscale-user-login": "owner@example.com" }, config);
  assert.deepEqual(who, { principal: "owner@example.com", subject: "tailscale:owner@example.com", via: "tailscale" });
  assert.equal(authenticate(`::ffff:${SERVE}`, { "tailscale-user-login": "owner@example.com" }, config)?.via, "tailscale");
});

test("tailnet login outside the allowlist is rejected", () => {
  assert.equal(authenticate(SERVE, { "tailscale-user-login": "someone@else.com" }, config), null);
});

test("an explicit Serve device grant is admitted only from the Serve proxy", () => {
  const caps = JSON.stringify({ "mitechconsult.com/cap/family-court": [{ access: true, principal: "tag:docker" }] });
  assert.equal(authenticate(SERVE, { "tailscale-app-capabilities": caps }, config)?.principal, "tag:docker");
  assert.equal(authenticate("100.100.1.1", { "tailscale-app-capabilities": caps }, config), null);
  const denied = JSON.stringify({ "mitechconsult.com/cap/family-court": [{ access: false, principal: "tag:docker" }] });
  assert.equal(authenticate(SERVE, { "tailscale-app-capabilities": denied }, config), null);
});

test("Authentik headers are admitted only from the Traefik peer", () => {
  const headers = { "x-authentik-uid": "abc123", "x-authentik-username": "owner" };
  assert.deepEqual(authenticate(TRAEFIK, headers, config), { principal: "owner", subject: "abc123", via: "authentik" });
  assert.equal(authenticate(SERVE, headers, config), null, "Serve peer must not accept Authentik headers");
  assert.equal(authenticate(TRAEFIK, { "x-authentik-username": "owner" }, config), null, "uid is required");
});

test("spoofed identity headers from any other peer are rejected", () => {
  const spoof = { "tailscale-user-login": "owner@example.com", "x-authentik-uid": "abc123", "x-authentik-username": "owner", "x-forwarded-for": SERVE, "x-real-ip": SERVE };
  for (const peer of ["100.100.1.1", "10.0.1.5", "127.0.0.1", undefined]) assert.equal(authenticate(peer, spoof, config), null, String(peer));
  assert.equal(authenticate(TRAEFIK, { "tailscale-user-login": "owner@example.com" }, config), null, "Traefik peer must not accept Tailscale headers");
});

test("no identity is rejected; control characters and oversized values are rejected", () => {
  assert.equal(authenticate(SERVE, {}, config), null);
  assert.equal(authenticate(TRAEFIK, {}, config), null);
  assert.equal(authenticate(TRAEFIK, { "x-authentik-uid": "a\nb", "x-authentik-username": "owner" }, config), null);
  assert.equal(authenticate(SERVE, { "tailscale-user-login": "x".repeat(300) }, config), null);
});

test("empty or malformed configuration fails closed", () => {
  const empty = loadAuthConfig({});
  assert.equal(authenticate(SERVE, { "tailscale-user-login": "owner@example.com" }, empty), null);
  assert.equal(authenticate(TRAEFIK, { "x-authentik-uid": "a", "x-authentik-username": "b" }, empty), null);
  assert.equal(parseCidrs("100.64.0.0/10", true), null, "Serve proxies must be single hosts");
  assert.equal(parseCidrs("not-an-ip/32", false), null);
  assert.ok(parseCidrs("100.72.169.40/32,10.0.0.0/8", false));
});
