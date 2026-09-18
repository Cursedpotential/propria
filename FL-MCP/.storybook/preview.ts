// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — shared theme parity.
import type { Preview } from "@storybook/react-vite";
import "../src/styles/globals.css";

const preview: Preview = {
  parameters: {
    controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
    a11y: { test: "todo" },
  },
  globalTypes: {
    theme: {
      description: "Global theme for components",
      defaultValue: "dark",
      toolbar: {
        title: "Theme",
        icon: "circlehollow",
        items: [
          { value: "dark", icon: "circle", title: "Dark" },
          { value: "light", icon: "circlehollow", title: "Light" },
        ],
        dynamicTitle: true,
      },
    },
  },
  decorators: [
    (Story, context) => {
      const theme = context.globals.theme ?? "dark";
      document.documentElement.setAttribute("data-theme", theme);
      document.documentElement.setAttribute("data-pr-theme", theme);
      document.documentElement.setAttribute("data-pr-experience", "general");
      document.documentElement.classList.toggle("theme-dark", theme === "dark");
      document.documentElement.classList.toggle("theme-light", theme === "light");
      return Story();
    },
  ],
};

export default preview;
