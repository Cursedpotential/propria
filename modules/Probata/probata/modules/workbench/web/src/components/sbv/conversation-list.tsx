// Ported from modules/forks/sbv/frontend/src/components/ConversationList.jsx
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: Bootstrap classes replaced with this app's Tailwind + shadcn tokens; `date-fns`'
// formatDistanceToNow replaced by Intl.RelativeTimeFormat (no new dependency); the phone-number
// formatting and the display-name rules (a conversation titled by its contact name, else its subject,
// else its formatted number) are carried over unchanged. Rows are links, not click handlers, so a
// conversation opens from a tap, a keyboard or a shared URL alike.
// Byline: Claude Code · Sonnet · 2026-10-02
"use client";

import { MessageSquareText, Phone } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { AppLink } from "@/lib/router-compat";
import { cn } from "@/lib/utils";

export interface ConversationListItem {
  /** Where the row opens. */
  href: string;
  /** The contact's name, when one is known. */
  contactName?: string | null;
  /** A phone number or comma-separated group of numbers. */
  address?: string | null;
  subject?: string | null;
  type?: "message" | "call";
  lastMessage?: string | null;
  lastDate?: string | null;
  count: number;
  /** An extra mark on the row (for example first-party / third-party). */
  tag?: string | null;
}

const relative = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

export function relativeDate(value: string | null | undefined) {
  if (!value) return "";
  const then = new Date(value).getTime();
  if (Number.isNaN(then)) return "";
  const seconds = Math.round((then - Date.now()) / 1000);
  const steps: [Intl.RelativeTimeFormatUnit, number][] = [["year", 31536000], ["month", 2592000], ["day", 86400], ["hour", 3600], ["minute", 60]];
  for (const [unit, size] of steps) {
    if (Math.abs(seconds) >= size) return relative.format(Math.round(seconds / size), unit);
  }
  return relative.format(seconds, "second");
}

function formatSingleNumber(number: string) {
  const cleaned = number.replace(/\D/g, "");
  if (cleaned.length === 11 && cleaned.startsWith("1")) return `+1 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7)}`;
  if (cleaned.length === 10) return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`;
  return number;
}

function formatPhoneNumber(number: string | null | undefined) {
  if (!number) return "Unknown";
  return number.includes(",") ? number.split(",").map((part) => formatSingleNumber(part.trim())).join(", ") : formatSingleNumber(number);
}

export function displayName(item: ConversationListItem) {
  const subject = item.subject && !item.subject.startsWith("proto:") ? item.subject : null;
  if (subject && (!item.contactName || item.contactName === "(Unknown)" || /^\d{8}$/.test(item.contactName))) return subject;
  if (!item.contactName || item.contactName === "(Unknown)") return formatPhoneNumber(item.address);
  return item.contactName;
}

function truncate(message: string | null | undefined, maxLength = 80) {
  if (!message) return "";
  return message.length <= maxLength ? message : `${message.slice(0, maxLength).trim()}...`;
}

export function ConversationList({ items, className }: { items: ConversationListItem[]; className?: string }) {
  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 px-6 py-12 text-center text-muted-foreground">
        <MessageSquareText className="size-12 opacity-50" aria-hidden="true" />
        <p className="font-medium text-foreground">No conversations found</p>
      </div>
    );
  }
  return (
    <ul className={cn("divide-y divide-border", className)} data-testid="conversation-list">
      {items.map((item) => (
        <li key={item.href}>
          <AppLink href={item.href} className="flex min-h-16 items-start gap-3 px-4 py-3 hover:bg-muted/60 active:bg-muted">
            <span className="mt-1 shrink-0 rounded-full bg-muted p-2 shadow-sm">
              {item.type === "call" ? <Phone className="size-5 text-emerald-600" aria-hidden="true" /> : <MessageSquareText className="size-5 text-primary" aria-hidden="true" />}
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex items-baseline justify-between gap-2">
                <span className="truncate text-sm font-semibold">{displayName(item)}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{relativeDate(item.lastDate)}</span>
              </span>
              <span className="block truncate text-[13px] text-muted-foreground">{truncate(item.lastMessage)}</span>
              <span className="mt-1 flex items-center gap-1">
                <Badge variant="secondary" className="text-[11px]">
                  {item.count.toLocaleString("en-US")} {item.type === "call" ? "call" : "message"}{item.count === 1 ? "" : "s"}
                </Badge>
                {item.tag ? <Badge variant="outline" className="text-[11px]">{item.tag}</Badge> : null}
              </span>
            </span>
          </AppLink>
        </li>
      ))}
    </ul>
  );
}
