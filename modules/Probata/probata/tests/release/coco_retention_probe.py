"""Actual pinned CocoIndex target writes with synthetic vectors; no model/network calls."""
import asyncio,os,sys,json
from pathlib import Path
root=Path(sys.argv[1]).resolve()
sys.path.insert(0,str(Path(__file__).parents[1]/'scripts/docstore'))
from scope import ROOTS
projects=[]
for name,(path,prefix) in ROOTS.items():
    directory=root/path; directory.mkdir(parents=True,exist_ok=True)
    (directory/'test.md').write_text('# '+name+'\n\nThis is a synthetic retained document for '+name+'.\n'*20)
    projects.append({'project_id':name,'source_root':path,'canonical_prefix':prefix,'domains':['docs'], 'registration_status':'active',
                     'ingestion_status':'current-full-source' if name=='probata' else 'pending-multi-root-cdc','included_patterns':['**/*.md'],
                     'excluded_patterns':['private/**','**/to_be_deleted/**']})
registry=root/'registry.json'; registry.write_text(json.dumps({'schema':'propria-docstore-source-registry-v1','monorepo_root':str(root),'projects':projects}))
os.environ.update(DOCSTORE_PROJECT_REGISTRY=str(registry),DOCSTORE_MULTI_ROOT_ENABLED='1',DOCSTORE_COCOINDEX_DB=str(root/'state/coco.db'),
                  DOCSTORE_MAX_RSS_MB='0',SURREAL_DOCS_URL='http://127.0.0.1:1',SURREAL_DOCS_USER='synthetic',SURREAL_DOCS_PASS='synthetic',NVIDIA_API_KEY='synthetic')
import flow_docs as flow
from surrealdb import AsyncSurreal
import numpy as np
from retention import RetainingConnection
from upgrade import SCHEMA,rows

async def main():
    db=AsyncSurreal('mem://'); await db.connect(); await db.use('test','docs')
    for name in ('000_analyzers.surql','010_documents.surql','020_chunks.surql','030_records.surql','040_graph.surql','050_events.surql'):
        await db.query((SCHEMA/name).read_text())
    class Connection(RetainingConnection):
        async def close(self): pass
    class Factory:
        async def acquire(self): return Connection(db)
    class Embedding:
        def __coco_memo_key__(self): return 'synthetic-vector-v1'
        async def __coco_vector_schema__(self):
            from cocoindex.resources.schema import VectorSchema
            return VectorSchema(dtype=np.dtype('float32'),size=2048)
        async def embed(self,text): return np.ones(2048,dtype=np.float32)
    provider=flow.DOCSTORE_ENV.context_provider
    provider.provide(flow.SURREAL_DB,Factory()); provider.provide(flow.EMBEDDER,Embedding())
    await flow._run_checked()
    documents=rows(await db.query('SELECT id,status,body FROM document;'))
    assert len(documents)==len(ROOTS),len(documents)
    original=root/ROOTS['probata'][0]/'test.md'; quarantine=root/'to_be_deleted'; quarantine.mkdir(exist_ok=True); original.replace(quarantine/'test.md')
    await flow._run_checked()
    retained=rows(await db.query('SELECT * FROM document:docs_test_md;'))
    assert retained[0]['status']=='retracted' and retained[0]['body']
    print(f'COCOINDEX_RETENTION_VERIFIED: {len(ROOTS)} documents, real connector, removed source retained, zero model calls')
    await db.close()
asyncio.run(main())
