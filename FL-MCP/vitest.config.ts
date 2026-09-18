// Byline: Claude Code · Sonnet 5 · 2026-09-07
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// Deliberately separate from vite.config.ts: the app build config carries the
// TanStack Router codegen plugin and Tauri-specific build target logic that
// vitest does not need and that slows every test run down for no benefit.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": new URL("./src", import.meta.url).pathname,
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    css: false,
    // Component tests only — sidecar/**/*.test.mjs are node:test files run
    // separately via `npm run test:sidecar` (they use node's own `test`
    // API, not Vitest's, and need Node's real fs/net, not jsdom).
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
