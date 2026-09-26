// Byline: Claude Code · Opus 5.5 · 2026-09-24
// n8n external hook: sign the owner in from a trusted proxy identity, so n8n never shows its own login screen.
// Owner 2026-09-24 05:10: on the tailnet, Tailscale is the only barrier; keep the app protected through the proxy.
// Owner 2026-09-24 05:27: off the tailnet, the Authentik login is the only login.
// n8n accounts are unchanged; a request with no trusted identity gets n8n's normal login.
//
// How: two proxies vouch for the caller.
//   - Tailscale Serve stamps tailnet requests with `Tailscale-User-Login` (allowlist N8N_TRUSTED_HEADER_LOGINS).
//   - Traefik's Authentik forward-auth on n8n.int.mitechconsult.com sets `X-authentik-username` from Authentik's
//     answer, replacing any value the client sent (allowlist N8N_TRUSTED_AUTHENTIK_USERS).
// When an allowlisted identity arrives without an n8n session, this issues the normal n8n session cookie for
// the instance owner (role global:owner) and lets the request continue.
// Known gap, deferred by the owner as a later hardening step: n8n also listens on the host's tailnet address,
// so a tailnet peer that reaches the port directly could set either header itself.
//
// Wiring (Coolify service casebible-n8n): EXTERNAL_HOOK_FILES=/hooks/tailnet-signin.js,
// N8N_TRUSTED_HEADER_LOGINS / N8N_TRUSTED_AUTHENTIK_USERS=<comma lists>, and the host file /data/probata/config/n8n/tailnet-signin.js mounted read-only.
// Written against n8n 2.36.6 (Express 5); it uses n8n internals, so re-check it after an n8n upgrade.

const N8N = '/usr/local/lib/node_modules/n8n';
const SOURCES = [
  { name: 'Tailscale', header: 'tailscale-user-login', env: 'N8N_TRUSTED_HEADER_LOGINS' },
  { name: 'Authentik', header: 'x-authentik-username', env: 'N8N_TRUSTED_AUTHENTIK_USERS' },
];
const COOKIE = 'n8n-auth';

function hasSession(req) {
  return (req.headers.cookie || '').split(';').some((c) => c.trim().startsWith(COOKIE + '='));
}

module.exports = {
  n8n: {
    ready: [
      async function (server) {
        // n8n treats a throwing hook as fatal; setup failure must leave n8n running with its normal login.
        try {
          await setup.call(this, server);
        } catch (e) {
          console.log(`[tailnet-signin] setup failed, normal login unchanged: ${e && e.message}`);
        }
      },
    ],
  },
};

async function setup(server) {
  const sources = SOURCES.map((src) => ({
    ...src,
    allowed: new Set((process.env[src.env] || '').split(',').map((s) => s.trim().toLowerCase()).filter(Boolean)),
  })).filter((src) => src.allowed.size);
  if (!sources.length) {
    console.log('[tailnet-signin] no trusted identities configured; hook inactive');
    return;
  }
  const { Container } = require(`${N8N}/node_modules/@n8n/di`);
  const { AuthService } = require(`${N8N}/dist/auth/auth.service`);
  const authService = Container.get(AuthService);
  const users = this.dbCollections.User;
  const warned = new Set();
  const seen = new Set();

  async function owner() {
    const all = await users.find({ relations: ['role'] });
    return all.find((u) => u.role && u.role.slug === 'global:owner' && !u.disabled);
  }

  async function signIn(req, res, next) {
    try {
      if (hasSession(req)) return next();
      const src = sources.find((x) => req.headers[x.header]);
      if (!src) return next();
      const login = String(req.headers[src.header]).trim().toLowerCase();
      if (!seen.has(src.name)) {
        seen.add(src.name);
        console.log(`[tailnet-signin] ${src.name} identity header received`);
      }
      if (!src.allowed.has(login)) {
        if (!warned.has(login)) console.log(`[tailnet-signin] ${src.name} login not in the allowlist: ${login}`);
        warned.add(login);
        return next();
      }
      const user = await owner();
      if (!user) return next();
      authService.issueCookie(res, user, false);
      // Let this same request through as signed in: copy the new session cookie onto the request.
      const set = [].concat(res.getHeader('set-cookie') || []).find((c) => String(c).startsWith(COOKIE + '='));
      if (set) {
        const pair = String(set).split(';')[0];
        req.headers.cookie = req.headers.cookie ? `${req.headers.cookie}; ${pair}` : pair;
      }
    } catch (e) {
      console.log(`[tailnet-signin] sign-in skipped: ${e && e.message}`);
    }
    return next();
  }

  const app = server.app;
  app.use(signIn);
  const stack = (app.router || app._router).stack;
  stack.unshift(stack.pop()); // run before n8n's own routes and auth
  console.log(`[tailnet-signin] active for ${sources.map((x) => `${x.name} (${x.allowed.size})`).join(', ')}`);
}
