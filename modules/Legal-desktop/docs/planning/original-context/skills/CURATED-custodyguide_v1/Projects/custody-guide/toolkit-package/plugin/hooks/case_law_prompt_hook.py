#!/usr/bin/env python3
"""Read-only Claude hook: add a narrow case-law research reminder or fail open."""
import json, re, sys
MAX_INPUT=65536
TERMS=re.compile(r"\b(case[ -]?law|citation|precedent|holding|reporter|docket|negative[ -]?treatment)\b",re.I)
def emit(value):
    sys.stdout.write(json.dumps(value,separators=(",",":")))
def main():
    try:
        raw=sys.stdin.buffer.read(MAX_INPUT+1)
        if len(raw)>MAX_INPUT: return emit({})
        data=json.loads(raw.decode("utf-8"))
        prompt=data.get("prompt","")
        if not isinstance(prompt,str) or not TERMS.search(prompt): return emit({})
        emit({"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"Case-law/citation request: use CourtListener only if connected or user-authorized; treat results as discovery. Verify exact case/citation/court/date/opinion URL/quoted text, precedential status, jurisdictional weight, and negative-treatment limits. Do not invent citations or claim Shepardization/KeyCite."}})
    except Exception:
        emit({})
if __name__=="__main__": main()
