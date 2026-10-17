"""Verify progressive agent endpoints, full human catalog and portable domain packaging.
Inputs: synthetic authorized catalogs. Outputs: assertions. No real tools or network are invoked.
"""
import asyncio,io,json,time,unittest,zipfile
from unittest.mock import patch
import httpx
import app,domains
from response_projection import normalize,catalog_page

class DomainTests(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self):
        self.calls=[];domains.SUBMISSIONS.clear()
        async def fake(value,method,params=None,endpoint=None):
            self.calls.append((method,params,endpoint))
            if value['access_token']=='invalid':raise app.HTTPException(401,'Invalid credential')
            if method=='ping':return {}
            d=next((key for key,val in domains.DOMAINS.items() if val['server_id'] in (endpoint or '')),'hub')
            if method=='tools/list':return {'tools':[{'name':d+'-read','description':'Read a record','inputSchema':{'type':'object','properties':{},'additionalProperties':False},'annotations':{'readOnlyHint':True}}, {'name':d+'-write','description':'Modify a record','inputSchema':{'type':'object','properties':{},'additionalProperties':False},'annotations':{'readOnlyHint':False}}]}
            if method=='tools/call':return {'content':[{'type':'text','text':json.dumps({'ok':True,'tool':params['name']})}],'isError':False}
            return {}
        self.patcher=patch.object(app,'mcp',side_effect=fake);self.patcher.start()
        self.client=httpx.AsyncClient(transport=httpx.ASGITransport(app=app.app),base_url='https://mcp.mitechconsult.com')
        self.sid='domain-test-session'
        app.SESSIONS[self.sid]={'access_token':'synthetic-owner','expires':time.time()+3600,'until':time.time()+3600,'name':'Test owner','csrf':'unit-csrf','run_lock':asyncio.Lock(),'refresh_lock':asyncio.Lock(),'tools':{},'used_ids':set()}
        self.prefix=app.PREFIX
    async def asyncTearDown(self):
        self.patcher.stop();app.SESSIONS.clear();await self.client.aclose()
    async def rpc(self,path,name,args=None,rid=1):
        return await self.client.post(path,headers={'Authorization':'Bearer synthetic-owner'},json={'jsonrpc':'2.0','id':rid,'method':name,'params':args or {}})
    def user(self):
        self.client.cookies.set(app.COOKIE,self.sid)
    async def test_anonymous_endpoint_and_metadata(self):
        r=await self.client.post('/n8n/mcp',json={'method':'initialize','id':1})
        self.assertEqual(r.status_code,401)
        self.assertIn('/.well-known/oauth-protected-resource/n8n/mcp',r.headers['www-authenticate'])
        meta=await self.client.get('/.well-known/oauth-protected-resource/n8n/mcp')
        self.assertEqual(meta.status_code,200)
        self.assertEqual(meta.json()['resource'],domains.BASE+'/n8n/mcp')
    async def test_invalid_credential_cannot_list_tools(self):
        r=await self.client.post('/admin/mcp',headers={'Authorization':'Bearer invalid'},json={'jsonrpc':'2.0','id':1,'method':'tools/list'})
        self.assertEqual(r.status_code,401)
    async def test_agent_initial_surface_is_small(self):
        for domain in domains.DOMAINS:
            r=await self.rpc('/'+domain+'/mcp','tools/list')
            self.assertEqual(r.status_code,200)
            self.assertEqual(len(r.json()['result']['tools']),5)
        root=await self.rpc('/mcp','tools/list')
        self.assertEqual(len(root.json()['result']['tools']),6)
    async def test_discovery_is_bounded_and_schemas_are_on_demand(self):
        r=await self.rpc('/n8n/mcp','tools/call',{'name':'discover','arguments':{'query':'read','limit':1}})
        data=r.json()['result']['structuredContent']
        self.assertEqual(data['processor'],'duckdb')
        self.assertEqual(len(data['tools']),1)
        self.assertNotIn('inputSchema',data['tools'][0])
        r=await self.rpc('/n8n/mcp','tools/call',{'name':'inspect','arguments':{'name':'n8n-read'}})
        self.assertIn('inputSchema',r.json()['result']['structuredContent'])
    async def test_domain_boundary_and_read_write_classification(self):
        r=await self.rpc('/n8n/mcp','tools/call',{'name':'read','arguments':{'name':'admin-read','arguments':{}}})
        self.assertEqual(r.status_code,404)
        r=await self.rpc('/n8n/mcp','tools/call',{'name':'read','arguments':{'name':'n8n-write','arguments':{}}})
        self.assertEqual(r.status_code,409)
        self.assertFalse(any(c[0]=='tools/call' for c in self.calls))
    async def test_authorized_execution_is_not_replayed(self):
        args={'name':'execute','arguments':{'name':'n8n-write','arguments':{},'confirmed':True,'request_id':'test-request-0000001'}}
        first=await self.rpc('/n8n/mcp','tools/call',args)
        second=await self.rpc('/n8n/mcp','tools/call',args)
        self.assertEqual(first.status_code,200);self.assertEqual(second.status_code,409)
        self.assertEqual(sum(c[0]=='tools/call' for c in self.calls),1)
    async def test_human_catalog_ignores_package_selection(self):
        self.user();app.SESSIONS[self.sid]['selected_domains']=['docstore']
        r=await self.client.get(self.prefix+'/api/tools')
        self.assertEqual(r.status_code,200)
        names={t['name'] for t in r.json()['tools']}
        self.assertIn('n8n-write',names);self.assertIn('admin-write',names)
        self.assertIn('think--thinking-first-principles',names)
        self.assertEqual(len(r.json()['coverage']),7)
    async def test_marketplace_cannot_modify_selection_without_csrf(self):
        self.user()
        r=await self.client.post(self.prefix+'/api/domains',json={'domains':['n8n']})
        self.assertEqual(r.status_code,403)
        r=await self.client.post(self.prefix+'/api/domains',headers={'Origin':app.ORIGIN,'X-CSRF-Token':'unit-csrf'},json={'domains':['n8n']})
        self.assertEqual(r.status_code,200)
        catalog=await self.client.get(self.prefix+'/api/tools')
        self.assertIn('admin-write',{t['name'] for t in catalog.json()['tools']})
    async def test_complete_think_method_and_controls(self):
        r=await self.rpc('/think/mcp','tools/call',{'name':'read','arguments':{'name':'think--thinking-first-principles','arguments':{'depth':'micro','scope':'conversation','output':'silent'}}})
        self.assertEqual(r.status_code,200)
        packet=r.json()['result']['structuredContent']
        if packet.get('truncated'):
            entry=domains.RESULTS[packet['result_uri'][9:]];packet=json.loads(entry['raw'])
        self.assertEqual(packet['selection']['depth'],'micro')
        self.assertGreater(len(packet['methodology']),500)
        self.assertGreater(len(packet['persistent_control']),500)
        self.assertTrue(packet['host_action_required'])
        self.assertFalse(packet['executed_by_remote_model'])
    def test_resources_cannot_escape_packaged_tree(self):
        with self.assertRaises(app.HTTPException):domains.resource({'access_token':'x'},'think','think://../../../anything.md')
    def test_one_package_contains_selected_servers_and_full_method_sources(self):
        with zipfile.ZipFile(io.BytesIO(domains.package_bytes(['n8n','think']))) as z:
            servers=json.loads(z.read('propria/mcp.json'))['mcpServers']
            self.assertEqual(set(servers),{'n8n','think'})
            self.assertEqual(servers['n8n']['url'],domains.BASE+'/n8n/mcp')
            names=z.namelist()
            self.assertTrue(any('/reflex/SKILL.md' in n for n in names))
            self.assertTrue(any('/workflows/references/plan.md' in n for n in names))
            self.assertFalse(any('.secrets' in n or n.endswith('.env') for n in names))
    def test_every_think_entry_has_unique_identity_and_preserved_source(self):
        self.assertEqual(len({x['id'] for x in domains.THINK}),len(domains.THINK))
        self.assertEqual(sum(x['kind']=='workflow' and x['id'].startswith('workflow-') for x in domains.THINK),6)
        for entry in domains.THINK:self.assertTrue((domains.LIBRARY/entry['path']).is_file())
    def test_duckdb_projection_does_not_remove_error_evidence(self):
        value={'content':[{'type':'text','text':'not JSON, an actual source excerpt'}],'isError':True}
        self.assertEqual(normalize(value),value)
        projected=domains.pack({'access_token':'fake'}, {'content':[{'type':'text','text':'{"error":"missing"}'}],'isError':True})
        self.assertTrue(projected['isError']);self.assertEqual(projected['structuredContent']['error'],'missing')

if __name__=='__main__':unittest.main()
