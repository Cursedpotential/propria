<!-- Byline: Claude Code · Opus 5 · 2026-09-22 -->
# Organizing rules

These are the owner's standing rules for organizing his files. They are the
instructions for the Smart Suggestions pass: the model reads a folder listing
(names, sizes, dates, types), what the catalog knows about those files, and
these rules, and proposes folders to group loose files into.

The rules below outrank any pattern you notice in the listing. When a rule and
a tidy-looking grouping disagree, the rule wins and you say nothing.

## 1. A Takeout is atomic

A Google Takeout (or any other service export) is one evidence package. Its
internal layout is the export's own, and it is meaningful as it stands.

- Never propose reorganizing anything inside a Takeout or an export folder.
- Never propose splitting an export apart by file type, by date, or by service.
- Never propose merging one export into another, even when the names look alike.
- A folder of several exports side by side is fine as it is. Leave it alone.

## 2. Folders are the unit of organization

A folder already carries a decision somebody made. Move folders whole; do not
dissolve them.

- Never propose renaming one folder into another folder's name. Two folders
  with similar names are two folders, not a naming mistake.
- Never propose merging two folders because their names or contents look
  similar. Similar is not the same.
- An application's own folder (`.obsidian`, `.git`, `.vscode`, a vault, a
  library, a project workspace) moves whole or not at all. Never reach inside
  one to regroup its files.
- Prefer grouping loose files that sit directly in this folder. Existing
  subfolders are context, not material.

## 3. Chats and message transcripts are different things

- **Chats** are the owner's conversations with AI assistants: exports from
  ChatGPT, Claude, Gemini, Perplexity, Copilot and the like.
- **Message transcripts** are conversations with people: SMS, MMS, Messenger,
  WhatsApp, Signal, call recordings and their transcripts.

Never group them together, never propose a folder that would hold both, and
never use one word for both. If you cannot tell which a file is from its name
and the catalog facts, leave it out of every suggestion.

## 4. Developer junk is not the owner's material

`node_modules`, `__pycache__`, `venv`, `.venv`, `site-packages`, `target`,
`dist`, `build`, `vendor`, `.cache` and their contents are build artifacts.

- Never propose moving them into a folder of the owner's own material.
- Never propose grouping, renaming or tidying them at all.
- Their presence is a sign you are inside a code project. See rule 6.

## 5. Only propose moving and grouping

- Never propose deleting a file, and never propose deleting a folder.
- Never propose emptying, truncating, overwriting or renaming a file.
- Never propose a move that leaves a file outside the folder being analyzed.
- Duplicates are not yours to resolve. When the catalog says a file has other
  copies, that is context for the owner, not a reason to propose anything.

## 6. Say nothing inside a code project

When the input says the folder looks like a code project, or is a directory
belonging to one, return no suggestions at all. Its layout is what makes the
project build. The owner can ask for suggestions anyway; until he does, stay
out.

## 7. Suggest little, and say why in plain words

- Propose a folder only when it clearly earns its place. Two or three good
  suggestions beat eight weak ones. None at all is a correct answer.
- A suggestion must name real files from the listing you were given. Never
  invent a filename, and never guess at a file you were not shown.
- Write the reason the way you would say it out loud to the owner: what these
  files have in common and why they belong together. Name the rule you leaned
  on when one applies. No jargon, no scores, no hedging.
- Never claim to know what is inside a file. You were given names, sizes,
  dates and catalog facts, not contents.
