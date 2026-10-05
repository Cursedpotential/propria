// Byline: Codex · GPT-6 · 2026-10-05. Bounded human original proxy checks without network or case records.
import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { proxyLibraryOriginal } from "../dist/library-original-proxy.js";

const binding = "library_file:" + "a".repeat(64);
const request = () => new URL("http://fixture/api/library/original?binding_id="+binding+"&version_id=exact%2Fversion");
const response = () => ({headersSent:false,status:0,headers:{},body:null,writeHead(status,headers){this.status=status;this.headers=headers;this.headersSent=true;},end(body){this.body=body;}});

test("original proxy preserves exact version and PDF bytes with server-only credential",async()=>{
  const prior = globalThis.fetch;
  const oldToken = process.env.TOOLKIT_LIBRARY_SYNC_TOKEN_FILE;
  const oldUrl = process.env.TOOLKIT_LIBRARY_SYNC_ORIGINAL_URL;
  const file = join(mkdtempSync(join(tmpdir(),"toolkit-original-proof-")),"token");
  writeFileSync(file,"synthetic-sync-service-credential-over32bytes",{mode:0o600});
  process.env.TOOLKIT_LIBRARY_SYNC_TOKEN_FILE=file;
  process.env.TOOLKIT_LIBRARY_SYNC_ORIGINAL_URL="http://100.91.190.107:8091/toolkit/library/files/";
  const bytes=Buffer.from("%PDF-synthetic-preserved"); let calls=0;
  globalThis.fetch=async(url,options)=>{
    calls++;
    assert.equal(url.searchParams.get("version_id"),"exact/version");
    assert.equal(options.redirect,"error");
    assert.equal(options.headers.Authorization,"Bearer synthetic-sync-service-credential-over32bytes");
    return new Response(bytes,{headers:{"Content-Type":"application/pdf","Content-Length":String(bytes.length)}});
  };
  try {
    const ok=response();await proxyLibraryOriginal(request(),ok);
    assert.equal(ok.status,200);assert.deepEqual(ok.body,bytes);assert.equal(ok.headers["Cache-Control"],"private, no-store");
    const invalid=request();invalid.searchParams.append("version_id","other");
    const no=response();await proxyLibraryOriginal(invalid,no);assert.equal(no.status,400);assert.equal(calls,1);
    globalThis.fetch=async()=>new Response("%PDF-small",{headers:{"Content-Type":"application/pdf","Content-Length":String(21*1024*1024)}});
    const oversized=response();await proxyLibraryOriginal(request(),oversized);assert.equal(oversized.status,502);
    globalThis.fetch=async()=>new Response("%PDF-short",{headers:{"Content-Type":"application/pdf","Content-Length":"999"}});
    const incomplete=response();await proxyLibraryOriginal(request(),incomplete);assert.equal(incomplete.status,502);
    globalThis.fetch=async()=>new Response("secret fixture error",{status:503});
    const unavailable=response();await proxyLibraryOriginal(request(),unavailable);assert.equal(unavailable.status,502);
    assert.ok(!unavailable.body.includes("secret"));
  } finally {
    globalThis.fetch=prior;
    if(oldToken===undefined)delete process.env.TOOLKIT_LIBRARY_SYNC_TOKEN_FILE;else process.env.TOOLKIT_LIBRARY_SYNC_TOKEN_FILE=oldToken;
    if(oldUrl===undefined)delete process.env.TOOLKIT_LIBRARY_SYNC_ORIGINAL_URL;else process.env.TOOLKIT_LIBRARY_SYNC_ORIGINAL_URL=oldUrl;
  }
});
