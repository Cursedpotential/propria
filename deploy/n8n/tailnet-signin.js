// Byline: Claude Code · Opus 5.5 · 2026-09-24
// n8n external hook: sign the owner in from Tailscale's identity header, so n8n shows no login screen on the tailnet.
// Owner 2026-09-24 05:10: on the tailnet, Tailscale is the only barrier; keep the app protected through the proxy.
// The public route stays behind Authentik forward-auth at Traefik, and n8n accounts are unchanged.
//
// How: Tailscale Serve stamps tailnet requests with `Tailscale-User-Login`. When that login is listed in
// N8N_TRUSTED_HEADER_LOGINS and the request carries no n8n session, this issues the normal n8n session cookie
// for the instance owner (role global:owner) and lets the request continue.
// Known gap, deferred by the owner as a later hardening step: n8n also listens on the host's tailnet address,
// so a tailnet peer that reaches the port directly could set the header itself.
//
// Wiring (Coolify service casebible-n8n): EXTERNAL_HOOK_FILES=/hooks/tailnet-signin.js,
// N8N_TRUSTED_HEADER_LOGINS=<comma list>, and the host file /data/probata/config/n8n/tailnet-signin.js mounted read-only.
// Written against n8n 2.36.6 (Express 5); it uses n8n internals, so re-check it after an n8n upgrade.

const N8N = '/usr/local/lib/node_modules/n8n';
const HEADER = 'tailscale-user-login';
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
  const allowed = new Set(
    (process.env.N8N_TRUSTED_HEADER_LOGINS || '')
      .split(',')
      .map((s) => s.trim().toLowerCase())
      .filter(Boolean),
  );
  if (!allowed.size) {
    console.log('[tailnet-signin] N8N_TRUSTED_HEADER_LOGINS is empty; hook inactive');
    return;
  }
  const { Container } = require(`${N8N}/node_modules/@n8n/di`);
  const { AuthService } = require(`${N8N}/dist/auth/auth.service`);
  const authService = Container.get(AuthService);
  const users = this.dbCollections.User;
  const warned = new Set();
  let seen = false;

  async function owner() {
    const all = await users.find({ relations: ['role'] });
    return all.find((u) => u.role && u.role.slug === 'global:owner' && !u.disabled);
  }

  async function signIn(req, res, next) {
    try {
      const login = String(req.headers[HEADER] || '').trim().toLowerCase();
      if (login && !seen) {
        seen = true;
        console.log('[tailnet-signin] Tailscale identity header received');
      }
      if (!login || hasSession(req)) return next();
      if (!allowed.has(login)) {
        if (!warned.has(login)) console.log(`[tailnet-signin] tailnet login not in the allowlist: ${login}`);
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
  console.log(`[tailnet-signin] active for ${allowed.size} tailnet login(s)`);
}
