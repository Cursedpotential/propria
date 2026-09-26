// Byline: Grok · grok-4.6 · 2026-08-18
// Vercel AI SDK lives here (Next). Domain mutations stay on Python legal-api.
// Paths come from GET /v1/routing so a framework swap is a JSON edit, not a rewrite.

import { legalApiBase } from "@/lib/api/client";

export const runtime = "nodejs";

type Routing = {
  chat: {
    run_path: string;
    confidential_path: string;
    default_confidential_model: string;
  };
};

const FALLBACK: Routing = {
  chat: {
    run_path: "/v1/agent-runs",
    confidential_path: "/v1/gateway:invoke",
    default_confidential_model: "ollama-cloud",
  },
};

async function loadRouting(): Promise<Routing> {
  try {
    const response = await fetch(`${legalApiBase()}/v1/routing`, { cache: "no-store" });
    if (!response.ok) return FALLBACK;
    const body = (await response.json()) as Routing;
    if (!body.chat?.run_path || !body.chat?.confidential_path) return FALLBACK;
    return body;
  } catch {
    return FALLBACK;
  }
}

export async function POST(request: Request) {
  const body = (await request.json()) as {
    messages?: Array<{ role: string; content: string }>;
    prompt?: string;
    confidential?: boolean;
    model?: string;
    surface?: { path?: string; label?: string; help?: string };
    live?: { path?: string; selection?: string; fields?: Array<{ label: string; value: string }> };
  };
  const last =
    body.prompt ??
    [...(body.messages ?? [])].reverse().find((item) => item.role === "user")?.content ??
    "";
  if (!last.trim()) {
    return Response.json({ error: "empty prompt" }, { status: 400 });
  }

  let saved = "";
  try {
    const path = encodeURIComponent(body.surface?.path || "/");
    const snap = await fetch(`${legalApiBase()}/v1/surface-context?path=${path}`, {
      cache: "no-store",
    });
    if (snap.ok) saved = JSON.stringify(await snap.json());
  } catch {
    saved = "";
  }

  const liveBits: string[] = [];
  if (body.live?.selection) liveBits.push(`Selected text:\n${body.live.selection}`);
  for (const field of body.live?.fields ?? []) {
    liveBits.push(`${field.label}:\n${field.value}`);
  }
  const liveBlock = liveBits.length
    ? `Unsaved on-screen work (not in the database yet):\n${liveBits.join("\n\n")}`
    : "No unsaved on-screen fields were sent.";

  const surfaceLabel = body.surface?.label?.trim() || "current page";
  const path = body.surface?.path || "/";
  let background = "";
  if (saved) {
    try {
      const snap = JSON.parse(saved) as { background?: string };
      background = snap.background?.trim() ?? "";
    } catch {
      background = "";
    }
  }
  if (!background && path.startsWith("/custody-factors")) {
    background =
      "Both-parent rule — assistant only. Address petitioner and respondent. Conduct, not diagnoses. Do not invent facts. Not a finding. Weighting is the owner's.";
  }
  const grounded = [
    `Looking at: ${surfaceLabel}${body.surface?.path ? ` (${body.surface.path})` : ""}`,
    body.surface?.help ?? "",
    background ? `Background (do not repeat to the owner):\n${background}` : "",
    liveBlock,
    saved ? `Saved workspace snapshot:\n${saved}` : "No saved snapshot attached.",
    last,
  ]
    .filter(Boolean)
    .join("\n\n");

  const routing = await loadRouting();

  if (body.confidential) {
    const model = body.model ?? routing.chat.default_confidential_model;
    const response = await fetch(`${legalApiBase()}${routing.chat.confidential_path}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ prompt: grounded, model, confidential: true }),
    });
    const payload = (await response.json().catch(() => ({}))) as {
      text?: string;
      detail?: string;
      model?: string;
    };
    if (!response.ok) {
      return Response.json({
        id: crypto.randomUUID(),
        role: "assistant",
        content:
          `[confidential · ${response.status} · ${payload.detail ?? "blocked"}]\n` +
          "No silent fallback to Claude/ChatGPT consumer. Not court-safe. " +
          "Not a privilege legal conclusion.",
      });
    }
    return Response.json({
      id: crypto.randomUUID(),
      role: "assistant",
      content: `[confidential · ${payload.model ?? model}]\n${payload.text ?? "no output"}`,
    });
  }

  const response = await fetch(`${legalApiBase()}${routing.chat.run_path}`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      intent: grounded,
      prompt: grounded,
      requested_model: "unevaluated-manual",
    }),
  });
  if (!response.ok) {
    return Response.json({ error: `legal-api ${response.status}` }, { status: 502 });
  }
  const run = (await response.json()) as { output: string; status: string; role: string };
  return Response.json({
    id: crypto.randomUUID(),
    role: "assistant",
    content: `[${run.role} · ${run.status}]\n${run.output}`,
  });
}
