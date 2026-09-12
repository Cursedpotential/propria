# Propria design-contract package

> _Byline: Codex · GPT-5 · 2026-09-12._

This directory is the portable styling contract for Propria's owner-facing surfaces. Version 1.0.0
keeps the accepted Carbon-Linen-Seal palette in one place and preserves Probata's owner-approved
two-surface structure:

- **General:** Evidence Operations Desk — everyday evidence work.
- **Advanced:** Modular Service Cockpit — deep forensic and governed service tooling.

Theme and experience are separate. Any product can use light or dark tokens. Only Probata defines the
General/Advanced Workbench switch; Intake, the portal and advocatio Legal Workdesk are General-first.

## One edit point

`tokens.json` is the single editable source for color, typography, effects, geometry and the
General/Advanced density values. Do not hand-edit the generated `tokens.css`. After changing JSON,
run:

```powershell
npm.cmd run build:tokens
npm.cmd run verify
```

For CI or a read-only drift check, run only:

```powershell
npm.cmd run verify
```

Verification first regenerates the expected CSS in memory and fails if the checked-in file differs.
It then checks package/token version parity, exact CSS/JSON color and density parity, required
adapters, forbidden `!important`, sample integration, and core contrast pairs. A successful run does
not replace browser, keyboard, zoom, high-contrast, or live-route verification.

## Consumption order

Load the files in this order:

```css
@import "@propria/design-contract/tokens.css";
@import "@propria/design-contract/theme.css";
@import "@propria/design-contract/adapters/probata.css";
/* Product-local component overrides follow. */
```

Until a private package pipeline exists, each independent repository should vendor a pinned copy of
the release rather than importing through a relative path into `Propria`. That preserves independent
builds and release cycles. The adapter files are migration maps; adding this directory does not claim
that any product has adopted them.

Set theme and experience independently on the application root:

```html
<html data-pr-theme="dark">
  <body data-pr-experience="advanced"></body>
</html>
```

Existing `html.dark`, `:root.theme-dark`, and `:root.theme-light` selectors remain compatible.

## Adapter intent

| Adapter | Maps | Boundary |
|---|---|---|
| `intake.css` | Intake's workspace, surface, shell, selection and status variables | Does not replace Explorer, preview, metadata or chat components |
| `probata.css` | shadcn/Tailwind-style roles plus General/Advanced density variables | Does not expose gated Advanced destinations or change evidence authority |
| `advocatio.css` | Legal Workdesk shell, status, confidential and document-font aliases | Does not make advocatio the Probata Advanced surface |
| `portal.css` | Portal/progress-board shell, paper, status and chart aliases | Does not turn the portal into an application or merge status sources |

See [CALLABILITY.md](CALLABILITY.md) before exposing a route or legal tool. Shared styling is not proof
that a cross-product launch, authentication exchange, backend connection or mutation path exists.

## Versioning

- **Patch:** tune a value while preserving semantic meaning and adapter shape.
- **Minor:** add a compatible semantic role or adapter alias.
- **Major:** rename/remove a role, change state meaning, or break an adapter contract.

Do not use `!important`. Product-local overrides load after the adapter. Status must include readable
text or an icon; color alone never communicates authority, completion, failure, or staleness.

## Visual reference

Open `examples/probata-two-surface-sample.html` to see the current palette applied to a compact
structural sample of the approved Evidence Operations Desk and Modular Service Cockpit layouts. The
original Probata mockups remain the structural authority; the sample is a token-integration aid.
