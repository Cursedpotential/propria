import type { NextConfig } from "next";

// Server app (NOT static export) — required for the /api/* route handlers that query
// the live traceiq DB. pg is externalized so Turbopack doesn't try to bundle its
// native/dynamic requires.
const nextConfig: NextConfig = {
  serverExternalPackages: ['pg'],
  turbopack: { root: process.cwd() },
};

export default nextConfig;
