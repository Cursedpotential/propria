/** Proves the actual template and worker write SQL under record-access identities on an isolated SurrealDB 3.2.4 process.
 * Inputs: exact server binary, template, repository.go, currency_review.go and optional Linux Go test executable.
 * Outputs: safe version/assertion summary. Effects: synthetic memory-only DB/process; no production connection, credentials or deployment.
 * Choose over embedded 3.0.x tests when checking native 3.2.4 permissions; retain files and stop only this fixture process.
 * Byline: Codex · GPT-6.1 · 2026-10-04.
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { spawn } from 'node:child_process';
import { once } from 'node:events';

const [binary, templateFile, repositoryFile, currencyFile, goTest] = process.argv.slice(2);
if (!currencyFile) throw new Error('explicit fixture paths required');
const endpoint = 'http://127.0.0.1:18045';
const namespace = 'toolkit_scope_fixture';
const database = 'case';
const rootPassword = 'synthetic-memory-only-root';
const recordPassword = 'synthetic-record-password';
const root = 'Basic ' + Buffer.from(`fixture_root:${rootPassword}`).toString('base64');
const processFixture = spawn(binary, ['start', '--bind', '127.0.0.1:18045', '--user', 'fixture_root', '--pass', rootPassword, 'memory'], { stdio: ['ignore','ignore','pipe'], env:{...process.env,...(process.env.TOOLKIT_SCOPE_CA_FILE?{SSL_CERT_FILE:process.env.TOOLKIT_SCOPE_CA_FILE}:{})} });
let startupError='';
processFixture.stderr.on('data',chunk=>{if(startupError.length<4096) startupError+=chunk;});
processFixture.on('error',error=>{startupError=error.code;});
let assertions = 0;
const key = '11111111-2222-3333-4444-555555555555';
const sourceFile = readFileSync(repositoryFile,'utf8');
const receiptSQL = /const commitSQL = `([\s\S]*?)`/.exec(sourceFile)?.[1];
const currencySQL = /const currencyCommitSQL = `([\s\S]*?)`/.exec(readFileSync(currencyFile,'utf8'))?.[1];
assert.ok(receiptSQL && currencySQL);

/** Executes every statement and preserves substantive THROW errors rather than trailing NotExecuted markers. */
async function query(auth,sql,variables={},selected=true) {
  const prefix=Object.keys(variables).length;
  if(Object.keys(variables).length){sql=Object.keys(variables).sort().map(key=>{assert.match(key,/^[A-Za-z][A-Za-z0-9_]{0,63}$/);return `LET $${key} = encoding::json::decode($toolkit_bound_json).${key};\n`;}).join('')+sql;variables={toolkit_bound_json:JSON.stringify(variables)};}
  const response = await fetch(endpoint+'/rpc',{method:'POST',headers:{Authorization:auth,'Content-Type':'application/json',...(selected?{'surreal-ns':namespace,'surreal-db':database}:{})},body:JSON.stringify({id:1,method:'query',params:[sql,variables]})});
  const value=await response.json();
  if(!response.ok||value.error) throw new Error('fixture query RPC failed: '+JSON.stringify(value.error ?? {status:response.status}));
  const errors=value.result.filter(r=>r.status!=='OK').map(r=>String(r.result));
  if(errors.length) throw new Error(errors.find(e=>!/not executed|notexecuted|failed transaction|cancelled/i.test(e))??errors[0]);
  return value.result.slice(prefix).map(r=>r.result);
}

/** Signs in a provisioned record principal; inputs: fixed access/name; outputs: private bearer header only. */
async function signin(access,username) {
  const response=await fetch(endpoint+'/signin',{method:'POST',headers:{'Content-Type':'application/json',Accept:'application/json'},body:JSON.stringify({NS:namespace,DB:database,AC:access,username,password:recordPassword})});
  const value=await response.json();
  assert.equal(response.status,200);assert.equal(typeof value.token,'string');assertions+=2;
  return 'Bearer '+value.token;
}

/** Confirms denied writes by root readback; an OK response with filtered empty results is not proof of a write. */
async function denied(auth,sql,readback,variables={}) {
  const before=JSON.stringify(await query(root,readback,variables));
  try { await query(auth,sql,variables); } catch { /* Permission rejection is expected, effect readback is authoritative. */ }
  assert.equal(JSON.stringify(await query(root,readback,variables)),before);assertions++;
}

try {
  let ready=false;
  for(let n=0;n<100;n++){try{const r=await fetch(endpoint+'/health');if(r.ok){ready=true;break;}}catch{}await new Promise(resolve=>setTimeout(resolve,100));}
  assert.ok(ready,'isolated server startup failed: '+startupError);assertions++;
  const version=await (await fetch(endpoint+'/version')).text();
  assert.match(version,/3\.2\.4/);assertions++;
  await query(root,'DEFINE NAMESPACE toolkit_scope_fixture; USE NS toolkit_scope_fixture; DEFINE DATABASE case;',{},false);
  await query(root,'DEFINE TABLE source SCHEMALESS; DEFINE TABLE reference SCHEMALESS; DEFINE TABLE library_proposal SCHEMALESS; DEFINE TABLE unrelated SCHEMALESS; CREATE source:official CONTENT {primary_url:"https://www.courts.michigan.gov/fixture"}; CREATE reference:fixture CONTENT {personal_note:"Synthetic full personal fields remain retained."}; CREATE unrelated:fixture CONTENT {body:"unrelated retained"};');
  const hashes=await query(root,'RETURN crypto::argon2::generate($password); RETURN crypto::argon2::generate($password);',{password:recordPassword});
  let template=readFileSync(templateFile,'utf8').replace(/CANCEL TRANSACTION;\s*$/,'COMMIT TRANSACTION;');
  await query(root,template,{validator_username:'fixture-validator',validator_password_hash:hashes[0],currency_username:'fixture-currency',currency_password_hash:hashes[1]});assertions++;
  const validator=await signin('toolkit_validator','fixture-validator');
  const currency=await signin('toolkit_currency','fixture-currency');
  const [sourceVersion,targetVersion]=await query(root,"RETURN 'sha256:' + crypto::sha256(<string> (SELECT * OMIT embedding FROM ONLY source:official)); RETURN 'sha256:' + crypto::sha256(<string> (SELECT * OMIT embedding FROM ONLY reference:fixture));");
  const citation={source_id:'source:official',source_version:sourceVersion,pinpoint:'page:1',claim:'Synthetic exact fixture claim.'};
  const proposed={personal_note:'Synthetic retained correction.'};
  const [proposedHash]=await query(root,"RETURN 'sha256:' + crypto::sha256(<string> $proposed);",{proposed});
  await query(root,"CREATE type::record('library_proposal',$key) CONTENT {target:reference:fixture,expected_version:$targetVersion,status:'pending_validation',proposed_hash:$proposedHash,proposed_record:$proposed,citations:[$citation]};",{key,targetVersion,proposedHash,proposed,citation});
  const [proposalVersion]=await query(root,"RETURN 'sha256:' + crypto::sha256(<string> object::remove((SELECT * OMIT embedding FROM ONLY type::record('library_proposal',$key)),['dispatch','dispatch_at']));",{key});
  const now=new Date(Date.now()-1000).toISOString();
  const expires=new Date(Date.now()+3600000).toISOString();
  const check={...citation,status:'BLOCKED',currency_status:'provisional',evidence_time:now,check_version:'fixture-check'};
  const receipt={proposal_id:'library_proposal:'+key,proposal_version:proposalVersion,proposed_hash:proposedHash,status:'BLOCKED',currency_status:'provisional',validator_version:'fixture-validator',completed_at:now,expires_at:expires,claims:[citation],claim_checks:[check],signature:'synthetic-signed-receipt',attempt_id:'fixture-run'};
  assert.deepEqual((await query(validator,"LET $p = (SELECT * OMIT embedding FROM ONLY type::record('library_proposal',$key)); RETURN ['sha256:' + crypto::sha256(<string> object::remove($p,['dispatch','dispatch_at'])) = $receipt.proposal_version, $p.proposed_hash = $receipt.proposed_hash, 'sha256:' + crypto::sha256(<string> $p.proposed_record) = $receipt.proposed_hash, $p.citations = $receipt.claims];",{key,receipt}))[1],[true,true,true,true]);assertions++;
  await query(validator,receiptSQL,{key,receipt});assertions++;
  await query(validator,receiptSQL,{key,receipt});assertions++;
  const artifact={uri:'b2://synthetic/derivative?versionId=v1',version_id:'v1',sha256:'sha256:'+'a'.repeat(64),bytes:100};
  const evidence={source_id:'source:official',source_version:sourceVersion,snapshot_sha256:'a'.repeat(64),primary_url:'https://www.courts.michigan.gov/fixture',release_url:'https://www.courts.michigan.gov/release-fixture',release_version:'synthetic-release',release_sha256:'b'.repeat(64),review_id:'synthetic-review',reviewer_id:'synthetic-authenticated-reviewer',decision_id:'synthetic-authorized-decision',release_pinpoint:'page:1',release_quote_sha256:'c'.repeat(64),approval_ref:artifact,source_snapshot_ref:artifact,release_snapshot_ref:artifact,effective_at:now,reviewed_through:now,checked_at:now,expires_at:expires,status:'cleared',signature:'synthetic-currency-signature'};
  await query(currency,currencySQL,{evidence,proposal_id:receipt.proposal_id,proposal_version:proposalVersion,proposed_hash:proposedHash});assertions++;
  await query(currency,currencySQL,{evidence,proposal_id:receipt.proposal_id,proposal_version:proposalVersion,proposed_hash:proposedHash});assertions++;
  for(const auth of [validator,currency]) {
    for(const table of ['source','reference','library_proposal','unrelated','toolkit_service']) {
      await denied(auth,`UPDATE ${table} SET forbidden_change = true;`,`SELECT * FROM ${table};`);
      await denied(auth,`DELETE ${table};`,`SELECT * FROM ${table};`);
      await denied(auth,`CREATE ${table}:forbidden CONTENT {forbidden_change:true};`,`SELECT * FROM ${table};`);
    }
  }
  await denied(validator,"UPDATE library_currency SET signature='forged';",'SELECT * FROM library_currency;');
  await denied(currency,"UPDATE library_validation SET signature='forged';",'SELECT * FROM library_validation;');
  for(const auth of [validator,currency]) for(const table of ['library_currency','library_validation']) await denied(auth,`DELETE ${table};`,`SELECT * FROM ${table};`);
  await denied('',"CREATE library_validation:ordinary CONTENT {status:'VERIFIED_PRIMARY'};",'SELECT * FROM library_validation;');
  await denied('',"UPDATE library_currency SET signature='ordinary-forged';",'SELECT * FROM library_currency;');
  assert.equal((await query(validator,'SELECT * FROM source;'))[0].length,1);assertions++;
  assert.equal((await query(currency,'SELECT * FROM source;'))[0].length,1);assertions++;
  await denied(validator,"CREATE library_currency:forbidden CONTENT $evidence;",'SELECT * FROM library_currency;',{evidence});
  await denied(currency,"CREATE library_validation:forbidden CONTENT $receipt;",'SELECT * FROM library_validation;',{receipt});
  // Demonstrate why a system EDITOR must never be called table-scoped.
  await query(root,"DEFINE USER fixture_editor ON DATABASE PASSWORD 'synthetic-record-password' ROLES EDITOR;");
  const editor='Basic '+Buffer.from('fixture_editor:'+recordPassword).toString('base64');
  const response=await fetch(endpoint+'/rpc',{method:'POST',headers:{Authorization:editor,'Content-Type':'application/json','surreal-ns':namespace,'surreal-db':database,'surreal-auth-ns':namespace,'surreal-auth-db':database},body:JSON.stringify({id:1,method:'query',params:["UPDATE unrelated:fixture SET editor_bypasses_permissions = true;",{}]})});
  assert.ok(response.ok);const editorResult=await response.json();assert.equal(editorResult.result[0].status,'OK');assertions++;
  assert.equal((await query(root,'RETURN unrelated:fixture.editor_bypasses_permissions;'))[0],true);assertions++;
  if(goTest){const go=spawn(goTest,['-test.run','^TestRecordAccessAgainstIsolatedSurreal324$','-test.v'],{env:{...process.env,TOOLKIT_VALIDATION_SCOPE_TEST_URL:endpoint},stdio:['ignore','pipe','pipe']});let output='';go.stdout.on('data',b=>output+=b);go.stderr.on('data',b=>output+=b);const [code]=await once(go,'exit');assert.equal(code,0,output);assertions++;}
  console.log(JSON.stringify({version:version.trim(),assertions,record_writer_scope:'enforced',system_editor_bypasses_permissions:true,actual_worker_sql:true,production_writes:0}));
} finally {
  if(processFixture.exitCode===null){processFixture.kill('SIGTERM');await once(processFixture,'exit');}
}
