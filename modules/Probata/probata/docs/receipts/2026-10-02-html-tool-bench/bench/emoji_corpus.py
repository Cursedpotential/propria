import glob, re, sys
from collections import Counter

sys.path.insert(0, ".")
import oracle

tot = Counter()
examples = {}
for f in sorted(glob.glob("../samples/**/message_*.html", recursive=True)) + glob.glob("../samples/**/_chat.html", recursive=True):
    html = open(f, encoding="utf-8").read()
    blocks = oracle.fb_blocks(html) if "message_" in f else []
    texts = [b["body"] for b in blocks] + [r for b in blocks for r in b["reactions"]]
    if not texts:
        texts = [oracle.visible_text(html)]
    for t in texts:
        for u in oracle.emoji_units(t):
            kind = "zwj_sequence" if "‍" in u else "skin_tone" if re.search("[\U0001F3FB-\U0001F3FF]", u) else "flag" if re.fullmatch("[\U0001F1E6-\U0001F1FF]{2}", u) else "keycap" if "⃣" in u else "vs16_symbol" if "️" in u else "astral_pictograph" if ord(u[0]) > 0xFFFF else "bmp_symbol"
            tot[kind] += 1
            examples.setdefault(kind, set())
            if len(examples[kind]) < 4:
                examples[kind].add(u)
    # accented letters (Latin-1 supplement and Latin Extended)
    tot["accented_letters"] += sum(len(re.findall("[À-ɏ]", t)) for t in texts)
    tot["curly_quotes_dashes"] += sum(len(re.findall("[‘’“”–—]", t)) for t in texts)
print(dict(tot))
print({k: sorted(v) for k, v in examples.items()})
