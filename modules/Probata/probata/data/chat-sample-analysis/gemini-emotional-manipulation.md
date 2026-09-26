# ‎Gemini - Emotional Manipulation in Relationships.md
> _Byline: Claude Code · Sonnet · 2026-07-11_

## Snapshot
- **This file is mislabeled — filename and content do not match.** The on-disk filename (with the same invisible U+200E left-to-right mark prefix as the other three "Gemini" files) and the Obsidian front-matter `title:` field both claim "Gemini - Emotional Manipulation in Relationships," but the actual body content is a **FamilyTreeNow.com public-records genealogy search-results page** for an address in Clio, MI — a completely different clipped webpage with no relation to the stated title.
- Source tool (actual, from front-matter): Obsidian Web Clipper capture of `https://www.familytreenow.com/search/genealogy/results?streetaddress=2090%20W%20Lake%20Rd%2C&citystatezip=Clio,%20MI` — **not a Gemini conversation at all** (no `author: [[Gemini]]`, no "Created with Gemini Advanced" description — unlike the other three files).
- Size: 4.4 KB / 35 lines
- Format: standard Obsidian Web Clipper front-matter (title/source/author/created/description/tags/updated) wrapping a raw HTML→Markdown table dump of the clipped page.
- Era: front-matter `created: 2025-06-24` (three days after the other three Gemini files, which are all `2025-06-21` — consistent with this being a separate, later, unrelated clip that overwrote or was misnamed into this filename)

## Structure & format
- **Not the Gemini export format at all** — no conversational turns, no role delimiters, no user/assistant content of any kind. It is a static webpage clip: a mostly-empty markdown table (site layout whitespace) containing one populated row of search-result data, followed by the source site's footer navigation links (Home / My Tree / About Us / Terms / Privacy Notice / Contact Us / Join / Sign In / Do Not Sell or Share My Personal Information / Notice at Collection) and copyright line.
- **PARSING NOTE (critical, and the main finding for this file):** file identity (filename + front-matter `title:`) does **not** guarantee content matches. This is a concrete, evidenced instance of that failure mode inside the sample set itself — a parser/ingest pipeline must verify body content against filename/title claims (e.g. content-type sniffing: does the body look like a Gemini share page, or like a generic Web Clipper capture of some other site?) rather than trusting either metadata field. The real "Emotional Manipulation in Relationships" Gemini conversation implied by this filename is **absent from this sample set** — it was likely overwritten, misnamed, or mis-clipped during export.

## Section-by-section breakdown (in order)
1. **Front-matter block** — carries the misleading title but the correct (and different) `source` URL, which is the tell that content ≠ title.
2. **Search-result table** — one populated row: Name "MindySheldon" (likely "Mindy Sheldon" with a lost space, a clipper rendering artifact), Born "Unknown," Age blank, Lives in "Clio, MI." Surrounding table cells are empty (page-layout whitespace captured by the clipper).
3. **Site footer/navigation chrome** — generic FamilyTreeNow.com links and copyright notice, no case-relevant content.

## Facets present (extraction-lane map)
| facet | present? | examples / notes |
|---|---|---|
| identity(who) | **yes** | A named third party: "Mindy Sheldon" (as rendered; likely a genealogy-site record for a person named Mindy Sheldon) |
| entities | **yes** | A real street address: "2090 W Lake Rd, Clio, MI" — same town (Clio, MI) as the residence addresses appearing in `gemini-timeline-analysis.md` |
| relationships | unknown | No relationship context given in the clip itself — this file alone doesn't establish how "Mindy Sheldon" relates to the case |
| timeline/events | none | — |
| life-history | none | — |
| legal-strategy | none | — |
| legal-artifacts | none | — |
| mood/sentiment (owner-self, low-pri) | none | — |
| psychiatric | none | — |
| code/app-dev/plans | none | — |
| work | none | — |

## Notable content
- **Real investigative lead**: this clip evidences the owner running a public-records/genealogy lookup on a person named "Mindy Sheldon" at an address in Clio, MI — the same town as the two residence addresses that appear in `gemini-timeline-analysis.md`. Worth surfacing as an entity-pointer for follow-up (who is this person, and what is their relationship to the case?), even though this file carries no AI-chat content to explain the "why."
- **Corpus-integrity finding**: this is direct, in-sample evidence that filename/title metadata in this chat-sample set cannot be trusted at face value — at least one file is a title/content mismatch. Any future ingest pipeline needs a content-verification step, not filename-based routing, and the cross-file synthesis / INDEX should flag that the true "Emotional Manipulation in Relationships" Gemini conversation is missing from the sample and may need to be re-sourced from the owner's Gemini history if it still exists.

## Sensitivity
LABEL, do not redact: contains a real third-party name ("Mindy Sheldon," public-records sourced) and a real street address (both moderate-sensitivity PII, sourced from a public people-search site rather than the owner's own conversation). No abuse, psychiatric, or emotional-manipulation content is actually present in this file despite its title — flag the title/content mismatch itself prominently so downstream consumers don't misclassify this file's sensitivity based on its filename alone.
