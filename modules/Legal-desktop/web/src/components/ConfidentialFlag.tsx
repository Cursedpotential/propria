"use client";

import { useEffect, useState } from "react";

// Byline: Grok · grok-4.6 · 2026-08-18
// Reads the shell toggle key. Does not decide privilege.

export const CONFIDENTIAL_KEY = "lw-confidential";

export function readConfidential(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(CONFIDENTIAL_KEY) === "1";
  } catch {
    return false;
  }
}

export function ConfidentialFlag() {
  const [on, setOn] = useState<boolean | null>(null);
  useEffect(() => {
    const update = () => setOn(readConfidential());
    update();
    const observer = new MutationObserver(update);
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-confidential"] });
    window.addEventListener("storage", update);
    return () => {
      observer.disconnect();
      window.removeEventListener("storage", update);
    };
  }, []);
  return (
    <span className="pr-status" data-pr-status="information">
      {on === null ? "Checking privacy setting" : `Confidential mode ${on ? "on" : "off"}`}
    </span>
  );
}
