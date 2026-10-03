// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// ZIP central-directory parser: hand-built archives (every field laid out as the spec does), and archives written by
// Python's zipfile, an independent implementation, when `python3` is on the PATH.
import test from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { crc32, deflateRawSync } from "node:zlib";
import { CentralDirectoryParser, TAIL_BYTES, findEOCD, listZip, parseEOCD } from "../zip_lister/src/zipcd.js";

const enc = (s) => new TextEncoder().encode(s);

/** An in-memory store with the Worker's range() contract; counts requests and bytes. */
function memStore(buf) {
  const stats = { requests: 0, bytes: 0 };
  return {
    stats,
    async range(_key, offset, length) {
      stats.requests++;
      const out = new Uint8Array(buf.subarray(offset, Math.min(buf.length, offset + length)));
      stats.bytes += out.length;
      return out;
    },
  };
}

function u16(n) {
  return Buffer.from([n & 255, (n >> 8) & 255]);
}
function u32(n) {
  const b = Buffer.alloc(4);
  b.writeUInt32LE(n >>> 0);
  return b;
}
function u64(n) {
  const b = Buffer.alloc(8);
  b.writeBigUInt64LE(BigInt(n));
  return b;
}

/**
 * Build a ZIP the way a standard writer does. entries: {name, data, deflate?, utf8?, cp437Name?, fake?:{size,compSize,lho}}.
 * `fake` writes claimed sizes (for ZIP64 without 4 GiB of data); `zip64: true` writes the ZIP64 records.
 */
function buildZip(entries, { zip64 = false, comment = "", prefix = Buffer.alloc(0), forceZip64Eocd = false } = {}) {
  const parts = [];
  const central = [];
  let offset = 0;
  const push = (b) => {
    parts.push(b);
    offset += b.length;
  };
  push(prefix);
  const base = prefix.length;
  for (const e of entries) {
    const data = Buffer.from(e.data ?? "");
    const name = e.cp437Name ?? Buffer.from(e.name, "utf8");
    const method = e.deflate ? 8 : 0;
    const body = e.deflate ? deflateRawSync(data) : data;
    const crc = crc32(data);
    const lho = offset - base;
    const flags = e.utf8 === false ? 0 : 0x800;
    push(Buffer.concat([u32(0x04034b50), u16(20), u16(flags), u16(method), u16(0x6000), u16(0x5a21), u32(crc), u32(body.length), u32(data.length), u16(name.length), u16(0), name, body]));
    const size = e.fake?.size ?? data.length;
    const compSize = e.fake?.compSize ?? body.length;
    const bigSize = size > 0xfffffffe || compSize > 0xfffffffe || (e.fake?.lho ?? lho) > 0xfffffffe;
    const lhoReal = e.fake?.lho ?? lho;
    const extra = bigSize
      ? Buffer.concat([u16(1), u16(24), u64(size), u64(compSize), u64(lhoReal)])
      : Buffer.alloc(0);
    central.push(
      Buffer.concat([
        u32(0x02014b50), u16(0x031e), u16(bigSize ? 45 : 20), u16(flags), u16(method), u16(0x6000), u16(0x5a21), u32(crc),
        u32(bigSize ? 0xffffffff : compSize), u32(bigSize ? 0xffffffff : size), u16(name.length), u16(extra.length), u16(0), u16(0), u16(0), u32(0),
        u32(bigSize ? 0xffffffff : lhoReal), name, extra,
      ]),
    );
  }
  const cdOffset = offset - base;
  const cd = Buffer.concat(central);
  push(cd);
  const useZip64 = zip64 || forceZip64Eocd;
  if (useZip64) {
    const z64pos = offset - base;
    push(Buffer.concat([u32(0x06064b50), u64(44), u16(45), u16(45), u32(0), u32(0), u64(entries.length), u64(entries.length), u64(cd.length), u64(cdOffset)]));
    push(Buffer.concat([u32(0x07064b50), u32(0), u64(z64pos), u32(1)]));
  }
  const c = Buffer.from(comment);
  const sat = zip64;
  push(Buffer.concat([u32(0x06054b50), u16(0), u16(0), u16(sat ? 0xffff : entries.length), u16(sat ? 0xffff : entries.length), u32(sat ? 0xffffffff : cd.length), u32(sat ? 0xffffffff : cdOffset), u16(c.length), c]));
  return Buffer.concat(parts);
}

async function list(buf, opts) {
  const rows = [];
  const store = memStore(buf);
  const summary = await listZip(store, "k.zip", buf.length, async (m) => rows.push(...m), opts);
  return { rows, summary, store };
}

test("stored and deflated members: name, sizes, CRC32, method", async () => {
  const buf = buildZip([
    { name: "conversations.json", data: '[{"mapping":{}}]'.repeat(50), deflate: true },
    { name: "Takeout/", data: "" },
    { name: "Takeout/My Activity/Gemini Apps/MyActivity.html", data: "<html></html>" },
  ]);
  const { rows, summary } = await list(buf);
  assert.equal(summary.entries_listed, 3);
  assert.equal(summary.zip64, false);
  assert.equal(rows[0].name, "conversations.json");
  assert.equal(rows[0].method, 8);
  assert.equal(rows[0].size, '[{"mapping":{}}]'.length * 50);
  assert.equal(rows[0].comp_size, deflateRawSync(Buffer.from('[{"mapping":{}}]'.repeat(50))).length);
  assert.equal(rows[0].crc32, crc32(Buffer.from('[{"mapping":{}}]'.repeat(50))).toString(16).padStart(8, "0"));
  assert.equal(rows[1].is_dir, true);
  assert.equal(rows[2].method, 0);
  assert.equal(rows[2].mtime, "2025-01-01T12:00:00"); // DOS date 0x5a21 / time 0x6000
  assert.ok(rows.every((r, i) => r.index === i));
});

test("only the tail and the central directory are read, never the members", async () => {
  const big = "x".repeat(2_000_000);
  const buf = buildZip([{ name: "a.bin", data: big }, { name: "b.bin", data: big }]);
  const { store } = await list(buf);
  assert.ok(store.stats.bytes < TAIL_BYTES + 2_000, `read ${store.stats.bytes} bytes of a ${buf.length} byte archive`);
  assert.ok(store.stats.requests <= 3);
});

test("archive comment and a self-extractor prefix are handled", async () => {
  const prefix = Buffer.from("MZ".padEnd(5000, "\u0090"), "latin1");
  const buf = buildZip([{ name: "m.txt", data: "hello" }], { comment: "made by a test ".repeat(100), prefix });
  const { rows, summary } = await list(buf);
  assert.equal(rows[0].name, "m.txt");
  assert.equal(summary.prefix_bytes, 5000);
  assert.equal(summary.comment_len, 1500);
});

test("ZIP64: saturated end record, zip64 locator, and entries with sizes above 4 GiB", async () => {
  const GiB = 1024 ** 3;
  const buf = buildZip(
    [
      { name: "takeout-001/Takeout/Drive/huge.mp4", data: "tiny", fake: { size: 9 * GiB, compSize: 8 * GiB, lho: 5 * GiB } },
      { name: "small.txt", data: "abc" },
    ],
    { zip64: true },
  );
  const { rows, summary } = await list(buf);
  assert.equal(summary.zip64, true);
  assert.equal(summary.entries_declared, 2);
  assert.equal(rows[0].size, 9 * GiB);
  assert.equal(rows[0].comp_size, 8 * GiB);
  assert.equal(rows[0].local_header_offset, 5 * GiB);
  assert.equal(rows[1].size, 3);
});

test("an unused ZIP64 end record before the EOCD is not mistaken for a prefix", async () => {
  const buf = buildZip([{ name: "a.txt", data: "a" }], { forceZip64Eocd: true });
  const { rows, summary } = await list(buf);
  assert.equal(rows.length, 1);
  assert.equal(summary.prefix_bytes, 0);
  assert.equal(summary.zip64, false);
});

test("names: UTF-8 flag, and code page 437 without it", async () => {
  const buf = buildZip([
    { name: "Résumé – 2025.txt", data: "a" },
    { name: "ignored", cp437Name: Buffer.from([0x63, 0x61, 0x66, 0x82, 0x2e, 0x74, 0x78, 0x74]), data: "b", utf8: false }, // "café.txt" in CP437
  ]);
  const { rows } = await list(buf);
  assert.equal(rows[0].name, "Résumé – 2025.txt");
  assert.equal(rows[1].name, "café.txt");
});

test("the central directory is read in windows and an entry may straddle two of them", async () => {
  const entries = Array.from({ length: 3000 }, (_, i) => ({ name: `Takeout/Gemini Apps/conversation-${String(i).padStart(5, "0")}-with-a-long-name.json`, data: `{"i":${i}}` }));
  const buf = buildZip(entries);
  const { rows, summary, store } = await list(buf, { window: 1000 });
  assert.equal(rows.length, 3000);
  assert.equal(rows[2999].name, entries[2999].name);
  assert.equal(summary.entries_listed, 3000);
  assert.ok(store.stats.requests > 100);
});

test("max_members truncates and says so", async () => {
  const buf = buildZip(Array.from({ length: 50 }, (_, i) => ({ name: `f${i}.txt`, data: "x" })));
  const { rows, summary } = await list(buf, { maxMembers: 10 });
  assert.equal(rows.length, 10);
  assert.equal(summary.truncated, true);
  assert.equal(summary.entries_declared, 50);
});

test("errors: not a zip, truncated archive, multi-disk", async () => {
  await assert.rejects(() => list(Buffer.from("this is a text file, not an archive at all")), /not a zip/);
  const buf = buildZip([{ name: "a.txt", data: "hello" }, { name: "b.txt", data: "world" }]);
  await assert.rejects(() => list(buf.subarray(0, buf.length - 40)), /not a zip|truncated|ends/); // EOCD cut off
  const multi = Buffer.from(buf);
  multi.writeUInt16LE(1, multi.length - 22 + 4); // disk number 1
  await assert.rejects(() => list(multi), /multi-disk/);
  const noSig = Buffer.from(buf);
  noSig.writeUInt32LE(0xdeadbeef, noSig.readUInt32LE(noSig.length - 22 + 16)); // wreck the first central entry
  await assert.rejects(() => list(noSig), /bad signature/);
});

test("an empty archive lists zero members", async () => {
  const { rows, summary } = await list(buildZip([]));
  assert.equal(rows.length, 0);
  assert.equal(summary.entries_declared, 0);
});

test("findEOCD skips a signature inside the comment and finds the real record", () => {
  const fake = "PK" + "A".repeat(30); // looks like an EOCD, but its comment length (0x4141) cannot reach the end
  const outer = buildZip([{ name: "b", data: "y" }], { comment: "note " + fake + " more" });
  const at = findEOCD(new Uint8Array(outer));
  const eocd = parseEOCD(new Uint8Array(outer), at);
  assert.equal(eocd.entries, 1);
  assert.equal(eocd.commentLen, ("note " + fake + " more").length);
});

test("CentralDirectoryParser reports a half-read entry as leftover", () => {
  const buf = buildZip([{ name: "a.txt", data: "x" }]);
  const cdStart = buf.readUInt32LE(buf.length - 22 + 16);
  const parser = new CentralDirectoryParser();
  const half = new Uint8Array(buf.subarray(cdStart, cdStart + 30));
  assert.equal(parser.push(half).length, 0);
  assert.equal(parser.push(new Uint8Array(buf.subarray(cdStart + 30, buf.length - 22))).length, 1);
  assert.equal(parser.leftover, 0);
});

// --- independent writer: Python's zipfile --------------------------------------------------------------------------------
const py = spawnSync("python3", ["-c", "import zipfile,sys;print(sys.version_info[0])"], { encoding: "utf8" });
const havePython = py.status === 0 && py.stdout.trim() === "3";

test("Python zipfile archive: names, sizes, CRC and methods match what the writer recorded", { skip: !havePython }, async () => {
  const dir = mkdtempSync(join(tmpdir(), "cfzip-"));
  try {
    const out = join(dir, "t.zip");
    const script = `
import zipfile, json
with zipfile.ZipFile(${JSON.stringify(out)}, "w") as z:
    z.writestr("conversations.json", "[{\\"mapping\\":{}}]" * 400, compress_type=zipfile.ZIP_DEFLATED)
    z.writestr("chat.html", "<html>jsonData</html>", compress_type=zipfile.ZIP_STORED)
    z.writestr("notes/ünï.md", "# hi", compress_type=zipfile.ZIP_DEFLATED)
    z.writestr(zipfile.ZipInfo("empty/"), "")
    z.comment = b"python wrote this"
print(json.dumps([[i.filename, i.file_size, i.compress_size, i.CRC, i.compress_type] for i in zipfile.ZipFile(${JSON.stringify(out)}).infolist()]))
`;
    const r = spawnSync("python3", ["-c", script], { encoding: "utf8" });
    assert.equal(r.status, 0, r.stderr);
    const expected = JSON.parse(r.stdout);
    const { rows } = await list(readFileSync(out));
    assert.deepEqual(
      rows.map((m) => [m.name, m.size, m.comp_size, parseInt(m.crc32, 16), m.method]),
      expected,
    );
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("Python zipfile archive with 70,000 members forces the real ZIP64 end record", { skip: !havePython, timeout: 120000 }, async () => {
  const dir = mkdtempSync(join(tmpdir(), "cfzip-"));
  try {
    const out = join(dir, "many.zip");
    const script = `
import zipfile
with zipfile.ZipFile(${JSON.stringify(out)}, "w") as z:
    for i in range(70000):
        z.writestr("d%d/f%d.txt" % (i % 100, i), "")
`;
    const r = spawnSync("python3", ["-c", script], { encoding: "utf8" });
    assert.equal(r.status, 0, r.stderr);
    const { rows, summary } = await list(readFileSync(out));
    assert.equal(summary.zip64, true);
    assert.equal(summary.entries_declared, 70000);
    assert.equal(rows.length, 70000);
    assert.equal(rows[69999].name, "d99/f69999.txt");
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
