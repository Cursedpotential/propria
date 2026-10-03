// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Content-signature format detection for the first bytes of an object. Pure functions, no I/O, so they run in a
// Worker and in a plain `node --test`.
//
// Nothing here is a new rule. It is a port of the three signature sources the owner named:
//   * Proffer's handler detection        modules/Probata/probata/modules/engine/postgres/handler_selection_store.go
//                                        (detectHandlerContent) and html_signature.go (detectHTMLContent)
//   * the AI-chat signature port         .../engine/postgres/handler_ai_chat_signature.go (detectAIChatContent)
//   * the original Case Bible probe      modules/Consignatio/casebible/tools/comm_timeline_mvp/elt/ai_chat_signature_probe.py
// The format ids are the Proffer registry's, so a result can be compared with handler_detected_format. The file name and
// path are never inputs: the bytes decide.
//
// Not ported (the Go sources define them elsewhere or need a full parse, so the Worker reports the generic id instead):
// facebook_messenger_json, messages_transcript (SMS/iMessage lines), csv.

export const RULESETS = ["proffer-v1", "casebible-probe-v1"];

const JSON_PROBE_BYTES = 64 * 1024; // a JSON export is recognised by its first object's keys
const TEXT_PROBE_BYTES = 16 * 1024; // the Case Bible probe's HEAD_BYTES

// --- speaker markers --------------------------------------------------------------------------------------------------
const PRE = String.raw`^[ \t]{0,3}(?:#{1,6}[ \t]*)?(?:>[ \t]*)?(?:[-*][ \t]*)?(?:\*\*|__)?[ \t]*(?:[^\x00-\x7F][ \t]*)*`;
const TS = String.raw`(?:[ \t]*\([^)\n]{0,48}\))?(?:[ \t]*\[[^\]\n]{0,48}\])?`;
const POST_INLINE = String.raw`[ \t]*(?:\*\*|__)?[ \t]*[:：]`;
const POST_HEAD = String.raw`[ \t]*(?:\*\*|__)?[ \t]*[:：]?[ \t]*$`;

const WORDS = {
  // Go port: ChatGPT, Claude, Gemini, Perplexity and Chat Memo; inline markers only.
  "proffer-v1": {
    owner: "(?:you said|user|human|prompt|question|you)",
    assistant: "(?:chatgpt said|gemini said|claude said|assistant|chatgpt|claude|gemini|perplexity|model|answer|response|ai)",
    post: `(?:${POST_INLINE})`,
  },
  // Original probe: a wider vocabulary and the heading shape ("## User", no colon).
  "casebible-probe-v1": {
    owner: "(?:you said|user|human|me|prompt|matt|question|you)",
    assistant:
      "(?:chatgpt said|gemini said|claude said|assistant|chatgpt|claude|gemini|bard|copilot|perplexity|deepseek|grok|qwen|gpt-?4o?|model|answer|response|ai)",
    post: `(?:${POST_INLINE}|${POST_HEAD})`,
  },
};

const MARKERS = {};
for (const [name, w] of Object.entries(WORDS)) {
  MARKERS[name] = {
    owner: new RegExp(`${PRE}${w.owner}${TS}${w.post}`, "gim"),
    assistant: new RegExp(`${PRE}${w.assistant}${TS}${w.post}`, "gim"),
  };
}

const CLIP_HOSTS = {
  "proffer-v1": String.raw`(?:gemini|bard)\.google\.com|chatgpt\.com|chat\.openai\.com|claude\.ai|(?:www\.|playground\.)?perplexity\.ai`,
  "casebible-probe-v1": String.raw`(?:gemini|bard)\.google\.com|chatgpt\.com|chat\.openai\.com|claude\.ai|(?:www\.|playground\.)?perplexity\.ai|copilot\.microsoft\.com|grok\.com|x\.com/i/grok|chat\.qwen\.ai|venice\.ai|chat\.deepseek\.com|poe\.com|you\.com`,
};
const CLIP_SOURCE = Object.fromEntries(
  Object.entries(CLIP_HOSTS).map(([k, hosts]) => [k, new RegExp(String.raw`(?:^|\n)\s*(?:source|url|link|permalink)\s*:\s*["'<]?https?://(?:www\.)?(?:${hosts})`, "i")]),
);
const PERPLEXITY_MARK = /r2cdn\.perplexity\.ai|LLM served by Perplexity/i;
const JSON_ROLE = /"role"\s*:\s*"(?:user|assistant|system|human|model)"/i;
const JSON_CONTENT = /"(?:content|text|parts)"\s*:/i;
const HTML_ROOT = /^(?:<\?xml[^>]*\?>\s*|<!--[\s\S]*?-->\s*)*(?:<!doctype\s+html\b|<html\b)/i;

// --- helpers ----------------------------------------------------------------------------------------------------------
const utf8 = new TextDecoder("utf-8");
const utf8Strict = new TextDecoder("utf-8", { fatal: true });

function isSpace(b) {
  return b === 0x20 || b === 0x09 || b === 0x0a || b === 0x0d || b === 0x0b || b === 0x0c;
}

/** Drop a UTF-8 byte order mark and surrounding ASCII whitespace (Go: bytes.TrimSpace(TrimPrefix(BOM))). */
export function trimHead(bytes) {
  let start = 0;
  let end = bytes.length;
  if (end >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) start = 3;
  while (start < end && isSpace(bytes[start])) start++;
  while (end > start && isSpace(bytes[end - 1])) end--;
  return bytes.subarray(start, end);
}

function startsWith(bytes, text) {
  if (bytes.length < text.length) return false;
  for (let i = 0; i < text.length; i++) if (bytes[i] !== text.charCodeAt(i)) return false;
  return true;
}

function hasNul(bytes) {
  return bytes.indexOf(0) >= 0;
}

/** True when the bytes are valid UTF-8, allowing the cut of a probe to split the last rune. */
function validUtf8(bytes) {
  for (let cut = 0; cut <= 3 && cut < bytes.length; cut++) {
    try {
      utf8Strict.decode(bytes.subarray(0, bytes.length - cut));
      return true;
    } catch {
      /* try one byte shorter */
    }
  }
  return false;
}

function countMatches(re, text) {
  return (text.match(re) || []).length;
}

function classToken(text, token) {
  return new RegExp(`class="[^"]*\\b${token.replace(/[.*+?^${}()|[\]\\-]/g, "\\$&")}\\b[^"]*"`).test(text);
}

function result(format, signatureKind, confidence, ruleSource) {
  return { format, signature_kind: signatureKind, confidence, rule_source: ruleSource };
}

// --- AI chat ----------------------------------------------------------------------------------------------------------
/**
 * Classify the head of an object as one of the AI-chat formats, or return null.
 *
 * Port of detectAIChatContent (Go) with the HTML cases of the original probe. `trimmed` is the BOM-and-whitespace-trimmed head.
 */
export function detectAIChat(trimmed, ruleset) {
  if (trimmed.length === 0) return null;
  const GO = "handler_ai_chat_signature.go";
  const PY = "ai_chat_signature_probe.py";
  const first = trimmed[0];
  if (first === 0x5b /* [ */ || first === 0x7b /* { */) {
    const text = utf8.decode(trimmed.subarray(0, JSON_PROBE_BYTES));
    if (text.includes('"mapping"') && (text.includes('"author"') || text.includes('"create_time"'))) {
      return result("chatgpt_official_json", "chatgpt_mapping_author_v1", 0.95, GO);
    }
    if (text.includes('"chat_messages"') && text.includes('"sender"')) {
      return result("claude_conversations_json", "claude_chat_messages_sender_v1", 0.95, GO);
    }
    if (text.includes('"header"') && text.includes('"time"') && (text.includes('"Gemini Apps"') || text.includes('"Bard"') || text.includes('"Gemini"'))) {
      return result("gemini_activity_json", "gemini_takeout_activity_v1", 0.9, GO);
    }
    if (JSON_ROLE.test(text) && JSON_CONTENT.test(text)) {
      return result("ai_generic_json", "ai_role_content_turns_v1", 0.7, GO);
    }
    return null;
  }
  const head = trimmed.subarray(0, TEXT_PROBE_BYTES);
  if (hasNul(head)) return null;
  const text = utf8.decode(head);
  if ((text.includes("Chat Memo") && text.includes("Total Conversations:")) || (text.includes("Total Conversations:") && text.includes("Platform:") && text.includes("URL:"))) {
    return result("ai_chat_memo_txt", "chat_memo_header_v1", 0.95, GO);
  }
  const markers = MARKERS[ruleset];
  const owner = countMatches(markers.owner, text);
  const assistant = countMatches(markers.assistant, text);
  if (owner >= 2 && assistant >= 2) {
    const margin = Math.min(owner, assistant) - 2;
    return result("ai_markdown_transcript", "ai_speaker_markers_v1", Math.round((0.7 + Math.min(0.2, 0.02 * margin)) * 100) / 100, ruleset === "proffer-v1" ? GO : PY);
  }
  if (CLIP_SOURCE[ruleset].test(text)) return result("ai_clipped_markdown", "ai_clip_front_matter_source_v1", 0.8, ruleset === "proffer-v1" ? GO : PY);
  if (PERPLEXITY_MARK.test(text)) return result("ai_clipped_markdown", "perplexity_watermark_v1", 0.6, ruleset === "proffer-v1" ? GO : PY);
  return null;
}

/** HTML exports the original probe knows and the Go port leaves to generic_html_document. */
function detectAIChatHTML(text) {
  const PY = "ai_chat_signature_probe.py";
  if (text.includes("jsonData") && text.includes('"mapping"')) return result("chatgpt_chat_html", "chatgpt_chat_html_jsondata_v1", 0.9, PY);
  const low = text.toLowerCase();
  if ((low.includes("gemini apps") || low.includes(">bard<") || low.includes("bard</")) && low.includes("mdl-")) {
    return result("gemini_activity_html", "gemini_takeout_activity_html_v1", 0.8, PY);
  }
  return null;
}

// --- generic Proffer detection ----------------------------------------------------------------------------------------
function ndjsonValues(text) {
  let values = 0;
  let firstLine = null;
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    let parsed;
    try {
      parsed = JSON.parse(line);
    } catch {
      return false;
    }
    if (values === 0) firstLine = parsed;
    values++;
  }
  const sms = firstLine && typeof firstLine === "object" && firstLine.thread != null && firstLine.source_pos != null && firstLine.kind != null;
  return values >= 2 || (values === 1 && sms);
}

/**
 * Detect the format of an object from its first bytes.
 *
 * `head` is the first bytes of the object; `truncated` says the object continues past them (the last line is then cut,
 * as signatureHead handles it in the Go engine). `ruleset` is "proffer-v1" (default; the Go port) or "casebible-probe-v1"
 * (the original probe's wider vocabulary and heading markers). Returns `{format, signature_kind, confidence, rule_source}`.
 * Confidence is assigned per rule here (0.95 structural key signatures down to 0.4 for plain text); the Go and Python
 * sources return no score.
 */
export function sniff(head, { ruleset = "proffer-v1", truncated = false } = {}) {
  if (!RULESETS.includes(ruleset)) throw new Error(`bad request: unknown ruleset ${ruleset}`);
  const HS = "handler_selection_store.go";
  const trimmed = trimHead(head);
  if (trimmed.length === 0) return result("empty", "empty_v1", 1, HS);
  if (startsWith(trimmed, "%PDF-")) return result("pdf", "pdf_header_v1", 1, HS);
  if (startsWith(trimmed, "PK\x03\x04") || startsWith(trimmed, "PK\x05\x06") || startsWith(trimmed, "PK\x07\x08")) {
    const text = utf8.decode(trimmed);
    if (text.includes("[Content_Types].xml") && text.includes("word/")) return result("docx", "office_open_xml_word_package_v1", 0.95, HS);
    return result("archive", "zip_container_v1", 1, HS);
  }
  if (
    (trimmed[0] === 0x1f && trimmed[1] === 0x8b) ||
    startsWith(trimmed, "7z\xbc\xaf\x27\x1c") ||
    startsWith(trimmed, "Rar!\x1a\x07") ||
    (trimmed.length > 262 && String.fromCharCode(...trimmed.subarray(257, 262)) === "ustar")
  ) {
    return result("archive", "archive_magic_v1", 1, HS);
  }
  if (trimmed[0] === 0x3c /* < */) {
    const text = utf8.decode(trimmed);
    if (HTML_ROOT.test(text.slice(0, 4096))) {
      if (classToken(text, "_a6-g") && classToken(text, "_a6-h") && classToken(text, "_a6-p") && classToken(text, "_a6-o") && !text.includes('"_a70f"')) {
        return result("facebook_messenger_html", "facebook_messenger_thread_html_v1", 0.95, "html_signature.go");
      }
      const chatHtml = detectAIChatHTML(text);
      if (chatHtml) return chatHtml;
      if (ruleset === "casebible-probe-v1") {
        const hits = detectAIChat(trimmed, ruleset); // the probe also read speaker markers and clip headers inside HTML
        if (hits) return hits;
      }
      return result("generic_html_document", "html_document_root_v1", 0.95, "html_signature.go");
    }
    const stripped = text.replace(/<\?[\s\S]*?\?>|<!--[\s\S]*?-->|<!DOCTYPE[^>]*>/gi, "");
    const m = /<([A-Za-z_][\w.\-:]*)/.exec(stripped);
    if (!m) return result("xml", "xml_prefix_v1", 0.5, HS);
    const local = m[1].split(":").pop().toLowerCase();
    if (local === "smses") return result("smsbackuprestore_xml", "sms_backup_restore_smses_root_v1", 0.95, HS);
    if (local === "calls") return result("callsbackuprestore_xml", "sms_backup_restore_calls_root_v1", 0.95, HS);
    return result("xml", "xml_root_v1", 0.9, HS);
  }
  // Order as in detectHandlerContent: JSON AI-chat keys, then ndjson, then the text AI-chat markers.
  const jsonLeading = trimmed[0] === 0x7b || trimmed[0] === 0x5b;
  if (jsonLeading) {
    const chat = detectAIChat(trimmed, ruleset);
    if (chat) return chat;
  }
  let probe = trimmed;
  if (truncated) {
    const cut = probe.lastIndexOf(0x0a);
    if (cut > 0) probe = probe.subarray(0, cut + 1);
  }
  if (!hasNul(probe)) {
    const text = utf8.decode(probe);
    if (ndjsonValues(text)) return result("ndjson", "newline_delimited_json_v1", 0.8, HS);
  }
  if (!jsonLeading) {
    const chat = detectAIChat(trimmed, ruleset);
    if (chat) return chat;
  }
  if (jsonLeading) return result("json", "json_container_prefix_v1", 0.5, HS);
  if (validUtf8(trimmed) && !hasNul(trimmed)) return result("text", "utf8_text_v1", 0.4, HS);
  return result("binary", "opaque_binary_v1", 0.6, HS);
}
