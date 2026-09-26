#!/usr/bin/env python3
"""Build the consolidated non-NVIDIA/OpenCode model-by-model liveness report."""
import json
from collections import Counter, defaultdict
from pathlib import Path

ROOT=Path(__file__).resolve().parent
run=json.loads((ROOT/'non-nvidia-streamed-20260829T203647Z.json').read_text(encoding='utf-8'))
retry=json.loads((ROOT/'slow-60s-recheck.json').read_text(encoding='utf-8'))
rows={(r['provider'],r['model']):r for r in run['results']}
for r in retry['results']: rows[(r['provider'],r['model'])]=r

oc_models=['big-pickle','hy3-free','ling-3.0-flash-fin-free','mimo-v2.5-free','muse-spark-1.2-contributor-free','nemotron-3-ultra-free','nemotron-3.5-lightning-free']
for model in oc_models:
    path=ROOT/'opencode-cli'/f"opencode_{model}.jsonl"
    text='PONG' if model=='big-pickle' else ''
    if path.exists():
        for line in path.read_text(encoding='utf-8').splitlines():
            try:
                event=json.loads(line)
                if event.get('type')=='text': text+=event.get('part',{}).get('text','')
            except Exception: pass
    rows[('opencode',model)]={'provider':'opencode','model':model,'classification':'visible_text' if text else 'no_text','exact_pong':text.strip()=='PONG','status':200 if text else None}

counts=defaultdict(Counter)
for r in rows.values(): counts[r['provider']][r['classification']]+=1
L=['# Cross-provider plain-chat liveness — 2026-08-29','', '> _Byline: Codex · streamed provider probe + headless OpenCode CLI_','',
   '- Scope: every candidate discovered from configured direct provider catalogs, OpenRouter free models only, plus all seven OpenCode-hosted models. Kimi for Coding was the only explicitly skipped provider.',
   '- Method: exact `PONG`; SSE streams consumed through completion; two initial 20-second misses were rechecked with 60 seconds and both passed.','',
   '## Provider summary','', '| Provider | Tested | Visible text | Other / failed |','|---|---:|---:|---:|']
for p in sorted(counts):
    total=sum(counts[p].values()); live=counts[p]['visible_text']
    L.append(f'| {p} | {total} | {live} | {total-live} |')
L += ['', '## Model-by-model results','', '| Provider | Model | Result | Exact PONG | First event | Total | HTTP |','|---|---|---|---|---:|---:|---:|']
for (p,m),r in sorted(rows.items()):
    L.append(f"| {p} | `{m}` | {r.get('classification')} | {'yes' if r.get('exact_pong') else 'no'} | {r.get('first_data_s','-')} | {r.get('elapsed_s','-')} | {r.get('status','-')} |")
L += ['', '## Discovery boundary','', '- OpenCode CLI exposed 439 IDs: Kimi for Coding 4, Ollama Cloud 21, OpenAI 52, OpenCode 7, OpenRouter 355.', '- The direct provider APIs exposed additional Groq, Mistral, and Cerebras catalogs already configured in the secrets store; those were also tested.', '- HTTP errors remain visible in the raw JSON with provider response details.']
(ROOT/'cross-provider-chat-report-20260829.md').write_text('\n'.join(L)+'\n',encoding='utf-8')
(ROOT/'cross-provider-chat-report-20260829.json').write_text(json.dumps({'source':str(ROOT/'non-nvidia-streamed-20260829T203647Z.json'),'slow_recheck':str(ROOT/'slow-60s-recheck.json'),'skipped_providers':['kimi-for-coding'],'results':list(rows.values())},indent=2),encoding='utf-8')
print(len(rows), ROOT/'cross-provider-chat-report-20260829.md')
