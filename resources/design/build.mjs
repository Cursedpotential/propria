// Propria deterministic CSS token generator | contract 1.0.0 | 2026-09-12
// Byline: Codex · GPT-5
import assert from "node:assert/strict";
import { readFile, writeFile } from "node:fs/promises";

const root = new URL("./", import.meta.url);
const tokenUrl = new URL("tokens.json", root);
const cssUrl = new URL("tokens.css", root);
const contract = JSON.parse(await readFile(tokenUrl, "utf8"));
const checkOnly = process.argv.includes("--check");

function kebab(value) {
  return value.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`);
}

function fontStack(values) {
  return values.map((value) => value.includes(" ") ? `"${value}"` : value).join(", ");
}

function themeLines(themeName, includeFoundation) {
  const theme = contract.themes[themeName];
  const lines = Object.entries(theme).map(([name, value]) => `  --pr-${kebab(name)}: ${value.toLowerCase()};`);
  if (includeFoundation) {
    lines.push(
      `  --pr-font-ui: ${fontStack(contract.typography.ui)};`,
      `  --pr-font-data: ${fontStack(contract.typography.data)};`,
      `  --pr-font-document: ${fontStack(contract.typography.document)};`,
      `  --pr-radius-control: ${contract.geometry.radiusPx.control}px;`,
      `  --pr-radius-panel: ${contract.geometry.radiusPx.panel}px;`,
      `  --pr-radius-dialog: ${contract.geometry.radiusPx.dialog}px;`,
    );
  }
  const effects = contract.effects[themeName];
  lines.push(
    `  --pr-shadow-panel: ${effects.shadowPanel};`,
    `  --pr-shadow-dialog: ${effects.shadowDialog};`,
  );
  return lines.join("\n");
}

function tierLines(tierName) {
  const tier = contract.experienceTiers[tierName];
  return [
    `  --pr-layout-gap: ${tier.layoutGapPx}px;`,
    `  --pr-panel-padding: ${tier.panelPaddingPx}px;`,
    `  --pr-data-row-height: ${tier.dataRowHeightPx}px;`,
    `  --pr-control-min-height: ${tier.controlMinHeightPx}px;`,
    `  --pr-context-strip-min-height: ${tier.contextStripMinHeightPx}px;`,
    `  --pr-label-size: ${tier.labelSizePx}px;`,
  ].join("\n");
}

const output = `/* Propria shared semantic tokens | contract ${contract.version} | ${contract.updated} | Byline: Codex · GPT-5
   GENERATED FROM tokens.json. Do not hand-edit this file. */

:root,
[data-pr-theme="light"],
:root.theme-light {
  color-scheme: light;
${themeLines("light", true)}
}

[data-pr-theme="dark"],
:root.theme-dark,
html.dark {
  color-scheme: dark;
${themeLines("dark", false)}
}

/* Experience tier is independent of light/dark theme. These values tune density and composition,
   never authority. General mirrors the Evidence Operations Desk; Advanced mirrors the Modular
   Service Cockpit. */
:root,
[data-pr-experience="general"] {
${tierLines("general")}
}

[data-pr-experience="advanced"] {
${tierLines("advanced")}
}
`;

if (checkOnly) {
  const current = await readFile(cssUrl, "utf8");
  assert.equal(current, output, "tokens.css drifted from tokens.json; run npm run build:tokens");
  console.log(`Verified generated tokens.css is current for ${contract.version}.`);
} else {
  await writeFile(cssUrl, output, "utf8");
  console.log(`Generated tokens.css from tokens.json for ${contract.version}.`);
}
