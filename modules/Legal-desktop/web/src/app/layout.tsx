// Byline: Grok · grok-4.6 · 2026-08-18
// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria design contract 1.0.0 adoption)
// Self-hosted SIL OFL fonts. No Google Fonts CDN.
import "@fontsource/playfair-display/400.css";
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";
import type { ReactNode } from "react";
import { TerminalShell } from "@/components/TerminalShell";
import "../../vendor/propria-design-contract/tokens.css";
import "../../vendor/propria-design-contract/theme.css";
import "../../vendor/propria-design-contract/adapters/advocatio.css";
import "./globals.css";

export const metadata = {
  title: "Legal Workspace",
  description: "Overview of your case for the Genesee custody case.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      data-theme="dark"
      data-pr-theme="dark"
      data-pr-experience="general"
    >
      <body className="pr-app">
        <TerminalShell>{children}</TerminalShell>
      </body>
    </html>
  );
}
