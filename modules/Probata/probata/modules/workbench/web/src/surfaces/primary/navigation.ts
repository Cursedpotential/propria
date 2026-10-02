// Byline: Codex · GPT-5.6-Sol · 2026-08-30
// Byline: Claude Code · Opus 5 · 2026-09-22 (Sources replaces Intake in the nav;
// /intake stays a reachable route, off the navigation.)
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (Case: people and identifiers over registry, step 6)
import { FileSearch, FolderTree, LayoutDashboard, Users } from "lucide-react";
import type { WorkbenchNavigationItem } from "@/platform-ui/navigation";

export const primaryNavigationItems = [
  {
    title: "Desk",
    pageTitle: "Context Intake Desk",
    href: "/",
    icon: LayoutDashboard,
    surface: "primary",
  },
  {
    title: "Sources",
    pageTitle: "Sources",
    href: "/sources",
    icon: FolderTree,
    surface: "primary",
  },
  {
    title: "Review",
    pageTitle: "Review extracted context",
    href: "/review",
    icon: FileSearch,
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
