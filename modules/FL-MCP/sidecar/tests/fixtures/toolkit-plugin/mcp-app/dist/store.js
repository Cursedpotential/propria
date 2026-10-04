// Byline: Codex · GPT-6 · 2026-10-04

/** Marks this inert fixture so the config test can verify module discovery. */
export const configFixture = true;

/**
 * Fails if a config test attempts to open a store connection.
 * Input: none. Output: a rejected promise. Side effects: none. Use only as the
 * inert fixture instead of a real toolkit database client.
 */
export async function getStore() {
  throw new Error("the config fixture must not connect to a store");
}
