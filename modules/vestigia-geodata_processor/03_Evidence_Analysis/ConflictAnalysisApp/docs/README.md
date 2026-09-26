# ConflictAnalysisApp — Conversational Conflict Analysis Toolkit

A desktop toolkit to parse message exports (PDF / HTML / XML), tag manipulative/abusive patterns,
and detect **multi-turn sequences** (e.g., reactive abuse cycles), built for high-conflict co‑parenting evidence work.

## Features

- Import **files or folders** (recursive) of exports: `.pdf`, `.html/.htm`, `.xml` (SMS Backup & Restore).
- Parse, **dedupe**, and **merge** into one timeline.
- **Rule tagging** from modular YAML packs (behaviors, MCL 722.23 factors, entities, spelling variants).
- **Multi-file rule auto-merge**: load every `*.yaml` in the rules folder.
- **Hot reload**: editing any YAML file re-applies rules without restarting the app.
- **Sequence detection** (from `rules/sequences.yaml`): reactive abuse cycles, stonewalling windows, alcohol+child risk proximity.
- **Preview & tables**: sortable message table and detected-sequences table.
- **Export**: master timeline, tagged, signal-only, and sequences JSON.

## Quick Start

```bash
# 1) (Optional) Create a virtual env
python -m venv .venv && source .venv/bin/activate   # Windows: .venv\Scripts\activate

# 2) Install dependencies
pip install -r requirements.txt

# 3) Run
python app.py
```

## Workflow

1. Click **Add Files…** or **Add Folder…** (toggle **Recursive** as needed).
2. Load your **rules directory** (default rules in `rules/` are included).
3. Set date range and PDF page limit (0 = parse all pages).
4. Click **Scan** → messages appear with tags.
5. Click **Detect Sequences** → sequence table populates.
6. Click **Export…** → CSVs and sequences JSON are written.

## Rule Packs (YAML)

- `rules/behaviors.yaml` — manipulations & abuse taxonomies (gaslighting, coercive control, alienation, sexual weaponization, social media deception, etc.).
- `rules/mcl.yaml` — MCL 722.23 factor indicators (`b, d, f, j, k`).
- `rules/entities.yaml` — people, places (e.g., **Huckleberry Junction/Huck**), platforms (Snapchat), substances (e.g., **Fireball**, Adderall).
- `rules/spelling_variants.yaml` — canonical → misspellings (e.g., `faggot`: `fagot`, `fagget`, `faggit`).
- `rules/sequences.yaml` — multi-message patterns (reactive‑abuse cycle, stonewalling window, alcohol+child risk window).
- `rules/severity.yaml` — default category weights.
- `rules/stopwords_custom.yaml` — optional de-noising terms.

> **Sensitive language disclaimer:** The rule sets include abusive, slur, and sexual terms strictly for **detection** and evidentiary tagging.

## Exports

- `master_messages.csv` — `datetime, sender, message, source`.
- `tagged_messages.csv` — above + `behavior_tags, mcl_factors` (if enabled).
- `signal_messages.csv` — subset where any tag present.
- `sequences.json` — detected multi-turn sequences with pointers to message rows.

## Privacy

No network calls. All processing is local. Your data never leaves your machine.


## Low-memory streaming (massive XML)

- Enable **Stream (low memory)** to parse **huge SMS XML** via `iterparse` without loading it into RAM.
- Use **Max rows in memory** to limit how many parsed rows are kept for on-screen preview (everything else can be exported).
- PDF parsing is page-by-page already; HTML is read whole-file, but you can split large exports beforehand or run the CLI.

## Party normalization & notes

- Add aliases/notes in `rules/parties.yaml`. The app normalizes the `sender` to `role` (e.g., `ParentA`, `ParentB`, `Child`) and lets you view party notes.
