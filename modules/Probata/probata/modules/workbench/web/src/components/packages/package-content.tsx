// Byline: Codex · GPT-6 · 2026-10-12.
import { formatValue } from "@/components/metadata/metadata-format";
import type { ContextPackageItem, PackageSourceRef } from "@/lib/context-package-client";
import { AppLink } from "@/lib/router-compat";

/** Allow only existing app/API links; inert source text never becomes an executable URL.
 * Input: recorded URL. Output: local source link or null. Effects: none.
 * Pick before rendering source links or embedded media from extracted fields.
 */
export function packageSourceHref(value: unknown): string | null {
  if (typeof value !== "string" || !/^\/(?:read(?:\?|$)|sources(?:\?|$)|api\/(?:context|proffer|imported)\/)/.test(value)) return null;
  const url = new URL(value, "https://context.invalid");
  return url.origin === "https://context.invalid" ? `${url.pathname}${url.search}${url.hash}` : null;
}

/** Show every supplied source coordinate without dropping native locator fields.
 * Inputs: native source refs. Output: exact version/locator JSON and supported app links.
 * Effects: clicked navigation only. Pick beside every item, event or edge.
 */
export function PackageSources({ sources }: { sources: PackageSourceRef[] }) {
  return <section aria-label="Source locators" className="space-y-2">
    <h3 className="platform-kicker">Source locators · {sources.length}</h3>
    {!sources.length && <p className="text-xs text-muted-foreground">No source locators returned for this item.</p>}
    <ol className="space-y-2">{sources.map((source, index) => {
      const href = packageSourceHref(typeof source === "string" ? source : source.href ?? source.url);
      return <li key={index} className="rounded border p-2 text-xs">
        {href && <AppLink className="mb-1 inline-block underline" href={href}>Open exact source</AppLink>}
        <pre className="whitespace-pre-wrap break-all font-mono text-[11px]">{typeof source === "string" ? source : JSON.stringify(source, null, 2)}</pre>
      </li>;
    })}</ol>
  </section>;
}

/** Render full recorded content and identity cues using the selected package item's fields.
 * Input: exact item. Output: complete source text/media and source-linked entity details.
 * Effects: supported same-origin media reads only. Pick for documents, chunks, messages,
 * screenshots and entity records; source account/device remain separately labeled fields.
 */
export function PackageItemContent({ item }: { item: ContextPackageItem }) {
  const textFields = Object.entries(item.fields).filter(([key, value]) => typeof value === "string" && /^(text|body|content|markdown|document_text|snippet|description|statement|summary|extracted_text|ocr_text)$/i.test(key));
  const media = packageSourceHref(item.fields.preview_url ?? item.fields.content_url);
  const mediaType = String(item.fields.media_type ?? item.fields.mime_type ?? "");
  const identityKeys = ["person_ref", "source_account", "account_ref", "source_device", "device_ref", "provenance_party", "party", "basis", "classification_basis"];
  return <article className="space-y-4" aria-label="Selected item content">
    <header><p className="platform-kicker">{item.kind} · {item.review_state}</p><h2 className="mt-1 break-words text-lg font-semibold">{item.label}</h2>
      <p className="mt-1 break-all font-mono text-[10px] text-muted-foreground">{item.id} · revision {item.revision}</p></header>
    {identityKeys.some((key) => key in item.fields) && <dl className="grid gap-2 rounded border p-3 text-xs">{identityKeys.filter((key) => key in item.fields).map((key) => <div key={key}><dt className="font-medium">{key.replaceAll("_", " ")}</dt><dd className="whitespace-pre-wrap break-all text-muted-foreground">{formatValue(item.fields[key])}</dd></div>)}</dl>}
    {media && mediaType.startsWith("image/") && <img className="max-h-[34rem] max-w-full rounded border object-contain" src={media} alt={item.label} />}
    {media && mediaType === "application/pdf" && <iframe src={media} className="h-[34rem] w-full border" title={item.label} sandbox="" />}
    {textFields.map(([key, text]) => <section key={key}><h3 className="platform-kicker mb-2">{key.replaceAll("_", " ")}</h3><p className="whitespace-pre-wrap break-words text-sm leading-6">{text as string}</p></section>)}
    {Array.isArray(item.fields.aliases) && <section><h3 className="platform-kicker mb-2">Aliases</h3><ul className="flex flex-wrap gap-1">{item.fields.aliases.map((alias, index) => <li key={index} className="rounded border px-2 py-1 text-xs">{typeof alias === "string" ? alias : JSON.stringify(alias)}</li>)}</ul></section>}
    <PackageSources sources={item.source_refs} />
  </article>;
}
