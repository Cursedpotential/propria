// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — reflow-safe Carbon-Linen-Seal header.
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Moon, Search, Sun } from "lucide-react";
import type * as React from "react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { authStatusQuery } from "@/lib/queries";
import { useTheme } from "./theme-provider";

export function Header() {
  const [q, setQ] = useState("");
  const navigate = useNavigate();
  const { data: auth } = useQuery(authStatusQuery());
  const { theme, toggle } = useTheme();

  function onSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!q.trim()) return;
    void navigate({ to: "/evidence", search: { q } as never });
  }

  return (
    <header className="flex min-h-12 shrink-0 flex-wrap items-center gap-3 border-b border-border bg-surface px-3 py-2">
      <form onSubmit={onSearch} className="flex min-w-56 max-w-md flex-1 items-center gap-2 max-[520px]:min-w-full">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-text-tertiary" aria-hidden />
          <Input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search events, messages, notes, exhibits…"
            className="pl-7"
            aria-label="Case search"
          />
        </div>
      </form>

      <div className="ml-auto flex items-center gap-2">
        <AuthChip
          configured={auth?.chat.available ?? false}
          source={auth?.chat.source}
          fix={"fix" in (auth?.chat ?? {}) ? (auth?.chat as { fix?: string }).fix : undefined}
        />
        <Button variant="ghost" size="icon" onClick={toggle} aria-label="Toggle theme">
          {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
        </Button>
      </div>
    </header>
  );
}

function AuthChip({ configured, source, fix }: { configured: boolean; source?: string; fix?: string }) {
  if (configured) {
    return (
      <Badge tone="good" title={`Auth source: ${source ?? "unknown"}`}>
        Claude connected
      </Badge>
    );
  }
  return (
    <Badge tone="warn" title={fix ?? "Not authenticated"}>
      Claude not connected
    </Badge>
  );
}
