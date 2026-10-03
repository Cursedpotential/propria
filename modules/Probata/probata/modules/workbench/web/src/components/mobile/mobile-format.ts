// Byline: Claude Code · Sonnet · 2026-10-02
// Pure formatters for the mobile shell.
import { ApiError } from "@/lib/api-client";

const numberFormat = new Intl.NumberFormat("en-US");
export const formatCount = (value: number | null | undefined) => numberFormat.format(value ?? 0);

export function formatDate(value: string | null | undefined) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

export function formatTime(value: string | null | undefined) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
}

export function formatDateTime(value: string | null | undefined) {
  if (!value) return "";
  return `${formatDate(value)}, ${formatTime(value)}`;
}

export function formatRange(first: string | null | undefined, last: string | null | undefined) {
  if (!first && !last) return "no dates";
  const a = formatDate(first);
  const b = formatDate(last);
  return a === b || !b ? a : `${a} to ${b}`;
}

export function formatDuration(seconds: number | null | undefined) {
  if (seconds === null || seconds === undefined) return "";
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return minutes >= 60 ? `${Math.floor(minutes / 60)}h ${minutes % 60}m` : `${minutes}m ${rest}s`;
}

export function errorText(error: unknown) {
  if (error instanceof ApiError) return error.message;
  return error instanceof Error ? error.message : "Something went wrong";
}

const STATUS_LABEL: Record<string, string> = {
  committed: "Done",
  awaiting_review: "Awaiting review",
  parked: "Parked",
  failed: "Failed",
  running: "Running",
  not_finished: "Not finished",
  skipped: "Skipped",
};

export const statusLabel = (status: string) => STATUS_LABEL[status] ?? status;

export function prettyNumber(number: string) {
  return number.length === 10 ? `(${number.slice(0, 3)}) ${number.slice(3, 6)}-${number.slice(6)}` : number;
}
