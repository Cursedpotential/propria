"""Every Chonkie chunker that fits a text-message stream, run on the 25 head-to-head windows.

Byline: Claude Code · Sonnet · 2026-10-02. Owner-approved 2026-10-02 09:54 (finish the 2026-09-24 chunking comparison) and
09:55 ("Chonkie has a ton of different chunk options, not just one": run every one that can apply).
Companion of chonkie_chunks.py (2026-09-24, semantic only); imports its stored-vector embedder and stream loader.

Windows: the 25 stretches in raw/h2h_v2/_run.json ("texts": one text per stretch, message lines "[j] day time who: text" plus
"[— 3.7 h no messages —]" gap lines), the same text the LLMs saw. Every method splits the same text; a chunk's start offset
is mapped back to the message it falls in, so each method becomes "message indexes where a new chunk starts".

Methods (chonkie 1.7.0 exports): Token, Fast, Sentence, Recursive, Semantic (3 settings), Late, Neural (2 models),
Slumber (kimi-k3 on NVIDIA NIM). Not run: Code (source code only), Table (markdown tables only), TeraflopAI (sends the text to
a third-party segmentation API: case messages are not sent out).
Semantic embeds one message per sentence with the vector already stored in Weaviate MsgEvents20260918 (text_nim); the
embedder only falls back to NIM for texts that have none (grouped windows, the f2024 stretch). Weaviate is only read.
Slumber's LLM is moonshotai/kimi-k3 (response_format json_object, max_tokens 2500, thinking off first; an empty, junk or
unparsable reply is retried once with thinking on; a call that still fails is counted and shown, never silent).

Runs in the ovh-files devbox, jev-eval venv (keys come from the container env: NVIDIA_API_KEY; never printed):
    .venv/bin/python code/chunk_all.py <outdir> [--only name,name] [--limit N] [--force]
Output: <outdir>/<method>.json = {"method","label","chunker","settings","windows":{key:{"chunks":[{first,last,n}],"secs"}},
"totals":{...}} and <outdir>/stream_<method>.json for the whole-stream counts of the cheap methods.
"""

import argparse
import concurrent.futures as cf
import json
import os
import pathlib
import re
import statistics
import threading
import sys
import time

import chonkie
from chonkie.genie import BaseGenie

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import block_review_llm as B  # noqa: E402
import chonkie_chunks as C  # noqa: E402

H2H = "raw/h2h_v2/_run.json"
MSG = re.compile(r"^\[(\d+)\] (\S+) (\S+) ([^:]+): (.*)$")
NIM_CHAT = "https://integrate.api.nvidia.com/v1/chat/completions"
KIMI = "moonshotai/kimi-k3"

SEM = {
    "semantic_t65_w3": {"threshold": 0.65, "similarity_window": 3, "min_characters_per_sentence": 24},  # earlier setting shown to the owner
    "semantic_t50_w3": {"threshold": 0.5, "similarity_window": 3, "min_characters_per_sentence": 24},  # earlier, coarsest
    "semantic_t50_w8_min1": {"threshold": 0.5, "similarity_window": 8, "min_characters_per_sentence": 1},  # wider window
}
LABELS = {
    "token_1000": "Token 1000 chars", "token_2500": "Token 2500 chars",
    "fast_1000": "Fast 1000 chars", "fast_2500": "Fast 2500 chars",
    "sentence_1000": "Sentence 1000 chars", "sentence_2500": "Sentence 2500 chars",
    "recursive_1000": "Recursive 1000 chars", "recursive_2500": "Recursive 2500 chars",
    "semantic_t65_w3": "Semantic t.65 win3 (earlier)", "semantic_t50_w3": "Semantic t.50 win3 (earlier)",
    "semantic_t50_w8_min1": "Semantic t.50 win8",
    "late_256": "Late (nomic modernbert) 256 tok",
    "neural_distilbert": "Neural/Chonky distilbert", "neural_modernbert": "Neural/Chonky modernbert-large",
    "slumber_kimi_k3": "Slumber (kimi-k3)",
}


# ---------------------------------------------------------------- windows
def load_windows(limit: int = 0) -> list[dict]:
    run = json.loads(pathlib.Path(H2H).read_text(encoding="utf-8"))
    stream = C.load_stream()
    first_k = {}
    for m in stream:
        first_k.setdefault(m["bout_id"], m["k"])
    out = []
    for key, tup in run["texts"].items():
        text, n, ff, ft = tup[0], tup[1], tup[2], tup[3]
        units, plain, msgs = [], [], []  # units: (msg index or None for a gap line, line)
        for ln in text.split("\n"):
            m = MSG.match(ln)
            if m:
                units.append((int(m.group(1)), ln))
                plain.append(" ".join(m.group(5).split()) or "(attachment)")
                msgs.append(int(m.group(1)))
            else:
                units.append((None, ln))
        assert msgs == list(range(n)), f"{key}: message indexes not 0..{n - 1}"
        ids = None
        if key in first_k:
            base = first_k[key] - ff
            ids = [m["msg_id"] for m in stream[base:base + n]]
        out.append({"key": key, "n": n, "ff": ff, "ft": ft, "units": units, "plain": plain, "msg_ids": ids,
                    "text": "\n".join(u[1] for u in units)})
    return out[:limit] if limit else out


def char_starts(units: list) -> list[int]:
    s, p = [], 0
    for _, ln in units:
        s.append(p)
        p += len(ln) + 1
    return s


def to_message_ranges(chunks, units: list, n: int) -> list[dict]:
    """Chunk start offsets -> the message each chunk starts in (a gap line counts for the next message). Chunks that
    start in the same message (a split inside one message) merge; ranges are first..last message index."""
    starts = char_starts(units)
    nxt = [None] * len(units)  # message index at or after each unit
    cur = n - 1
    for i in range(len(units) - 1, -1, -1):
        if units[i][0] is not None:
            cur = units[i][0]
        nxt[i] = cur
    firsts = []
    for c in chunks:
        li = max(i for i, s in enumerate(starts) if s <= c.start_index)
        j = nxt[li]
        if not firsts or j > firsts[-1]:
            firsts.append(j)
    if not firsts or firsts[0] != 0:
        firsts.insert(0, 0)
    return [{"first": f, "last": (firsts[i + 1] - 1 if i + 1 < len(firsts) else n - 1),
             "n": (firsts[i + 1] if i + 1 < len(firsts) else n) - f} for i, f in enumerate(firsts)]


# ---------------------------------------------------------------- Slumber's LLM
class KimiGenie(BaseGenie):
    """kimi-k3 on NVIDIA NIM. Thinking off first (short prompts); empty/junk/unparsable -> one retry with thinking on."""

    def __init__(self):
        self.calls = self.retried = self.failed = 0

    def _post(self, prompt: str, thinking_off: bool) -> str:
        body = {"model": KIMI, "messages": [{"role": "user", "content": prompt}], "max_tokens": 2500, "temperature": 0.0,
                "response_format": {"type": "json_object"}}
        if thinking_off:
            body["chat_template_kwargs"] = {"thinking": False}
        d = B.post(NIM_CHAT, body, {"Authorization": f"Bearer {os.environ['NVIDIA_API_KEY']}"}, timeout=300)
        return (d["choices"][0]["message"].get("content") or "").strip()

    @staticmethod
    def _index(txt: str):
        if not txt or len(set(txt)) <= 2 and len(txt) > 5:  # empty, or a run of one repeated character
            return None
        try:
            return int(json.loads(txt)["split_index"])
        except (ValueError, KeyError, TypeError):
            return None

    def generate_json(self, prompt: str, schema) -> dict:
        self.calls += 1
        p = prompt + '\n\nAnswer with a JSON object only: {"split_index": <integer>}'
        for attempt, off in enumerate((True, False)):
            try:
                idx = self._index(self._post(p, off))
            except Exception:  # noqa: BLE001  network/HTTP: fall to the retry, then fail loudly below
                idx = None
            if idx is not None:
                return {"split_index": idx}
            if attempt == 0:
                self.retried += 1
        self.failed += 1
        raise RuntimeError("kimi-k3: empty/junk reply twice")

    def generate(self, prompt: str) -> str:
        raise NotImplementedError("json mode only")


# ---------------------------------------------------------------- methods
def build_methods(emb) -> dict:
    m = {}
    for size in (1000, 2500):
        m[f"token_{size}"] = ("chonkie.TokenChunker", {"tokenizer": "character", "chunk_size": size, "chunk_overlap": 0},
                              lambda s=size: chonkie.TokenChunker(tokenizer="character", chunk_size=s, chunk_overlap=0), "text")
        m[f"fast_{size}"] = ("chonkie.FastChunker", {"chunk_size": size, "delimiters": "\n"},
                             lambda s=size: chonkie.FastChunker(chunk_size=s, delimiters="\n"), "text")
        m[f"sentence_{size}"] = ("chonkie.SentenceChunker", {"chunk_size": size, "delim": ["\n"]},
                                 lambda s=size: chonkie.SentenceChunker(tokenizer="character", chunk_size=s, delim=["\n"],
                                                                        min_characters_per_sentence=1), "text")
        m[f"recursive_{size}"] = ("chonkie.RecursiveChunker", {"chunk_size": size, "rules": "default"},
                                  lambda s=size: chonkie.RecursiveChunker(tokenizer="character", chunk_size=s,
                                                                          min_characters_per_chunk=1), "text")
    for name, cfg in SEM.items():
        m[name] = ("chonkie.SemanticChunker", cfg,
                   lambda c=cfg: chonkie.SemanticChunker(embedding_model=emb, chunk_size=100000, delim=["\n"],
                                                         include_delim="prev", **c), "plain")
    m["late_256"] = ("chonkie.LateChunker", {"embedding_model": "nomic-ai/modernbert-embed-base", "chunk_size": 256},
                     lambda: chonkie.LateChunker(embedding_model="nomic-ai/modernbert-embed-base", chunk_size=256,
                                                 min_characters_per_chunk=1), "text")
    m["neural_distilbert"] = ("chonkie.NeuralChunker", {"model": "mirth/chonky_distilbert_base_uncased_1"},
                              lambda: chonkie.NeuralChunker(model="mirth/chonky_distilbert_base_uncased_1",
                                                            min_characters_per_chunk=1), "text")
    m["neural_modernbert"] = ("chonkie.NeuralChunker", {"model": "mirth/chonky_modernbert_large_1"},
                              lambda: chonkie.NeuralChunker(model="mirth/chonky_modernbert_large_1",
                                                            min_characters_per_chunk=1), "text")
    m["slumber_kimi_k3"] = ("chonkie.SlumberChunker", {"llm": KIMI, "chunk_size": 1500, "candidate_size": 120},
                            None, "text")
    return m


def summarize(wins: dict) -> dict:
    sizes = [c["n"] for w in wins.values() for c in w.get("chunks", [])]
    done = sum(1 for w in wins.values() if w.get("chunks"))
    return {"windows_done": done, "windows_failed": sum(1 for w in wins.values() if w.get("error")),
            "n_chunks": len(sizes), "median_msgs": statistics.median(sizes) if sizes else None,
            "mean_msgs": round(statistics.mean(sizes), 2) if sizes else None, "max_msgs": max(sizes) if sizes else None,
            "singletons": sum(1 for s in sizes if s == 1)}


def run_method(name, spec, windows, out, emb):
    chunker_name, settings, factory, mode = spec
    path = out / f"{name}.json"
    wins, genie = {}, None
    t0 = time.time()
    if name == "slumber_kimi_k3":
        genie = KimiGenie()

        part_path = out / "slumber_kimi_k3.partial.json"  # finished windows survive a restarted devbox
        part = json.loads(part_path.read_text(encoding="utf-8")) if part_path.exists() else {}
        lock = threading.Lock()

        def one(w):
            if w["key"] in part:
                return w["key"], part[w["key"]]
            r = one_run(w)
            if r[1].get("chunks"):
                with lock:
                    part[w["key"]] = r[1]
                    part_path.write_text(json.dumps(part), encoding="utf-8")
            return r

        def one_run(w):
            sl = chonkie.SlumberChunker(genie=genie, tokenizer="character", chunk_size=1500, candidate_size=120,
                                        min_characters_per_chunk=1, verbose=False)
            s = time.time()
            try:
                return w["key"], {"chunks": to_message_ranges(sl.chunk(w["text"]), w["units"], w["n"]),
                                  "secs": round(time.time() - s, 1)}
            except Exception as e:  # noqa: BLE001  recorded per window, never silent
                return w["key"], {"error": f"{type(e).__name__}: {e}"[:300]}
        with cf.ThreadPoolExecutor(5) as ex:
            for k, v in ex.map(one, windows):
                wins[k] = v
                print(f"  {name} {k}: {len(v.get('chunks', []))} chunks {v.get('error', '')[:100]}", flush=True)
    else:
        ch = factory()
        for w in windows:
            s = time.time()
            try:
                if mode == "plain":
                    units = [(j, ln) for j, ln in enumerate(w["plain"])]
                    chunks = ch.chunk("\n".join(w["plain"]))
                else:
                    units = w["units"]
                    chunks = ch.chunk(w["text"])
                wins[w["key"]] = {"chunks": to_message_ranges(chunks, units, w["n"]), "secs": round(time.time() - s, 1)}
            except Exception as e:  # noqa: BLE001
                wins[w["key"]] = {"error": f"{type(e).__name__}: {e}"[:300]}
            print(f"  {name} {w['key']}: {len(wins[w['key']].get('chunks', []))} chunks "
                  f"{wins[w['key']].get('error', '')[:100]}", flush=True)
    rec = {"method": name, "label": LABELS[name], "chunker": chunker_name, "settings": settings, "windows": wins,
           "totals": summarize(wins), "seconds": round(time.time() - t0, 1)}
    if genie:
        rec["llm_calls"] = {"calls": genie.calls, "retried_with_thinking": genie.retried, "failed": genie.failed}
    path.write_text(json.dumps(rec), encoding="utf-8")
    print(f"{name}: {rec['totals']}" + (f" llm {rec['llm_calls']}" if genie else ""), flush=True)


def run_stream(name, spec, out):
    """Whole 2024 stream (23,030 messages) for the methods that need no model: counts only."""
    if spec[3] != "text" or name.split("_")[0] not in ("token", "fast", "sentence", "recursive"):
        return
    stream = C.load_stream()
    units = [(k, f"{m['day']} {m['ts_local']} {m['who']}: {m['text']}") for k, m in enumerate(stream)]
    text = "\n".join(u[1] for u in units)
    ch = spec[2]()
    t0 = time.time()
    rng = to_message_ranges(ch.chunk(text), units, len(stream))
    sizes = sorted(r["n"] for r in rng)
    rec = {"method": name, "n_messages": len(stream), "n_chunks": len(rng), "median_msgs": sizes[len(sizes) // 2],
           "max_msgs": sizes[-1], "seconds": round(time.time() - t0, 1)}
    (out / f"stream_{name}.json").write_text(json.dumps(rec), encoding="utf-8")
    print(f"stream {name}: {rec}", flush=True)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("outdir")
    ap.add_argument("--only", default="")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--force", action="store_true")
    ap.add_argument("--stream", action="store_true", help="also chunk the whole stream with the model-free methods")
    a = ap.parse_args()
    out = pathlib.Path(a.outdir)
    out.mkdir(parents=True, exist_ok=True)
    print(f"chonkie {chonkie.__version__}", flush=True)
    windows = load_windows(a.limit)
    ids = [i for w in windows for i in (w["msg_ids"] or [])]
    vecs = C.stored_vectors(ids)
    by_text = {}
    for w in windows:
        for t, i in zip(w["plain"], w["msg_ids"] or [None] * w["n"]):
            if i in vecs and t != "(attachment)":
                by_text[t] = vecs[i]
    print(f"{len(windows)} windows, {sum(w['n'] for w in windows)} messages, stored vectors for {len(by_text)} texts", flush=True)
    emb = C.StoredThenNim(by_text, out / "embed_cache.jsonl")
    methods = build_methods(emb)
    only = [x for x in a.only.split(",") if x]
    for name, spec in methods.items():
        if only and name not in only:
            continue
        if (out / f"{name}.json").exists() and not a.force:
            print(f"{name}: already done, skipped", flush=True)
            continue
        emb.reused = emb.fresh = emb.cached = 0
        print(f"== {name}", flush=True)
        run_method(name, spec, windows, out, emb)
        if name.startswith("semantic"):
            print(f"   vectors reused {emb.reused}, new {emb.fresh}, cached {emb.cached}", flush=True)
        if a.stream:
            run_stream(name, spec, out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
