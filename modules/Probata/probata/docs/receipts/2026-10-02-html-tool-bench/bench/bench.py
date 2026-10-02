# Byline: Claude Code · Sonnet · 2026-10-02
"""HTML tool bench. Usage:
  python bench.py run  <samples_dir> <results.jsonl> [--venv NAME=python ...]   run tools (each in a subprocess)
  python bench.py child <mode> <tool> <file> <out.json>                          (internal)
Scoring happens in child (needs the oracle) so parent stays tiny; one JSON line per (tool,file).
"""
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time
from collections import Counter

import psutil

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

FAMILY_RULES = [
    ("fb_messenger_thread", r"message_\d+\.html$"),
    ("fb_other_section", r"(fbsec|account_activity|your_friends|logins_and_logouts|/30\.html)"),
    ("google_voice", r"Voice/"),
    ("google_my_activity", r"MyActivity\.html$"),
    ("imessage_export", r"imessage exports"),
    ("whatsapp_chat", r"_chat\.html$"),
    ("number_named_export", r"Court & Legal Project"),
    ("snapchat_export", r"/snap/"),
    ("software_docs_and_misc", r".*"),
]


def family(path: str) -> str:
    p = path.replace("\\", "/")
    for name, rx in FAMILY_RULES:
        if re.search(rx, p):
            return name
    return "software_docs_and_misc"


def words(text: str) -> Counter:
    return Counter(w for w in re.findall(r"\w+", (text or "").lower()) if len(w) >= 2)


def recall(truth: Counter, got: Counter) -> float:
    tot = sum(truth.values())
    if not tot:
        return float("nan")
    return sum(min(v, got.get(k, 0)) for k, v in truth.items()) / tot


def refs_of(html: str) -> set[str]:
    from html.parser import HTMLParser

    out: set[str] = set()

    class P(HTMLParser):
        def handle_starttag(self, tag, attrs):
            for k, v in attrs:
                if k in ("href", "src") and v and not v.startswith(("data:", "#", "javascript:")):
                    out.add(v)

    P(convert_charrefs=True).feed(html)
    return out


def load_truth(path: str, html: str) -> dict:
    """Oracle facts for one file, computed once (the stdlib oracle takes minutes on a 60 MB page) and cached."""
    import hashlib

    import oracle

    cache_dir = os.path.join(HERE, "truth_cache")
    os.makedirs(cache_dir, exist_ok=True)
    key = hashlib.sha1((path.replace("\\", "/") + str(os.path.getsize(path)) + "v2").encode()).hexdigest()
    cache = os.path.join(cache_dir, key + ".json")
    if os.path.exists(cache):
        return json.load(open(cache, encoding="utf-8"))
    text = oracle.visible_text(html)
    truth = {
        "words": dict(words(text)),
        "emoji": dict(Counter(oracle.emoji_units(text))),
        "refs": sorted(refs_of(html)),
        "fb": [{"ts": b["ts"], "body": b["body"]} for b in oracle.fb_blocks(html) if b["ts"]] if re.search(r"message_\d+\.html$", path.replace("\\", "/")) else [],
    }
    json.dump(truth, open(cache, "w", encoding="utf-8"), ensure_ascii=False)
    return truth


def child(mode: str, tool: str, path: str, out_json: str) -> None:
    import oracle
    import tools

    t0 = time.time()
    fn = (tools.GENERIC if mode == "generic" else tools.FB)[tool]
    result = fn(path)
    elapsed = time.time() - t0
    html = open(path, "rb").read().decode("utf-8", "replace")
    row: dict = {"elapsed_s": round(elapsed, 3)}
    if mode == "generic":
        truth = load_truth(path, html)
        text = result if isinstance(result, str) else json.dumps(result, ensure_ascii=False)
        # Treat markdown/link syntax as noise-free: strip url targets before token counting
        text_nolinks = re.sub(r"\]\([^)]*\)", "]", text)
        tw = Counter(truth["words"])
        row["text_recall"] = round(recall(tw, words(text_nolinks)), 4)
        row["noise_ratio"] = round(sum(words(text_nolinks).values()) / max(1, sum(tw.values())), 3)
        te = Counter(truth["emoji"])
        ge = Counter(oracle.emoji_units(text))
        row["emoji_truth"] = sum(te.values())
        row["emoji_recall"] = round(sum(min(v, ge.get(k, 0)) for k, v in te.items()) / sum(te.values()), 4) if te else None
        refs = truth["refs"]
        row["refs_truth"] = len(refs)
        # Token-set lookup: a substring scan per reference is O(refs x text) and never finishes on a 60 MB page.
        tokens = set(re.findall(r"[^\s()\[\]<>\"'`]+", text))
        row["refs_recall"] = round(sum(1 for r in refs if r in tokens or os.path.basename(r) in tokens) / len(refs), 4) if refs else None
        row["out_chars"] = len(text)
        if truth.get("fb"):
            blocks = truth["fb"]
            ntext = oracle.norm(text)
            row["fb_msgs_truth"] = len(blocks)
            row["fb_body_found"] = round(sum(1 for b in blocks if b["body"] and b["body"] in ntext) / max(1, sum(1 for b in blocks if b["body"])), 4)
            row["fb_ts_found"] = round(sum(1 for b in blocks if b["ts"] in ntext) / max(1, len(blocks)), 4)
    else:
        blocks = oracle.fb_blocks(html)
        truth = [b for b in blocks if b["ts"]]
        got = [{**r, "body": oracle.norm(r["body"]), "sender": oracle.norm(r["sender"]), "ts": oracle.norm(r["ts"])} for r in result]
        row["records_out"] = len(got)
        row["blocks_truth_with_ts"] = len(truth)
        key = lambda r: (r["sender"], r["ts"], r["body"])  # noqa: E731
        gk = Counter(key(r) for r in got)
        tk = Counter(key(b) for b in truth)
        row["exact_match_recall"] = round(sum(min(v, gk.get(k, 0)) for k, v in tk.items()) / max(1, len(truth)), 4)
        row["sender_ok"] = round(sum(1 for b in truth if any(r["sender"] == b["sender"] for r in got[:0])) , 4)  # placeholder, replaced below
        gts = Counter((r["ts"], r["body"]) for r in got)
        tts = Counter((b["ts"], b["body"]) for b in truth)
        row["ts_body_recall"] = round(sum(min(v, gts.get(k, 0)) for k, v in tts.items()) / max(1, len(truth)), 4)
        gsb = Counter((r["sender"], r["body"]) for r in got)
        tsb = Counter((b["sender"], b["body"]) for b in truth)
        row["sender_body_recall"] = round(sum(min(v, gsb.get(k, 0)) for k, v in tsb.items()) / max(1, len(truth)), 4)
        del row["sender_ok"]
        emo = [b for b in truth if oracle.emoji_units(b["body"])]
        row["emoji_msgs_truth"] = len(emo)
        gb = Counter((r["ts"], r["body"]) for r in got)
        row["emoji_msg_exact"] = round(sum(1 for b in emo if gb.get((b["ts"], b["body"]), 0) > 0) / len(emo), 4) if emo else None
        react = [u for b in blocks for u in oracle.emoji_units(" ".join(b["reactions"]))]
        gr = Counter(u for r in got for u in oracle.emoji_units(" ".join(r["reactions"]) if isinstance(r["reactions"], list) else ""))
        tr = Counter(react)
        row["reaction_emoji_truth"] = sum(tr.values())
        row["reaction_emoji_recall"] = round(sum(min(v, gr.get(k, 0)) for k, v in tr.items()) / sum(tr.values()), 4) if tr else None
        att = sum(len(b["attach"]) for b in truth)
        row["attach_truth"] = att
        row["attach_out"] = sum(len(r["attach"]) if isinstance(r["attach"], list) else (1 if r["attach"] else 0) for r in got)
    json.dump(row, open(out_json, "w"))


def run(samples: str, results: str, tool_list: list[tuple[str, str, str]], timeout: int) -> None:
    files = []
    for root, _, fs in os.walk(samples):
        for f in fs:
            if f.lower().endswith((".html", ".htm")):
                files.append(os.path.join(root, f))
    files.sort(key=os.path.getsize)
    done = set()
    if os.path.exists(results):
        for line in open(results, encoding="utf-8"):
            r = json.loads(line)
            done.add((r["file"], r["tool"], r["mode"]))
    with open(results, "a", encoding="utf-8") as out:
        for path in files:
            fam = family(path)
            size = os.path.getsize(path)
            for mode, tool, py in tool_list:
                if mode == "fb" and fam != "fb_messenger_thread":
                    continue
                if (os.path.relpath(path, samples).replace("\\", "/")[-90:], tool, mode) in done:
                    continue
                tmp = results + ".tmp.json"
                if os.path.exists(tmp):
                    os.remove(tmp)
                t0 = time.time()
                proc = subprocess.Popen([py, os.path.join(HERE, "bench.py"), "child", mode, tool, path, tmp],
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding="utf-8", errors="replace")
                peak = 0
                capped = False
                try:
                    ps = psutil.Process(proc.pid)
                    while proc.poll() is None:
                        try:
                            rss = ps.memory_info().rss + sum(c.memory_info().rss for c in ps.children(recursive=True))
                            peak = max(peak, rss)
                        except (psutil.Error, OSError):
                            pass
                        if peak > 6 * 1024 ** 3:
                            capped = True
                            proc.kill()
                            break
                        if time.time() - t0 > timeout:
                            proc.kill()
                            break
                        time.sleep(0.02)
                except (psutil.Error, OSError):
                    pass
                stdout, stderr = proc.communicate()
                rec = {"family": fam, "file": os.path.relpath(path, samples).replace("\\", "/")[-90:], "bytes": size, "mode": mode, "tool": tool,
                       "wall_s": round(time.time() - t0, 2), "peak_rss_mb": round(peak / 1048576)}
                if capped:
                    rec["status"] = "memory_cap_6GB"
                elif time.time() - t0 > timeout:
                    rec["status"] = "timeout"
                elif proc.returncode != 0 or not os.path.exists(tmp):
                    rec["status"] = "error"
                    rec["error"] = (stderr or "").strip().splitlines()[-1][:200] if stderr else "no output"
                else:
                    rec["status"] = "ok"
                    rec.update(json.load(open(tmp)))
                out.write(json.dumps(rec, ensure_ascii=False) + "\n")
                out.flush()
                print(rec["family"], rec["tool"], rec["status"], rec.get("wall_s"), rec.get("peak_rss_mb"), flush=True)


if __name__ == "__main__":
    if sys.argv[1] == "child":
        child(*sys.argv[2:6])
    else:
        samples, results = sys.argv[2], sys.argv[3]
        timeout = int(os.environ.get("BENCH_TIMEOUT", "300"))
        spec = json.load(open(sys.argv[4]))  # [[mode, tool, python], ...]
        run(samples, results, [tuple(x) for x in spec], timeout)
