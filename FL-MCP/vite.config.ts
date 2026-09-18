// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// Tauri expects a fixed dev-server port and will fail if it is already in
// use — see src-tauri/tauri.conf.json's `build.devUrl`. Keep these in sync.
const DEV_PORT = 5183;

export default defineConfig(() => ({
  plugins: [
    tanstackRouter({ target: "react", autoCodeSplitting: true, routesDirectory: "./src/routes", generatedRouteTree: "./src/routeTree.gen.ts" }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": new URL("./src", import.meta.url).pathname,
    },
  },
  // Tauri-specific build tuning (see https://v2.tauri.app/start/frontend/#vite).
  clearScreen: false,
  server: {
    port: DEV_PORT,
    strictPort: true,
    host: "127.0.0.1",
    watch: {
      ignored: ["**/src-tauri/**"],
    },
  },
  envPrefix: ["VITE_", "TAURI_"],
  build: {
    // A single modern target (Tauri's WebView2 on Windows / WebKit on macOS
    // are both evergreen) rather than branching on TAURI_ENV_PLATFORM: the
    // old "safari13" fallback used by some Tauri starter templates hits an
    // esbuild limitation transforming certain destructuring patterns down
    // that far (verified against this codebase 2026-09-07 — build failed on
    // routes/memos.tsx's `const { data } = ...`). es2022 covers both
    // webview engines Tauri actually ships with no such issue.
    target: "es2022",
    minify: (process.env.TAURI_ENV_DEBUG ? false : "esbuild") as "esbuild" | false,
    sourcemap: !!process.env.TAURI_ENV_DEBUG,
    outDir: "dist",
  },
}));
