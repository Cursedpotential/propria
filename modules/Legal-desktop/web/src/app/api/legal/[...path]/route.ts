// Same-origin browser/server adapter for the private advocatio API.
// Byline: Codex · GPT-5 · 2026-09-12

import { signBffRequest } from "@/lib/auth/bff-signature";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ path?: string[] }> };

const REQUEST_HEADERS = ["accept", "content-type", "if-match", "if-none-match"];
const RESPONSE_HEADERS = [
  "content-disposition",
  "content-length",
  "content-type",
  "etag",
  "last-modified",
];

function normalizedHost(value: string | null): string {
  return (value ?? "").trim().toLowerCase().replace(/:\d+$/, "");
}

function isPublicIngress(request: Request): boolean {
  const configured = normalizedHost(process.env.LEGAL_PUBLIC_HOST ?? null);
  return Boolean(configured) && normalizedHost(request.headers.get("host")) === configured;
}

function upstreamBase(): string {
  const configured = process.env.LEGAL_API_INTERNAL_URL ?? "http://127.0.0.1:8010";
  const url = new URL(configured);
  if (!['http:', 'https:'].includes(url.protocol)) {
    throw new Error("LEGAL_API_INTERNAL_URL must use http or https");
  }
  return url.toString().replace(/\/$/, "");
}

async function forward(request: Request, context: RouteContext): Promise<Response> {
  const segments = (await context.params).path ?? [];
  if (segments.some((segment) => segment === "." || segment === "..")) {
    return Response.json({ detail: "invalid API path" }, { status: 400 });
  }
  const pathname = `/${segments.map(encodeURIComponent).join("/")}`;
  const incoming = new URL(request.url);
  const target = `${pathname}${incoming.search}`;
  const upstream = `${upstreamBase()}${target}`;
  const body = request.method === "GET" || request.method === "HEAD"
    ? new Uint8Array()
    : new Uint8Array(await request.arrayBuffer());

  const authentikJwt = request.headers.get("x-authentik-jwt")?.trim() ?? "";
  if (isPublicIngress(request) && !authentikJwt) {
    return Response.json({ detail: "Authentik identity required" }, { status: 401 });
  }

  const headers = new Headers();
  for (const name of REQUEST_HEADERS) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  if (authentikJwt) {
    // The private API performs cryptographic OIDC validation. This bearer never
    // appears in browser code, storage or a response.
    headers.set("authorization", `Bearer ${authentikJwt}`);
  } else {
    const signature = signBffRequest(request.method, target, body);
    for (const [name, value] of Object.entries(signature)) headers.set(name, value);
  }

  let response: Response;
  try {
    response = await fetch(upstream, {
      method: request.method,
      headers,
      body: body.byteLength ? body : undefined,
      cache: "no-store",
      redirect: "manual",
    });
  } catch {
    return Response.json({ detail: "legal-api unavailable" }, { status: 502 });
  }

  const outgoing = new Headers({ "cache-control": "no-store" });
  for (const name of RESPONSE_HEADERS) {
    const value = response.headers.get(name);
    if (value) outgoing.set(name, value);
  }
  return new Response(request.method === "HEAD" ? null : response.body, {
    status: response.status,
    headers: outgoing,
  });
}

export const GET = forward;
export const POST = forward;
export const PUT = forward;
export const PATCH = forward;
export const DELETE = forward;
export const HEAD = forward;
