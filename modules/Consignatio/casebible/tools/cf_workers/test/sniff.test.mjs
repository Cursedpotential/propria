// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Format sniffer rules on real-shaped bytes (the head of each export the owner's AI-chat files come from).
import test from "node:test";
import assert from "node:assert/strict";
import { sniff, detectAIChat, trimHead } from "../format_sniffer/src/sniff.js";

const enc = (s) => new TextEncoder().encode(s);
const fmt = (s, opts) => sniff(typeof s === "string" ? enc(s) : s, opts).format;

test("ChatGPT export: mapping + author", () => {
  const head = '[{"title":"Custody timeline","create_time":1731439894.1,"update_time":1731440000.0,"mapping":{"a1":{"id":"a1","message":{"id":"m1","author":{"role":"user","name":null},"content":{"content_type":"text","parts":["hi"]}},"parent":null,"children":["a2"]}},"moderation_results":[]}';
  const r = sniff(enc(head), { truncated: true });
  assert.equal(r.format, "chatgpt_official_json");
  assert.equal(r.signature_kind, "chatgpt_mapping_author_v1");
  assert.ok(r.confidence >= 0.9);
});

test("Claude export: chat_messages + sender", () => {
  const head = '[{"uuid":"1a2b","name":"Hearing prep","created_at":"2025-11-12T19:31:34.000000Z","account":{"uuid":"9f"},"chat_messages":[{"uuid":"m1","text":"hello","sender":"human","created_at":"2025-11-12T19:31:35Z"}]}]';
  assert.equal(fmt(head), "claude_conversations_json");
});

test("Gemini Takeout activity JSON: header + time", () => {
  const head = '[{"header":"Gemini Apps","title":"Prompted what is MCL 722.23","time":"2025-11-12T19:31:34.123Z","products":["Gemini Apps"]}]';
  assert.equal(fmt(head), "gemini_activity_json");
});

test("generic role/content turns", () => {
  assert.equal(fmt('{"messages":[{"role":"user","content":"a"},{"role":"assistant","content":"b"}]}'), "ai_generic_json");
});

test("a JSON file with no chat keys stays json", () => {
  assert.equal(fmt('{"a":1,"b":[1,2,3]}'), "json");
});

test("a markdown design note that quotes chat_messages is not a Claude export", () => {
  const md = "# Foreman work order\n\nThe Claude export carries `\"chat_messages\"` and `\"sender\"` keys.\n";
  const r = sniff(enc(md));
  assert.notEqual(r.format, "claude_conversations_json");
  assert.equal(r.format, "text");
});

test("Chat Memo header", () => {
  const head = "Chat Memo - All Conversations\nExported: 2025-11-12\nTotal Conversations: 214\n\nPlatform: ChatGPT\nURL: https://chatgpt.com/c/abc\n";
  assert.equal(fmt(head), "ai_chat_memo_txt");
});

test("speaker-marked transcript needs two owner and two assistant markers", () => {
  const two = "**User:** how do I file\n\n**Assistant:** here is how\n\n**User:** and the fee\n\n**Assistant:** it is 175 dollars\n";
  const r = sniff(enc(two));
  assert.equal(r.format, "ai_markdown_transcript");
  assert.equal(r.signature_kind, "ai_speaker_markers_v1");
  const one = "**User:** how do I file\n\n**Assistant:** here is how\n";
  assert.equal(fmt(one), "text");
});

test("SMS-style transcript is not an AI chat", () => {
  const sms = "[11/12/2025, 7:31 PM] Matt: ok\n[11/12/2025, 7:32 PM] Katrina: fine\n[11/12/2025, 7:33 PM] Matt: thanks\n";
  assert.notEqual(fmt(sms), "ai_markdown_transcript");
});

test("heading-shaped transcript (no colon) is only found by the original probe's ruleset", () => {
  const md = "### User\nhello\n### Assistant\nhi\n### User\nmore\n### Assistant\nsure\n";
  assert.equal(fmt(md, { ruleset: "proffer-v1" }), "text");
  assert.equal(fmt(md, { ruleset: "casebible-probe-v1" }), "ai_markdown_transcript");
});

test("clipped markdown is found by its front-matter source host or the Perplexity watermark", () => {
  assert.equal(fmt("---\ntitle: Notes\nsource: https://claude.ai/chat/123\n---\nbody\n"), "ai_clipped_markdown");
  assert.equal(fmt("Some answer.\n\nLLM served by Perplexity Labs\n"), "ai_clipped_markdown");
  assert.equal(fmt("---\nsource: https://example.com/page\n---\nbody\n"), "text");
  assert.equal(fmt("---\nsource: https://poe.com/chat/1\n---\nbody\n", { ruleset: "proffer-v1" }), "text");
  assert.equal(fmt("---\nsource: https://poe.com/chat/1\n---\nbody\n", { ruleset: "casebible-probe-v1" }), "ai_clipped_markdown");
});

test("HTML: ChatGPT chat.html, Gemini activity html, generic html, Facebook thread", () => {
  const chat = '<!DOCTYPE html><html><head><title>ChatGPT Data Export</title></head><body><script>var jsonData = [{"title":"t","mapping":{}}];</script></body></html>';
  assert.equal(fmt(chat), "chatgpt_chat_html");
  const gem = '<!DOCTYPE html><html><body><div class="mdl-grid"><p class="mdl-typography--title">Gemini Apps<br></p></div></body></html>';
  assert.equal(fmt(gem), "gemini_activity_html");
  assert.equal(fmt("<html><body><p>hello</p></body></html>"), "generic_html_document");
  assert.equal(fmt('<?xml version="1.0"?><!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0//EN"><html><body>voice</body></html>'), "generic_html_document");
  const fb = '<html><body><div class="_a6-g"><div class="_a6-h _a6-i">Matt</div><div class="_a6-p">hi</div><div class="_a6-o">Nov 12, 2025</div></div></body></html>';
  assert.equal(fmt(fb), "facebook_messenger_html");
  assert.equal(fmt(fb.replace("</body>", '<div class="_a70f">section</div></body>')), "generic_html_document");
});

test("magic numbers: pdf, zip, docx, gzip, 7z, xml roots", () => {
  assert.equal(fmt("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n1 0 obj"), "pdf");
  assert.equal(fmt(new Uint8Array([0x50, 0x4b, 3, 4, 20, 0, 0, 0, 8, 0])), "archive");
  const docx = new Uint8Array([...new Uint8Array([0x50, 0x4b, 3, 4]), ...enc("....[Content_Types].xml....word/document.xml")]);
  assert.equal(fmt(docx), "docx");
  assert.equal(fmt(new Uint8Array([0x1f, 0x8b, 8, 0, 0, 0])), "archive");
  assert.equal(fmt('<?xml version="1.0"?><smses count="2"><sms/></smses>'), "smsbackuprestore_xml");
  assert.equal(fmt("<calls count='1'><call/></calls>"), "callsbackuprestore_xml");
  assert.equal(fmt("<root><a/></root>"), "xml");
});

test("NDJSON: a cut last line is dropped when the probe is truncated", () => {
  const body = '{"thread":"1","source_pos":"a","kind":"sms"}\n{"thread":"2","source_pos":"b","kind":"sms"}\n{"thread":"3","sou';
  assert.equal(fmt(body, { truncated: true }), "ndjson");
  assert.equal(fmt(body, { truncated: false }), "json"); // the cut line is invalid JSON: the failure signatureHead exists to avoid
});

test("BOM and leading whitespace are ignored; empty and binary heads are named", () => {
  const withBom = new Uint8Array([0xef, 0xbb, 0xbf, ...enc('\n  [{"chat_messages":[{"sender":"human"}]}]')]);
  assert.equal(fmt(withBom), "claude_conversations_json");
  assert.equal(fmt(""), "empty");
  assert.equal(fmt(new Uint8Array([0, 1, 2, 255, 254, 0, 9])), "binary");
  assert.equal(trimHead(enc("  x \n")).length, 1);
});

test("a probe cut inside a multibyte rune is still text", () => {
  const bytes = enc("café ".repeat(10));
  assert.equal(fmt(bytes.subarray(0, bytes.length - 1)), "text");
});

test("an unknown ruleset is rejected", () => {
  assert.throws(() => sniff(enc("x"), { ruleset: "nope" }), /unknown ruleset/);
  assert.equal(detectAIChat(new Uint8Array(0), "proffer-v1"), null);
});
