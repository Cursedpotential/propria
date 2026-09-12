// Byline: Claude Code · Sonnet 5 · 2026-09-07
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
      document.documentElement.setAttribute("data-theme", context.globals.theme ?? "dark");
      return Story();
    },
  ],
};

export default preview;
