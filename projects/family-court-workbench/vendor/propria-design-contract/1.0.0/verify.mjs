// Propria design-contract verification | contract 1.0.0 | 2026-09-12
// Byline: Codex · GPT-5
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

const root = new URL("./", import.meta.url);
const contract = JSON.parse(await readFile(new URL("tokens.json", root), "utf8"));
const packageContract = JSON.parse(await readFile(new URL("package.json", root), "utf8"));
const css = await readFile(new URL("tokens.css", root), "utf8");
const themeCss = await readFile(new URL("theme.css", root), "utf8");
const sampleHtml = await readFile(new URL("examples/probata-two-surface-sample.html", root), "utf8");

const requiredRoles = [
  "canvas", "surface", "surfaceMuted", "ink", "inkMuted", "border", "borderStrong",
  "shell", "shellSurface", "shellText", "action", "actionText", "actionHover",
  "actionSoft", "focus", "positive", "caution", "destructive", "information",
];
const adapters = ["intake", "probata", "advocatio", "portal"];

function cssName(role) {
  return role.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function selectorBlock(source, selector) {
  const selectorAt = source.indexOf(selector);
  assert.notEqual(selectorAt, -1, `Missing selector ${selector}`);
  const openAt = source.indexOf("{", selectorAt);
  let depth = 0;
  for (let index = openAt; index < source.length; index += 1) {
    if (source[index] === "{") depth += 1;
    if (source[index] === "}") depth -= 1;
    if (depth === 0) return source.slice(openAt + 1, index);
  }
  assert.fail(`Unclosed selector ${selector}`);
}

function cssValue(block, customProperty) {
  const match = block.match(new RegExp(`${escapeRegExp(customProperty)}\\s*:\\s*([^;]+);`, "i"));
  assert.ok(match, `Missing ${customProperty}`);
  return match[1].trim();
}

function luminance(hex) {
  const channels = hex.slice(1).match(/.{2}/g).map((value) => Number.parseInt(value, 16) / 255);
  const linear = channels.map((value) =>
    value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4,
  );
  return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2];
}

function contrast(a, b) {
  const [bright, dark] = [luminance(a), luminance(b)].sort((left, right) => right - left);
  return (bright + 0.05) / (dark + 0.05);
}

assert.equal(packageContract.version, contract.version, "package.json and tokens.json versions differ");
assert.equal(contract.status, "accepted_for_initial_adoption", "Unexpected contract status");

for (const themeName of ["light", "dark"]) {
  const theme = contract.themes[themeName];
  assert.ok(theme, `Missing ${themeName} theme`);
  const block = selectorBlock(css, `[data-pr-theme="${themeName}"]`);

  for (const role of requiredRoles) {
    const jsonColor = theme[role];
    assert.match(jsonColor ?? "", /^#[0-9A-F]{6}$/i, `${themeName}.${role} must be a hex color`);
    const cssColor = cssValue(block, `--pr-${cssName(role)}`);
    assert.equal(cssColor.toUpperCase(), jsonColor.toUpperCase(), `${themeName}.${role} CSS/JSON drift`);
  }

  for (const [foreground, background] of [
    ["ink", "canvas"], ["ink", "surface"], ["inkMuted", "surface"],
    ["shellText", "shell"], ["actionText", "action"],
  ]) {
    const ratio = contrast(theme[foreground], theme[background]);
    assert.ok(ratio >= 4.5, `${themeName} ${foreground}/${background} is ${ratio.toFixed(2)}, below 4.5:1`);
  }

  const focusRatio = contrast(theme.focus, theme.surface);
  assert.ok(focusRatio >= 3, `${themeName} focus/surface is ${focusRatio.toFixed(2)}, below 3:1`);
}

const tierPropertyMap = {
  layoutGapPx: "--pr-layout-gap",
  panelPaddingPx: "--pr-panel-padding",
  dataRowHeightPx: "--pr-data-row-height",
  controlMinHeightPx: "--pr-control-min-height",
  contextStripMinHeightPx: "--pr-context-strip-min-height",
  labelSizePx: "--pr-label-size",
};

for (const tierName of ["general", "advanced"]) {
  const tier = contract.experienceTiers[tierName];
  assert.ok(tier, `Missing ${tierName} experience tier`);
  const block = selectorBlock(css, `[data-pr-experience="${tierName}"]`);
  for (const [jsonName, customProperty] of Object.entries(tierPropertyMap)) {
    assert.equal(cssValue(block, customProperty), `${tier[jsonName]}px`, `${tierName}.${jsonName} CSS/JSON drift`);
  }
}

const cssFiles = [{ name: "tokens.css", value: css }, { name: "theme.css", value: themeCss }];
for (const adapter of adapters) {
  const adapterCss = await readFile(new URL(`adapters/${adapter}.css`, root), "utf8");
  assert.match(adapterCss, /var\(--pr-/, `${adapter}.css does not map to Propria tokens`);
  cssFiles.push({ name: `adapters/${adapter}.css`, value: adapterCss });
}
for (const file of cssFiles) {
  assert.doesNotMatch(file.value, /!important/i, `${file.name} must not use !important`);
}

assert.match(themeCss, /prefers-reduced-motion/, "theme.css must handle reduced motion");
assert.match(themeCss, /forced-colors/, "theme.css must handle forced colors");
assert.match(sampleHtml, /\.\.\/tokens\.css/, "Sample must consume the shared token file");
assert.match(sampleHtml, /data-pr-experience="general"/, "Sample must show the General surface");
assert.match(sampleHtml, /data-pr-experience="advanced"/, "Sample must show the Advanced surface");
assert.match(sampleHtml, /ADVOCATIO/, "Sample must show the aligned legal surface");
assert.doesNotMatch(sampleHtml, /#[0-9a-f]{3,8}\b/i, "Sample must not fork the shared color palette");

console.log(
  `Verified Propria design contract ${contract.version}: exact token/density parity, adapters, sample integration, CSS policy, and core contrast pairs pass.`,
);
