import re, json
from pathlib import Path
ROOT = Path(__file__).resolve().parents[1]
# Every ${CLAUDE_PLUGIN_ROOT}/... path mentioned anywhere a model will read must exist on disk.
DOCS = list((ROOT/'skills').rglob('SKILL.md')) + list((ROOT/'commands').glob('*.md')) + list((ROOT/'agents').glob('*.md'))
PAT = re.compile(r'\$\{CLAUDE_PLUGIN_ROOT\}(/[^\s`)\'"*,;]+)')

def test_every_referenced_plugin_root_path_exists():
    missing = []
    for doc in DOCS:
        for rel in PAT.findall(doc.read_text(encoding='utf-8')):
            rel = rel.rstrip('.,;:')
            # a documented placeholder like <script>.py: validate the directory that holds it
            target = rel.rsplit('/', 1)[0] if ('<' in rel or '>' in rel) else rel
            if not (ROOT / target.lstrip('/')).exists():
                missing.append((doc.name, rel))
    assert not missing, f"unresolvable ${{CLAUDE_PLUGIN_ROOT}} paths: {missing}"

def test_custody_packet_bundled_resources_present():
    C = ROOT/'skills/custody-packet/content/custody-guide'
    for rel in ['GUARDRAILS.md','verification_ledger.md','CHEAT-SHEET.md','DRAFTING-HANDOFF.md',
                'custody_guide_outline_v2.md','critical_review_outline_v2.md','master_source_directory.md',
                'research_interference_coercive_control.md','sources/fetch-sources.sh',
                'sources/primary/SHA256SUMS','draft/P1-translation-card.md','draft/P1-worked-examples.md']:
        assert (C/rel).is_file(), rel
    # Primary law ships as structured markdown (pymupdf4llm), not the source PDFs:
    # smaller AND more usable than the old flat text, and a model can't read a PDF
    # anyway. sources/fetch-sources.sh restores the PDFs on demand; the PDF
    # originals live at the repo top level.
    mds = list((C/'sources/primary').glob('*.md')); pdfs = list((C/'sources/primary').glob('*.pdf'))
    assert len(mds) >= 6, f"expected >=6 markdown sources, got {len(mds)}"
    assert len(pdfs) == 0, "source PDFs must stay out of the plugin to keep it uploadable"
    assert (C/'sources/fetch-sources.sh').is_file(), "the PDF-refresh script must ship so PDFs are recoverable"

def test_components_autodiscover_from_default_locations():
    # The manifest declares no component paths; Claude Code scans the default
    # directories. Assert those directories hold loadable components so the
    # auto-discovery the manifest relies on actually finds something.
    man = json.loads((ROOT/'.claude-plugin/plugin.json').read_text())
    assert not ({'skills','commands','agents','mcpServers','hooks'} & set(man)), \
        "manifest should declare no component paths and rely on auto-discovery"
    assert len(list((ROOT/'commands').glob('*.md'))) >= 9, "commands/ must hold the slash commands"
    assert len(list((ROOT/'agents').glob('*.md'))) >= 4, "agents/ must hold the subagents"
    # every skills/ subdir must carry a SKILL.md so it is actually loaded
    skill_dirs = [d for d in (ROOT/'skills').iterdir() if d.is_dir()]
    assert len(skill_dirs) >= 2
    for d in skill_dirs: assert (d/'SKILL.md').is_file(), d.name

def test_bundled_helper_scripts_are_executable_python():
    import py_compile
    for s in (ROOT/'skills/toolkit/content/toolkit/scripts').glob('*.py'):
        py_compile.compile(str(s), doraise=True)
