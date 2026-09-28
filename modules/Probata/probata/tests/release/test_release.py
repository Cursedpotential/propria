import hashlib
import json
import re
from pathlib import Path
from types import SimpleNamespace
import pytest
from surrealdb import AsyncSurreal
import sq,upgrade,adr,knowledge
from context_pack import pack
from retention import retained_sql,RetainingConnection
from scope import ROOTS
from source_registry import load_sources
from source_sync import operation as sync,validated


@pytest.fixture
async def database(monkeypatch):
    db=AsyncSurreal('mem://'); await db.connect(); await db.use('test','docs')
    for name in ('000_analyzers.surql','010_documents.surql','020_chunks.surql','030_records.surql','040_graph.surql','050_events.surql'):
        await db.query((upgrade.SCHEMA/name).read_text())
    class Lease:
        query=db.query
        async def close(self): pass
    async def connect(*args): return Lease()
    monkeypatch.setattr(sq,'connect',connect)
    plan=await upgrade.operation('plan'); assert not plan['destructive']
    assert (await upgrade.operation('apply',plan['plan_id']))['verified']
    yield db
    await db.close()


def registry(root):
    projects=[]
    for project,(path,prefix) in ROOTS.items():
        (root/path).mkdir(parents=True)
        projects.append({'project_id':project,'title':project,'source_root':path,'canonical_prefix':prefix,'domains':['docs'],
                         'registration_status':'active','ingestion_status':'current-full-source' if project=='probata' else 'pending-multi-root-cdc',
                         'included_patterns':['**/*.md'],'excluded_patterns':['private/**','**/to_be_deleted/**'],'required':True})
    p=root/'registry.json'; p.write_text(json.dumps({'schema':'propria-docstore-source-registry-v1','monorepo_root':str(root),'projects':projects}))
    return p


def test_packing_dedupe_budget_and_provenance():
    result=pack([{'id':'a','snippet':'text','score':1},{'id':'a','snippet':'other','score':2},{'id':'a','source':'claude','snippet':'other','score':1}],budget=2000)
    assert len(result['results'])==2
    assert result['results'][0]['snippet']=='other'
    assert result['packing']['omitted_rows']==1
    assert pack([{'id':'a','snippet':'x'*4000}],budget=256)['results']==[]


def test_scope_rejects_broadened_or_partial_roots(tmp_path):
    p=registry(tmp_path)
    sources,_=load_sources(p,tmp_path/'docs',multi_root_enabled=True)
    assert len(sources)==len(ROOTS)  # never a literal: this read 5 while scope.py had grown to 7
    data=json.loads(p.read_text()); data['projects'][0]['source_root']='.'; p.write_text(json.dumps(data))
    with pytest.raises(ValueError): load_sources(p,tmp_path/'docs',multi_root_enabled=True)
    with pytest.raises(ValueError): load_sources(None,tmp_path/'docs',multi_root_enabled=False)


def test_source_sync_stale_plan_hash_and_quarantine(tmp_path,monkeypatch):
    p=registry(tmp_path)
    monkeypatch.setenv('DOCSTORE_PROJECT_REGISTRY',str(p)); monkeypatch.setenv('DOCSTORE_SYNC_LOCK',str(tmp_path/'state/sync.lock'))
    files=[{'project':project,'path':'note.md','content':'# '+project,'sha256':hashlib.sha256(('# '+project).encode()).hexdigest()} for project in ROOTS]
    plan=sync('plan',{'files':files})
    assert not list((tmp_path/'docs').glob('*.md'))
    with pytest.raises(ValueError): sync('apply',{'files':files,'plan_id':'stale'})
    assert sync('apply',{'files':files,'plan_id':plan['plan_id']})['verified']
    files[0]['path']='new.md'; plan=sync('plan',{'files':files})
    assert plan['retracted_sources'] and set(plan['retracted_hashes'])==set(plan['retracted_sources'])
    copy=sync('read',{'paths':plan['retracted_sources']})  # 0.8.1-r3: restore source for a guarded retraction
    assert not copy['missing'] and copy['files'][0]['sha256']==plan['retracted_hashes'][copy['files'][0]['key']]
    with pytest.raises(ValueError): sync('apply',{'files':files,'plan_id':plan['plan_id']})
    assert not list((tmp_path/'to_be_deleted').rglob('note.md'))
    sync('apply',{'files':files,'plan_id':plan['plan_id'],'retract':plan['retracted_sources']})
    assert list((tmp_path/'to_be_deleted').rglob('note.md'))
    files[0]['path']='../escape.md'
    with pytest.raises(ValueError): validated(files)


async def test_upgrade_idempotent_and_stale_plan(database):
    state=await upgrade.operation('verify'); assert state['verified']
    assert (await upgrade.operation('apply',state['plan_id']))['verified']
    with pytest.raises(ValueError): await upgrade.operation('apply','wrong')
    await database.query('UPDATE docstore_migration SET checksum="tampered";')
    with pytest.raises(ValueError): await upgrade.operation('verify')


async def test_adr_lifecycle_conflict_and_projection(database):
    value=await adr.operation('create',{'number':1,'title':'Retain','decision':'Keep history','context':'Context'})
    assert value['projections'][0]['content'].startswith('<!-- Generated')
    await adr.operation('update',{'id':'adr:propria_0001','expected_version':1,'decision':'Keep every version'})
    with pytest.raises(Exception): await adr.operation('update',{'id':'adr:propria_0001','expected_version':1,'decision':'stale'})
    rows=upgrade.rows(await database.query('SELECT * FROM adr:propria_0001;'))
    assert rows[0]['version']==2 and rows[0]['decision']=='Keep every version'
    assert not upgrade.rows(await database.query('SELECT * FROM document;'))


async def make_document(db,rid='document:legacy',path='docs/legacy.md'):
    await db.query('CREATE type::record($id) SET source_path=$path,content_hash=$hash,title="Legacy",body="## Context\nOld\n## Decision\nKeep `SurrealDB` history",doc_type="decision",project="probata",status="active",domains=["docs"];',{'id':rid,'path':path,'hash':hashlib.sha256(path.encode()).hexdigest()})


async def test_legacy_import_preserves_source_and_does_not_promote_authority(database):
    await make_document(database)
    plan=await adr.operation('migration-plan')
    result=await adr.operation('migration-apply',{'plan_id':plan['plan_id']})
    assert result['imported']==1
    assert len(await database.query('SELECT * FROM document;'))==1
    assert upgrade.rows(await database.query('SELECT * FROM adr;'))[0]['status']=='proposed'
    assert (await adr.operation('migration-plan'))['imports']==[]


async def test_coco_connector_retains_canonical_document(database):
    await make_document(database)
    connection=RetainingConnection(database)
    await connection.query('BEGIN TRANSACTION;\nDELETE document:legacy;\nCOMMIT TRANSACTION;')
    row=upgrade.rows(await database.query('SELECT * FROM document:legacy;'))[0]
    assert row['status']=='retracted' and 'Keep' in row['body']
    with pytest.raises(ValueError): retained_sql('DELETE document WHERE true;')
    with pytest.raises(ValueError): retained_sql('REMOVE TABLE document;')


async def test_graph_resolution_path_and_native_read(database):
    a=await knowledge.graph('entity-upsert',{'name':'SurrealDB','kind':'service','aliases':['Surreal DB']})
    again=await knowledge.graph('entity-upsert',{'name':'Surreal DB','kind':'service'})
    assert a==again
    b=await knowledge.graph('entity-upsert',{'name':'CocoIndex','kind':'library'})
    await knowledge.graph('relate',{'start':a['id'],'end':b['id']})
    path=await knowledge.graph('path',{'start':a['id'],'end':b['id']})
    assert path['path']==[a['id'],b['id']]
    assert (await knowledge.surrealql_read('SELECT name FROM entity LIMIT 10;'))['results']
    with pytest.raises(ValueError): await knowledge.surrealql_read('SELECT fn::remember({}) FROM entity LIMIT 10;')


async def test_incremental_enrichment_and_shared_entities(database,monkeypatch):
    monkeypatch.delenv('DOCSTORE_LLM_BASE_URL',raising=False)
    await make_document(database)
    first=await knowledge.enrich_changed([SimpleNamespace(source_path='docs/legacy.md')])
    assert first['changed_documents']==1
    assert (await knowledge.enrich_changed([SimpleNamespace(source_path='docs/legacy.md')]))['changed_documents']==0
    assert await database.query('SELECT * FROM about;')
    assert await database.query('SELECT * FROM statement;')


async def test_scope_retraction_preserves_independently_authored_docs(database):
    from cdc_verify import retire_unexpected_projection,stable_id
    path='docs/removed.md'
    await make_document(database,'document:'+stable_id(path),path)
    await make_document(database,'document:authored','docs/new-receipt.md')
    plan=await retire_unexpected_projection((),dry_run=True)
    assert plan['retired_paths']==[path]
    held=await retire_unexpected_projection(())  # 0.8.1-r3: nothing named, nothing retracted
    assert held['held_paths']==[path] and held['retired_count']==0
    assert upgrade.rows(await database.query('SELECT * FROM type::record($id);',{'id':'document:'+stable_id(path)}))[0]['status']=='active'
    await retire_unexpected_projection((),allowed={path})
    assert upgrade.rows(await database.query('SELECT * FROM document:authored;'))[0]['status']=='active'
    assert upgrade.rows(await database.query('SELECT * FROM type::record($id);',{'id':'document:'+stable_id(path)}))[0]['status']=='retracted'


async def test_semantic_retry_never_accepts_truncated_json(monkeypatch):
    for key,value in {'DOCSTORE_LLM_BASE_URL':'https://provider.invalid/v1','DOCSTORE_LLM_MODEL':'test','DOCSTORE_LLM_API_KEY':'synthetic'}.items():
        monkeypatch.setenv(key,value)
    requests=[]
    class Client:
        def __init__(self,**kwargs):pass
        async def __aenter__(self):return self
        async def __aexit__(self,*args):pass
        async def post(self,url,**kwargs):
            requests.append(kwargs['json'])
            content='{"summary":' if len(requests)==1 else json.dumps({'summary':'Supported claim','classification':'reference','entities':[],'statements':[]})
            return SimpleNamespace(raise_for_status=lambda:None,json=lambda:{'choices':[{'finish_reason':'length' if len(requests)==1 else 'stop','message':{'content':content}}]})
    monkeypatch.setattr(knowledge.httpx,'AsyncClient',Client)
    result=await knowledge.extract('x'*50000)
    assert result['method']=='remote-llm' and len(requests)==2
    assert len(requests[1]['messages'][1]['content'])==6000
    assert result['coverage']['truncated'] is True


async def test_semantic_retry_reshapes_misshaped_reply(monkeypatch):
    # 0.8.1-r4 (Claude Code · Opus 5.5, 2026-09-26): statements as one string is retried, not a document failure.
    for key,value in {'DOCSTORE_LLM_BASE_URL':'https://provider.invalid/v1','DOCSTORE_LLM_MODEL':'test','DOCSTORE_LLM_API_KEY':'synthetic'}.items():
        monkeypatch.setenv(key,value)
    requests=[]
    class Client:
        def __init__(self,**kwargs):pass
        async def __aenter__(self):return self
        async def __aexit__(self,*args):pass
        async def post(self,url,**kwargs):
            requests.append(kwargs['json'])
            statements='one; two' if len(requests)==1 else ['one','two']
            content=json.dumps({'summary':'s','classification':'reference','entities':[],'statements':statements})
            return SimpleNamespace(raise_for_status=lambda:None,json=lambda:{'choices':[{'finish_reason':'stop','message':{'content':content}}]})
    monkeypatch.setattr(knowledge.httpx,'AsyncClient',Client)
    result=await knowledge.extract('short body')
    assert len(requests)==2 and result['statements']==['one','two']


async def test_enrichment_bounds_parallel_provider_work(database,monkeypatch):
    import asyncio
    active=0; peak=0
    async def extract(body):
        nonlocal active,peak
        active+=1;peak=max(peak,active)
        await asyncio.sleep(.02)
        active-=1
        return {'summary':'test','classification':'reference','entities':[],'statements':[],'method':'test','model':'synthetic'}
    monkeypatch.setattr(knowledge,'extract',extract)
    monkeypatch.setenv('DOCSTORE_ENRICH_CONCURRENCY','2')
    snapshot=[]
    for n in range(5):
        path=f'docs/parallel-{n}.md'
        await make_document(database,f'document:parallel_{n}',path)
        snapshot.append(SimpleNamespace(source_path=path))
    assert (await knowledge.enrich_changed(snapshot))['changed_documents']==5
    assert peak==2 and active==0


async def test_one_provider_failure_retains_other_completed_enrichment(database,monkeypatch):
    await make_document(database,'document:good','docs/good.md')
    await make_document(database,'document:bad','docs/bad.md')
    await database.query('UPDATE document:bad SET body="provider-fails";')
    async def extract(body):
        if body=='provider-fails':raise ValueError('synthetic provider failure')
        return {'summary':'retained','classification':'reference','entities':[],'statements':[],'method':'test','model':'synthetic'}
    monkeypatch.setattr(knowledge,'extract',extract)
    with pytest.raises(knowledge.EnrichmentIncomplete) as failure:
        await knowledge.enrich_changed([SimpleNamespace(source_path='docs/bad.md'),SimpleNamespace(source_path='docs/good.md')])
    assert failure.value.result['changed_documents']==1
    assert failure.value.result['failed_documents']==[{'source_path':'docs/bad.md','error_type':'ValueError'}]
    assert len(upgrade.rows(await database.query('SELECT * FROM docstore_enrichment;')))==1


def test_no_module_carries_its_own_copy_of_the_source_roots():
    """scope.ROOTS is the only place the indexed roots may be written down.

    api.py and plugins/docstore/control/server.py each kept a private five-root list on
    pre-2026-09-20 module paths, and adr.py joined one of those paths directly. They went stale
    the moment scope.py grew to seven roots, so /health advertised roots the pipeline had not
    indexed for eight days -- and rebuilding from git did not help, because the copies were in
    git. Docstrings and comments may still discuss the old paths; executable strings may not.
    """
    import ast
    stale = re.compile(r'^(Propria/docs|Probata/probata(/docs)?|Consignatio(/Intake)?/docs'
                       r'|Legal-desktop/docs)$')
    root = Path(__file__).resolve().parents[1]
    offenders = []
    for directory in ('scripts/docstore', 'plugins/docstore/control'):
        for source in sorted((root / directory).rglob('*.py')):
            tree = ast.parse(source.read_text(encoding='utf-8'))
            documentation = set()
            for node in ast.walk(tree):
                body = getattr(node, 'body', None)
                if isinstance(node, (ast.Module, ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)) \
                        and body and isinstance(body[0], ast.Expr) \
                        and isinstance(body[0].value, ast.Constant) and isinstance(body[0].value.value, str):
                    documentation.add(id(body[0].value))
            for node in ast.walk(tree):
                if isinstance(node, ast.Constant) and isinstance(node.value, str) \
                        and id(node) not in documentation and stale.match(node.value.strip()):
                    offenders.append(f'{source.relative_to(root)}:{node.lineno} {node.value!r}')
    assert not offenders, 'source roots must come from scope.ROOTS, not a literal: ' + '; '.join(offenders)


def test_embed_safe_defuses_every_data_uri_and_never_sends_blank():
    """One data: URI in one chunk fails the whole NIM batch, and a failed batch fails the run.

    The 2026-09-28 sync died after 269 s on 872 sources because exactly two shipped documents
    mention a bare `data:image/`. strip_data_uris only matches the `;base64,` form, and the old
    embed_safe only looked at the start of a chunk -- and even there it prefixed the text while
    leaving the `data:` token standing, so it never actually helped.
    """
    import os
    os.environ.setdefault('SURREAL_DOCS_URL', 'http://127.0.0.1:1')
    os.environ.setdefault('SURREAL_DOCS_USER', 'synthetic')
    os.environ.setdefault('SURREAL_DOCS_PASS', 'synthetic')
    os.environ.setdefault('NVIDIA_API_KEY', 'synthetic')
    from flow_docs import embed_safe

    introducer = re.compile(r'data:[a-zA-Z0-9.+-]+/')
    for chunk in ('see the icon data:image/png in the spec',       # the real case, mid-chunk
                  'inline data:image/svg+xml,<svg/> here',         # no base64
                  'x data:image/png;base64,AAAA y',
                  'data:image/png is first',                       # what the old guard aimed at
                  'DATA:IMAGE/PNG upper case'):
        assert not introducer.search(embed_safe(chunk)), chunk
    assert embed_safe('   ').strip(), 'a blank input is rejected by NIM too'
    assert embed_safe('ordinary text') == 'ordinary text', 'must not rewrite ordinary text'


def test_no_shipped_document_can_break_an_embedding_batch():
    """Every markdown root actually shipped in this image survives embed_safe.

    A unit test on invented strings would not have caught the two real documents, because the
    failure depends on what is in the corpus.
    """
    import os
    os.environ.setdefault('SURREAL_DOCS_URL', 'http://127.0.0.1:1')
    os.environ.setdefault('SURREAL_DOCS_USER', 'synthetic')
    os.environ.setdefault('SURREAL_DOCS_PASS', 'synthetic')
    os.environ.setdefault('NVIDIA_API_KEY', 'synthetic')
    from flow_docs import embed_safe

    docs = Path(__file__).resolve().parents[1] / 'docs'
    if not docs.is_dir():
        pytest.skip('documentation roots are assembled into the image, not the checkout')
    introducer = re.compile(r'data:[a-zA-Z0-9.+-]+/')
    offenders = [str(f) for f in docs.rglob('*.md')
                 if introducer.search(embed_safe(f.read_text(encoding='utf-8', errors='ignore')))]
    assert not offenders, 'these would fail the NIM batch: ' + '; '.join(offenders[:5])
