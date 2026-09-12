# Propria design contract pin

> _Byline: OpenAI Codex · GPT-5 · 2026-09-12._

- Contract: `propria-surface-design`
- Version: `1.0.0`
- Source repository: `E:/AI_Workspace/Projects/Propria`
- Source commit: `c6da141`
- Source subtree: `resources/design`
- Source subtree tree ID: `9dcb135efedfb05b423e8e7e4b64529b09b556a9`
- Adoption lane: `SDA-04`

The sibling `1.0.0/` directory is a byte-for-byte vendored snapshot of the source subtree at the
commit above. It is deliberately repository-owned; the application must not import the Propria
router's live `resources/design` directory at runtime.

`1.0.0/tokens.json` is the single editable token source inside this pinned copy. Regenerate
`tokens.css` with `npm.cmd run design:build` from the Family Court Workbench root. Verify drift,
contract parity, adapters, sample integration, CSS policy, and core contrast with
`npm.cmd run design:verify`. Never hand-edit generated `tokens.css`.
