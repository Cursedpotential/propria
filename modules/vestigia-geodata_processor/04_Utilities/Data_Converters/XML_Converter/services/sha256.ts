/**
 * A streaming-friendly SHA-256 implementation.
 * We cannot use crypto.subtle.digest for streaming as it requires the full buffer.
 * This class allows incremental updates.
 */
export class Sha256 {
    private h: Uint32Array;
    private buffer: Uint8Array;
    private bufferLength: number;
    private totalLength: number;
  
    constructor() {
      this.h = new Uint32Array([
        0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
        0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19
      ]);
      this.buffer = new Uint8Array(64);
      this.bufferLength = 0;
      this.totalLength = 0;
    }
  
    update(input: Uint8Array): void {
      let i = 0;
      this.totalLength += input.length;
  
      while (i < input.length) {
        const remaining = 64 - this.bufferLength;
        const chunk = input.subarray(i, i + remaining);
        this.buffer.set(chunk, this.bufferLength);
        this.bufferLength += chunk.length;
        i += chunk.length;
  
        if (this.bufferLength === 64) {
          this.processBlock(this.buffer);
          this.bufferLength = 0;
        }
      }
    }
  
    hex(): string {
      const finalBuffer = new Uint8Array(this.bufferLength < 56 ? 64 : 128);
      finalBuffer.set(this.buffer.subarray(0, this.bufferLength));
      finalBuffer[this.bufferLength] = 0x80;
      
      const view = new DataView(finalBuffer.buffer);
      // Append total length in bits as BigInt64 at the end
      const bitLen = BigInt(this.totalLength) * 8n;
      view.setBigUint64(finalBuffer.length - 8, bitLen, false); // Big-endian
      
      let offset = 0;
      while (offset < finalBuffer.length) {
          this.processBlock(finalBuffer.subarray(offset, offset + 64));
          offset += 64;
      }
  
      return Array.from(this.h)
        .map(x => x.toString(16).padStart(8, '0'))
        .join('');
    }
  
    private processBlock(block: Uint8Array): void {
      const k = [
        0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
        0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
        0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
        0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
        0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
        0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
        0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
        0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
      ];
  
      const w = new Uint32Array(64);
      const view = new DataView(block.buffer, block.byteOffset, block.byteLength);
  
      for (let i = 0; i < 16; i++) w[i] = view.getUint32(i * 4, false);
      for (let i = 16; i < 64; i++) {
        const s0 = this.rotr(w[i - 15], 7) ^ this.rotr(w[i - 15], 18) ^ (w[i - 15] >>> 3);
        const s1 = this.rotr(w[i - 2], 17) ^ this.rotr(w[i - 2], 19) ^ (w[i - 2] >>> 10);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1) >>> 0;
      }
  
      let [a, b, c, d, e, f, g, h] = this.h;
  
      for (let i = 0; i < 64; i++) {
        const S1 = this.rotr(e, 6) ^ this.rotr(e, 11) ^ this.rotr(e, 25);
        const ch = (e & f) ^ (~e & g);
        const temp1 = (h + S1 + ch + k[i] + w[i]) >>> 0;
        const S0 = this.rotr(a, 2) ^ this.rotr(a, 13) ^ this.rotr(a, 22);
        const maj = (a & b) ^ (a & c) ^ (b & c);
        const temp2 = (S0 + maj) >>> 0;
  
        h = g; g = f; f = e; e = (d + temp1) >>> 0;
        d = c; c = b; b = a; a = (temp1 + temp2) >>> 0;
      }
  
      this.h[0] = (this.h[0] + a) >>> 0;
      this.h[1] = (this.h[1] + b) >>> 0;
      this.h[2] = (this.h[2] + c) >>> 0;
      this.h[3] = (this.h[3] + d) >>> 0;
      this.h[4] = (this.h[4] + e) >>> 0;
      this.h[5] = (this.h[5] + f) >>> 0;
      this.h[6] = (this.h[6] + g) >>> 0;
      this.h[7] = (this.h[7] + h) >>> 0;
    }
  
    private rotr(x: number, n: number): number {
      return (x >>> n) | (x << (32 - n));
    }
}