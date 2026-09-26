import json,re
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def test_micourt_is_not_an_mcp_or_api_connector():
 mcp=json.loads((ROOT/'.mcp.json').read_text());assert set(mcp['mcpServers'])=={'courtlistener'}
 manifest=json.loads((ROOT/'.claude-plugin/plugin.json').read_text());assert 'micourt' not in json.dumps(manifest).lower()
 assert not any('micourt' in p.name.lower() for p in (ROOT/'hooks').glob('*'))
def test_micourt_assessment_and_guarded_command_exist():
 doc=(ROOT/'MICOURT-CASE-SEARCH-INTEGRATION.md').read_text();cmd=(ROOT/'commands/case-lookup.md').read_text()
 for phrase in ['Not implemented, not configured, and not activated','Jis-Api-Subscription-Key','OAuth2 client-credentials','Starter','first 10 search results','disallows expansion','CourtListener remains the active case-law/opinion discovery MCP']:
  assert phrase in doc
 for phrase in ['no MiCOURT connectivity','must not make API calls','request credentials','sealed, suppressed, adoption, nonpublic','not case-law authority']:
  assert phrase in cmd
def test_no_real_micourt_credentials_or_secret_placeholders():
 text='\n'.join(p.read_text(encoding='utf-8',errors='ignore') for p in ROOT.rglob('*') if p.is_file() and p.suffix in {'.md','.json','.py'})
 # Terms may be documented, but no assigned/pasted credential values are permitted.
 assert not re.search(r'(?i)(?:jis-api-subscription-key|client_secret|client_id|authorization)\s*[:=]\s*["\']?(?!\{|your |<|\[|\$)[A-Za-z0-9_\-]{12,}',text)
 assert 'https://developer.micourt.courts.michigan.gov' in text
def test_docket_metadata_is_not_confused_with_case_law():
 text=(ROOT/'MICOURT-CASE-SEARCH-INTEGRATION.md').read_text()
 for phrase in ['does not establish a holding','opinion text','reporter citation','precedential status','negative treatment','CourtListener']:
  assert phrase in text
