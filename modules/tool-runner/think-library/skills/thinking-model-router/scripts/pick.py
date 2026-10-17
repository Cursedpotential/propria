#!/usr/bin/env python3
"""Cheap offline shortlist helper for think/skill-registry.json.

This is a scout, not the final router. It ranks entries using aliases/tags/description
and returns a small candidate list. The model/router still decides fit, depth, ordering,
and whether a combination is actually useful.
"""
from __future__ import annotations
import json,re,sys
from pathlib import Path
STOP={"the","a","an","and","or","of","to","in","on","for","is","are","was","were","be","this","that","it","with","as","by","at","from","what","which","how","should","can","could","would","need","use","using","before","after","whether","about","into","over","under","through","think","changing","change","but","one","two","thing","things","different","wrong"}
SYN={
    "reversed":"reversibility","reverse":"reversibility","reversible":"reversibility",
    "downstream":"second-order","consequence":"second-order","consequences":"second-order",
    "root":"root-cause","cause":"root-cause","causes":"root-cause",
    "sometimes":"intermittent","sporadic":"intermittent",
    "docs":"documentation","doc":"documentation",
    "rollback":"reversibility","undo":"reversibility",
    "mismatch":"map-territory","reality":"map-territory",
    "bottleneck":"theory-of-constraints","throughput":"theory-of-constraints",
}
def toks(s):
    out=set()
    for w in re.findall(r"[a-z0-9-]+",s.lower()):
        if len(w)<=2 or w in STOP: continue
        out.add(w)
        if w in SYN: out.add(SYN[w])
        if w.endswith("ing") and len(w)>6: out.add(w[:-3])
        if w.endswith("ed") and len(w)>5: out.add(w[:-2])
        if w.endswith("s") and len(w)>4: out.add(w[:-1])
    return out
def root(): return Path(__file__).resolve().parents[3]
def main(argv):
    if not argv:
        print('usage: pick.py "<task>" [--family family] [--top N]'); return 2
    fam=None; top=6; parts=[]; i=0
    while i<len(argv):
        if argv[i]=='--family' and i+1<len(argv): fam=argv[i+1]; i+=2; continue
        if argv[i]=='--top' and i+1<len(argv): top=int(argv[i+1]); i+=2; continue
        parts.append(argv[i]); i+=1
    q=' '.join(parts); qt=toks(q)
    data=json.loads((root()/'skill-registry.json').read_text())
    scored=[]
    qlow=q.lower()
    phrase_concepts={
        "roll back":"reversibility", "can we undo":"reversibility",
        "downstream effect":"second-order", "and then what":"second-order",
        "docs differ":"map-territory", "documentation says":"map-territory",
        "observed behavior":"map-territory", "expectation mismatch":"map-territory",
        "root cause":"root-cause", "what could go wrong":"pre-mortem",
        "blast radius":"systems", "find weaknesses":"red-team",
    }
    for phrase, concept in phrase_concepts.items():
        if phrase in qlow: qt.add(concept)
    for e in data['skills']:
        if e.get('kind')!='model': continue
        if fam and e.get('family')!=fam: continue
        aliases=[a.lower() for a in e.get('aliases',[]) if len(a)>3]
        alias_hit=any(a in qlow for a in aliases)
        name_t=toks(e.get('name',''))
        tag_t=set(e.get('tags',[]))
        desc_t=toks(e.get('description',''))
        name_ov=qt & name_t
        tag_ov=qt & tag_t
        desc_ov=qt & desc_t
        ov=name_ov | tag_ov | desc_ov
        exact_name=e.get('name','').lower() in qlow
        score=(14 if exact_name else 0)+(10 if alias_hit else 0)+(4*len(name_ov))+(2*len(tag_ov))+len(desc_ov)
        if score: scored.append((score,e,sorted(ov),alias_hit))
    scored.sort(key=lambda x:(-x[0],x[1]['name']))
    for score,e,ov,ah in scored[:top]:
        print(f"[{score:>2}] {e['name']} ({e['family']})  depth={e['default_depth']}")
        print('     '+e['description'])
        print('     path: '+e['path'])
        if ov: print('     match: '+', '.join(ov))
    if not scored:
        print('No strong registry match. Use the router decision matrix rather than forcing a model.')
    return 0
if __name__=='__main__': raise SystemExit(main(sys.argv[1:]))
