// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// ZIP central-directory reader that needs only ranged reads: the end of the file, then the central directory.
// Pure functions over byte buffers, so they run in a Worker and in a plain `node --test`. No member is ever decompressed.
//
// Format reference: PKWARE APPNOTE 6.3.x (end of central directory, ZIP64 end record and locator, central file header,
// extra fields 0x0001 zip64, 0x5455 extended timestamp, 0x7075 Info-ZIP unicode path).

const SIG_EOCD = 0x06054b50;
const SIG_ZIP64_LOCATOR = 0x07064b50;
const SIG_ZIP64_EOCD = 0x06064b50;
const SIG_CENTRAL = 0x02014b50;
const EOCD_MIN = 22;
const MAX_COMMENT = 0xffff;

/** How many bytes from the end of the file must be read to be sure of containing the end-of-central-directory record. */
export const TAIL_BYTES = EOCD_MIN + MAX_COMMENT + 20 + 4096;

// IBM code page 437, bytes 0x80..0xFF: the encoding a ZIP name has when flag bit 11 is clear (as Python's zipfile assumes).
const CP437_HIGH =
  "ÇüéâäàåçêëèïîìÄÅÉæÆôöòûùÿÖÜ¢£¥₧ƒáíóúñÑªº¿⌐¬½¼¡«»░▒▓│┤╡╢╖╕╣║╗╝╜╛┐└┴┬├─┼╞╟╚╔╩╦╠═╬╧╨╤╥╙╘╒╓╫╪┘┌█▄▌▐▀αßΓπΣσµτΦΘΩδ∞φε∩≡±≥≤⌠⌡÷≈°∙·√ⁿ²■ ";

function view(buf) {
  return new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
}

/**
 * Find the end-of-central-directory record in the tail of a file.
 *
 * Scans backwards for the signature and keeps a candidate whose comment length reaches exactly the end of the buffer; if
 * none does, the last signature found is used (a file with trailing bytes). Returns the index in `tail`, or -1.
 */
export function findEOCD(tail) {
  const dv = view(tail);
  let fallback = -1;
  for (let i = tail.length - EOCD_MIN; i >= 0; i--) {
    if (dv.getUint32(i, true) !== SIG_EOCD) continue;
    const commentLen = dv.getUint16(i + 20, true);
    if (i + EOCD_MIN + commentLen === tail.length) return i;
    if (fallback < 0) fallback = i;
  }
  return fallback;
}

/** Decode the 22-byte end-of-central-directory record at `at`. */
export function parseEOCD(tail, at) {
  const dv = view(tail);
  return {
    disk: dv.getUint16(at + 4, true),
    cdDisk: dv.getUint16(at + 6, true),
    entriesOnDisk: dv.getUint16(at + 8, true),
    entries: dv.getUint16(at + 10, true),
    cdSize: dv.getUint32(at + 12, true),
    cdOffset: dv.getUint32(at + 16, true),
    commentLen: dv.getUint16(at + 20, true),
  };
}

/** True when the EOCD fields are saturated, which means the real values are in the ZIP64 end record. */
export function needsZip64(eocd) {
  return eocd.entries === 0xffff || eocd.cdSize === 0xffffffff || eocd.cdOffset === 0xffffffff || eocd.entriesOnDisk === 0xffff;
}

/** Read the ZIP64 end-of-central-directory locator that ends right before the EOCD; returns the absolute offset of the ZIP64 record, or -1. */
export function parseZip64Locator(tail, eocdAt) {
  const at = eocdAt - 20;
  if (at < 0) return -1;
  const dv = view(tail);
  if (dv.getUint32(at, true) !== SIG_ZIP64_LOCATOR) return -1;
  return Number(dv.getBigUint64(at + 8, true));
}

/** Decode a ZIP64 end-of-central-directory record (at least 56 bytes), or null if the signature is wrong. */
export function parseZip64EOCD(buf) {
  if (buf.length < 56) return null;
  const dv = view(buf);
  if (dv.getUint32(0, true) !== SIG_ZIP64_EOCD) return null;
  return {
    disk: dv.getUint32(16, true),
    cdDisk: dv.getUint32(20, true),
    entriesOnDisk: Number(dv.getBigUint64(24, true)),
    entries: Number(dv.getBigUint64(32, true)),
    cdSize: Number(dv.getBigUint64(40, true)),
    cdOffset: Number(dv.getBigUint64(48, true)),
  };
}

function decodeName(bytes, utf8Flag, extra) {
  const unicodePath = findExtra(extra, 0x7075);
  if (unicodePath && unicodePath.length > 5 && unicodePath[0] === 1) return new TextDecoder("utf-8").decode(unicodePath.subarray(5));
  if (utf8Flag) return new TextDecoder("utf-8").decode(bytes);
  let out = "";
  for (const b of bytes) out += b < 0x80 ? String.fromCharCode(b) : CP437_HIGH[b - 0x80];
  return out;
}

function findExtra(extra, id) {
  const dv = view(extra);
  let at = 0;
  while (at + 4 <= extra.length) {
    const fieldId = dv.getUint16(at, true);
    const size = dv.getUint16(at + 2, true);
    if (at + 4 + size > extra.length) return null;
    if (fieldId === id) return extra.subarray(at + 4, at + 4 + size);
    at += 4 + size;
  }
  return null;
}

function dosToISO(date, time) {
  if (date === 0) return null;
  const y = 1980 + (date >> 9);
  const mo = (date >> 5) & 0xf;
  const d = date & 0x1f;
  const h = time >> 11;
  const mi = (time >> 5) & 0x3f;
  const s = (time & 0x1f) * 2;
  const p = (n) => String(n).padStart(2, "0");
  return `${y}-${p(mo)}-${p(d)}T${p(h)}:${p(mi)}:${p(s)}`;
}

/**
 * Incremental central-directory parser.
 *
 * `push(chunk)` takes the next bytes of the central directory in order (chunks may split an entry anywhere) and returns the
 * members completed so far. After the last chunk `leftover` is the number of bytes of an unfinished entry (0 when clean).
 * Each member is `{index, name, comp_size, size, crc32, method, flags, encrypted, is_dir, local_header_offset, mtime}`;
 * `crc32` is 8 lowercase hex digits and `mtime` an ISO local time (the extended timestamp, in UTC with a Z, when present).
 */
export class CentralDirectoryParser {
  constructor() {
    this.pending = new Uint8Array(0);
    this.index = 0;
    this.leftover = 0;
  }

  push(chunk) {
    let buf = chunk;
    if (this.pending.length) {
      buf = new Uint8Array(this.pending.length + chunk.length);
      buf.set(this.pending, 0);
      buf.set(chunk, this.pending.length);
    }
    const members = [];
    let at = 0;
    while (buf.length - at >= 46) {
      const dv = new DataView(buf.buffer, buf.byteOffset + at, buf.length - at);
      if (dv.getUint32(0, true) !== SIG_CENTRAL) throw new Error(`central directory entry ${this.index}: bad signature at byte ${at}`);
      const nameLen = dv.getUint16(28, true);
      const extraLen = dv.getUint16(30, true);
      const commentLen = dv.getUint16(32, true);
      const total = 46 + nameLen + extraLen + commentLen;
      if (buf.length - at < total) break;
      members.push(this.#entry(buf.subarray(at, at + total), dv, nameLen, extraLen));
      this.index++;
      at += total;
    }
    this.pending = buf.slice(at);
    this.leftover = this.pending.length;
    return members;
  }

  #entry(raw, dv, nameLen, extraLen) {
    const flags = dv.getUint16(8, true);
    const method = dv.getUint16(10, true);
    const time = dv.getUint16(12, true);
    const date = dv.getUint16(14, true);
    const crc = dv.getUint32(16, true);
    let compSize = dv.getUint32(20, true);
    let size = dv.getUint32(24, true);
    const diskStart = dv.getUint16(34, true);
    let lho = dv.getUint32(42, true);
    const name = raw.subarray(46, 46 + nameLen);
    const extra = raw.subarray(46 + nameLen, 46 + nameLen + extraLen);
    const z64 = findExtra(extra, 0x0001);
    if (z64) {
      const zv = view(z64);
      let o = 0;
      const next = () => {
        if (o + 8 > z64.length) throw new Error(`central directory entry ${this.index}: truncated zip64 extra field`);
        const value = Number(zv.getBigUint64(o, true));
        o += 8;
        return value;
      };
      if (size === 0xffffffff) size = next();
      if (compSize === 0xffffffff) compSize = next();
      if (lho === 0xffffffff) lho = next();
    } else if (size === 0xffffffff || compSize === 0xffffffff || lho === 0xffffffff) {
      throw new Error(`central directory entry ${this.index}: sizes saturated but no zip64 extra field`);
    }
    let mtime = dosToISO(date, time);
    const ts = findExtra(extra, 0x5455);
    if (ts && ts.length >= 5 && (ts[0] & 1)) mtime = new Date(view(ts).getUint32(1, true) * 1000).toISOString();
    const decoded = decodeName(name, (flags & 0x800) !== 0, extra);
    return {
      index: this.index,
      name: decoded,
      comp_size: compSize,
      size,
      crc32: crc.toString(16).padStart(8, "0"),
      method,
      flags,
      encrypted: (flags & 1) !== 0,
      is_dir: decoded.endsWith("/"),
      local_header_offset: lho,
      disk_start: diskStart,
      mtime,
    };
  }
}

/**
 * List the members of a ZIP stored in an object store, with ranged reads only.
 *
 * `store.range(key, offset, length)` is the only I/O. `size` is the object size. Options: `maxMembers` (default 200000) and
 * `maxCdBytes` (default 64 MiB) bound how much of a huge directory is read; `window` is the bytes per read (default 4 MiB).
 * `onMembers(rows)` is awaited for each parsed batch. Returns the archive summary
 * `{entries_declared, entries_listed, cd_offset, cd_size, zip64, comment_len, prefix_bytes, truncated, requests, bytes_read}`;
 * it throws with a plain message for a file that is not a ZIP, is truncated, or spans several disks.
 */
export async function listZip(store, key, size, onMembers, { maxMembers = 200000, maxCdBytes = 64 * 1024 * 1024, window = 4 * 1024 * 1024 } = {}) {
  if (size < EOCD_MIN) throw new Error("not a zip: smaller than an end-of-central-directory record");
  let requests = 0;
  let bytesRead = 0;
  const read = async (offset, length) => {
    const bytes = await store.range(key, offset, length);
    requests++;
    bytesRead += bytes.length;
    return bytes;
  };
  const tailLen = Math.min(size, TAIL_BYTES);
  const tailStart = size - tailLen;
  const tail = await read(tailStart, tailLen);
  const at = findEOCD(tail);
  if (at < 0) throw new Error("not a zip: no end-of-central-directory record");
  const eocdPos = tailStart + at;
  let { entries, cdSize, cdOffset, disk, cdDisk, commentLen } = parseEOCD(tail, at);
  let zip64 = false;
  let endOfCd = eocdPos; // where the central directory must end, before any prefix correction
  const z64pos = parseZip64Locator(tail, at);
  if (z64pos >= 0) endOfCd = z64pos; // a ZIP64 record sits between the central directory and the EOCD even when unused
  if (needsZip64({ entries, cdSize, cdOffset, entriesOnDisk: entries })) {
    if (z64pos < 0) throw new Error("zip64 end-of-central-directory locator missing");
    const record = parseZip64EOCD(await read(z64pos, 1024));
    if (!record) throw new Error("zip64 end-of-central-directory record not found at the located offset");
    ({ entries, cdSize, cdOffset } = record);
    disk = record.disk;
    cdDisk = record.cdDisk;
    zip64 = true;
  }
  if (disk !== 0 || cdDisk !== 0) throw new Error("multi-disk archives are not supported");
  const summary = { entries_declared: entries, entries_listed: 0, cd_offset: cdOffset, cd_size: cdSize, zip64, comment_len: commentLen, prefix_bytes: 0, truncated: false };
  if (entries === 0 || cdSize === 0) return { ...summary, requests, bytes_read: bytesRead };
  const delta = endOfCd - (cdOffset + cdSize);
  if (delta < 0) throw new Error("central directory lies beyond the end record: truncated archive");
  if (delta > 0) {
    // bytes were prepended (self-extractor, mail wrapper): every offset in the file is shifted by `delta`
    summary.prefix_bytes = delta;
    cdOffset += delta;
    summary.cd_offset = cdOffset;
  }
  const parser = new CentralDirectoryParser();
  const limit = Math.min(cdSize, maxCdBytes);
  if (limit < cdSize) summary.truncated = true;
  let done = 0;
  while (done < limit && summary.entries_listed < maxMembers) {
    const chunk = await read(cdOffset + done, Math.min(window, limit - done));
    if (chunk.length === 0) throw new Error("central directory ended early: truncated archive");
    done += chunk.length;
    let members = parser.push(chunk);
    if (summary.entries_listed + members.length > maxMembers) {
      members = members.slice(0, maxMembers - summary.entries_listed);
      summary.truncated = true;
    }
    summary.entries_listed += members.length;
    if (members.length) await onMembers(members);
  }
  if (!summary.truncated && summary.entries_listed < entries) {
    if (parser.leftover > 0) throw new Error("central directory ends inside an entry: truncated archive");
    throw new Error(`central directory holds ${summary.entries_listed} entries but the end record declares ${entries}`);
  }
  if (summary.entries_listed < entries) summary.truncated = true;
  return { ...summary, requests, bytes_read: bytesRead };
}
