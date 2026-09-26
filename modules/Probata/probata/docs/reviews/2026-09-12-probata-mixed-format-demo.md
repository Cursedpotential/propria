# Probata mixed-format demonstration receipt — 2026-09-12

> Byline: Codex · GPT-5 · 2026-09-12

## Result

A bounded synthetic fixture set was run through real Probata interfaces. Each
source was SHA-256 hashed before parsing. The run emitted no message bodies in
its report and performed zero source mutations, PostgreSQL writes, object-store
writes, or canonical evidence promotions.

| Input | Boundary | Parser | Records | SHA-256 |
|---|---|---|---:|---|
| `imessage-thread.txt` | evidence parser proof | `messages.imessage-txt` | 4 | `fbb8db88fa09d44639370b30f3f3937d405c93db06151cd95d7d26c4aee4943c` |
| `sms-backup.xml` | evidence parser proof | `messages.sms-xml` | 4 | `3a3290be5d26e4928d11514300b8999a8b9a24aadd0d53c1d2f81ec9f644a88d` |
| `chatgpt-conversations.json` | context-chat parser proof | `transcripts.chatgpt-official` | 2 | `4d35fb943938d7048999b517a2169050f358d74671c28747428e2b3d15720f59` |
| `claude-conversations.json` | context-chat full dry run | `transcripts.claude-ai-export` | 2 | `f5428d09153217d62b9a7945c8074571722ae60fb8745e2deb043edc670365a9` |

The iMessage result contained message, call, and event records. The SMS result
retained all four message events, including MMS/empty-body cases. The Claude
dry run parsed one conversation, classified one chunk assignment into the
`context` lane, and stored zero records. The ChatGPT sample used its registered
Probata parser and was correctly labeled context-chat, not evidence.

AI-chat exports are intentionally fenced out of Probata's evidence custody
lane. This run preserves that D-082 boundary: AI chats are organizational
context and do not become evidence merely because a parser can read them.

## Reproduction

From the Probata repository root:

```powershell
$env:TEMP = 'E:\AI_Workspace\.tmp'
$env:TMP = 'E:\AI_Workspace\.tmp'
$env:UV_CACHE_DIR = 'E:\AI_Workspace\.uv-cache'
uv run --no-sync python scripts\run_mixed_probata_demo.py tests\fixtures\probata_mixed_demo
```

The demonstrator was executed repeatedly; hashes and record counts were
identical. Its final execution exited 0. Ruff also exited 0.

The capability-focused regression command passed **19 tests**:

```powershell
uv run --no-sync -- python -m pytest -q -p no:cacheprovider `
  tests/test_imessage_txt.py `
  tests/test_sms_xml.py `
  tests/test_context_chat_ingest.py::test_dry_run_exercises_parse_chunk_classify_without_database `
  tests/test_transcript_tools.py::test_chatgpt_official_wrapper_end_to_end `
  tests/test_custody_ai_chat_fence.py
```

One broader focused run produced 37 passes and one unrelated failure in
`test_existing_builtin_tools_make_no_unverified_format_or_quality_claims`: the
test expected literal tool version `unversioned`, while a current registered
tool reported `1.0.0`. That global registry-contract drift was not changed as
part of this demonstration.

## Proof boundary

This is a local, synthetic, non-persistent parser/context-pipeline proof. It is
not proof of a live canonical evidence ingest, PostgreSQL storage, R2/B2
transfer, production deployment, or a completed H1/H2/H3 custody chain. The
SHA-256 values are pre-parse H1-style raw-byte hashes, but no custody rows or
chain heads were written. The first context dry run fetched model artifacts
into the required E: cache; dry run means no governed data-store write, not
zero dependency/cache activity.
