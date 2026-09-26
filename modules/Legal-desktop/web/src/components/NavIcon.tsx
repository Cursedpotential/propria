// Byline: Grok · grok-4.6 · 2026-08-18
// lucide-react (ISC) — same icon set legal-terminal-master uses.
import type { LucideIcon } from "lucide-react";
import {
  Activity,
  BookOpen,
  Calendar,
  CheckSquare,
  ClipboardCheck,
  Clock,
  Eye,
  FileText,
  FolderOpen,
  HelpCircle,
  Home,
  ListChecks,
  Lock,
  MessageSquare,
  Package,
  Paperclip,
  PenTool,
  Quote,
  Scale,
  Search,
  ShieldAlert,
  Swords,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  home: Home,
  message: MessageSquare,
  scale: Scale,
  list: ListChecks,
  book: BookOpen,
  search: Search,
  quote: Quote,
  file: FileText,
  pen: PenTool,
  eye: Eye,
  package: Package,
  clipboard: ClipboardCheck,
  folder: FolderOpen,
  paperclip: Paperclip,
  help: HelpCircle,
  check: CheckSquare,
  calendar: Calendar,
  clock: Clock,
  lock: Lock,
  swords: Swords,
  activity: Activity,
  shield: ShieldAlert,
};

export function NavIcon({ name, size = 14 }: { name?: string; size?: number }) {
  const Icon = ICONS[name ?? ""] ?? FileText;
  return <Icon size={size} aria-hidden />;
}
