"""Register the existing Superindex search APIs in the existing scoped Search MCP server.
Inputs: authorized catalog helper, live OpenAPI contracts, fixed private Superindex origin.
Outputs: exact source-backed tool schemas and a bounded verification receipt.
Effects with --apply: add only three REST tool registrations and their Search membership;
no index creation, source edits, authentication weakening, or unrelated server changes.
"""
from __future__ import annotations
import argparse,copy,hashlib,importlib.util,json,time,urllib.error,urllib.request
from pathlib import Path

ROOT=Path(__file__).resolve().parent
SOURCE='http://100.91.190.107:8765'
ROUTES={'superindex-chunks-search':'/chunks/search','superindex-filesystem-search':'/filesystem/search','superindex-images-search':'/images/search'}


def request(method,url,body=None,token=None):
    """Request a fixed known API without printing credentials, source records, or raw failures."""
    headers={'Content-Type':'application/json','Accept':'application/json'}
    if token:headers['Authorization']='Bearer '+token
    req=urllib.request.Request(url,method=method,headers=headers,data=None if body is None else json.dumps(body).encode())
    try:
        with urllib.request.urlopen(req,timeout=35) as response:
            raw=response.read(4_000_001)
            if len(raw)>4_000_000:raise RuntimeError('API response exceeded bound')
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f'API rejected {method} with HTTP {exc.code}') from None


def expand(schema,document,seen=()):
    """Inline only source-local OpenAPI references; never fetch a schema URL."""
    if isinstance(schema,list):return [expand(s,document,seen) for s in schema]
    if not isinstance(schema,dict):return schema
    if '$ref' in schema:
        ref=schema['$ref']
        if not ref.startswith('#/components/schemas/') or ref in seen:raise RuntimeError('Unsupported schema reference')
        target=document['components']['schemas'][ref.split('/')[-1]]
        return expand(target,document,seen+(ref,))
    return {k:expand(v,document,seen) for k,v in schema.items()}


def main():
    """Inspect first, apply idempotently when requested, and verify through the real Search MCP path."""
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--apply',action='store_true')
    parser.add_argument('--catalog-helper',default=r'E:\AI_Workspace\plugins\plugins\propria-toolbox\scripts\generate_catalog.py')
    args=parser.parse_args()
    spec=importlib.util.spec_from_file_location('existing_catalog',args.catalog_helper)
    helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
    token=helper.cf_token();base=helper.CF_URL
    call=lambda method,path,body=None:request(method,base+path,body,token)
    data=call('GET','/servers?include_inactive=true')
    servers=data if isinstance(data,list) else data['servers']
    selected=[s for s in servers if s['name']=='search']
    if len(selected)!=1:raise RuntimeError('Expected one existing Search server')
    server=selected[0]
    if not server.get('enabled') or not server.get('oauthEnabled'):raise RuntimeError('Search authorization is not ready')
    data=call('GET','/tools?include_inactive=true&limit=0')
    tools=data if isinstance(data,list) else data['tools']
    baseline=next(t for t in tools if t['name']=='memsearch-search')
    existing={t['name']:t for t in tools}
    api=request('GET',SOURCE+'/openapi.json')
    proposals=[]
    for name,path in ROUTES.items():
        operation=api['paths'][path]['post']
        inputs=expand(operation['requestBody']['content']['application/json']['schema'],api)
        inputs['additionalProperties']=False
        description=operation.get('description') or operation['summary']+'.'
        description+='\n\nInputs: the native Superindex query, mode, limits and lane-specific filters. Outputs: native hits, source locators, scores and coverage. Effects: read-only search of existing indexes; no source or index mutation. Choose this lane directly rather than scanning journals or creating a replacement index.'
        definition={'name':name,'title':operation['summary'],'description':description,'url':SOURCE+path,'integration_type':'REST','request_type':'POST','inputSchema':inputs,
           'annotations':{'readOnlyHint':True,'destructiveHint':False,'idempotentHint':True,'openWorldHint':True},
           'headers':{'Content-Type':'application/json','Accept':'application/json'},'timeout_ms':35000,'expose_passthrough':False,
           'tags':['propria','search','superindex'],'visibility':'team' if baseline.get('teamId') else 'private'}
        if baseline.get('teamId'):definition['team_id']=baseline['teamId']
        if baseline.get('ownerEmail'):definition['owner_email']=baseline['ownerEmail']
        old=existing.get(name)
        if old and (old.get('url')!=definition['url'] or old.get('integrationType',old.get('integration_type'))!='REST'):
            raise RuntimeError('Existing tool with this name targets a different source; no changes')
        proposals.append(definition)
    print(json.dumps({'mode':'apply' if args.apply else 'plan','server':server['name'],'tools':[{'name':p['name'],'required':p['inputSchema']['required'],'inputs':list(p['inputSchema']['properties'])} for p in proposals]}),flush=True)
    if not args.apply:return
    backup=Path.home()/'.secrets'/('superindex-search-membership-'+time.strftime('%Y%m%d-%H%M%S')+'.json')
    before=list(server.get('associatedToolIds',[]))
    backup.write_text(json.dumps({'server_id':server['id'],'associatedTools':before},indent=2),encoding='utf-8')
    created=[];incoming=[]
    try:
        for definition in proposals:
            found=existing.get(definition['name'])
            if found is None:
                found=call('POST','/tools',{'tool':definition,'team_id':baseline.get('teamId')})
                if isinstance(found,dict) and 'tool' in found:found=found['tool']
                if not isinstance(found,dict) or not found.get('id'):
                    data=call('GET','/tools?include_inactive=true&limit=0');all_tools=data if isinstance(data,list) else data['tools']
                    found=next(t for t in all_tools if t['name']==definition['name'])
                created.append(found['id'])
            incoming.append(found['id'])
        union=list(dict.fromkeys(before+incoming))
        if union!=before:call('PUT','/servers/'+server['id'],{'associatedTools':union})
        check=call('GET','/servers/'+server['id'])
        if set(check.get('associatedToolIds',[]))!=set(union) or not check.get('oauthEnabled'):
            raise RuntimeError('Search membership verification failed')
    except Exception:
        call('PUT','/servers/'+server['id'],{'associatedTools':before})
        for identifier in created:call('DELETE','/tools/'+identifier)
        raise
    receipt={'registered':True,'created':len(created),'source_indexes_created':0,'native_tools':len(union),'calls':[]}
    for definition in proposals:
        message={'jsonrpc':'2.0','id':1,'method':'tools/call','params':{'name':'read','arguments':{'name':definition['name'],'arguments':{'query':'Propria','limit':1,'mode':'keyword'}}}}
        try:
            result=request('POST','https://mcp.mitechconsult.com/search/mcp',message,token)['result']
            payload=result.get('structuredContent',{})
            receipt['calls'].append({'tool':definition['name'],'is_error':result.get('isError',False),'fields':list(payload),'hits':len(payload.get('hits',[])) if isinstance(payload.get('hits'),list) else None})
        except Exception as exc:receipt['calls'].append({'tool':definition['name'],'verified':False,'error':str(exc)[:120]})
    registry_path=ROOT/'domain_registry.json'
    registry=json.loads(registry_path.read_text(encoding='utf-8'))
    if registry['domains']['search']['server_id']!=server['id']:raise RuntimeError('Local domain identity differs; refused registry edit')
    registry['domains']['search']['native_count']=len(union)
    registry_path.write_text(json.dumps(registry,indent=2)+'\n',encoding='utf-8')
    receipt['verified_at']=time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime())
    (ROOT/'superindex-registration-receipt.json').write_text(json.dumps(receipt,indent=2),encoding='utf-8')
    print(json.dumps(receipt),flush=True)

if __name__=='__main__':main()
