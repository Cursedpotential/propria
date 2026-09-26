"use client";

import { useEffect, useState } from "react";
import { legalApiBase } from "@/lib/api/client";
import {
  type Surface,
  SURFACES,
  enrichCatalog,
  setSurfaceCatalog,
  surfaceCatalog,
} from "@/lib/surfaces";

// Byline: Grok · grok-4.6 · 2026-08-18
// Shell reads pages from GET /v1/routing.surfaces.

export function useSurfaceCatalog(): Surface[] {
  const [rows, setRows] = useState<Surface[]>(surfaceCatalog());

  useEffect(() => {
    void (async () => {
      try {
        const response = await fetch(`${legalApiBase()}/v1/routing`, { cache: "no-store" });
        if (!response.ok) return;
        const body = (await response.json()) as { surfaces?: Surface[] };
        if (Array.isArray(body.surfaces) && body.surfaces.length > 0) {
          const merged = enrichCatalog(body.surfaces);
          setSurfaceCatalog(merged);
          setRows(merged);
        }
      } catch {
        setRows(SURFACES);
      }
    })();
  }, []);

  return rows;
}
