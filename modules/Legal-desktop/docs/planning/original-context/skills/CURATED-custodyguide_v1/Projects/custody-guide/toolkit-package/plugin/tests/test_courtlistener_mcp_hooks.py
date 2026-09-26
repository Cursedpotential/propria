import hashlib,json,os,stat,subprocess,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
HOOKS=ROOT/'hooks'
def run(script,payload):
 before={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in HOOKS.iterdir() if p.is_file()}
 r=subprocess.run([sys.executable,str(HOOKS/script)],input=payload,capture_output=True,text=True,timeout=3)
 assert r.returncode==0 and r.stderr==''
 out=json.loads(r.stdout);assert json.dumps(out,separators=(',',':'))==r.stdout
 after={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in HOOKS.iterdir() if p.is_file()}
 assert before==after
 return out
def test_mcp_schema_and_no_secrets():
 d=json.loads((ROOT/'.mcp.json').read_text());assert d=={'mcpServers':{'courtlistener':{'type':'http','url':'https://mcp.courtlistener.com/'}}}
def test_hook_config_has_required_safe_matchers_and_commands():
 d=json.loads((HOOKS/'hooks.json').read_text())['hooks'];assert len(d['UserPromptSubmit'])==1 and 'matcher' not in d['UserPromptSubmit'][0]
 expected={'^mcp__plugin_family-court-toolkit_courtlistener__.*$','^mcp__courtlistener__.*$'}
 for event in ['PostToolUse','PostToolUseFailure']:
  assert {x['matcher'] for x in d[event]}==expected
  for x in d[event]:
   h=x['hooks'][0];assert h['type']=='command';assert '"${CLAUDE_PLUGIN_ROOT}/hooks/' in h['command'];assert h['timeout']==5
 for p in [HOOKS/'case_law_prompt_hook.py',HOOKS/'case_law_tool_hook.py']:
  assert p.read_text().startswith('#!/usr/bin/env python3\n');assert p.stat().st_mode&stat.S_IXUSR
def test_prompt_hook_relevant_and_irrelevant_and_fail_open():
 o=run('case_law_prompt_hook.py',json.dumps({'hook_event_name':'UserPromptSubmit','prompt':'Find negative treatment for a Michigan holding'}));c=o['hookSpecificOutput']['additionalContext'];assert 'CourtListener' in c and 'invent citations' in c
 assert run('case_law_prompt_hook.py',json.dumps({'hook_event_name':'UserPromptSubmit','prompt':'Help organize exhibits'}))=={}
 assert run('case_law_prompt_hook.py','not-json')=={}
 assert run('case_law_prompt_hook.py','{'+'x'*70000)=={}
def test_tool_hooks_success_failure_and_no_output_replacement():
 for name in ['mcp__plugin_family-court-toolkit_courtlistener__search','mcp__courtlistener__get_opinion']:
  o=run('case_law_tool_hook.py',json.dumps({'hook_event_name':'PostToolUse','tool_name':name,'tool_response':{'ignored':'original remains'}}));h=o['hookSpecificOutput'];assert h['hookEventName']=='PostToolUse';assert 'exact citation' in h['additionalContext'];assert 'updatedToolOutput' not in json.dumps(o) and 'updatedMCPToolOutput' not in json.dumps(o)
  f=run('case_law_tool_hook.py',json.dumps({'hook_event_name':'PostToolUseFailure','tool_name':name,'error':'network'}));assert 'Do not auto-retry' in f['hookSpecificOutput']['additionalContext']
 assert run('case_law_tool_hook.py',json.dumps({'hook_event_name':'PostToolUse','tool_name':'Read'}))=={}
 assert run('case_law_tool_hook.py','[]')=={}
def test_hook_scripts_do_not_read_transcript_or_persist_data():
 for p in [HOOKS/'case_law_prompt_hook.py',HOOKS/'case_law_tool_hook.py']:
  text=p.read_text().lower();assert 'transcript_path' not in text and 'open(' not in text and 'requests' not in text and 'urllib' not in text
  assert '.write' not in text.replace('sys.stdout.write','')
