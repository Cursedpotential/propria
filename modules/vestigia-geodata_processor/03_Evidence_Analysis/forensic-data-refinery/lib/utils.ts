
// A simple, time-sortable unique ID generator inspired by UUIDv7.
// This is not a full UUIDv7 implementation but serves the purpose of creating
// time-ordered, reasonably unique IDs on the client-side.
export function generateUUIDv7(): string {
  const timestamp = Date.now();
  const randomPart = Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
  return `${timestamp.toString(16).padStart(12, '0')}-${randomPart}`.slice(0, 36);
}

// Computes the SHA-256 hash of a given string or ArrayBuffer.
export async function sha256(data: string | ArrayBuffer): Promise<string> {
  const buffer = typeof data === 'string'
    ? new TextEncoder().encode(data)
    : data;
  
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  return hashHex;
}
