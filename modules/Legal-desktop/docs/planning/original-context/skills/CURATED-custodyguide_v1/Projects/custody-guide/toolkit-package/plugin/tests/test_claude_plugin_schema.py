import json,re,yaml,pytest
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
# Fields Claude Code recognizes in plugin.json. The component-path fields
# (skills/commands/agents/mcpServers/hooks/...) are VALID but intentionally
# UNUSED: this plugin relies on default auto-discovery of skills/, commands/,
# agents/, hooks/hooks.json, and .mcp.json so the manifest can never drift out
# of sync with the files on disk. See MANIFEST-SCHEMA-NOTES.md.
ALLOWED_PLUGIN={'$schema','name','displayName','version','description','author','license','keywords',
                'skills','commands','agents','hooks','mcpServers','outputStyles','lspServers',
                'homepage','repository','metadata','defaultEnabled','experimental','dependencies'}
COMPONENT_FIELDS={'skills','commands','agents','hooks','mcpServers','lspServers','outputStyles'}
ALLOWED_SKILL={'name','description','when_to_use','argument-hint','allowed-tools','user-invocable','disable-model-invocation','license','compatibility','metadata'}
ALLOWED_COMMAND={'description','argument-hint','allowed-tools','disable-model-invocation'}
ALLOWED_AGENT={'name','description','tools','disallowedTools','model','maxTurns','skills'}
def fm(p):
 s=Path(p).read_text(encoding='utf-8');assert s.startswith('---\n'),p;e=s.find('\n---\n',4);assert e>4,p;return yaml.safe_load(s[4:e]),s[e+5:]
def test_manifest_identity_and_autodiscovery():
 d=json.loads((ROOT/'.claude-plugin/plugin.json').read_text())
 assert set(d)<=ALLOWED_PLUGIN, f"unrecognized manifest fields: {set(d)-ALLOWED_PLUGIN}"
 assert d['name']=='family-court-toolkit'
 assert d['displayName']=='Family Court Toolkit'
 assert re.fullmatch(r'\d+\.\d+\.\d+',d['version']) and d['version']=='1.3.0'
 assert d['author']=={'name':'Matthew Salem'} and isinstance(d['keywords'],list)
 # Components are auto-discovered, not enumerated: no component-path field is set,
 # so nothing in the manifest can go stale when a command or skill is added.
 assert not (COMPONENT_FIELDS & set(d)), f"component-path fields should be omitted for auto-discovery, found {COMPONENT_FIELDS & set(d)}"
 # The files those defaults scan must actually be present.
 assert (ROOT/'hooks/hooks.json').is_file()
 assert (ROOT/'.mcp.json').is_file()
 for rel in ['commands/packet.md','commands/case-law.md','commands/case-lookup.md',
             'agents/case-law-researcher.md','skills/custody-packet/SKILL.md',
             'skills/toolkit/SKILL.md']:
  assert (ROOT/rel).is_file(), rel
 # no command filename may carry the stripped genesee- prefix anymore
 assert not list((ROOT/'commands').glob('genesee-*.md')), "commands still carry the genesee- prefix"
def test_skill_command_agent_frontmatter_supported_only():
 for skill in ['skills/toolkit/SKILL.md','skills/custody-packet/SKILL.md']:
  f,_=fm(ROOT/skill);assert set(f)<=ALLOWED_SKILL, skill
  assert f['user-invocable'] is True and f['disable-model-invocation'] is False and f['argument-hint']
 for p in (ROOT/'commands').glob('*.md'):
  x,_=fm(p);assert set(x)<=ALLOWED_COMMAND;assert x['disable-model-invocation'] is False
 for p in (ROOT/'agents').glob('*.md'):
  x,_=fm(p);assert set(x)<=ALLOWED_AGENT;assert x['model']=='inherit';assert isinstance(x['maxTurns'],int) and x['maxTurns']<=14;assert x['skills']==['toolkit'];assert not {'hooks','mcpServers','permissionMode'}&set(x)
def test_scoped_mcp_name_tracks_plugin_name_everywhere():
 # The scoped MCP tool name `mcp__plugin_<name>_courtlistener__` is wired into the
 # hook config, the hook script, and the agents' tool grants. A rename must reach
 # ALL of them or the case-law integration silently loses access. This guards the
 # whole surface, not just hooks.json.
 name=json.loads((ROOT/'.claude-plugin/plugin.json').read_text())['name']
 # the name-bearing fragment; the hook script splits `mcp__` off in its regex,
 # so match on the piece all four files share verbatim.
 good=f'plugin_{name}_courtlistener'
 targets=[ROOT/'hooks/hooks.json',ROOT/'hooks/case_law_tool_hook.py',
          ROOT/'agents/case-law-researcher.md',ROOT/'agents/michigan-source-verifier.md']
 for p in targets:
  t=p.read_text();assert good in t, f"{p.name} missing scoped name {good}"
 # no file anywhere in the plugin may still carry a stale scoped name.
 # Assemble the needle from fragments so this guard never matches itself.
 stale='plugin_'+'genesee-family-court-toolkit'+'_courtlistener'
 import subprocess
 # exclude bytecode: the compiler constant-folds this file's fragments back into
 # the full literal in its own .pyc, and .pyc never ships in the plugin anyway.
 hits=subprocess.run(['grep','-rl','--exclude=*.pyc','--exclude-dir=__pycache__',stale,str(ROOT)],
                     capture_output=True,text=True).stdout.strip()
 assert not hits, f"stale scoped MCP name remains in: {hits}"
def test_no_secret_or_external_service_configuration():
 assert (ROOT/'.mcp.json').is_file()
 text=(ROOT/'.mcp.json').read_text().lower();assert not any(x in text for x in ['api_key','token','secret','authorization','header','clientid','client_id','user_id'])
def test_plugin_root_runtime_paths_are_used():
 for p in [ROOT/'skills/toolkit/SKILL.md', ROOT/'commands/family-court.md', ROOT/'commands/verify-sources.md']:
  t=p.read_text(); assert '${CLAUDE_PLUGIN_ROOT}' in t
def test_custody_packet_skill_and_marketplace():
 f,body=fm(ROOT/'skills/custody-packet/SKILL.md');assert set(f)<=ALLOWED_SKILL;assert f['name']=='custody-packet'
 assert f['user-invocable'] is True and f['disable-model-invocation'] is False and f['when_to_use'] and f['argument-hint']
 assert '${CLAUDE_PLUGIN_ROOT}' in body
 for rel in ['GUARDRAILS.md','verification_ledger.md','CHEAT-SHEET.md','DRAFTING-HANDOFF.md','sources/fetch-sources.sh','draft/P1-translation-card.md']:
  assert (ROOT/'skills/custody-packet/content/custody-guide'/rel).is_file(), rel
 REPO=next((q for q in ROOT.parents if (q/'.claude-plugin/marketplace.json').is_file()),None)
 if REPO is None: pytest.skip('no marketplace manifest above plugin root')
 m=json.loads((REPO/'.claude-plugin/marketplace.json').read_text())
 assert m['name'] and m['owner']['name'] and isinstance(m['plugins'],list)
 e=[p for p in m['plugins'] if p['name']=='family-court-toolkit'][0]
 assert (REPO/e['source'][2:]/'.claude-plugin/plugin.json').is_file()
 assert e['version']==json.loads((ROOT/'.claude-plugin/plugin.json').read_text())['version']
