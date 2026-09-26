// Server-only request authentication for the private advocatio API.
// Byline: Codex · GPT-5 · 2026-09-12

import { createHash, createHmac, randomBytes } from "node:crypto";

export type BffSignatureHeaders = {
  "x-legal-bff-timestamp": string;
  "x-legal-bff-nonce": string;
  "x-legal-bff-signature": string;
};

export function signBffRequest(method: string, target: string, body: Uint8Array): BffSignatureHeaders {
  const secret = process.env.LEGAL_BFF_SIGNING_SECRET ?? "";
  if (Buffer.byteLength(secret, "utf8") < 32) {
    throw new Error("LEGAL_BFF_SIGNING_SECRET must be at least 32 bytes");
  }
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const nonce = randomBytes(16).toString("hex");
  const bodyHash = createHash("sha256").update(body).digest("hex");
  const message = [timestamp, nonce, method.toUpperCase(), target, bodyHash].join("\n");
  const signature = createHmac("sha256", secret).update(message).digest("hex");
  return {
    "x-legal-bff-timestamp": timestamp,
    "x-legal-bff-nonce": nonce,
    "x-legal-bff-signature": signature,
  };
}
