// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import type { StorybookConfig } from "@storybook/react-vite";
import { mergeConfig } from "vite";

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(ts|tsx)"],
  addons: ["@storybook/addon-a11y", "@storybook/addon-docs"],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  // Storybook builds its OWN Vite config (it does not reuse ../vite.config.ts,
  // which also carries the TanStack Router codegen plugin Storybook doesn't
  // need) — Tailwind v4's plugin and the "@/*" path alias have to be added
  // here explicitly so component stories resolve the same imports the app does.
  viteFinal: async (viteConfig) =>
    mergeConfig(viteConfig, {
      plugins: [tailwindcss()],
      resolve: {
        alias: {
          "@": fileURLToPath(new URL("../src", import.meta.url)),
        },
      },
    }),
};

export default config;
