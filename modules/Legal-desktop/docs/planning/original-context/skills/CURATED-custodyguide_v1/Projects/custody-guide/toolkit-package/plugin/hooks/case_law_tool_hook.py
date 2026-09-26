#!/usr/bin/env python3
"""Read-only Claude hook: preserve CourtListener output and add verification/fallback context."""
import json, re, sys
MAX_INPUT=65536
TOOLS=re.compile(r"^mcp__(?:plugin_family-court-toolkit_courtlistener|courtlistener)__.*$")
def emit(value):
    sys.stdout.write(json.dumps(value,separators=(",",":")))
def main():
    try:
        raw=sys.stdin.buffer.read(MAX_INPUT+1)
        if len(raw)>MAX_INPUT: return emit({})
        data=json.loads(raw.decode("utf-8")); tool=data.get("tool_name",""); event=data.get("hook_event_name","")
        if not isinstance(tool,str) or not TOOLS.fullmatch(tool): return emit({})
        if event=="PostToolUse":
            msg="CourtListener result is discovery evidence only: verify exact citation, court, date, direct source URL, quotation, published/precedential status, jurisdictional weight, and negative-treatment limitation. Do not claim Shepardization/KeyCite; cross-check Michigan opinions with official Michigan Courts material when available."
        elif event=="PostToolUseFailure":
            msg="CourtListener was unavailable or failed. Do not auto-retry. Use bundled references or official Michigan Courts/other primary sources manually, state the fallback and remaining verification limits."
        else: return emit({})
        emit({"hookSpecificOutput":{"hookEventName":event,"additionalContext":msg}})
    except Exception:
        emit({})
if __name__=="__main__": main()
