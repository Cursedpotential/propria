"""Serve small agent discovery surfaces and a complete, independent human tool catalog.
Inputs: verified domain registry, caller OAuth identity and explicit tool selections.
Outputs: scoped MCP metadata/results, source-backed Think instructions and plugin packages.
Effects: only selected native tools execute; no retries, source mutation on discovery, or admin impersonation.
"""
import asyncio,hashlib,io,json,secrets,time,zipfile
from pathlib import Path
from urllib.parse import quote
from jsonschema import Draft202012Validator
from referencing import Registry
from starlette.exceptions import HTTPException
from starlette.responses import JSONResponse,Response
from starlette.routing import Route

ROOT=Path(__file__).parent
CONFIG=json.loads((ROOT/'domain_registry.json').read_text(encoding='utf-8')) if (ROOT/'domain_registry.json').exists() else {'domains':{}}
DOMAINS=CONFIG['domains']
BASE=CONFIG.get('base_url','https://mcp.mitechconsult.com')
LIBRARY=ROOT/'think-library'
THINK=json.loads((LIBRARY/'index.json').read_text(encoding='utf-8')) if (LIBRARY/'index.json').exists() else []
RESULTS={}
SUBMISSIONS={}
from response_projection import catalog_page, normalize
LIMIT=12000
HOST=None


def target(domain):
    """Resolve only operator-configured server IDs; caller input can never specify a URL."""
    identifier=CONFIG['auth_server_id'] if domain=='hub' else DOMAINS[domain]['server_id']
    if not isinstance(identifier,str) or len(identifier)!=32 or any(c not in '0123456789abcdef' for c in identifier):
        raise HTTPException(503,'Domain registry identity is invalid.')
    return BASE+'/servers/'+identifier+'/mcp'


def valid_domain(domain):
    """Reject unknown domain selections without silently routing to the old combined server."""
    if domain not in DOMAINS:raise HTTPException(400,'Select a registered domain.')
    return domain


def think_tools():
    """Describe every preserved Think process/workflow for direct human selection."""
    return [{'name':'think--'+item['id'],'title':item['title'],'description':item['description'],
             'inputSchema':{'type':'object','properties':{
               'depth':{'type':'string','enum':['adaptive','micro','focused','full'],'default':'adaptive'},
               'scope':{'type':'string','enum':['task','conversation'],'default':'conversation'},
               'output':{'type':'string','enum':['silent','normal','report'],'default':'silent'}},'additionalProperties':False},
             'annotations':{'readOnlyHint':True,'destructiveHint':False},'_kind':item['kind'],
             '_domain':'think','_source':item['source'],'_sha256':item['sha256']}
            for item in THINK if item.get('selectable',True)]


async def native_tools(value,domain):
    """Obtain caller-authorized schemas internally; only the human catalog returns all of them."""
    valid_domain(domain)
    if domain=='think':
        await HOST.mcp(value,'ping',endpoint=target(domain))
        return think_tools()
    result=await HOST.mcp(value,'tools/list',endpoint=target(domain))
    return [dict(t,_domain=domain) for t in result.get('tools',[]) if not t['name'].startswith('octopoda-')]


async def complete_catalog(value):
    """Collect all domains for the human runner, preserving failures and duplicate membership."""
    outcomes=await asyncio.gather(*[native_tools(value,d) for d in DOMAINS],return_exceptions=True)
    by_name={};coverage=[]
    for d,outcome in zip(DOMAINS,outcomes):
        if isinstance(outcome,BaseException):
            coverage.append({'domain':d,'state':'unavailable','reason':getattr(outcome,'detail','Connection failed')})
            continue
        coverage.append({'domain':d,'state':'loaded','count':len(outcome)})
        for t in outcome:
            if t['name'] in by_name:
                by_name[t['name']]['_domains'].append(d)
            else:by_name[t['name']]=dict(t,_domains=[d])
    return {'tools':list(by_name.values()),'coverage':coverage}


async def human_catalog(request):
    """List the FULL human catalog; no agent discovery sequence or install selection filters it."""
    value=HOST.session(request)
    result=await complete_catalog(value)
    value['tools']={t['name']:t for t in result['tools']}
    return JSONResponse(result)


def domain_cards():
    """Return one concise domain entry each, not the full tool or resource definitions."""
    return [{'id':d,'name':v['name'],'mcp_url':BASE+v['path'],'native_tools':len(think_tools()) if d=='think' else v['native_count'],
             'state':v['availability'],'skills':d,'resources':True} for d,v in DOMAINS.items()]


async def marketplace(request):
    """Show selectable domain packages without changing infrastructure or hiding human tools."""
    value=HOST.session(request)
    return JSONResponse({'domains':domain_cards(),'selected':value.get('selected_domains',list(DOMAINS)),
                         'single_connection':BASE+'/mcp','selection_changes':'package inclusion only'})


async def select_domains(request):
    """Record this runner session's package choice after CSRF validation; not a global rule."""
    value=HOST.session(request);HOST.check_csrf(request,value)
    payload=await bounded_json(request,8192)
    selected=payload.get('domains')
    if not isinstance(selected,list) or not selected or len(selected)>len(DOMAINS) or any(d not in DOMAINS for d in selected):
        raise HTTPException(400,'Select one or more valid domains.')
    value['selected_domains']=list(dict.fromkeys(selected))
    return JSONResponse({'selected':value['selected_domains'],'servers_stopped':0,'human_catalog_filtered':False})


def package_bytes(selected):
    """Build one portable multi-MCP plugin with scoped skills and preserved Think resources."""
    buf=io.BytesIO()
    with zipfile.ZipFile(buf,'w',zipfile.ZIP_DEFLATED) as z:
        manifest={'$schema':'https://agent-plugins.org/schemas/1.0.0/plugin.schema.json','name':'propria','version':'0.2.0',
                  'description':'Domain MCP servers, progressive agent skills and the complete human runner.','author':{'name':'Matthew Salem'}}
        z.writestr('propria/plugin.json',json.dumps(manifest,indent=2))
        servers={d:{'type':'streamable-http','url':BASE+DOMAINS[d]['path']} for d in selected}
        z.writestr('propria/mcp.json',json.dumps({'$schema':'https://agent-plugins.org/schemas/1.0.0/mcp.schema.json','mcpServers':servers},indent=2))
        for domain in selected:
            z.writestr('propria/skills/'+domain+'/SKILL.md',skill(domain))
        z.writestr('propria/README.md','# Propria\n\nInstall this one package to configure the selected domain MCP servers.\nHuman runner: '+BASE+'/runner/\nSingle-URL alternative: '+BASE+'/mcp\nSeparate OAuth grants can still be required by the host. Package inclusion is not authorization.\n')
        if 'think' in selected:
            for path in LIBRARY.rglob('*'):
                if path.is_file():z.writestr('propria/skills/think/references/'+path.relative_to(LIBRARY).as_posix(),path.read_bytes())
    return buf.getvalue()


async def package_download(request):
    """Export exactly the selected domain connections as one plugin; never include credentials."""
    value=HOST.session(request)
    data=package_bytes(value.get('selected_domains',list(DOMAINS)))
    return Response(data,media_type='application/zip',headers={'Content-Disposition':'attachment; filename="propria-domain-plugin.zip"'})


def skill(domain):
    """Generate domain entry guidance from the actual discovery interface, without flat tool dumps."""
    text=f'''---\nname: propria-{domain}\ndescription: Use the Propria {domain} domain. Discover only the needed native tool or Think process; preserve explicit human choices.\n---\n# Propria {domain}\n\nCall `discover` with a focused query. Call `inspect` for the chosen exact tool name before execution. Use `read` only for explicitly read-only tools; `execute` retains write/admin confirmations and native permissions. The endpoint is `{BASE}{DOMAINS[domain]['path']}`.\n\nDo not enumerate every domain's full schemas. Do not substitute a different store or tool after an availability failure. Registered is not invocation-verified.\n\nThe human runner at `{BASE}/runner/` deliberately shows ALL tools and bypasses agent discovery. Its Connections tab chooses which domain connections go into one installable package; it does not stop servers or filter human access.\n\nTool results retain errors, source references and explicit truncation. Use the returned result URI to expand omitted content. Never treat missing/partial data as a clean result. Local CCC/CNF require their authorized workstation bridge; do not create remote replacement indexes.\n'''
    if domain=='think':text+='\nChoose individual process, integrated workflow, or router explicitly. Depth (adaptive/Micro/Focused/Full), scope (task/conversation), and output (silent/normal/report) are independent. Read the complete selected method and persistent Reflex resources before applying it. Micro performs the real method silently; it is not a keyword or a shorter report. Manual selection is not overridden by routing. A returned activation packet supplies instructions to the host; it does not prove another conversation adopted the process. Preserve the selected process in this conversation until changed or ended.\n'
    return text


async def bounded_json(request,limit=64000):
    """Read bounded JSON without loading oversized requests or resolving arbitrary URLs."""
    raw=bytearray()
    async for chunk in request.stream():
        raw.extend(chunk)
        if len(raw)>limit:raise HTTPException(413,'Input exceeds limit.')
    try:data=json.loads(raw)
    except ValueError:raise HTTPException(400,'Invalid JSON.')
    if not isinstance(data,dict):raise HTTPException(400,'Expected a JSON object.')
    return data


def summary_schema(name,description,properties,required=(),write=False):
    """Expose a small, truthfully annotated interface rather than every native tool definition."""
    return {'name':name,'description':description,'inputSchema':{'type':'object','properties':properties,'required':list(required),'additionalProperties':False},
            'annotations':{'readOnlyHint':not write,'destructiveHint':write,'idempotentHint':not write,'openWorldHint':write}}


def interface(domain):
    """Return stable progressive MCP tools; executions never masquerade as read-only discovery."""
    choose={'domain':{'type':'string','enum':list(DOMAINS)}} if domain=='hub' else {}
    required=['domain'] if domain=='hub' else []
    common={'name':{'type':'string','minLength':1}}
    tools=[summary_schema('discover','Find native tools or Think processes in one domain; summaries only.',{**choose,'query':{'type':'string'},'offset':{'type':'integer','minimum':0},'limit':{'type':'integer','minimum':1,'maximum':50}},required),
           summary_schema('inspect','Load the exact selected tool schema, effects and source guidance.',{**choose,**common},required+['name']),
           summary_schema('read','Execute one native tool declared read-only, or load a complete Think activation packet.',{**choose,**common,'arguments':{'type':'object'}},required+['name']),
           summary_schema('execute','Execute a selected native operation; it may modify/delete data or deploy infrastructure. Inspect first and obtain user authorization.',{**choose,**common,'arguments':{'type':'object'},'confirmed':{'type':'boolean'},'request_id':{'type':'string','minLength':16,'maxLength':100}},required+['name','request_id','confirmed'],True),
           summary_schema('resource','Read domain guidance or expand a result/Think resource by exact URI.',{**choose,'uri':{'type':'string'},'offset':{'type':'integer','minimum':0},'limit':{'type':'integer','minimum':1,'maximum':12000}},required+['uri'])]
    if domain=='hub':tools.insert(0,summary_schema('domains','List domain connections and open the full human runner or marketplace.',{}))
    return tools


def owner_key(value):
    """Bind stored response expansions to the same credential without storing it in a result URI."""
    return hashlib.sha256(value['access_token'].encode()).hexdigest()


def pack(value,result):
    """Return bounded data while retaining the unchanged native result for authorized expansion."""
    native_error=bool(isinstance(result,dict) and result.get('isError'))
    result=normalize(result)
    raw=json.dumps(result,ensure_ascii=False)
    if len(raw)<=LIMIT:return {'content':[{'type':'text','text':raw}],'structuredContent':result if isinstance(result,dict) else {'items':result},'isError':native_error}
    now=time.time()
    for key in list(RESULTS):
        if RESULTS[key]['until']<now:RESULTS.pop(key,None)
    while len(RESULTS)>=32:RESULTS.pop(next(iter(RESULTS)))
    ref=secrets.token_urlsafe(24)
    RESULTS[ref]={'owner':owner_key(value),'raw':raw,'until':now+1800}
    data={'truncated':True,'native_is_error':native_error,'original_characters':len(raw),'preview':raw[:4000],'result_uri':'result://'+ref,'expires_in_seconds':1800,'next_offset':0}
    return {'content':[{'type':'text','text':json.dumps(data)}],'structuredContent':data,'isError':data['native_is_error']}


def resource(value,domain,uri,offset=0,limit=12000):
    """Read whitelisted packaged instructions or caller-owned result slices, never arbitrary files."""
    if uri=='skill://'+domain:raw=skill(domain)
    elif uri.startswith('think://') and domain=='think':
        relative=uri[8:]
        path=(LIBRARY/relative).resolve()
        if not path.is_relative_to(LIBRARY.resolve()) or not path.is_file() or path.suffix!='.md':raise HTTPException(404,'Resource not found.')
        raw=path.read_text(encoding='utf-8')
    elif uri.startswith('result://'):
        saved=RESULTS.get(uri[9:])
        if not saved or saved['until']<time.time() or saved['owner']!=owner_key(value):raise HTTPException(404,'Result expired or belongs to a different connection.')
        raw=saved['raw']
    else:raise HTTPException(404,'Unknown resource URI.')
    return {'uri':uri,'text':raw[offset:offset+limit],'total_characters':len(raw),'next_offset':offset+limit if offset+limit<len(raw) else None}


def activate(name,args):
    """Return the original selected methodology and Reflex instructions; never claim to execute a model's thinking."""
    entry=next((x for x in THINK if 'think--'+x['id']==name),None)
    if not entry:raise HTTPException(404,'Think process is unavailable.')
    args={'depth':'adaptive','scope':'conversation','output':'silent',**args}
    text=(LIBRARY/entry['path']).read_text(encoding='utf-8')
    reflex=next((x for x in THINK if x['id']=='reflex'),None)
    control=(LIBRARY/reflex['path']).read_text(encoding='utf-8') if reflex else ''
    workflow=(LIBRARY/'skills/workflows/SKILL.md').read_text(encoding='utf-8') if entry['id'].startswith('workflow-') else ''
    return {'kind':'think-activation','selected':entry['id'],'selection':args,'source':entry['source'],'sha256':entry['sha256'],'workflow_control':workflow,
      'methodology':text,'persistent_control':control,'resources':['think://'+entry['path']]+(['think://'+reflex['path']] if reflex else [])+(['think://skills/workflows/SKILL.md'] if workflow else []),
      'instruction':'Apply the complete selected process in the requested scope. Depth and output are independent. Micro is real silent reasoning, not a checklist/report. Preserve manual choice; do not route away without permission. Maintain conversation scope until the user changes it.',
      'host_action_required':True,'executed_by_remote_model':False}


async def invoke(value,domain,name,args,read_only=False,confirmed=False):
    """Validate against current caller-authorized native schema, then execute only the exact chosen operation."""
    catalog=await native_tools(value,domain)
    tool=next((t for t in catalog if t['name']==name),None)
    if not tool:raise HTTPException(404,'Tool not available in this domain for this caller.')
    try:Draft202012Validator(tool['inputSchema'],registry=Registry()).validate(args)
    except Exception:raise HTTPException(422,'Inputs do not match the selected tool schema.')
    declared_read=tool.get('annotations',{}).get('readOnlyHint') is True
    if read_only and not declared_read:raise HTTPException(409,'This tool is not declared read-only; use the explicitly authorized execute action.')
    if not declared_read and not confirmed:raise HTTPException(409,'Explicit authorization is required for this operation.')
    if domain=='think':return activate(name,args)
    return await HOST.mcp(value,'tools/call',{'name':name,'arguments':args},endpoint=target(domain))


async def progressive(request):
    """Serve scoped agent MCP with OAuth delegated to its native ContextForge resource; no service identity substitution."""
    domain=request.path_params.get('domain','hub')
    if domain!='hub':valid_domain(domain)
    public=BASE+('/mcp' if domain=='hub' else DOMAINS[domain]['path'])
    if request.headers.get('origin') not in (None,BASE):raise HTTPException(403,'Origin not allowed.')
    header=request.headers.get('authorization','')
    if not header.startswith('Bearer '):
        return JSONResponse({'error':'Authentication required'},401,headers={'WWW-Authenticate':'Bearer resource_metadata="'+BASE+'/.well-known/oauth-protected-resource'+('/mcp' if domain=='hub' else DOMAINS[domain]['path'])+'"'})
    value={'access_token':header[7:],'expires':time.time()+600,'refresh_lock':asyncio.Lock()}
    # The gateway validates issuer, audience and access. Never trust mere presence of a JWT.
    await HOST.mcp(value,'ping',endpoint=target(domain))
    data=await bounded_json(request)
    method=data.get('method');rid=data.get('id');params=data.get('params') or {}
    if method=='notifications/initialized':return Response(status_code=202)
    if method=='initialize':result={'protocolVersion':'2025-06-18','capabilities':{'tools':{},'resources':{}},'serverInfo':{'name':'propria-'+domain,'version':'0.2.0'},'instructions':'Use domains/discover, inspect only the selected tool, then read or execute. Human runner shows all tools directly. Never load all schemas by default. Octopoda is excluded. Preserve manual Think selections and full methodology.'}
    elif method=='ping':result={}
    elif method=='tools/list':result={'tools':interface(domain)}
    elif method=='resources/list':result={'resources':[{'uri':'skill://'+d,'name':d+' instructions','mimeType':'text/markdown'} for d in (DOMAINS if domain=='hub' else [domain])]}
    elif method=='tools/call':
        action=params.get('name');args=params.get('arguments') or {}
        defined=next((t for t in interface(domain) if t['name']==action),None)
        if not defined:raise HTTPException(400,'Unknown discovery action.')
        try:Draft202012Validator(defined['inputSchema']).validate(args)
        except Exception:raise HTTPException(422,'Invalid discovery inputs.')
        chosen=valid_domain(args.get('domain')) if domain=='hub' and action!='domains' else domain
        if action=='domains':output={'domains':domain_cards(),'human_runner':BASE+'/runner/','marketplace':BASE+'/runner/?view=connections'}
        elif action=='resource':output=resource(value,chosen,args['uri'],args.get('offset',0),args.get('limit',12000))
        elif action in ('discover','inspect'):
            catalog=await native_tools(value,chosen)
            if action=='inspect':
                output=next((t for t in catalog if t['name']==args['name']),None)
                if not output:raise HTTPException(404,'Tool unavailable in this domain.')
            else:
                q=args.get('query','');start=args.get('offset',0);size=args.get('limit',20)
                matches,total=catalog_page(catalog,q,start,size)
                output={'domain':chosen,'total':total,'processor':'duckdb','tools':[{'name':t['name'],'title':t.get('title',t['name']),'description':(t.get('description') or '').split('\n\n')[0][:240],'read_only':t.get('annotations',{}).get('readOnlyHint') is True} for t in matches],'next_offset':start+size if start+size<total else None}
        else:
            if rid is None:raise HTTPException(400,'Tool execution requires a JSON-RPC request ID.')
            if action=='execute':
                now=time.time()
                for k in list(SUBMISSIONS):
                    if SUBMISSIONS[k]<now:SUBMISSIONS.pop(k,None)
                key=(owner_key(value),args['request_id'])
                if key in SUBMISSIONS:raise HTTPException(409,'This request was already submitted; it will not run again.')
                if len(SUBMISSIONS)>=5000:raise HTTPException(429,'Submission receipt limit reached.')
                SUBMISSIONS[key]=now+3600
            output=await invoke(value,chosen,args['name'],args.get('arguments',{}),action=='read',args.get('confirmed',False))
        result=pack(value,output)
    elif method=='resources/read':
        uri=params.get('uri','');chosen=valid_domain(uri[8:]) if uri.startswith('skill://') else domain
        text=resource(value,chosen,uri,limit=1_000_000)['text'];result={'contents':[{'uri':uri,'mimeType':'text/markdown','text':text}]}
    else:return JSONResponse({'jsonrpc':'2.0','id':rid,'error':{'code':-32601,'message':'Unsupported method'}})
    return JSONResponse({'jsonrpc':'2.0','id':rid,'result':result})


async def metadata(request):
    """Publish short-URL protected-resource metadata with the existing authorization server."""
    suffix=request.path_params.get('resource','mcp')
    allowed=['mcp']+[d+'/mcp' for d in DOMAINS]
    if suffix not in allowed:raise HTTPException(404,'Unknown protected resource.')
    return JSONResponse({'resource':BASE+'/'+suffix,'authorization_servers':CONFIG.get('oauth',{}).get('authorization_servers',[HOST.ISSUER]),'scopes_supported':CONFIG.get('oauth',{}).get('scopes_supported',['openid','profile','email','offline_access']),'bearer_methods_supported':['header']})


def install(host):
    """Extend the existing runner application in place; keep human and machine surfaces separate."""
    global HOST;HOST=host
    for route in host.app.routes:
        if getattr(route,'path','')==host.PREFIX+'/api/tools':route.endpoint=human_catalog;route.app=Route(route.path,human_catalog).app
    host.app.router.routes.extend([
      Route('/mcp',progressive,methods=['POST']),Route('/{domain}/mcp',progressive,methods=['POST']),
      Route('/.well-known/oauth-protected-resource/{resource:path}',metadata),
      Route(host.PREFIX+'/api/domains',marketplace),Route(host.PREFIX+'/api/domains',select_domains,methods=['POST']),
      Route(host.PREFIX+'/api/package',package_download)])
