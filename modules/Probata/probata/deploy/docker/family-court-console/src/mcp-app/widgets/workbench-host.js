// Byline: OpenAI Codex · GPT-5 · 2026-09-12
// Shared MCP host-context bridge. The embedding host, not the OS media query,
// is authoritative for theme and supplied design tokens.
const { applyDocumentTheme, applyHostStyleVariables, applyHostFonts } = globalThis.ExtApps;

const applyWorkbenchHostContext = (context = {}) => {
  if (context.theme) applyDocumentTheme(context.theme);
  if (context.styles?.variables) applyHostStyleVariables(context.styles.variables);
  if (context.styles?.css?.fonts) applyHostFonts(context.styles.css.fonts);
};

const connectWorkbenchApp = async (app) => {
  app.onhostcontextchanged = applyWorkbenchHostContext;
  await app.connect();
  applyWorkbenchHostContext(app.getHostContext?.() ?? {});
};
