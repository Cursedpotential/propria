/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_WORKBENCH_API_URL?: string;
  /** Exact http(s) origin of the embedded Xplorer shell; absent means disabled. */
  readonly VITE_INTAKE_SELECTION_BRIDGE_ORIGIN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module "*.css";
