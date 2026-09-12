// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Resolves the loopback base URL of the Node sidecar (see ../../sidecar).
// - Under Tauri: asks the Rust shell for the ephemeral port it captured
//   from the sidecar's stdout (see src-tauri/src/main.rs, `sidecar_port`).
// - Under plain browser dev (`npm run dev`, no Tauri shell): falls back to
//   VITE_SIDECAR_PORT / the sidecar's fixed dev default (4177).
//
// The auth token itself NEVER reaches this module or anything downstream of
// it — only the sidecar process holds it (see docs/ARCHITECTURE.md).

let cachedBaseUrl: string | null = null;

async function isTauriRuntime(): Promise<boolean> {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export async function resolveSidecarBaseUrl(): Promise<string> {
  if (cachedBaseUrl) return cachedBaseUrl;

  if (await isTauriRuntime()) {
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      const port = await invoke<number>("sidecar_port");
      cachedBaseUrl = `http://127.0.0.1:${port}`;
      return cachedBaseUrl;
    } catch (err) {
      console.error("Failed to resolve sidecar port via Tauri; falling back to dev default.", err);
    }
  }

  const devPort = import.meta.env.VITE_SIDECAR_PORT ?? "4177";
  cachedBaseUrl = `http://127.0.0.1:${devPort}`;
  return cachedBaseUrl;
}

/** Test-only escape hatch. */
export function __resetSidecarBaseUrlCacheForTests() {
  cachedBaseUrl = null;
}
