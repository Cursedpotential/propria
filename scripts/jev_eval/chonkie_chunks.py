"""Chonkie chunking of the 2024 Katrina text stream, reusing the vectors already in Weaviate.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 22:03 "try chonky?"; 22:05 "aren't these indexed? didn't we put them
into Weaviate? aren't these searchable semantically?"; 22:02 the 30-minute bouts were never real chunks.
- Stream: bouts/c2024_bouts_v2.jsonl flattened in time order: the same order and indexes items_h2h.py uses.
- semantic: chonkie SemanticChunker, one message per sentence. A single message's vector is the one already stored in
  MsgEvents20260918 (named vector text_nim = nvidia/nemotron-3-embed-1b, input_type=passage, over the stripped body),
  found by content_key = msg_id. Chonkie also embeds the window and group texts it builds; those are embedded with the
  same NIM model and settings and cached in <outdir>/embed_cache.jsonl, so a rerun re-embeds nothing.
- slumber (--slumber <gemini model>): chonkie SlumberChunker, an LLM choosing the split points, on the head-to-head
  windows (time + speaker lines, as the models saw them; read from <h2h_run.json>).
- Output: <outdir>/stream.json (index -> msg_id, day, time, who) and <outdir>/<run>.json per configuration, chunks as
  first/last stream index plus how many vectors were reused vs embedded.
Runs in the ovh-files devbox, jev-eval venv, --env-file h2h.env (NVIDIA_API_KEY, GEMINI_*); keys are never printed:
    .venv/bin/python code/chonkie_chunks.py <outdir> [--slumber gemini-2.5-flash --h2h raw/h2h_v2/_run.json]
"""

import argparse
import json
import os
import pathlib
import urllib.request

import numpy as np
from chonkie import BaseEmbeddings, SemanticChunker, SlumberChunker
from chonkie.genie import BaseGenie

import block_review_llm as B

WV = os.environ.get("WEAVIATE_URL", "http://100.91.190.107:8082").rstrip("/")
COLL = "MsgEvents20260918"
NIM = "https://integrate.api.nvidia.com/v1/embeddings"
MODEL = "nvidia/nemotron-3-embed-1b"
DIM = 2048
BOUTS = "bouts/c2024_bouts_v2.jsonl"
SEMANTIC_RUNS = {  # name: SemanticChunker settings (chonkie 1.7 defaults: threshold 0.8, window 3, min chars 24)
    "semantic_t80_w3": {"threshold": 0.8, "similarity_window": 3, "min_characters_per_sentence": 24},
    "semantic_t65_w3": {"threshold": 0.65, "similarity_window": 3, "min_characters_per_sentence": 24},
    "semantic_t50_w3": {"threshold": 0.5, "similarity_window": 3, "min_characters_per_sentence": 24},
    "semantic_t65_w5_min1": {"threshold": 0.65, "similarity_window": 5, "min_characters_per_sentence": 1},
}


def load_stream() -> list[dict]:
    bouts = [json.loads(x) for x in pathlib.Path(BOUTS).read_text(encoding="utf-8").split("\n") if x.strip()]
    bouts.sort(key=lambda b: b["start_local"])
    return [{"k": k, "bout_id": b["bout_id"], "msg_id": m["msg_id"], "day": b["day"], "ts_local": m["ts_local"],
             "who": m["who"],
             "text": " ".join((m["text"] or "").split()) or "(attachment)"}
            for k, (b, m) in enumerate((b, m) for b in bouts for m in b["messages"])]


def gql(query: str) -> dict:
    req = urllib.request.Request(f"{WV}/v1/graphql", data=json.dumps({"query": query}).encode(),
                                 headers={"content-type": "application/json"})
    with urllib.request.urlopen(req, timeout=120) as r:
        d = json.loads(r.read().decode())
    if d.get("errors"):
        raise RuntimeError(json.dumps(d["errors"])[:500])
    return d["data"]


def stored_vectors(msg_ids: list[str]) -> dict[str, np.ndarray]:
    """content_key -> stored text_nim vector. Several records can share a content_key (the same message from another
    backup or device: corroboration, not duplicates); their vectors are the same body embedded the same way."""
    out = {}
    for i in range(0, len(msg_ids), 40):
        ids = msg_ids[i:i + 40]
        q = ('{Get{%s(limit:400, where:{path:["content_key"],operator:ContainsAny,valueText:%s})'
             '{content_key _additional{vectors{text_nim}}}}}' % (COLL, json.dumps(ids)))
        for o in gql(q)["Get"][COLL]:
            v = (o["_additional"].get("vectors") or {}).get("text_nim")
            if v and o["content_key"] not in out:
                out[o["content_key"]] = np.asarray(v, dtype=np.float32)
    return out


class StoredThenNim(BaseEmbeddings):
    """One message's text -> its stored Weaviate vector; any other text (Chonkie's windows and groups) -> NIM with the
    settings the stored vectors were made with, cached on disk."""

    def __init__(self, by_text: dict[str, np.ndarray], cache_path: pathlib.Path):
        super().__init__()
        self.by_text, self.cache_path, self.cache = by_text, cache_path, {}
        self.reused = self.fresh = self.cached = 0
        if cache_path.exists():
            for line in cache_path.read_text(encoding="utf-8").splitlines():
                t, v = json.loads(line)
                self.cache[t] = np.asarray(v, dtype=np.float32)

    def _nim(self, texts: list[str]) -> list[np.ndarray]:
        out = []
        for i in range(0, len(texts), 64):
            part = texts[i:i + 64]
            d = B.post(NIM, {"model": MODEL, "input": part, "input_type": "passage", "encoding_format": "float",
                             "truncate": "END"}, {"Authorization": f"Bearer {os.environ['NVIDIA_API_KEY']}"})
            vs = [x["embedding"] for x in sorted(d["data"], key=lambda x: x["index"])]
            assert len(vs) == len(part) and all(len(v) == DIM for v in vs)
            with self.cache_path.open("a", encoding="utf-8") as f:
                for t, v in zip(part, vs):
                    f.write(json.dumps([t, v]) + "\n")
                    self.cache[t] = np.asarray(v, dtype=np.float32)
            out += [self.cache[t] for t in part]
        return out

    def embed_batch(self, texts: list[str]) -> list[np.ndarray]:
        keys = [t.strip() for t in texts]
        need = sorted({k for k in keys if k not in self.by_text and k not in self.cache})
        if need:
            self._nim(need)
            self.fresh += len(need)
        out = []
        for k in keys:
            if k in self.by_text:
                self.reused += 1
                out.append(self.by_text[k])
            else:
                self.cached += 1
                out.append(self.cache[k])
        return out

    def embed(self, text: str) -> np.ndarray:
        return self.embed_batch([text])[0]

    @property
    def dimension(self) -> int:
        return DIM

    def get_tokenizer(self):
        return "word"


class GeminiRotatingGenie(BaseGenie):
    """SlumberChunker's LLM, through block_review_llm.post_gemini (moves to the next key on 503/429/403)."""

    def __init__(self, model: str):
        self.model = model

    def generate(self, prompt: str) -> str:
        d = B.post_gemini(self.model, {"contents": [{"role": "user", "parts": [{"text": prompt}]}],
                                       "generationConfig": {"temperature": 0.0}})
        return "".join(p.get("text", "") for p in d["candidates"][0]["content"]["parts"])

    def generate_json(self, prompt: str, schema) -> dict:
        s = schema.model_json_schema() if hasattr(schema, "model_json_schema") else schema
        d = B.post_gemini(self.model, {"contents": [{"role": "user", "parts": [{"text": prompt}]}],
                                       "generationConfig": {"temperature": 0.0, "responseMimeType": "application/json",
                                                            "responseJsonSchema": s}})
        return json.loads("".join(p.get("text", "") for p in d["candidates"][0]["content"]["parts"]))


def to_ranges(chunks, starts: list[int], base: int = 0) -> list[dict]:
    """Chonkie chunk character offsets -> first/last line index (line i starts at starts[i])."""
    out = []
    for c in chunks:
        first = max(i for i, s in enumerate(starts) if s <= c.start_index)
        last = max(i for i, s in enumerate(starts) if s < c.end_index)
        out.append({"first": base + first, "last": base + last, "n": last - first + 1})
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("outdir")
    ap.add_argument("--slumber", default="")
    ap.add_argument("--h2h", default="raw/h2h_v2/_run.json")
    a = ap.parse_args()
    out = pathlib.Path(a.outdir)
    out.mkdir(parents=True, exist_ok=True)
    stream = load_stream()
    (out / "stream.json").write_text(json.dumps([{k: m[k] for k in ("k", "msg_id", "day", "ts_local", "who")}
                                                 for m in stream]), encoding="utf-8")
    vecs = stored_vectors([m["msg_id"] for m in stream])
    by_text = {m["text"]: vecs[m["msg_id"]] for m in stream if m["msg_id"] in vecs and m["text"] != "(attachment)"}
    print(f"stream {len(stream)} messages; stored vectors found for {len(vecs)}", flush=True)
    emb = StoredThenNim(by_text, out / "embed_cache.jsonl")
    lines = [m["text"] for m in stream]
    text = "\n".join(lines)
    starts, pos = [], 0
    for ln in lines:
        starts.append(pos)
        pos += len(ln) + 1
    for name, cfg in SEMANTIC_RUNS.items():
        emb.reused = emb.fresh = emb.cached = 0
        ch = SemanticChunker(embedding_model=emb, chunk_size=100000, delim=["\n"], include_delim="prev", **cfg)
        ranges = to_ranges(ch.chunk(text), starts)
        rec = {"run": name, "chunker": "chonkie.SemanticChunker", "settings": cfg, "n_messages": len(stream),
               "n_chunks": len(ranges), "vectors_reused": emb.reused, "texts_embedded_new": emb.fresh,
               "texts_from_cache": emb.cached, "chunks": ranges}
        (out / f"{name}.json").write_text(json.dumps(rec), encoding="utf-8")
        sizes = sorted(r["n"] for r in ranges)
        print(f"{name}: {len(ranges)} chunks, median {sizes[len(sizes) // 2]} msgs, max {sizes[-1]}; "
              f"reused {emb.reused}, new {emb.fresh}, cached {emb.cached}", flush=True)
    if a.slumber:
        run = json.loads(pathlib.Path(a.h2h).read_text(encoding="utf-8"))
        first_k = {}
        for m in stream:
            first_k.setdefault(m["bout_id"], m["k"])
        genie = GeminiRotatingGenie(a.slumber)
        windows = {}
        for bout_id, (_, n, ff, ft) in run["texts"].items():
            if bout_id not in first_k:  # f2024 items come from another stream
                continue
            base = first_k[bout_id] - ff  # the same window items_h2h.py sent to the models
            wl = [f"[{j}] {m['day']} {m['ts_local']} {m['who']}: {m['text']}" for j, m in enumerate(stream[base:base + n])]
            wstarts, p = [], 0
            for ln in wl:
                wstarts.append(p)
                p += len(ln) + 1
            try:
                sl = SlumberChunker(genie=genie, tokenizer="character", chunk_size=4000, candidate_size=256,
                                    min_characters_per_chunk=1, verbose=False)
                windows[bout_id] = {"base": base, "n": n, "focus_from": ff, "focus_to": ft,
                                    "chunks": to_ranges(sl.chunk("\n".join(wl)), wstarts, base)}
            except Exception as e:  # recorded per window, never silent
                windows[bout_id] = {"base": base, "n": n, "error": f"{type(e).__name__}: {e}"[:500]}
            print(f"slumber {bout_id}: {len(windows[bout_id].get('chunks', []))} chunks "
                  f"{windows[bout_id].get('error', '')[:120]}", flush=True)
        (out / f"slumber_{a.slumber}.json").write_text(json.dumps({"run": f"slumber_{a.slumber}",
                                                                   "chunker": "chonkie.SlumberChunker",
                                                                   "windows": windows}), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
