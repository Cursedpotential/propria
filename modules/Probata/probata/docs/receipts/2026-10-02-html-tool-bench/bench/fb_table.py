# Byline: Claude Code · Sonnet · 2026-10-02
"""Markdown table of the Facebook-specific (structured) tools over the real Messenger thread files."""
import json
import statistics
import sys
from collections import defaultdict

rows = defaultdict(list)
seen = set()
for path in sys.argv[1:]:
    for line in open(path, encoding="utf-8"):
        r = json.loads(line)
        if r["mode"] != "fb":
            continue
        k = (r["file"], r["tool"])
        if k in seen:
            continue
        seen.add(k)
        rows[r["tool"]].append(r)

NAMES = {
    "webbed_xpath_template": "DuckDB webbed XPath template (facebook_messenger_html_v1 query shape)",
    "lxml_xpath": "lxml + XPath on the `_a6-*` cards",
    "selectolax_css": "selectolax (lexbor) + CSS on the `_a6-*` cards",
    "bs4_css": "BeautifulSoup4 + CSS on the `_a6-*` cards",
    "casebible_regex_template_v1": "Case Bible `elt_fb_messenger_html_v1` (DuckDB regex, 2026-09-18)",
    "repo_python_port": "repo `facebook_messenger_html.py` before the 2026-10-02 fix (port of dial-stack FacebookExportParser)",
    "chat_history_manager": "realdeveloperongithub/chat-history-manager `MessengerParser` selectors (GPL-3.0)",
}
print("| tool | files ok/total | messages exact (sender+time+body) | sender+body only | emoji messages exact | reaction emoji recall | media refs out vs truth | median s | peak RSS MB |")
print("|---|---|---|---|---|---|---|---|---|")
out = []
for tool, rs in rows.items():
    ok = [r for r in rs if r["status"] == "ok"]

    def avg(key):
        vals = [r[key] for r in ok if r.get(key) is not None]
        return statistics.mean(vals) if vals else None

    ex = avg("exact_match_recall") if ok else 0
    out.append((ex or 0, tool, f"| {NAMES.get(tool, tool)} | {len(ok)}/{len(rs)} | {('%.3f' % ex) if ex is not None else 'n/a'} | {('%.3f' % avg('sender_body_recall')) if ok else 'n/a'} | "
                f"{('%.3f' % avg('emoji_msg_exact')) if avg('emoji_msg_exact') is not None else 'n/a'} | "
                f"{('%.3f' % avg('reaction_emoji_recall')) if avg('reaction_emoji_recall') is not None else 'n/a'} | "
                f"{sum(r.get('attach_out', 0) for r in ok)}/{sum(r.get('attach_truth', 0) for r in ok)} | "
                f"{statistics.median([r['elapsed_s'] if 'elapsed_s' in r else r['wall_s'] for r in rs]):.2f} | {max(r['peak_rss_mb'] for r in rs)} |"))
for _, _, line in sorted(out, key=lambda t: -t[0]):
    print(line)
