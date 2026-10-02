# HTML tool bench, 2026-10-02

> _Byline: Claude Code · Sonnet · 2026-10-02_
> Owner order 2026-10-02 11:59 EDT: "We need an HTML parsing tool for Facebook and other files", then "find the
> best one", "score per file type", "emoji issues". Everything here was run on real casevault files. The code
> that produced every number is in `bench/`; the raw rows are in `results/`; the ranks that went into the tool
> registry are `ranks.json` (generated into
> `server/tools/extractors/html_text/_ranks.py`).

## What was decided

| Question | Answer |
|---|---|
| Facebook Messenger thread (`message_N.html`) | **DuckDB webbed XPath template `facebook_messenger_html_v1`.** every message matched on sender, time and body (the bench query shape 100%; the shipped template 9,997 to 10,000 of 10,000 per big thread against the oracle, the 0 to 3 differences being a block-nesting corner of the oracle), every emoji message and every reaction emoji intact, a 10,000-message thread in about 2 to 9 s inside the live pg_duckdb. The Python tools below only return text; this one returns records. |
| Other HTML (saved pages, docs, Google Voice, WhatsApp, Snapchat, Facebook section pages) | **DuckDB webbed template `generic_html_document_v1`** as the engine default (text blocks into the document path); the seven Python tools are selectable alternatives and fallbacks, ranked per file type below. |
| Best text tool, per file type | `html2text` for Facebook threads and sections, WhatsApp, Snapchat and Takeout My Activity; `markitdown` for generic documents; `docling` for Google Voice. The full per-type tables are below; every tool stays selectable. |
| The iMessage HTML export | No HTML text tool reads it: the whole conversation is an HTML string inside a JavaScript template literal (`const rawHtml = ...`). It needs its own template (the Case Bible `elt_imessage_html_v3` is the sibling). Not built here. |
| Facebook-specific parsers off the shelf | None works on a current export. See the survey. |
| Emoji and accents | The Facebook **HTML** export is real UTF-8: 62,292 strings across the five threads, 5,922 with non-ASCII, zero mojibake. The mojibake repair is for the **JSON** export only. See the emoji section. |

## Sample set (24 files fetched from the vault, 4 families' worth excluded and why)

20 usable files, by family (how each was found: catalog `raw_duck.vault_objects` by name and path, then B2):

| Family | Files | Notes |
|---|---|---|
| Facebook Messenger thread HTML | 5 (14 KB to 8.1 MB) | 2024 layout (`div` cards, timestamp `div`) and 2025 layout (`section`/`h2`/`footer`); 10,000 messages in the three big ones |
| Facebook export section pages | 3 | logins and logouts, search history, notifications tab |
| Google Voice (XHTML) | 3 | call, text, voicemail |
| WhatsApp `_chat.html` | 1 | |
| iMessage export | 1 (8.5 MB) | messages inside a script string |
| Snapchat chat history | 1 | |
| Takeout My Activity | 1 (62 MB) | the large-file stress test |
| Documentation and misc (saved pages, Python docs, Javadoc, an exported TeraCopy page, a Skype page) | 5 | |

Families the catalog does not hold as HTML, so they could not be scored: **ChatGPT `chat.html`** (the only
`chat.html` is WhatsApp's `_chat.html`), **email HTML bodies** (email is mbox; the "email" name matches I looked at were Google
Voice voicemail pages). **Five files were excluded because they are scrambled bytes in the vault itself**: Facebook
`account_activity.html`, `your_friends.html`, `your_post_audiences.html`, `30.html` and a Takeout `MyActivity.html`
(entropy 7.99 bits per byte, no markup; the catalog sha1 equals the scrambled bytes). See
`modules/Consignatio/docs/receipts/2026-10-02-scrambled-files-README.md`.

## Method (so the numbers can be re-run)

- **Oracle.** A Python standard-library `html.parser` pass (`bench/oracle.py`), so no tool grades its own output: the
  visible text of the page (for the iMessage export, also the text inside the script string), and for Facebook
  threads the list of message blocks (sender, time, body, reactions, media references). Emoji are counted as
  units (flags, keycaps, skin-tone modifiers, ZWJ sequences stay one unit).
- **Generic score, per file type.** Fidelity = 0.60 x text recall + 0.25 x emoji recall (when the type has emoji) +
  0.15 x link and media-reference recall (when it has references), averaged over the files of that type; a file
  the tool failed on (error, timeout, memory cap) counts 0; a tool whose output holds more than 1.3x the visible
  text (CSS and script leakage) loses 10%. **primary** = best fidelity; **fallback** = within 0.15; **experimental**
  = the rest. That rule is what produced `_ranks.py`; the weights are in `bench/rank_from_bench.py`.
- **Structured score (Facebook threads).** Messages matched exactly on sender, time string and body; emoji messages
  matched exactly; reaction emoji recall; media references.
- **Resources.** Each tool runs in its own process; wall time and peak RSS (process plus children) are measured by the
  parent. Timeouts 300 to 600 s, memory cap 6 GB. Desktop, Windows, Python 3.13, DuckDB 1.5.6 with the current webbed
  build. The engine itself runs the templates in the DuckDB 1.4.3 inside pg_duckdb; both were checked (see below).
- Speed figures include process start and library import, which is why Docling costs 20 to 30 s and unstructured 8 to 14 s even on a
  1 KB file.

## Tools surveyed

Evaluated on real files: DuckDB webbed (`read_html`, `read_html_blocks`, `html_extract_text`, and `parse_html` + XPath),
`lxml`, `BeautifulSoup4`, `selectolax` (lexbor), `html2text`, `MarkItDown`, `trafilatura`, `readability-lxml`,
`Docling` (HTML backend), `unstructured` (`partition_html`). **Not evaluated:** Rust `scraper` / `html5ever` (no Rust
toolchain on this machine; `selectolax` is the native-parser representative in the table); `html5lib` (no case needed it).
`trafilatura` and `readability-lxml` are article extractors that discard what looks like boilerplate: on chat pages
they dropped 19% to 68% of the messages, so they are not registered.

## Per-file-type tables

#### fb_messenger_thread -> `facebook_messenger_html`  (5 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| html2text | 5/5 | 1.000 | 1.000 | 0.832 | 1.000x | 7.15 | 89 | 0.966 |
| markitdown | 5/5 | 1.000 | 1.000 | 0.831 | 1.000x | 12.65 | 303 | 0.966 |
| selectolax_lexbor | 5/5 | 1.000 | 1.000 | 0.086 | 1.000x | 4.92 | 129 | 0.843 |
| lxml | 5/5 | 1.000 | 1.000 | 0.086 | 1.000x | 5.10 | 127 | 0.843 |
| beautifulsoup4 | 5/5 | 1.000 | 1.000 | 0.086 | 1.000x | 7.28 | 241 | 0.843 |
| webbed_read_html_blocks | 5/5 | 1.000 | 1.000 | 0.086 | 1.016x | 5.99 | 447 | 0.843 |
| webbed_generic_template | 5/5 | 1.000 | 1.000 | 0.085 | 1.000x | 50.79 | 304 | 0.843 |
| unstructured | 5/5 | 0.999 | 0.942 | 0.086 | 1.000x | 119.65 | 383 | 0.834 |
| trafilatura | 5/5 | 0.811 | 0.985 | 0.719 | 0.875x | 12.13 | 338 | 0.819 |
| docling | 5/5 | 0.893 | 0.987 | 0.085 | 0.900x | 45.77 | 661 | 0.776 |
| webbed_html_extract_text | 5/5 | 0.819 | 1.000 | 0.019 | 4.218x | 5.40 | 145 | 0.634 |
| readability_lxml | 5/5 | 0.324 | 0.330 | 0.005 | 0.324x | 5.01 | 189 | 0.257 |

#### fb_other_section -> `facebook_export_section_html`  (3 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| html2text | 3/3 | 0.929 | 1.000 | 0.499 | 1.067x | 0.36 | 31 | 0.843 |
| markitdown | 3/3 | 0.929 | 1.000 | 0.499 | 1.067x | 2.39 | 93 | 0.843 |
| selectolax_lexbor | 3/3 | 0.929 | 1.000 | 0.000 | 1.067x | 0.27 | 30 | 0.760 |
| lxml | 3/3 | 0.929 | 1.000 | 0.000 | 1.067x | 0.32 | 34 | 0.760 |
| webbed_read_html_blocks | 3/3 | 0.929 | 1.000 | 0.000 | 1.125x | 0.53 | 55 | 0.760 |
| beautifulsoup4 | 3/3 | 0.929 | 1.000 | 0.000 | 1.067x | 0.71 | 41 | 0.760 |
| webbed_generic_template | 3/3 | 0.890 | 1.000 | 0.000 | 1.028x | 0.94 | 60 | 0.729 |
| docling | 3/3 | 0.760 | 0.833 | 0.168 | 0.898x | 17.45 | 355 | 0.678 |
| unstructured | 3/3 | 0.754 | 1.000 | 0.000 | 0.880x | 7.54 | 268 | 0.621 |
| webbed_html_extract_text | 3/3 | 0.775 | 1.000 | 0.000 | 7.551x | 0.55 | 51 | 0.583 |
| trafilatura | 3/3 | 0.585 | 0.833 | 0.000 | 0.712x | 1.31 | 55 | 0.505 |
| readability_lxml | 3/3 | 0.182 | 0.000 | 0.000 | 0.186x | 0.69 | 51 | 0.145 |

#### google_my_activity -> `google_takeout_activity_html`  (1 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| html2text | 1/1 | 1.000 | 1.000 | 1.000 | 1.000x | 36.15 | 732 | 1.000 |
| selectolax_lexbor | 1/1 | 1.000 | 1.000 | 0.028 | 1.000x | 9.44 | 781 | 0.854 |
| lxml | 1/1 | 1.000 | 1.000 | 0.028 | 1.000x | 13.85 | 659 | 0.854 |
| webbed_read_html_blocks | 1/1 | 1.000 | 1.000 | 0.028 | 1.000x | 27.46 | 3462 | 0.854 |
| beautifulsoup4 | 1/1 | 1.000 | 1.000 | 0.028 | 1.000x | 51.92 | 1680 | 0.854 |
| unstructured | 1/1 | 1.000 | 1.000 | 0.028 | 1.000x | 576.06 | 1161 | 0.854 |
| webbed_html_extract_text | 1/1 | 0.808 | 1.000 | 0.000 | 0.909x | 8.54 | 655 | 0.735 |
| readability_lxml | 1/1 | 0.000 | 0.000 | 0.000 | 0.000x | 112.34 | 1130 | 0.000 |
| webbed_generic_template | 0/1 | n/a | n/a | n/a | n/ax | 420.00 | 1333 | 0.000 |
| markitdown | 0/1 | n/a | n/a | n/a | n/ax | 420.01 | 733 | 0.000 |
| trafilatura | 0/1 | n/a | n/a | n/a | n/ax | 420.21 | 1596 | 0.000 |
| docling | 0/1 | n/a | n/a | n/a | n/ax | 600.03 | 1072 | 0.000 |

#### google_voice -> `google_voice_html`  (3 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| docling | 3/3 | 1.000 | 1.000 | 0.556 | 1.007x | 16.00 | 350 | 0.922 |
| markitdown | 3/3 | 1.000 | 1.000 | 0.333 | 1.005x | 1.86 | 88 | 0.881 |
| html2text | 3/3 | 1.000 | 1.000 | 0.667 | 1.601x | 0.25 | 29 | 0.850 |
| selectolax_lexbor | 3/3 | 1.000 | 1.000 | 0.000 | 1.000x | 0.24 | 29 | 0.817 |
| lxml | 3/3 | 1.000 | 1.000 | 0.000 | 1.000x | 0.31 | 31 | 0.817 |
| beautifulsoup4 | 3/3 | 1.000 | 1.000 | 0.000 | 1.000x | 0.66 | 42 | 0.817 |
| readability_lxml | 3/3 | 0.999 | 1.000 | 0.000 | 0.999x | 0.67 | 48 | 0.816 |
| unstructured | 3/3 | 0.998 | 1.000 | 0.000 | 1.003x | 9.11 | 268 | 0.815 |
| trafilatura | 3/3 | 0.980 | 1.000 | 0.056 | 1.009x | 1.16 | 75 | 0.811 |
| webbed_read_html_blocks | 3/3 | 0.982 | 1.000 | 0.000 | 1.039x | 0.53 | 49 | 0.806 |
| webbed_html_extract_text | 3/3 | 1.000 | 1.000 | 0.000 | 2.426x | 0.53 | 49 | 0.735 |
| webbed_generic_template | 3/3 | 0.440 | 1.000 | 0.000 | 0.440x | 0.61 | 53 | 0.373 |

#### imessage_export -> `imessage_export_html`  (1 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| webbed_html_extract_text | 1/1 | 1.000 | 1.000 | 0.000 | 1.770x | 3.68 | 236 | 0.765 |
| webbed_read_html_blocks | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 0.61 | 83 | 0.001 |
| html2text | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 0.64 | 125 | 0.001 |
| lxml | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 0.66 | 65 | 0.001 |
| beautifulsoup4 | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 0.90 | 115 | 0.001 |
| webbed_generic_template | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 0.95 | 105 | 0.001 |
| readability_lxml | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 0.99 | 107 | 0.001 |
| trafilatura | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 1.70 | 162 | 0.001 |
| markitdown | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 1.96 | 175 | 0.001 |
| selectolax_lexbor | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 3.62 | 86 | 0.001 |
| docling | 1/1 | 0.000 | 0.003 | 0.000 | 0.000x | 16.28 | 443 | 0.001 |
| unstructured | 1/1 | 0.000 | 0.000 | 0.000 | 0.000x | 9.91 | 256 | 0.000 |

#### number_named_export -> `generic_html_document`  (1 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| html2text | 1/1 | 0.880 | n/a | 1.000 | 1.085x | 0.29 | 28 | 0.904 |
| trafilatura | 1/1 | 0.880 | n/a | 1.000 | 1.085x | 1.35 | 48 | 0.904 |
| markitdown | 1/1 | 0.832 | n/a | 1.000 | 1.171x | 2.03 | 89 | 0.865 |
| selectolax_lexbor | 1/1 | 0.880 | n/a | 0.500 | 1.090x | 0.25 | 28 | 0.804 |
| lxml | 1/1 | 0.880 | n/a | 0.500 | 1.085x | 0.31 | 31 | 0.804 |
| webbed_read_html_blocks | 1/1 | 0.880 | n/a | 0.500 | 1.100x | 0.52 | 48 | 0.804 |
| webbed_generic_template | 1/1 | 0.880 | n/a | 0.500 | 1.085x | 0.61 | 49 | 0.804 |
| beautifulsoup4 | 1/1 | 0.880 | n/a | 0.500 | 1.085x | 0.66 | 39 | 0.804 |
| unstructured | 1/1 | 0.880 | n/a | 0.500 | 1.085x | 9.29 | 263 | 0.804 |
| docling | 1/1 | 0.880 | n/a | 0.500 | 1.085x | 16.48 | 353 | 0.804 |
| readability_lxml | 1/1 | 0.771 | n/a | 0.000 | 0.963x | 0.69 | 50 | 0.617 |
| webbed_html_extract_text | 0/1 | n/a | n/a | n/a | n/ax | 0.54 | 48 | 0.000 |

#### snapchat_export -> `snapchat_export_html`  (1 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| html2text | 1/1 | 1.000 | n/a | 1.000 | 1.000x | 0.24 | 28 | 1.000 |
| markitdown | 1/1 | 1.000 | n/a | 1.000 | 1.000x | 1.84 | 89 | 1.000 |
| selectolax_lexbor | 1/1 | 1.000 | n/a | 0.000 | 1.000x | 0.28 | 27 | 0.800 |
| lxml | 1/1 | 1.000 | n/a | 0.000 | 1.000x | 0.31 | 31 | 0.800 |
| beautifulsoup4 | 1/1 | 1.000 | n/a | 0.000 | 1.000x | 0.63 | 38 | 0.800 |
| trafilatura | 1/1 | 1.000 | n/a | 0.000 | 1.000x | 1.57 | 75 | 0.800 |
| webbed_read_html_blocks | 1/1 | 0.991 | n/a | 0.000 | 0.991x | 0.50 | 48 | 0.793 |
| webbed_generic_template | 1/1 | 0.991 | n/a | 0.000 | 0.991x | 0.54 | 50 | 0.793 |
| unstructured | 1/1 | 0.991 | n/a | 0.000 | 0.991x | 8.97 | 263 | 0.793 |
| docling | 1/1 | 0.545 | n/a | 0.000 | 0.554x | 24.64 | 349 | 0.436 |
| webbed_html_extract_text | 1/1 | 0.500 | n/a | 0.000 | 3.696x | 0.50 | 48 | 0.360 |
| readability_lxml | 1/1 | 0.116 | n/a | 0.000 | 0.116x | 0.64 | 48 | 0.093 |

#### software_docs_and_misc -> `generic_html_document`  (4 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| markitdown | 4/4 | 0.996 | 1.000 | 0.635 | 0.758x | 1.94 | 90 | 0.731 |
| selectolax_lexbor | 4/4 | 0.984 | 1.000 | 0.247 | 0.774x | 0.28 | 30 | 0.665 |
| lxml | 4/4 | 0.984 | 1.000 | 0.247 | 0.774x | 0.32 | 33 | 0.665 |
| beautifulsoup4 | 4/4 | 0.984 | 1.000 | 0.247 | 0.774x | 0.72 | 41 | 0.665 |
| webbed_read_html_blocks | 4/4 | 0.976 | 1.000 | 0.247 | 1.034x | 0.54 | 50 | 0.661 |
| webbed_generic_template | 4/4 | 0.962 | 1.000 | 0.247 | 0.756x | 0.57 | 56 | 0.655 |
| html2text | 4/4 | 0.994 | 0.000 | 0.635 | 0.754x | 0.30 | 30 | 0.605 |
| docling | 4/4 | 0.770 | 1.000 | 0.363 | 0.594x | 16.62 | 351 | 0.559 |
| webbed_html_extract_text | 4/4 | 0.969 | 1.000 | 0.000 | 23.296x | 0.57 | 49 | 0.546 |
| unstructured | 4/4 | 0.787 | 0.500 | 0.000 | 0.591x | 8.77 | 269 | 0.437 |
| trafilatura | 4/4 | 0.700 | 0.000 | 0.234 | 0.537x | 1.19 | 74 | 0.380 |
| readability_lxml | 4/4 | 0.419 | 0.500 | 0.000 | 0.315x | 0.64 | 49 | 0.251 |

#### whatsapp_chat -> `whatsapp_chat_html`  (1 files)

| tool | files ok/total | text recall | emoji recall | link recall | text volume vs truth | median wall s | peak RSS MB | fidelity |
|---|---|---|---|---|---|---|---|---|
| html2text | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.28 | 30 | 1.000 |
| selectolax_lexbor | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.28 | 30 | 1.000 |
| lxml | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.35 | 35 | 1.000 |
| webbed_read_html_blocks | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.54 | 52 | 1.000 |
| webbed_html_extract_text | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.57 | 51 | 1.000 |
| webbed_generic_template | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.61 | 54 | 1.000 |
| beautifulsoup4 | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.65 | 42 | 1.000 |
| readability_lxml | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 0.73 | 49 | 1.000 |
| trafilatura | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 1.19 | 52 | 1.000 |
| markitdown | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 2.12 | 91 | 1.000 |
| unstructured | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 7.28 | 235 | 1.000 |
| docling | 1/1 | 1.000 | 1.000 | n/a | 1.000x | 16.58 | 354 | 1.000 |

## Facebook Messenger threads: the structured tools

Real message files, 5 threads (the three large ones hold 10,000 messages each; 2024 and 2025 layouts). The measure
is whether the tool returns the message records, not just text.

| tool | files ok/total | messages exact (sender+time+body) | sender+body only | emoji messages exact | reaction emoji recall | media refs out vs truth | median s | peak RSS MB |
|---|---|---|---|---|---|---|---|---|
| DuckDB webbed XPath template (facebook_messenger_html_v1 query shape) | 5/5 | 1.000 | 1.000 | 1.000 | 1.000 | 5618/5610 | 1.99 | 126 |
| selectolax (lexbor) + CSS on the `_a6-*` cards | 5/5 | 1.000 | 1.000 | 1.000 | 1.000 | 5618/5610 | 0.38 | 114 |
| BeautifulSoup4 + CSS on the `_a6-*` cards | 5/5 | 1.000 | 1.000 | 1.000 | 1.000 | 5618/5610 | 4.99 | 225 |
| lxml + XPath on the `_a6-*` cards | 5/5 | 0.996 | 0.996 | 0.991 | 1.000 | 5618/5610 | 2.28 | 145 |
| Case Bible `elt_fb_messenger_html_v1` (DuckDB regex, 2026-09-18) | 4/5 | 0.000 | 0.457 | 0.000 | 0.000 | 0/4846 | 0.31 | 77 |
| repo `facebook_messenger_html.py` before the 2026-10-02 fix (port of dial-stack FacebookExportParser) | 5/5 | 0.000 | 0.000 | 0.000 | 0.000 | 0/5610 | 3.50 | 235 |
| realdeveloperongithub/chat-history-manager `MessengerParser` selectors (GPL-3.0) | 5/5 | 0.000 | 0.000 | 0.000 | 0.000 | 0/5610 | 3.00 | 235 |

What the numbers mean:

- The existing **Case Bible `elt_fb_messenger_html_v1`** (2026-09-18) returns an empty timestamp on every current file
  (its timestamp pattern expects text directly inside `div._3-94._a6-o`; the export now nests the time in a child
  `div._a72d`), and it matches senders only by the `div` tag, so on the 2025 layout (`h2` sender) it fails outright.
  `facebook_messenger_html_v1` keeps its approach and timezone ruling (the time is local Eastern, read as
  America/Detroit and recorded as inferred) and replaces the regexes with XPath, so both layouts read.
- The repo's **Python port** `facebook_messenger_html.py` read only the pre-2024 layout (timestamp in a sibling `div`);
  it returned 0 messages from every current file. Fixed in this change (both layouts, reactions kept out of the
  body) and covered by tests.
- **chat-history-manager** (GPL-3.0) selects `div._3-95 _a6-g` and an old `_a6-o` sibling: 0 messages from current
  files, and its licence is a copyleft one.
- `lxml` XPath and `selectolax` CSS are fast alternatives with the same selectors; the engine template stays the
  default because DuckDB is the owner-ruled engine and it already carries the timezone, attachment-locator and
  participant logic. `lxml` XPath matched 99.6% of the messages exactly (the remaining 0.4% was not investigated).

## Facebook-specific parsers on GitHub and PyPI

All tested on a real 2024 thread (4 messages) and a real 2025 thread (10,003 blocks) where they run at all.

| Name and link | Licence | Last commit | Covers | Mojibake | Result on a real sample |
|---|---|---|---|---|---|
| [fbchat-archive-parser](https://github.com/ownaginatious/fbchat-archive-parser) | MIT | 2018-04-29 | the old single-page `messages.htm` export; JSON/CSV out | n/a (pre-JSON HTML) | installs only on Python 3.11 or older (the setup uses `SafeConfigParser`, removed in 3.12); **0 messages** from both files |
| [fbparser](https://github.com/arcward/fbparser) | MIT | 2017-10-21 | `messages.htm`, CSV/JSON/text out | n/a | **0 messages** from both files (same pre-2018 layout) |
| [chat-history-manager](https://github.com/realdeveloperongithub/chat-history-manager) (Messenger parser) | GPL-3.0 | 2024-08-18 | Messenger HTML plus WhatsApp, Instagram, Line, Kakao, WeChat, Google Chat, for Telegram import | HTML is UTF-8, none needed | **0 messages** (selectors target an older card layout) |
| [fb-json2table](https://github.com/numbersprotocol/fb-json2table) | GPL-3.0, archived | 2020-02-19 | any DYI **JSON** file to a table | not documented | not run (JSON only; the engine's JSON template already covers Messenger JSON) |
| [ByeByeMeta](https://github.com/rubillos/ByeByeMeta) | none stated | 2026-04-17 | Facebook/Instagram **posts**, rebuilt as one browsable page | n/a | not a record parser; not run |
| [fbpull / terry-facebook](https://github.com/terryum/terry-facebook) | none stated | 2026-05-25 | Facebook **JSON posts** into Obsidian notes | JSON | not run (posts only, needs model APIs) |
| [facebook-dyi-parser](https://github.com/lezsakdomi/facebook-dyi-parser) and [facebook-data-parser](https://github.com/hanford/facebook-data-parser) | none | 2021-03-13 and 2018-10-20 | JavaScript, old export formats | not documented | not run (no licence, abandoned, JavaScript) |
| [fbmessengerexport](https://github.com/karlicoss/fbmessengerexport) | MIT | 2026-09-18 | live Messenger API export, not the download | n/a | not applicable |
| Dial-stack `FacebookExportParser.ts` port (repo `facebook_messenger_html.py`) | ours | fixed 2026-10-02 | legacy and card layouts | n/a | 0 messages before the fix, 100% after |
| Case Bible `elt_fb_messenger_html_v1` | ours | 2026-09-18 | message cards | n/a | empty timestamps on current files |

Posts, comments and reactions in the Download-Your-Information archive: no maintained open-source parser covers
comments or reactions; reactions on **Messenger messages** are covered by our template (`reactions`: emoji, reactor,
and in the 2025 layout the time). Posts and comments are other sections of the archive (JSON or section HTML) and are
not built here; the generic template carries their text as document blocks until a section template exists.

**Registered as its own toolbox handler:** the DuckDB template `facebook_messenger_html_v1` (the Proffer handler for
the detected format `facebook_messenger_html`), with the repaired Python `messages.facebook-html` as its Python
fallback. No off-the-shelf Facebook-specific tool met the bar.

## Emoji, accents and the encoding history

- **What was learned before, and reused.** The Facebook **JSON** export writes UTF-8 as one `\u00XX` escape per byte, so
  text is garbled unless re-read as UTF-8; a naive fix breaks real non-ASCII text. The engine template
  `facebook_messenger_json_v1` (d1cb113a) repairs a value only when every character is at most U+00FF and the bytes are
  valid UTF-8 (`fbMojibakeFix`), and the Python `facebook_messenger_json.py` does the same per string
  (`s.encode('latin-1').decode('utf-8')`). Case Bible and casekit Phase 5b carry the same ruling. **None of this is
  applied to HTML**, on purpose.
- **The HTML export has no mojibake.** Over all 62,292 sender, body and reaction strings of the five threads: 5,922
  contain non-ASCII, 0 carry the mojibake signature (U+00C3/U+00C2/U+00E2/U+00F0 followed by a C1 byte), and a
  latin-1-to-UTF-8 round trip would change 0 of them. Applying the JSON fix to HTML would therefore do nothing useful
  and risks harm; the template reads the text as written.
- **Emoji corpus in those real threads:** 3,446 astral pictographs, 18 ZWJ sequences (for example a man shrugging
  with a skin tone, a head-in-clouds face), 8 skin-tone sequences, 44 symbols with variation selector 16, 532 BMP
  symbols, 5,802 curly quotes and dashes. Reactions arrive as the emoji followed by the reactor's name (and in the 2025
  layout the time in brackets).
- **Result for the template, run inside the live pg_duckdb:** 1,157 messages containing emoji, all exact; 2,220
  reaction emoji, all recovered. Counts per file are in `results/`.
- **Result for the generic text tools (emoji recall on those threads):** html2text, MarkItDown, BeautifulSoup4,
  selectolax, lxml and webbed `read_html_blocks` 1.000; unstructured 0.942; Docling 0.987; trafilatura 0.985;
  readability 0.330. A real defect found while testing: `lxml` given UTF-8 bytes without a `<meta charset>` guesses
  Latin-1 and turns every emoji into mojibake. The registered `html.lxml` tool states the encoding explicitly
  (covered by a test on a page with no charset declaration); the engine templates do not use lxml.
- **Not checked:** how Messenger itself renders a message. There is no Messenger session here, so "what Messenger
  shows" was approximated by the export's own text, parsed by an independent standard-library parser.

## The engine and the registry

- Detection: `facebook_messenger_thread_html_v1` (the card classes `_a6-g`, `_a6-h`, `_a6-p`, `_a6-o` **without** the
  section description paragraph `_a70f` that every other Facebook export page carries under its title; section
  pages share the card classes, so the cards alone are not a thread signature) and `html_document_root_v1`
  (DOCTYPE html or an html root, also XHTML behind an `<?xml?>` prolog, which used to be detected as bare xml).
  Verified on the 20 real files: 5 of 5 threads, 15 of 15 others.
- Both templates run inside the live pg_duckdb (DuckDB 1.4.3, webbed `d974555`), which lacks `read_html_blocks`; the
  templates therefore use `parse_html` + XPath and `html_to_duck_blocks`, which exist in both builds.
- The templates read the page whole in PostgreSQL's memory. Measured: a 62 MB page ran past 300 s and 960 MB, so
  the engine refuses an HTML page above 24 MB (`DUCKDB_HTML_MAX_BYTES`) with a permanent, explained failure instead of
  starving the database.
- A file that is not valid UTF-8 (a stray Windows-1252 byte inside a script) no longer fails: the template falls back to
  one character per byte.
- Python registry: seven tools `html.docling`, `html.unstructured`, `html.markitdown`, `html.html2text`,
  `html.beautifulsoup4`, `html.lxml`, `html.selectolax` (capability `extract.html_text`), each with an exact version pin
  and a per-file-type `primary` / `fallback` / `experimental` rank, each also a Temporal Activity on the
  `evidence-pipeline` queue (`extract_html_<tool>_activity`). The registry already carried this per-format rank
  (`quality`), so no new mechanism was needed. Proffer's handler selection is signature-based, one handler per
  signature, and has no fourth execution path for a Python tool; routing a Proffer run to one of these Python tools
  (rather than calling them directly or as Activities) is a design step that needs an owner decision and is not done.
- Worker image: `docling-slim[convert-core,format-html]` is Docling's HTML backend without torch, transformers or OCR
  (the full `docling` wheel adds about 900 MB); desktop environment sizes: docling-slim 242 MB against full docling 1,131
  MB, unstructured 286 MB. The temporal-worker image was 1.16 GB before; the measured size after the next Coolify
  build is recorded in the report that accompanies the deploy.
