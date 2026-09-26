# Vendored Propria design contract

> _Byline: Codex · GPT-5 · 2026-09-12._

- Upstream repository: `E:/AI_Workspace/Projects/Propria`
- Upstream commit: `c6da1419e2baaea0c481047083f178d46ac14ca7`
- Upstream tree: `resources/design` at `9dcb135efedfb05b423e8e7e4b64529b09b556a9`
- Contract version: `1.0.0`
- Adoption lane: `SDA-03 advocatio Legal Workdesk`

The `propria-design-contract/` directory was originally copied from the upstream tree at the pinned commit.
Its palette and generated CSS were synchronized with Propria design-contract on 2026-09-24
using Probata application colors, at the owner's direction. The original palette is superseded. Runtime and build imports resolve only inside this advocatio repository; they do
not reach across repositories into Propria.

To verify the vendored contract itself:

```powershell
Set-Location web/vendor/propria-design-contract
npm.cmd run verify
```

The contract verifier proves token generation parity, adapters, required semantic roles, CSS policy,
and core contrast pairs. It does not replace browser, keyboard, zoom, forced-colors, or legal-state
verification in the consuming Workdesk.
