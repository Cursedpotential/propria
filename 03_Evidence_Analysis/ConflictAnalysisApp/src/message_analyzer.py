#  use this code to anaklyze the files i give you.. if you can make improvememnts or expand it do so

# Let's verify which files exist and (re)generate missing ones end-to-end.
import os, re, json
import pandas as pd
import numpy as np
from pathlib import Path
from zipfile import ZipFile, ZIP\DEFLATED
# -------------------------------
# Helper functions
# -------------------------------
def load\_csv\_any(path: Path):
    try:
        return pd.read\_csv(path)
    except Exception:
        try:
            return pd.read\_csv(path, sep=";")
        except Exception:
            return pd.read\_csv(path, engine="python")
def normalize\_ts\_any(col):
    dt = pd.to\_datetime(col, errors="coerce", utc=True)
    return dt.dt.tz\_convert("UTC").dt.tz\_localize(None)
def only\_digits(s):
    return re.sub(r"\\D+", "", str(s))
def to\_e164\_candidates(num\_str):
    d = only\_digits(num\_str)
    cands = set()
    if not d:
        return cands
    if len(d) == 10:
        cands.update({d, "+1"+d, "+1" + d, "+" + "1" + d})
    elif len(d) == 11 and d.startswith("1"):
        cands.update({d, "+"+d, d[1:], "+1"+d[1:]})
    else:
        cands.update({d, "+"+d})
    return cands
# -------------------------------
# Paths & whitelists
# -------------------------------
pdf\_path = Path("/mnt/data/messages\_from\_pdfs\_2and3\_eastern\_clean\_RERUN.csv")
sms\_path = Path("/mnt/data/sms\_export Matt & Katrina.csv")
fb\_path  = Path("/mnt/data/Combine FB Messages - combined\_messages (2).csv")
# number whitelists (user-specified)
raw\_number\_map = {
    "8102959302": "Matthew",
    "8102959303": "Katrina",
    "8103533592": "Katrina",
    "8108532989": "Katrina",
    "8102689630": "Katrina",
}
num\_to\_party = {}
for n, party in raw\_number\_map.items():
    for cand in to\_e164\_candidates(n):
        num\_to\_party[cand] = party
# alias list for Katrina
katrina\_aliases = [
    "katrina",
    "baby mama",
    "baby mother",
    "baby mother.",
    "baby-mama",
    "big spoon",
]
# -------------------------------
# Rebuild CORPUS from source files
# -------------------------------
# PDF-derived
df\_pdf = load\_csv\_any(pdf\_path)
df\_pdf["ts"] = normalize\_ts\_any(df\_pdf.get("timestamp\_eastern", df\_pdf.get("timestamp", pd.NaT)))
df\_pdf["timestamp\_display"] = df\_pdf.get("timestamp\_eastern", df\_pdf.get("timestamp", ""))
df\_pdf["source\_file"] = pdf\_path.name
df\_pdf["idx\_in\_source"] = df\_pdf.index
def norm\_pdf\_sender(s):
    s = str(s).strip().lower()
    if s == "katrina" or any(a in s for a in katrina\_aliases):
        return "Katrina"
    if s in {"matt","matthew","me"}:
        return "Matthew"
    return "Unknown"
df\_pdf["sender\_norm"] = df\_pdf.get("sender", "Unknown").map(norm\_pdf\_sender)
df\_pdf["message\_norm"] = df\_pdf.get("message", "").astype(str)
# SMS (owner = Katrina)
df\_sms\_raw = load\_csv\_any(sms\_path)
records = []
current\_conv = None
for raw\_idx, row in df\_sms\_raw.iterrows():
    col0 = str(row.iloc[0])
    if col0.startswith("Conversation with +") or col0.startswith("Conversation with "):
        current\_conv = col0.replace("Conversation with ", "").strip()
        continue
    if str(row.iloc[0]).lower() == "address" and str(row.get("Unnamed: 1","")).lower() == "readable\_date":
        continue
    body = row.get("Unnamed: 3")
    if pd.isna(body):
        continue
    rec = {
        "src\_row\_idx": raw\_idx,
        "conversation\_with": current\_conv,
        "readable\_date": row.get("Unnamed: 1", np.nan),
        "type": row.get("Unnamed: 2", np.nan), # 'Sent'/'Received' relative to Katrina's device
        "body": str(body),
        "read": row.get("Unnamed: 4", np.nan),
        "contact\_name": row.get("Unnamed: 5", np.nan),
    }
    records.append(rec)
df\_sms = pd.DataFrame(records)
df\_sms["ts"] = normalize\_ts\_any(df\_sms["readable\_date"])
df\_sms["timestamp\_display"] = df\_sms["readable\_date"].astype(str)
df\_sms["source\_file"] = sms\_path.name
df\_sms["idx\_in\_source"] = df\_sms.index
# Baseline: owner is Katrina
df\_sms["sender\_norm"] = np.where(df\_sms["type"].astype(str).str.lower()=="sent", "Katrina", "Matthew")
# Override with whitelist when the conversation partner is known
def override\_sms\_sender(row):
    conv = str(row.get("conversation\_with",""))
    party = num\_to\_party.get(conv) or num\_to\_party.get(only\_digits(conv)) or num\_to\_party.get("+1"+only\_digits(conv))
    if party == "Matthew" and str(row.get("type","")).lower()=="received":
        return "Matthew"
    if party == "Katrina" and str(row.get("type","")).lower()=="received":
        return "Katrina"
    return row["sender\_norm"]
df\_sms["sender\_norm"] = df\_sms.apply(override\_sms\_sender, axis=1)
df\_sms["message\_norm"] = df\_sms["body"].astype(str)
# FB combined
if fb\_path.exists():
    df\_fb = load\_csv\_any(fb\_path)
    lc = {c.lower(): c for c in df\_fb.columns}
    sender\_col = lc.get("sender\_name") or lc.get("sender") or lc.get("from") or list(df\_fb.columns)[0]
    content\_col = lc.get("content") or lc.get("message") or lc.get("body") or list(df\_fb.columns)[-1]
    ts\_col = lc.get("timestamp\_ms") or lc.get("timestamp") or lc.get("date") or lc.get("time")
    df\_fb["sender\_name\_norm"] = df\_fb[sender\_col].astype(str)
    df\_fb["message\_norm"] = df\_fb[content\_col].astype(str)
    if ts\_col:
        if ts\_col.lower()=="timestamp\_ms":
            df\_fb["ts"] = pd.to\_datetime(df\_fb[ts\_col], unit="ms", errors="coerce", utc=True).dt.tz\_convert("UTC").dt.tz\_localize(None)
            df\_fb["timestamp\_display"] = pd.to\_datetime(df\_fb[ts\_col], unit="ms", errors="coerce").astype(str)
        else:
            df\_fb["ts"] = normalize\_ts\_any(df\_fb[ts\_col])
            df\_fb["timestamp\_display"] = df\_fb[ts\_col].astype(str)
    else:
        df\_fb["ts"] = pd.NaT
        df\_fb["timestamp\_display"] = ""
    df\_fb["source\_file"] = fb\_path.name
    df\_fb["idx\_in\_source"] = df\_fb.index
    def fb\_sender\_map(name):
        s = str(name).strip().lower()
        if any(alias in s for alias in katrina\_aliases) or "katrina" in s:
            return "Katrina"
        return "Matthew"
    df\_fb["sender\_norm"] = df\_fb["sender\_name\_norm"].map(fb\_sender\_map)
else:
    df\_fb = pd.DataFrame(columns=["ts","timestamp\_display","sender\_norm","message\_norm","source\_file","idx\_in\_source"])
# Build CORPUS
pdf\_cols = ["ts","timestamp\_display","sender\_norm","message\_norm","source\_file","idx\_in\_source"]
sms\_cols = ["ts","timestamp\_display","sender\_norm","message\_norm","source\_file","idx\_in\_source"]
fb\_cols  = ["ts","timestamp\_display","sender\_norm","message\_norm","source\_file","idx\_in\_source"]
CORPUS = pd.concat([df\_pdf[pdf\_cols], df\_sms[sms\_cols], df\_fb[fb\_cols]], ignore\_index=True)
CORPUS = CORPUS.sort\_values("ts").reset\_index(drop=True)
# Save dataset backup
backup\_csv = Path("/mnt/data/corpus\_master\_backup.csv")
CORPUS.rename(columns={"ts":"timestamp","sender\_norm":"sender","message\_norm":"message"}).to\_csv(backup\_csv, index=False)
# -------------------------------
# Rebuild special passes
# -------------------------------
# A) Death-wish phrases (Katrina only)
die\_rx = re.compile("|".join([
    r"\\bi\\s+hope\\s+you\\s+die\\b",
    r"\\bi\\s+wish\\s+you\\s+were\\s+dead\\b",
    r"\\bgo\\s+die\\b",
    r"\\bdrop\\s+dead\\b",
    r"\\byou\\s+should\\s+die\\b",
    r"\\bi\\s+want\\s+you\\s+dead\\b",
    r"\\bkill\\s+yourself\\b",
    r"\\byou\\s+deserve\\s+to\\s+die\\b",
    r"\\bwish\\s+you\\s+were\\s+dead\\b",
    r"\\bhope\\s+u\\s+die\\b",
]), re.I)
die\_hits = CORPUS[(CORPUS["sender\_norm"]=="Katrina") & (CORPUS["message\_norm"].astype(str).str.contains(die\_rx))].copy()
die\_hits = die\_hits.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
die\_hits["matched\_text"] = die\_hits["message"].str.extract(re.compile("(" + die\_rx.pattern + ")", re.I), expand=False)
die\_hits = die\_hits[["timestamp","sender","message","matched\_text","source\_file","idx\_in\_source"]].sort\_values("timestamp")
die\_summary = pd.DataFrame({"metric":["told\_me\_to\_die\_count"], "count":[len(die\_hits)]})
# B) Blocking mentions
block\_rx = re.compile("|".join([
    r"\\bblock(?:ed|ing)?\\b",
    r"\\bi'?m\\s+going\\s+to\\s+block\\b",
    r"\\bi\\s+blocked\\s+you\\b",
    r"\\bunblock(?:ed)?\\b",
    r"\\bmute(?:d)?\\b|\\bmuting\\b",
    r"\\bstop\\s+texting\\s+me\\b",
    r"\\bdo\\s+not\\s+text\\b|\\bdon'?t\\s+text\\b",
    r"\\bchange\\s+my\\s+number\\b",
    r"\\bi\\s+changed\\s+my\\s+number\\b",
]), re.I)
block\_hits = CORPUS[(CORPUS["sender\_norm"]=="Katrina") & (CORPUS["message\_norm"].astype(str).str.contains(block\_rx))].copy()
block\_hits = block\_hits.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
block\_hits["matched\_text"] = block\_hits["message"].str.extract(re.compile("(" + block\_rx.pattern + ")", re.I), expand=False)
block\_hits = block\_hits[["timestamp","sender","message","matched\_text","source\_file","idx\_in\_source"]].sort\_values("timestamp")
block\_summary = pd.DataFrame({"metric":["blocking\_mentions\_count"], "count":[len(block\_hits)]})
# C) Alienation/Gatekeeping (Katrina only)
kid\_names = r"(?:k(?:aila|yla|ylah|ailah))"
alienate\_rx = re.compile("|".join([
    rf"\\byou\\s+won'?t\\s+see\\s+(?:her|your\\s+daughter|{kid\_names})\\b",
    rf"\\bi'?m\\s+not\\s+bringing\\s+(?:her|{kid\_names})\\b",
    rf"\\byou\\s+can'?t\\s+have\\s+(?:her|{kid\_names})\\b",
    rf"\\bi'?m\\s+keeping\\s+(?:her|{kid\_names})\\b",
    r"\\bno\\s+visit(?:ation)?\\b",
    r"\\bnot\\s+your\\s+day\\b",
    r"\\bi'?m\\s+cancell?ing\\b.\*\\b(?:visit|time|drop\\s\*off|pick\\s\*up|custody|parenting)\\b",
    r"\\bi\\s+changed\\s+my\\s+mind\\b.\*\\b(?:visit|time|drop\\s\*off|pick\\s\*up|seeing)\\b",
]), re.I)
alienate\_hits = CORPUS[(CORPUS["sender\_norm"]=="Katrina") & (CORPUS["message\_norm"].astype(str).str.contains(alienate\_rx))].copy()
alienate\_hits = alienate\_hits.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
alienate\_hits["matched\_text"] = alienate\_hits["message"].str.extract(re.compile("(" + alienate\_rx.pattern + ")", re.I), expand=False)
alienate\_hits = alienate\_hits[["timestamp","sender","message","matched\_text","source\_file","idx\_in\_source"]].sort\_values("timestamp")
alienate\_summary = pd.DataFrame({"metric":["alienation\_gatekeeping\_count"], "count":[len(alienate\_hits)]})
# D) Requests to see child (Matthew only)
request\_rx = re.compile("|".join([
    rf"\\b(?:can|could|may)\\s+i\\s+(?:see|have|get)\\s+(?:her|my\\s+(?:baby|daughter)|{kid\_names})\\b",
    rf"\\bwhen\\s+can\\s+i\\s+(?:see|have|get)\\s+(?:her|my\\s+(?:baby|daughter)|{kid\_names})\\b",
    rf"\\bbring\\s+(?:her|{kid\_names})\\b",
    r"\\bdrop\\s+her\\s+off\\b|\\bdropoff\\b",
    r"\\bpick\\s\*up\\b|\\bpickup\\b",
    r"\\b(?:my\\s+parenting\\s+time|my\\s+time)\\b",
    rf"\\bsee\\s+(?:her|my\\s+(?:baby|daughter)|{kid\_names})\\b",
    rf"\\bvisit(?:ation)?\\b.\*\\b(?:her|my\\s+(?:baby|daughter)|{kid\_names})\\b",
]), re.I)
request\_hits = CORPUS[(CORPUS["sender\_norm"]=="Matthew") & (CORPUS["message\_norm"].astype(str).str.contains(request\_rx))].copy()
request\_hits = request\_hits.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
request\_hits["matched\_text"] = request\_hits["message"].str.extract(re.compile("(" + request\_rx.pattern + ")", re.I), expand=False)
request\_hits = request\_hits[["timestamp","sender","message","matched\_text","source\_file","idx\_in\_source"]].sort\_values("timestamp")
request\_summary = pd.DataFrame({"metric":["requests\_to\_see\_child\_count"], "count":[len(request\_hits)]})
# E) Insults / cut-downs (Katrina only)
insult\_terms = {
    r"\\bain['’]?\\s\*t\\s+shit\\b": "ain't shit",
    r"\\bbitch[-\\s]\*made\\b": "bitch made",
    r"\\bfag+(?:ot|got|g[ao]t)?\\b": "faggot/fagget/fag",
    r"\\bpiece\\s+of\\s+shit\\b": "piece of shit",
    r"\\bstupid\\s+mother\\s\*fuck(?:er|a)\\b": "stupid motherfucker (var.)",
    r"\\bloser\\b": "loser",
}
insult\_rx = re.compile("|".join(insult\_terms.keys()), re.I)
insult\_hits = CORPUS[(CORPUS["sender\_norm"]=="Katrina") & (CORPUS["message\_norm"].astype(str).str.contains(insult\_rx))].copy()
insult\_hits = insult\_hits.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
def extract\_label(msg):
    for pat, label in insult\_terms.items():
        if re.search(pat, str(msg), re.I):
            return label
    return None
insult\_hits["insult\_label"] = insult\_hits["message"].apply(extract\_label)
insult\_hits = insult\_hits[["timestamp","sender","message","insult\_label","source\_file","idx\_in\_source"]].sort\_values("timestamp")
insult\_counts = insult\_hits.groupby("insult\_label")["message"].count().reset\_index(name="count").sort\_values("count", ascending=False)
insult\_total = pd.DataFrame({"metric":["insult\_total\_count"], "count":[len(insult\_hits)]})
# F) Court interactions (both sides) + contexts
court\_rx = re.compile("|".join([
    r"\\bcourt\\b",
    r"\\bfoc\\b", r"\\bfriend\\s+of\\s+the\\s+court\\b",
    r"\\bjudge\\b",
    r"\\border\\b", r"\\bparenting\\s\*time\\s\*order\\b",
    r"\\bcustody\\b",
    r"\\bhearing\\b",
    r"\\bcontempt\\b",
    r"\\bmotion\\b",
    r"\\bcase\\s\*(?:no\\.?|number)?\\b",
    r"\\battorney\\b", r"\\blawyer\\b",
    r"\\bmediator\\b",
    r"\\bguardian\\s+ad\\s+litem\\b",
    r"\\bsubpoena\\b",
    r"\\bdocket\\b",
    r"\\bprobation\\b",
    r"\\bppo\\b|\\bprotection\\s+order\\b",
]), re.I)
court\_hits = CORPUS[CORPUS["message\_norm"].astype(str).str.contains(court\_rx)].copy()
court\_hits = court\_hits.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
court\_hits["matched\_text"] = court\_hits["message"].str.extract(re.compile("(" + court\_rx.pattern + ")", re.I), expand=False)
court\_hits = court\_hits[["timestamp","sender","message","matched\_text","source\_file","idx\_in\_source"]].sort\_values("timestamp")
def make\_context(df\_hits, C\_base, window\_messages=15, window\_minutes=15):
    ctx\_msg\_rows, ctx\_time\_rows = [], []
    for \_, h in df\_hits.iterrows():
        mask = (
            (C\_base["ts"]==h["timestamp"]) &
            (C\_base["source\_file"]==h["source\_file"]) &
            (C\_base["idx\_in\_source"]==h["idx\_in\_source"]) &
            (C\_base["message\_norm"]==h["message"])
        )
        if not mask.any():
            mask = (
                (C\_base["ts"]==h["timestamp"]) &
                (C\_base["source\_file"]==h["source\_file"]) &
                (C\_base["idx\_in\_source"]==h["idx\_in\_source"])
            )
        if not mask.any():
            continue
        i = mask[mask].index[0]
        # message window
        start = max(0, i - window\_messages)
        end   = min(len(C\_base)-1, i + window\_messages)
        block = C\_base.iloc[start:end+1].copy()
        block["is\_hit\_line"] = False
        block.loc[i, "is\_hit\_line"] = True
        block = block.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
        block\_out = block[[
            "timestamp","timestamp\_display","sender","message","source\_file","idx\_in\_source","is\_hit\_line"
        ]].copy()
        block\_out["hit\_index"] = i
        ctx\_msg\_rows.append(block\_out)
        # time window
        ts0 = h["timestamp"]
        wstart, wend = ts0 - pd.Timedelta(minutes=window\_minutes), ts0 + pd.Timedelta(minutes=window\_minutes)
        tblock = C\_base[(C\_base["ts"]>=wstart) & (C\_base["ts"]<=wend)].copy()
        tblock["is\_hit\_line"] = (
            (tblock["ts"]==h["timestamp"]) &
            (tblock["source\_file"]==h["source\_file"]) &
            (tblock["idx\_in\_source"]==h["idx\_in\_source"]) &
            (tblock["message\_norm"]==h["message"])
        )
        tblock = tblock.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
        tblock\_out = tblock[[
            "timestamp","timestamp\_display","sender","message","source\_file","idx\_in\_source","is\_hit\_line"
        ]].copy()
        tblock\_out["hit\_index"] = i
        ctx\_time\_rows.append(tblock\_out)
    ctx\_msg\_df = pd.concat(ctx\_msg\_rows, ignore\_index=True) if ctx\_msg\_rows else pd.DataFrame()
    ctx\_time\_df = pd.concat(ctx\_time\_rows, ignore\_index=True) if ctx\_time\_rows else pd.DataFrame()
    return ctx\_msg\_df, ctx\_time\_df
court\_ctx\_msg, court\_ctx\_time = make\_context(court\_hits, CORPUS, 15, 15)
# -------------------------------
# Save the requested workbooks
# -------------------------------
# special\_passes.xlsx (death/block/alienation/requests)
special\_xlsx = Path("/mnt/data/court\_special\_passes.xlsx")
with pd.ExcelWriter(special\_xlsx, engine="xlsxwriter") as writer:
    summary = pd.DataFrame({
        "metric": ["told\_me\_to\_die\_count","blocking\_mentions\_count","alienation\_gatekeeping\_count","requests\_to\_see\_child\_count"],
        "count": [len(die\_hits), len(block\_hits), len(alienate\_hits), len(request\_hits)]
    })
    summary.to\_excel(writer, index=False, sheet\_name="Summary")
    die\_hits.to\_excel(writer, index=False, sheet\_name="ToldMeToDie")
    block\_hits.to\_excel(writer, index=False, sheet\_name="BlockingMentions")
    alienate\_hits.to\_excel(writer, index=False, sheet\_name="Alienation\_ByKatrina")
    request\_hits.to\_excel(writer, index=False, sheet\_name="Requests\_ByMatthew")
# special\_passes\_v2.xlsx (insults + court)
special\_v2 = Path("/mnt/data/court\_special\_passes\_v2.xlsx")
with pd.ExcelWriter(special\_v2, engine="xlsxwriter") as writer:
    # insults
    insult\_hits.to\_excel(writer, index=False, sheet\_name="Insults\_ByKatrina")
    insult\_counts.to\_excel(writer, index=False, sheet\_name="Insults\_Counts")
    insult\_total.to\_excel(writer, index=False, sheet\_name="Insults\_Total")
    # court (hits + contexts)
    court\_hits.to\_excel(writer, index=False, sheet\_name="Court\_Hits\_AllSenders")
    court\_ctx\_msg.to\_excel(writer, index=False, sheet\_name="Court\_Context\_Messages")
    court\_ctx\_time.to\_excel(writer, index=False, sheet\_name="Court\_Context\_Time")
# -------------------------------
# Recreate the earlier line-by-line labeled workbook (±5) and WIDE (±15)
# -------------------------------
# Minimal behavior rules for evidence labeling
RULES = [
    # 2.1 Denial
    {"rule\_id":"2.1","category":"Reality Distortion & Memory Revisionism","name":"Categorical Denial",
     "patterns":[
         r"really\\s+known\\s+you\\s+7\\s+years\\s+never\\s+locked\\s+a\\s+door\\s+in\\s+7\\s+years",
         r"didn.?t\\s+lock\\s+the\\s+son\\s+of\\s+a\\s+bitch\\s+the\\s+whole\\s+time\\s+i\\s+was\\s+there",
         r"\\bnever\\s+locked\\s+a\\s+door\\b",
         r"\\b(lock(ed)?|door)\\b.\*\\bnever\\b",
     ]},
    # 2.2 Blame reframe
    {"rule\_id":"2.2","category":"Reality Distortion & Memory Revisionism","name":"Subtle Shift / Blame Reframe",
     "patterns":[
         r"did\\s+you\\s+tell\\s+me\\s+to\\s+be\\s+home\\s+at\\s+a\\s+certain\\s+time\\s+no\\s+no\\s+you\\s+fucking\\s+didn.?t",
         r"did\\s+you\\s+tell\\s+me\\s+to\\s+be\\s+home.\*(time)?",
         r"did\\s+you\\s+tell\\s+me.\*(deadline|time)",
         r"you\\s+didn.?t\\s+tell\\s+me",
         r"that.?s\\s+your\\s+fault",
         r"if\\s+you\\s+had\\s+just",
         r"because\\s+you\\s+didn.?t",
     ]},
    # 3.1 substance accusation
    {"rule\_id":"3.1","category":"Attacking Credibility & Sanity","name":"Substance Accusation ('Crazy' via Drug/Alcohol)",
     "patterns":[
         r"you.?re\\s+a\\s+methed\\s+up\\s+piece\\s+of\\s+shit",
         r"maybe\\s+one\\s+day\\s+when\\s+you.?re\\s+off\\s+the\\s+meth\\s+we\\s+can\\s+have\\s+a\\s+conversation",
         r"what\\s+hurts\\s+is\\s+that\\s+you.?re\\s+still\\s+high",
         r"\\bmeth(ed)?\\b",
         r"\\byou.?re\\s+high\\b",
         r"\\boff\\s+the\\s+meth\\b",
         r"\\btweaker\\b",
         r"\\bpill.?head\\b",
     ]},
    # 4.1 provocation defense
    {"rule\_id":"4.1","category":"Shifting Blame & Evading Responsibility","name":"Provocation Defense",
     "patterns":[
         r"you\\s+wanna\\s+call\\s+me\\s+a\\s+cunt\\s+so\\s+i.?m\\s+gonna\\s+show\\s+you\\s+a\\s+fucking\\s+cunt",
         r"\\byou\\s+hung\\s+up\\s+first\\b",
         r"\\byou\\s+were\\s+rude\\b",
     ]},
    # 5.1 threats
    {"rule\_id":"5.1","category":"Psychological Control & Manipulation","name":"Volatility Cycle (Hot/Cold) — THREAT",
     "patterns":[
         r"i\\s+will\\s+literally\\s+remove\\s+your\\s+fucking\\s+testicles\\s+and\\s+let\\s+you\\s+bleed\\s+to\\s+death",
         r"i\\s+will\\s+remove\\s+your\\s+.\*testicles.\*bleed\\s+to\\s+death",
         r"beat\\s+the\\s+living\\s+shit\\s+out\\s+of\\s+you",
         r"(kill|hurt|hit|beat)\\s+you",
         r"i(?:'m| am)?\\s+going\\s+to\\s+(kill|hurt|hit|beat)\\s+you",
         r"i(?:'m| am)?\\s+going\\s+to\\s+.\*\\byou\\b",
     ]},
    # 5.2 abandonment
    {"rule\_id":"5.2","category":"Psychological Control & Manipulation","name":"Threat of Abandonment",
     "patterns":[
         r"i\\s+might\\s+not\\s+come\\s+tonight\\s+so\\s+don.?t\\s+get\\s+your\\s+hopes\\s+up",
         r"yeah\\s+i.?m\\s+just\\s+going\\s+home\\s+you\\s+have\\s+a\\s+good\\s+night",
         r"\\bi\\s+might\\s+not\\s+come\\b",
         r"don.?t\\s+get\\s+your\\s+hopes\\s+up",
         r"\\bi.?m\\s+just\\s+going\\s+home\\b",
         r"\\bi.?m\\s+leaving\\b|\\bi.?m\\s+done\\b|\\bit.?s\\s+over\\b",
         r"\\bnot\\s+coming\\s+back\\b|\\bnot\\s+coming\\s+home\\b",
         r"\\bchange\\s+my\\s+number\\b",
         r"\\bblock(ing|ed)?\\b|\\bmute\\b|\\bstop\\s+texting\\s+me\\b",
     ]},
    # 5.3 goalposts
    {"rule\_id":"5.3","category":"Psychological Control & Manipulation","name":"Moving the Goalposts / Circular Conversation",
     "patterns":[
         r"that.?s\\s+not\\s+the\\s+point",
         r"we.?re\\s+talking\\s+about\\s+something\\s+else",
         r"you\\s+didn.?t\\s+say\\s+that",
         r"you\\s+should.?ve\\s+(told|said)",
     ]},
    # X.1 hostile shutdown
    {"rule\_id":"X.1","category":"Dismissive Command / Shutdown","name":"Hostile Command or Contempt (explicit)",
     "patterns":[
         r"shut\\s+the\\s+fuck\\s+up",
         r"shut\\s+up",
         r"\\bsounds\\s+good\\b",
     ]},
    # X.2 parental leverage
    {"rule\_id":"X.2","category":"Parental Leverage / Alienation","name":"Using Child as Leverage",
     "patterns":[
         rf"\\byou\\s+won'?t\\s+see\\s+(?:her|your\\s+daughter|{kid\_names})\\b",
         rf"\\bi'?m\\s+not\\s+bringing\\s+(?:her|{kid\_names})\\b",
         rf"\\byou\\s+can'?t\\s+have\\s+(?:her|{kid\_names})\\b",
         rf"\\bi'?m\\s+keeping\\s+(?:her|{kid\_names})\\b",
     ]},
]
compiled\_rules = [(r["rule\_id"], r["category"], r["name"], [re.compile(p, re.I) for p in r["patterns"]]) for r in RULES]
def label\_hits(corpus):
    rows = []
    corpus = corpus.copy()
    corpus["prev\_msg"] = corpus["message\_norm"].shift(1)
    corpus["prev\_sender"] = corpus["sender\_norm"].shift(1)
    corpus["next\_msg"] = corpus["message\_norm"].shift(-1)
    corpus["next\_sender"] = corpus["sender\_norm"].shift(-1)
    for i, row in corpus.iterrows():
        if row["sender\_norm"] != "Katrina":
            continue
        text = str(row["message\_norm"])
        for rule\_id, category, name, pats in compiled\_rules:
            if any(p.search(text) for p in pats):
                rows.append({
                    "timestamp": row["ts"],
                    "timestamp\_display": row["timestamp\_display"],
                    "source\_file": row["source\_file"],
                    "idx\_in\_source": row["idx\_in\_source"],
                    "sender": "Katrina",
                    "message": text,
                    "behavior\_category": category,
                    "behavior\_name": name,
                    "rule\_id": rule\_id,
                    "prev\_sender": row["prev\_sender"],
                    "prev\_message": row["prev\_msg"],
                    "next\_sender": row["next\_sender"],
                    "next\_message": row["next\_msg"],
                })
                break
    return pd.DataFrame(rows).sort\_values("timestamp").reset\_index(drop=True)
evidence = label\_hits(CORPUS)
# Build context workbooks
def make\_context\_workbook(evidence\_df, corpus, path, window=5):
    ev = evidence\_df.copy()
    ev['match\_id'] = range(1, len(ev)+1)
    # Build keys
    corpus = corpus.copy()
    corpus['key'] = (
        corpus['source\_file'].astype(str) + '|' +
        corpus['idx\_in\_source'].astype(str) + '|' +
        corpus['ts'].astype(str) + '|' +
        corpus['message\_norm'].astype(str)
    )
    ev['key'] = (
        ev['source\_file'].astype(str) + '|' +
        ev['idx\_in\_source'].astype(str) + '|' +
        ev['timestamp'].astype(str) + '|' +
        ev['message'].astype(str)
    )
    key\_to\_idx = {k:i for i,k in enumerate(corpus['key'])}
    ctx\_rows = []
    for \_, hit in ev.iterrows():
        idx = key\_to\_idx.get(hit['key'])
        if idx is None:
            # fallback by ts+source+idx
            candidates = corpus.index[(corpus['source\_file']==hit['source\_file']) &
                                      (corpus['idx\_in\_source']==hit['idx\_in\_source']) &
                                      (corpus['ts']==hit['timestamp'])].tolist()
            idx = candidates[0] if candidates else None
        if idx is None:
            continue
        start = max(0, idx - window)
        end   = min(len(corpus)-1, idx + window)
        block = corpus.iloc[start:end+1].copy()
        block['is\_hit\_line'] = False
        block.loc[idx, 'is\_hit\_line'] = True
        block = block.rename(columns={'ts':'timestamp','sender\_norm':'sender','message\_norm':'message'})
        block\_out = block[[
            'timestamp','timestamp\_display','sender','message','source\_file','idx\_in\_source','is\_hit\_line'
        ]].copy()
        block\_out['match\_id'] = hit['match\_id']
        block\_out['behavior\_category'] = hit['behavior\_category']
        block\_out['behavior\_name'] = hit['behavior\_name']
        block\_out['rule\_id'] = hit['rule\_id']
        ctx\_rows.append(block\_out)
    context\_df = pd.concat(ctx\_rows, ignore\_index=True) if ctx\_rows else pd.DataFrame()
    labeled\_sheet = ev[[
        'match\_id','timestamp','timestamp\_display','sender','message','behavior\_category','behavior\_name','rule\_id',
        'source\_file','idx\_in\_source','prev\_sender','prev\_message','next\_sender','next\_message'
    ]]
    with pd.ExcelWriter(path, engine="xlsxwriter") as writer:
        labeled\_sheet.to\_excel(writer, index=False, sheet\_name="Labeled\_Behaviors")
        context\_df.to\_excel(writer, index=False, sheet\_name="Context\_Both\_Sides")
# Save 5-message window
wb5 = Path("/mnt/data/court\_line\_by\_line\_with\_context.xlsx")
make\_context\_workbook(evidence, CORPUS, wb5, window=5)
# Save 15-message window
wb15 = Path("/mnt/data/court\_line\_by\_line\_with\_context\_WIDE.xlsx")
make\_context\_workbook(evidence, CORPUS, wb15, window=15)
# -------------------------------
# Bundle ZIP of artifacts
# -------------------------------
bundle\_path = "/mnt/data/court\_artifacts\_bundle.zip"
candidate\_files = [
    str(backup\_csv),
    str(special\_xlsx),
    str(special\_v2),
    str(wb5),
    str(wb15),
]
with ZipFile(bundle\_path, "w", ZIP\_DEFLATED) as zf:
    for fp in candidate\_files:
        p = Path(fp)
        if p.exists():
            zf.write(p, arcname=p.name)
# -------------------------------
# Show the list of files now present (name + size)
# -------------------------------
def list\_files(dir\_path="/mnt/data"):
    rows = []
    for name in sorted(os.listdir(dir\_path)):
        p = Path(dir\_path) / name
        if p.is\_file():
            rows.append({"file": name, "size\_bytes": p.stat().st\_size})
    return pd.DataFrame(rows)
from caas\_jupyter\_tools import display\_dataframe\_to\_user
display\_dataframe\_to\_user("Files currently available", list\_files())
print("Ready:")
print("Dataset CSV Backup:", backup\_csv.as\_posix())
print("Special Passes (v1):", special\_xlsx.as\_posix())
print("Special Passes (v2):", special\_v2.as\_posix())
print("Line-by-line (±5):", wb5.as\_posix())
print("Line-by-line (±15):", wb15.as\_posix())
print("Artifacts Bundle ZIP:", bundle\_path)
