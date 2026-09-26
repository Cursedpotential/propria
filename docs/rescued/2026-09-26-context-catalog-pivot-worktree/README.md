# Rescued files — orphaned `context-catalog-pivot` worktree (2026-09-26)

> _Byline: Claude Code · Opus 5.5 · 2026-09-26_

## Where these came from, and why they're parked here

The folder `modules/Probata/probata-worktrees/context-catalog-pivot` was a linked
git worktree whose `.git` file pointed at
`E:/AI_Workspace/Projects/the-platform-workspace/probata/.git/worktrees/context-catalog-pivot`
— a retired repository path that no longer exists, so no git command could run
inside that folder and it no longer appeared in this repository's
`git worktree list`. Its branch, `codex/context-catalog-pivot` (tip `38a3ea3`),
was already fully merged into `main` (verified with
`git merge-base --is-ancestor`). The open question was narrower than "is the
branch merged": which *file contents physically sitting in that folder* do not
exist anywhere in this repository's object database — i.e. genuinely unrecorded
work, as opposed to committed content the folder simply still happened to hold
a working copy of.

To answer that, every tracked path at the branch tip plus every untracked,
non-ignored path on disk (7,007 files that actually existed on disk; one
tracked path, `modules/forks/sbv`, is a submodule gitlink with no file content
to compare) was hashed exactly as `git hash-object` would hash it if it were
being added from that location right now — filters on (`core.autocrlf=true`
from this repository's config, combined with the orphaned worktree's own
`.gitattributes`) — and checked against this repository's object database with
`git cat-file --batch-check`. Files whose filtered hash still came back missing
were additionally checked against a manually LF-normalized rehash, to catch any
case where the filter pass didn't reproduce the exact historical normalization.
An earlier pass (`git hash-object --no-filters`) had flagged 472 files as
missing, but that was a false-positive storm from comparing raw CRLF-checkout
bytes against LF-normalized blobs under `core.autocrlf=true`; redoing it with
filters on brought the real gap down to the 8 files listed below.

All 8 were scanned for secret-shaped content (API keys, tokens, passwords,
private-key blocks) before being copied. None matched — the `api_key` fields in
the Portkey gateway configs are `$OPENROUTER_API_KEY` / `$NVIDIA_API_KEY` /
`$HF_TOKEN` environment-variable placeholders, and the test fixtures use literal
placeholder strings (`test-provider-key`, `test-portkey-key`). **No files were
withheld.**

These 8 files are parked here byte-for-byte under their original relative path,
not restored to their live path: `main` has moved 497 commits past this
branch's fork point, live paths such as `deploy/` are deployed in place by
Coolify from the current tree, and none of these 8 paths exist on `main` today
(see table). Restoring them live without review could silently reintroduce a
two-week-old, never-reviewed slice of a Portkey gateway / multimodal-context
feature into paths other work now occupies or depends on. Parking them here
preserves the content, with provenance, for a human or a later task to
deliberately integrate.

## Files

| Original path | Bytes | On `main` today? | `main`'s last commit touching this path |
|---|---|---|---|
| `deploy/docker/gateway/portkey/configs/context-omni-extract.json` | 1,577 | No | none |
| `deploy/docker/gateway/portkey/configs/context-vl-embed.json` | 1,593 | No | none |
| `deploy/docker/gateway/portkey/configs/context-vl-rerank.json` | 1,709 | No | none |
| `deploy/docker/gateway/portkey/configs/document-extract-granite.json` | 580 | No | none |
| `docs/handoffs/HANDOFF-2026-09-09-context-catalog-pivot.md` | 4,672 | No | none |
| `server/core/multimodal_gateway.py` | 16,897 | No | none |
| `tests/test_context_multimodal_gateway.py` | 6,526 | No | none |
| `tests/test_context_portkey_configs.py` | 2,036 | No | none |

"On `main` today?" and "last commit touching this path" were checked against
this repository's `main` branch history (`git log -1 -- <path>`); none of these
8 paths have ever been touched by a commit reachable from `main`.

## Source and disposition

- Source folder: `E:/AI_Workspace/Projects/Propria/modules/Probata/probata-worktrees/context-catalog-pivot`
- Source branch (fully merged into `main`, tip `38a3ea3`): `codex/context-catalog-pivot`
- The source folder's own handoff doc (rescued alongside it, above) is
  `docs/handoffs/HANDOFF-2026-09-09-context-catalog-pivot.md`, dated 2026-09-09,
  byline Codex · GPT-5, describing the in-flight design at the time this
  worktree went orphaned.
- After this rescue commit merges into `main`, the source folder is moved
  (never deleted) to
  `E:/AI_Workspace/Projects/Propria/to_be_deleted/2026-09-26-probata-orphan-worktree-context-catalog-pivot/`.
