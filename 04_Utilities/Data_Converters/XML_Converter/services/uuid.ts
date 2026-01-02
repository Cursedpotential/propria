/**
 * Generates a UUID v7 (Time-ordered).
 * Structure: 48-bit timestamp | 4-bit version | 12-bit rand | 2-bit variant | 62-bit rand
 */
export class UuidV7 {
  private static lastTimestamp = -1;
  
  static generate(): string {
    const value = new Uint8Array(16);
    
    // 1. Timestamp (48 bits)
    let timestamp = Date.now();
    
    // Simple monotonic check (collision protection within same ms)
    if (timestamp <= UuidV7.lastTimestamp) {
        timestamp = UuidV7.lastTimestamp + 1;
    }
    UuidV7.lastTimestamp = timestamp;

    const high = Math.floor(timestamp / 0x100000000);
    const low = timestamp % 0x100000000;

    value[0] = (high >> 24) & 0xff;
    value[1] = (high >> 16) & 0xff;
    value[2] = (high >> 8) & 0xff;
    value[3] = high & 0xff;
    value[4] = (low >> 24) & 0xff;
    value[5] = (low >> 16) & 0xff;

    // 2. Randomness (80 bits total, minus version/variant bits)
    crypto.getRandomValues(value.subarray(6));

    // 3. Version (0111 for v7) at 7th byte
    value[6] = (value[6] & 0x0f) | 0x70;

    // 4. Variant (10xx) at 9th byte
    value[8] = (value[8] & 0x3f) | 0x80;

    return UuidV7.bytesToHex(value);
  }

  private static bytesToHex(bytes: Uint8Array): string {
    return Array.from(bytes)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('-') // UUID formatting
      .replace(/(.{8})-(.{4})-(.{4})-(.{4})-(.{12})/, '$1-$2-$3-$4-$5');
  }
}