import { Pool } from 'pg';

// Server-side only. One pooled connection to the live traceiq DB (tailnet).
// Never import this from a client component.
declare global {
  // eslint-disable-next-line no-var
  var __traceiqPool: Pool | undefined;
}

export const pool =
  global.__traceiqPool ??
  new Pool({
    connectionString: process.env.TRACEIQ_DSN,
    max: 5,
    idleTimeoutMillis: 30_000,
  });

if (process.env.NODE_ENV !== 'production') global.__traceiqPool = pool;
