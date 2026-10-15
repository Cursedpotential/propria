import type { NextConfig } from "next";
import path from "node:path";

const nextConfig: NextConfig = {
  output: "standalone",
  experimental: {
    // Existing provider requests can exceed Next's default30s proxy timeout.
    proxyTimeout: 600_000,
  },
  /** Proxy API calls without buffering or changing the existing text stream.
   * Inputs: same-origin API path and body. Outputs: unchanged FastAPI response.
   * Effects: loopback HTTP only. Pick for the bundled deployment; explicit
   * legacy public API URLs remain supported during cutover.
   */
  async rewrites() {
    return [{ source: "/api/probe/:path*", destination: "http://127.0.0.1:8000/:path*" }];
  },
  turbopack: {
    root: path.join(__dirname),
  },
};

export default nextConfig;
