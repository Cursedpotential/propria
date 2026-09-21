// Ported from modules/forks/sbv/frontend/src/components/VCardPreview.jsx
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: SBV received `vcfText` already fetched by its LazyMedia parent (a plain
// axios GET against SBV's own `/api/media`). This version fetches the vcf text
// itself from this app's post-ingest derived-media endpoint
// (getProfferPreviewMediaUrl) and falls back to a metadata card on any fetch or
// parse failure — same "no fake data" rule as attachment-preview.tsx: pre-ingest
// vCards have no derived file yet, so the fetch 404s and the fallback shows.
// Bootstrap classes and inline SVGs replaced with lucide-react icons and this
// app's own tokens; the "Download Contact" button is unchanged in spirit (a real
// client-side Blob download of the vcf text already in hand).
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { Building2, Cake, Contact2, Download, Globe, Mail, MapPin, Phone, StickyNote, TriangleAlert } from "lucide-react";
import { useEffect, useState } from "react";

import { getProfferPreviewMediaUrl } from "@/lib/api-client";
import { formatAddress, formatBirthday, parseVCard, type VCardContact } from "@/lib/sbv/vcf-parser";
import type { MatterMode, ProfferPreviewAttachment } from "@/lib/shared/types";

interface VCardPreviewProps {
  attachment: ProfferPreviewAttachment;
  previewHandle: string;
  mode: MatterMode;
}

type State =
  | { status: "loading" }
  | { status: "unavailable" }
  | { status: "ready"; contact: VCardContact; raw: string };

export function VCardPreview({ attachment, previewHandle, mode }: VCardPreviewProps) {
  const url = attachment.sha256 ? getProfferPreviewMediaUrl(previewHandle, mode, attachment.sha256) : null;
  const [state, setState] = useState<State>(() => (url ? { status: "loading" } : { status: "unavailable" }));
  // Adjusting state during render (React's own pattern for "reset state when a prop
  // changes") instead of a synchronous setState-in-effect for the no-url case; the
  // effect below only runs the actual async fetch.
  const [syncedUrl, setSyncedUrl] = useState(url);
  if (url !== syncedUrl) {
    setSyncedUrl(url);
    setState(url ? { status: "loading" } : { status: "unavailable" });
  }

  useEffect(() => {
    if (!url) return;
    let cancelled = false;
    fetch(url)
      .then((response) => {
        if (!response.ok) throw new Error(`vCard fetch failed: ${response.status}`);
        return response.text();
      })
      .then((text) => {
        if (cancelled) return;
        setState({ status: "ready", contact: parseVCard(text), raw: text });
      })
      .catch(() => {
        if (!cancelled) setState({ status: "unavailable" });
      });
    return () => {
      cancelled = true;
    };
  }, [url]);

  if (state.status === "loading") {
    return <p className="text-xs text-muted-foreground" role="status">Loading contact…</p>;
  }

  if (state.status === "unavailable") {
    return (
      <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-2 py-1.5 text-xs text-muted-foreground" data-testid="vcard-preview-unavailable">
        <TriangleAlert className="size-3.5 shrink-0" aria-hidden="true" />
        {attachment.filename ?? "Contact"} · preview unavailable (derived media not found)
      </div>
    );
  }

  const { contact, raw } = state;

  function download() {
    const blob = new Blob([raw], { type: "text/vcard" });
    const url2 = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url2;
    a.download = `${contact.name || "contact"}.vcf`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url2);
  }

  return (
    <div className="max-w-sm rounded-md border bg-card p-3 shadow-sm" data-testid="vcard-preview">
      <div className="flex items-center gap-3">
        {contact.photo ? (
          <img src={contact.photo} alt={contact.name} className="size-12 rounded-full object-cover" />
        ) : (
          <div className="flex size-12 items-center justify-center rounded-full bg-primary text-lg font-bold text-primary-foreground">
            {contact.name ? contact.name.charAt(0).toUpperCase() : <Contact2 className="size-5" />}
          </div>
        )}
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold">{contact.name || contact.formattedName || "Unknown contact"}</p>
          {contact.title && <p className="truncate text-xs text-muted-foreground">{contact.title}</p>}
          {contact.organization && <p className="truncate text-xs text-muted-foreground">{contact.organization}</p>}
        </div>
      </div>

      <div className="mt-3 space-y-2 text-xs">
        {contact.phoneNumbers.length > 0 && (
          <div>
            <p className="flex items-center gap-1 font-medium text-muted-foreground"><Phone className="size-3" /> Phone</p>
            {contact.phoneNumbers.map((phone, index) => (
              <p key={index} className="ml-4">
                <span className="text-muted-foreground">{phone.type}:</span>{" "}
                <a href={`tel:${phone.number}`} className="text-primary hover:underline">{phone.number}</a>
              </p>
            ))}
          </div>
        )}
        {contact.emails.length > 0 && (
          <div>
            <p className="flex items-center gap-1 font-medium text-muted-foreground"><Mail className="size-3" /> Email</p>
            {contact.emails.map((email, index) => (
              <p key={index} className="ml-4">
                <span className="text-muted-foreground">{email.type}:</span>{" "}
                <a href={`mailto:${email.address}`} className="text-primary hover:underline">{email.address}</a>
              </p>
            ))}
          </div>
        )}
        {contact.addresses.length > 0 && (
          <div>
            <p className="flex items-center gap-1 font-medium text-muted-foreground"><MapPin className="size-3" /> Address</p>
            {contact.addresses.map((address, index) => {
              const formatted = formatAddress(address);
              return formatted ? (
                <p key={index} className="ml-4">
                  <span className="text-muted-foreground">{address.type}:</span> {formatted}
                </p>
              ) : null;
            })}
          </div>
        )}
        {contact.birthday && (
          <div>
            <p className="flex items-center gap-1 font-medium text-muted-foreground"><Cake className="size-3" /> Birthday</p>
            <p className="ml-4">{formatBirthday(contact.birthday)}</p>
          </div>
        )}
        {contact.url && (
          <div>
            <p className="flex items-center gap-1 font-medium text-muted-foreground"><Globe className="size-3" /> Website</p>
            <p className="ml-4"><a href={contact.url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">{contact.url}</a></p>
          </div>
        )}
        {contact.note && (
          <div>
            <p className="flex items-center gap-1 font-medium text-muted-foreground"><StickyNote className="size-3" /> Note</p>
            <p className="ml-4 text-muted-foreground">{contact.note}</p>
          </div>
        )}
        {contact.organization === "" && contact.phoneNumbers.length === 0 && contact.emails.length === 0 && (
          <p className="flex items-center gap-1 text-muted-foreground"><Building2 className="size-3" /> No further contact details recorded</p>
        )}
      </div>

      <button
        type="button"
        onClick={download}
        className="mt-3 flex w-full items-center justify-center gap-1.5 rounded-md border px-2 py-1.5 text-xs font-medium hover:bg-accent"
      >
        <Download className="size-3.5" /> Download contact
      </button>
    </div>
  );
}
