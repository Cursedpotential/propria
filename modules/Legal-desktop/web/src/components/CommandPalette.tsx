"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import {
  type Surface,
  SURFACES,
  navHelp,
  navLabel,
} from "@/lib/surfaces";

// Byline: Grok · grok-4.6 · 2026-08-18

const RECENT_KEY = "lw-recent-pages";

export function CommandPalette({
  open,
  onClose,
  surfaces = SURFACES,
}: {
  open: boolean;
  onClose: () => void;
  surfaces?: Surface[];
}) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [recent, setRecent] = useState<string[]>([]);
  const modules = surfaces;

  useEffect(() => {
    if (!open) {
      setQuery("");
      return;
    }
    try {
      const raw = window.localStorage.getItem(RECENT_KEY);
      if (raw) setRecent(JSON.parse(raw) as string[]);
    } catch {
      setRecent([]);
    }
  }, [open]);

  function go(item: Surface) {
    const next = [item.path, ...recent.filter((path) => path !== item.path)].slice(0, 5);
    try {
      window.localStorage.setItem(RECENT_KEY, JSON.stringify(next));
    } catch {
      /* private mode */
    }
    router.push(item.path);
    onClose();
    setQuery("");
  }

  return (
    <Command.Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      label="Go to a page"
      overlayClassName="command-palette-overlay"
      contentClassName="command-palette-dialog"
    >
      <Command.Input
        value={query}
        onValueChange={setQuery}
        placeholder="Type a page name"
      />
      <Command.List>
        <Command.Empty>No pages found.</Command.Empty>
        {recent.length > 0 ? (
          <Command.Group heading="Recent">
            {recent.map((path) => {
              const item = surfaces.find((row) => row.path === path);
              if (!item) return null;
              return (
                <Command.Item
                  key={`recent-${path}`}
                  value={`recent ${navLabel(item)} ${navHelp(item)}`}
                  keywords={[navLabel(item), navHelp(item), item.group]}
                  onSelect={() => go(item)}
                >
                  {navLabel(item)}
                </Command.Item>
              );
            })}
          </Command.Group>
        ) : null}
        <Command.Group heading="Pages">
          {modules.map((item) => (
            <Command.Item
              key={item.path}
              value={`${navLabel(item)} ${navHelp(item)}`}
              keywords={[navLabel(item), navHelp(item), item.group]}
              onSelect={() => go(item)}
            >
              {navLabel(item)}
            </Command.Item>
          ))}
        </Command.Group>
      </Command.List>
    </Command.Dialog>
  );
}
