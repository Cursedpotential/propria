// Defense-in-depth for the public Authentik/Traefik ingress only.
// Tailnet hosts remain direct and are deliberately not redirected to Authentik.
// Byline: Codex · GPT-5 · 2026-09-12

import { NextRequest, NextResponse } from "next/server";

function normalizedHost(value: string | null): string {
  return (value ?? "").trim().toLowerCase().replace(/:\d+$/, "");
}

export async function proxy(request: NextRequest): Promise<NextResponse> {
  const publicHost = normalizedHost(process.env.LEGAL_PUBLIC_HOST ?? null);
  const requestHost = normalizedHost(request.headers.get("host"));

  if (publicHost && requestHost === publicHost) {
    const token = request.headers.get("x-authentik-jwt")?.trim();
    if (!token) {
      return NextResponse.json(
        { detail: "Authenticated public UI session required" },
        { status: 401, headers: { "cache-control": "no-store" } },
      );
    }

    try {
      const apiBase = process.env.LEGAL_API_INTERNAL_URL ?? "http://127.0.0.1:8010";
      const identity = await fetch(`${apiBase.replace(/\/$/, "")}/v1/auth/whoami`, {
        headers: { authorization: `Bearer ${token}` },
        cache: "no-store",
      });
      if (!identity.ok) {
        return NextResponse.json(
          { detail: "Invalid or expired public UI session" },
          { status: 401, headers: { "cache-control": "no-store" } },
        );
      }
    } catch {
      return NextResponse.json(
        { detail: "Public UI authorization unavailable" },
        { status: 503, headers: { "cache-control": "no-store" } },
      );
    }
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
