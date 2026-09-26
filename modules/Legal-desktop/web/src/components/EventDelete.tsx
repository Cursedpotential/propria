"use client";

import { useRouter } from "next/navigation";
import { legalApiBase } from "@/lib/api/client";

export function EventDelete({ eventId }: { eventId: string }) {
  const router = useRouter();

  async function remove() {
    const response = await fetch(`${legalApiBase()}/v1/docket-events/${eventId}`, {
      method: "DELETE",
    });
    if (response.ok) router.refresh();
  }

  return (
    <button type="button" onClick={() => void remove()} style={{ marginLeft: 8 }}>
      Remove
    </button>
  );
}
