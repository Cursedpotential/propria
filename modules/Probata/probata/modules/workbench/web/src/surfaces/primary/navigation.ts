// Byline: Codex · GPT-5.6-Sol · 2026-08-30
// Byline: Claude Code · Opus 5 · 2026-09-22 (Sources replaces Intake in the nav;
// /intake stays a reachable route, off the navigation.)
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (Case: people and identifiers over registry, step 6)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02 (Conversations: Extract and Send to Surreal on the desktop)
// Byline: Codex · GPT-6 · 2026-10-06 — the ratified Sources → Activity → Read workflow.
import { Activity, BookOpen, FolderTree, Users } from "lucide-react";
import type { WorkbenchNavigationItem } from "@/platform-ui/navigation";

export const primaryNavigationItems = [
  {
    title: "Sources",
    pageTitle: "Sources",
    href: "/sources",
    icon: FolderTree,
    surface: "primary",
  },
  {
    title: "Activity",
    pageTitle: "Activity",
    href: "/activity",
    icon: Activity,
    surface: "primary",
  },
  {
    title: "Read",
    pageTitle: "Read",
    href: "/read",
    icon: BookOpen,
    surface: "primary",
  },
  {
    title: "Case",
    pageTitle: "Case — people and identifiers",
    href: "/case",
    icon: Users,
    surface: "primary",
  },
] as const satisfies readonly WorkbenchNavigationItem[];
