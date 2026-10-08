"""Inspect and recover the existing Case Bible catalog runtime on the authorized trial host.

Inputs: an explicit bounded operation; protected transport configuration stays outside source.
Outputs: nonsecret metadata and durable server receipts. Side effects: only the selected operation.
Use for the October 8 OVH recovery, never for corpus copies, extraction or a replacement catalog.
"""
from __future__ import annotations

import argparse
import base64
import importlib.util
import json
import os
import subprocess
import uuid
from pathlib import Path

DATA = "100.108.135.88"
LAKE = "consignatio/_system/lake/refresh-01a1164b-f8f0-7688-9429-3d4895f03b2f/"


def transport():
    """Load the owner's existing protected SSH/API helper without copying credentials.

    Inputs: none. Output: helper module. Side effects: module load only; prefer over shell secrets.
    """
    spec = importlib.util.spec_from_file_location("trial_ops", "D:/WSL/trial-ovh/ops.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def inspect():
    """Read bounded trial-host runtime metadata and the supplied B2 lake generation listing.

    Inputs: fixed host and generation. Output: names, sizes and manifest text; no credentials.
    Side effects: remote reads only. Choose before selecting a compatible catalog recovery route.
    """
    script = r'''set -e
sudo docker ps --format '{{.Names}} {{.Image}} {{.Status}}'
sudo docker images --format '{{.Repository}}:{{.Tag}}' | head -30
sudo ss -lnt | awk '$4 ~ /:(5433|5475)$/ {print}'
sudo rclone --config /data/probata/secrets/rclone.conf lsf 'b2:salem-data/LAKEPATH' --max-depth 2 --files-only --format ps
'''.replace("LAKEPATH", LAKE)
    result = transport().ssh(DATA, script, timeout=55)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1500:])
        raise SystemExit(result.returncode)


def metadata():
    """Stage only schema and small registry/placement metadata on the trial server for inspection.

    Inputs: fixed B2 generation. Output: selected schema definitions and manifest headers.
    Side effects: bounded metadata files under the recovery receipt directory; no corpus import.
    """
    script = r'''set -e
sudo mkdir -p /data/probata/volumes/catalog-recovery-20261008
for f in schema.json manifest.csv catalog_registry.parquet atomic_units.parquet atomic_unit_members.parquet casevault_placement.parquet export_units_v1.parquet; do
  sudo rclone --config /data/probata/secrets/rclone.conf copyto 'b2:salem-data/LAKEPATH'"$f" "/data/probata/volumes/catalog-recovery-20261008/$f" --immutable --retries 1
done
sudo python3 - <<'PY'
import json,pathlib
p=pathlib.Path('/data/probata/volumes/catalog-recovery-20261008')
j=json.loads((p/'schema.json').read_text())
print('SCHEMA_TYPE',type(j).__name__)
print('SCHEMA_KEYS',list(j)[:20] if isinstance(j,dict) else len(j))
if isinstance(j,dict):
    for t in j.get('tables',[]):
        if t['table'] in ('source_occurrences','atomic_units','atomic_unit_members','casevault_placement','catalog_registry'):
            print('TABLE',json.dumps(t))
else:
    print('SCHEMA_SAMPLE',json.dumps(j)[:2000])
print('MANIFEST_HEADER', '\n'.join((p/'manifest.csv').read_text().splitlines()[:4]))
PY
sudo docker exec trial-postgres-c6jfifewfnzkemjethckeuox psql -U ai -d postgres -X -Atc "select version(); select name,default_version from pg_available_extensions where name in ('pg_duckdb','duckdb','vector','pgcrypto');"
'''.replace("LAKEPATH", LAKE)
    result = transport().ssh(DATA, script, timeout=55)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1500:])
        raise SystemExit(result.returncode)


def prepare():
    """Prepare declared host binds and private catalog credentials without starting containers.

    Inputs: existing cached PostgreSQL image and staged lake metadata. Output: image pin and paths.
    Side effects: exclusive secret/config files and recovery directories; no database or source mutation.
    Use before Coolify service creation so all binds exist and credentials never enter the transcript.
    """
    script = r"""set -e
sudo python3 - <<'PY'
import csv,hashlib,json,pathlib,secrets,subprocess,urllib.parse
p=pathlib.Path('/data/probata/volumes/catalog-recovery-20261008')
manifest={r['table_name']:r for r in csv.DictReader((p/'manifest.csv').open())}
for name in ('catalog_registry','atomic_units','atomic_unit_members','casevault_placement','export_units_v1'):
    f=p/(name+'.parquet')
    assert hashlib.sha256(f.read_bytes()).hexdigest()==manifest[name]['sha256'],name+' checksum mismatch'
images=json.loads(subprocess.check_output(['docker','image','inspect','ybqk7yzbpzkqiz07bqbb4v1j:latest']))
image=images[0]['Id']
print('CACHED_IMAGE',image)
secret=pathlib.Path('/data/probata/secrets/casebible-catalog'); secret.mkdir(mode=0o700,parents=True,exist_ok=True)
secret.chmod(0o700)
dbsecret=secret/'database.env'
if not dbsecret.exists():
    dbsecret.write_text('POSTGRES_PASSWORD='+secrets.token_urlsafe(40)+'\n')
dbsecret.chmod(0o600)
writer=secret/'writer.env'
if not writer.exists():
    password=secrets.token_urlsafe(40)
    dsn='postgresql://casebible_toolkit_recovery_service:'+urllib.parse.quote(password,safe='')+'@100.108.135.88:5433/casebible?sslmode=disable'
    writer.write_text('CATALOG_DATABASE_URL='+dsn+'\nCASEBIBLE_TOOLKIT_RECOVERY_WRITER_DSN='+dsn+'\n')
writer.chmod(0o600)
config=pathlib.Path('/data/probata/config/casebible-catalog');config.mkdir(parents=True,exist_ok=True)
data=pathlib.Path('/data/probata/volumes/casebible-pg18');data.mkdir(parents=True,exist_ok=True)
uid=int(subprocess.check_output(['docker','exec','trial-postgres-c6jfifewfnzkemjethckeuox','id','-u','postgres']))
gid=int(subprocess.check_output(['docker','exec','trial-postgres-c6jfifewfnzkemjethckeuox','id','-g','postgres']))
assert data.resolve()==pathlib.Path('/data/probata/volumes/casebible-pg18')
if not any(data.iterdir()):
    import os
    os.chown(data,uid,gid)
data.chmod(0o700)
compose='''services:
  casebible-pg:
    image: IMAGE
    pull_policy: never
    restart: unless-stopped
    command: ["postgres", "-c", "shared_preload_libraries=pg_duckdb"]
    ports:
      - "127.0.0.1:5475:5432"
    volumes:
      - /data/probata/volumes/casebible-pg18:/var/lib/postgresql
      - /data/probata/volumes/catalog-recovery-20261008:/catalog-recovery:ro
    env_file:
      - /data/probata/secrets/casebible-catalog/database.env
    environment:
      POSTGRES_USER: postgres
      PGUSER: postgres
      POSTGRES_DB: casebible
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d casebible"]
      interval: 10s
      timeout: 5s
      retries: 6
      start_period: 30s
    networks:
      coolify:
        aliases:
          - fgz1n7useplhk0t91uk7k1aw
networks:
  coolify:
    external: true
'''.replace('IMAGE',image)
compose_path=config/'compose.yaml'
if compose_path.exists():
    assert compose_path.read_text()==compose,'existing compose differs'
else:
    compose_path.write_text(compose)
subprocess.run(['docker','compose','-f',str(compose_path),'config','--quiet'],check=True)
print(json.dumps({'image':image,'compose_path':str(compose_path),'secret_path':str(writer),'compose':compose,'metadata_hashes_verified':5}))
PY
"""
    result = transport().ssh(DATA, script, timeout=55)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1500:])
        raise SystemExit(result.returncode)


def restore():
    """Restore existing table definitions, small historical metadata and the original narrow SQL API.

    Inputs: verified staged schema/Parquet and SQL from pinned commit 7022601759.
    Outputs: server receipt with restored versus omitted history counts and schema fingerprints.
    Side effects: additive schema/roles and five metadata tables in the new trial catalog only.
    Choose for minimum runtime recovery; source-occurrence history and corpus bytes remain untouched.
    """
    original = subprocess.check_output([
        "git", "show", "7022601759:modules/Consignatio/casebible/tools/ai_workproduct_source_occurrence_api.sql"
    ], cwd=Path(__file__).resolve().parents[4])
    encoded = base64.b64encode(original).decode("ascii")
    script = r'''set -e
sudo python3 - <<'PY'
import base64,csv,datetime,hashlib,json,pathlib,subprocess,urllib.parse
p=pathlib.Path('/data/probata/volumes/catalog-recovery-20261008')
names=subprocess.check_output(['docker','ps','--format','{{.Names}}'],text=True).splitlines()
containers=[n for n in names if n.startswith('casebible-pg-h6wtbmvvqibfreutqtjlgxum')]
assert len(containers)==1,containers
container=containers[0]
def sql(query):
    result=subprocess.run(['docker','exec','-i',container,'psql','-X','-A','-t','-v','ON_ERROR_STOP=1','-U','postgres','-d','casebible'],input=query,text=True,capture_output=True)
    if result.returncode: raise RuntimeError(result.stderr[:1800])
    return result.stdout.strip()
sql('CREATE EXTENSION IF NOT EXISTS pg_duckdb;')
print('PARQUET_READER_COUNT',sql("SELECT count(*) FROM read_parquet('/catalog-recovery/catalog_registry.parquet');"))
schema=json.loads((p/'schema.json').read_text())
tables={t['table']:t for t in schema['tables']}
selected=['catalog_registry','atomic_units','atomic_unit_members','casevault_placement','export_units_v1','source_occurrences']
allowed={'text','bigint','integer','smallint','boolean','jsonb','json','numeric','double precision','real','timestamp with time zone','timestamp without time zone','date','bytea','uuid'}
ddl=['CREATE SCHEMA IF NOT EXISTS raw_duck;']
for name in selected:
    t=tables[name]; cols=[]
    for c in t['columns']:
        assert c['pg_data_type'] in allowed,(name,c['pg_data_type'])
        assert c['name'].replace('_','').isalnum()
        spec='"'+c['name']+'" '+c['pg_data_type']
        if not c['nullable']: spec+=' NOT NULL'
        if c['pg_default'] is not None:
            assert c['pg_default'] in ("now()","''::text","'b2_server_side_copy'::text"),(name,c['pg_default'])
            spec+=' DEFAULT '+c['pg_default']
        cols.append(spec)
    for key in t['keys']:
        assert key['type'] in ('PRIMARY KEY','UNIQUE')
        cols.append('CONSTRAINT "'+key['name']+'" '+key['type']+' ('+','.join('"'+c+'"' for c in key['columns'])+')')
    ddl.append('CREATE TABLE IF NOT EXISTS raw_duck."'+name+'" ('+', '.join(cols)+');')
ddltext='BEGIN;\n'+'\n'.join(ddl)+'\nCOMMIT;\n'
(p/'restored-schema.sql').write_text(ddltext)
sql(ddltext)
counts={}
for name in selected[:-1]:
    expected=tables[name]['rows']
    before=int(sql('SELECT count(*) FROM raw_duck."'+name+'";'))
    if before==0:
        csv_bytes=subprocess.check_output(['docker','exec',container,'psql','-X','-q','--csv','-v','ON_ERROR_STOP=1','-U','postgres','-d','casebible','-c','SELECT * FROM read_parquet(\'/catalog-recovery/'+name+'.parquet\');'])
        result=subprocess.run(['docker','exec','-i',container,'psql','-X','-q','-v','ON_ERROR_STOP=1','-U','postgres','-d','casebible','-c','COPY raw_duck."'+name+'" FROM STDIN WITH (FORMAT CSV, HEADER TRUE);'],input=csv_bytes,capture_output=True)
        if result.returncode:raise RuntimeError(result.stderr.decode()[:1800])
    count=int(sql('SELECT count(*) FROM raw_duck."'+name+'";'))
    assert count==expected,(name,count,expected)
    counts[name]=count
source_count=int(sql('SELECT count(*) FROM raw_duck.source_occurrences;'))
assert source_count==0,'new catalog already contains occurrence writes; inspect before replay'
writer=pathlib.Path('/data/probata/secrets/casebible-catalog/writer.env')
dsn=dict(line.split('=',1) for line in writer.read_text().splitlines())['CATALOG_DATABASE_URL']
password=urllib.parse.unquote(urllib.parse.urlsplit(dsn).password)
sql("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='casebible_toolkit_recovery_writer') THEN CREATE ROLE casebible_toolkit_recovery_writer NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='casebible_toolkit_recovery_service') THEN CREATE ROLE casebible_toolkit_recovery_service LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '"+password.replace("'","''")+"'; END IF; END $$; GRANT casebible_toolkit_recovery_writer TO casebible_toolkit_recovery_service;")
original=base64.b64decode('ORIGINALSQL')
(p/'ai_workproduct_source_occurrence_api-pinned.sql').write_bytes(original)
sql(original.decode())
receipt={'observed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'execution_host':'100.108.135.88','container':container,'database':'casebible','lake_prefix':'LAKEPATH','metadata_counts':counts,'source_occurrences_restored':source_count,'source_occurrences_history_omitted':tables['source_occurrences']['rows'],'full_catalog_restore':False,'bulk_corpus_import':False,'schema_sha256':hashlib.sha256(ddltext.encode()).hexdigest(),'original_api_commit':'7022601759','original_api_sha256':hashlib.sha256(original).hexdigest(),'original_api_scope':'local/F-Downloads Markdown only; not generic B2 AI package admission','writer_secret_file':str(writer),'writer_role':'casebible_toolkit_recovery_service'}
out=p/'runtime-schema-receipt.json'
if out.exists():
    assert json.loads(out.read_text())['schema_sha256']==receipt['schema_sha256']
else:out.write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(receipt))
PY
'''.replace("ORIGINALSQL", encoded).replace("LAKEPATH", LAKE)
    result = transport().ssh(DATA, script, timeout=55)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-2500:])
        raise SystemExit(result.returncode)


def status():
    """Read the new service's state and bounded PostgreSQL startup diagnostics.

    Inputs: fixed service UUID. Output: status and last startup lines. Side effects: read only.
    Use to diagnose lifecycle failures without manipulating Coolify-owned containers directly.
    """
    script = r'''set -e
C=$(sudo docker ps -a --filter name=casebible-pg-h6wtbmvvqibfreutqtjlgxum --format '{{.Names}}')
sudo docker inspect --format '{{json .State}}' "$C"
sudo docker logs --tail 35 "$C" 2>&1
'''
    result = transport().ssh(DATA, script, timeout=30)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1500:])
        raise SystemExit(result.returncode)


def install_api():
    """Install the native package SQL pair and expose the recovered existing catalog over tailnet.

    Inputs: reviewed adjacent SQL and existing writer secret. Output: API/ACL and route receipts.
    Side effects: SQL functions/grants, plain DSN projection and the server's catalog TCP forward.
    Choose after schema recovery; no synthetic package rows or broad writer grants are created.
    """
    encoded = base64.b64encode(Path(__file__).with_name("ai_context_package_catalog_api.sql").read_bytes()).decode("ascii")
    script = r'''set -e
sudo python3 - <<'PY'
import base64,hashlib,json,pathlib,subprocess
p=pathlib.Path('/data/probata/volumes/catalog-recovery-20261008')
sql_bytes=base64.b64decode('NATIVESQL')
(p/'ai_context_package_catalog_api.sql').write_bytes(sql_bytes)
container='casebible-pg-h6wtbmvvqibfreutqtjlgxum'
def sql(query):
    r=subprocess.run(['docker','exec','-i',container,'psql','-X','-q','-A','-t','-v','ON_ERROR_STOP=1','-U','postgres','-d','casebible'],input=query,text=True,capture_output=True)
    if r.returncode:raise RuntimeError(r.stderr[:1800])
    return r.stdout.strip()
sql(sql_bytes.decode())
secret=pathlib.Path('/data/probata/secrets/casebible-catalog')
dsn=dict(line.split('=',1) for line in (secret/'writer.env').read_text().splitlines())['CATALOG_DATABASE_URL']+'\n'
plain=secret/'writer.dsn'
if plain.exists():assert plain.read_text()==dsn,'existing plain DSN differs'
else:plain.write_text(dsn)
plain.chmod(0o600)
checks=json.loads(sql("SELECT json_build_object('role',rolname,'super',rolsuper,'bypass',rolbypassrls,'createdb',rolcreatedb,'createrole',rolcreaterole,'replication',rolreplication,'raw_usage',has_schema_privilege(rolname,'raw_duck','USAGE'),'api_usage',has_schema_privilege(rolname,'source_occurrence_api','USAGE'),'register_execute',has_function_privilege(rolname,'source_occurrence_api.register_ai_context_package(text,text)','EXECUTE'),'read_execute',has_function_privilege(rolname,'source_occurrence_api.read_ai_context_package(text,text)','EXECUTE'),'private_execute',has_function_privilege(rolname,'source_occurrence_api.validate_ai_context_package(text,text)','EXECUTE')) FROM pg_roles WHERE rolname='casebible_toolkit_recovery_service';"))
assert checks['api_usage'] and checks['register_execute'] and checks['read_execute']
assert not any(checks[k] for k in ('super','bypass','createdb','createrole','replication','raw_usage','private_execute'))
# An invalid pinned batch must roll back without adding any native package or placement.
before=sql("SELECT count(*) FROM raw_duck.source_occurrences;")
invalid=subprocess.run(['docker','exec','-i',container,'psql','-X','-q','-v','ON_ERROR_STOP=1','-U','postgres','-d','casebible'],input="BEGIN; SET ROLE casebible_toolkit_recovery_service; SELECT source_occurrence_api.register_ai_context_package('{}','"+'0'*64+"'); COMMIT;",text=True,capture_output=True)
assert invalid.returncode!=0 and before==sql("SELECT count(*) FROM raw_duck.source_occurrences;")
receipt={'native_sql_sha256':hashlib.sha256(sql_bytes).hexdigest(),'writer_secret_file':str(plain),'privileges':checks,'invalid_payload_rejected_without_rows':True,'actual_package_registration_verified':False}
(p/'native-api-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(receipt))
current=json.loads(subprocess.check_output(['tailscale','serve','status','--json']))
if '5433' in current.get('TCP',{}):
    assert current['TCP']['5433'].get('TCPForward')=='127.0.0.1:5475',current['TCP']['5433']
else:
    subprocess.run(['tailscale','serve','--bg','--tcp=5433','tcp://127.0.0.1:5475'],check=True)
route=json.loads(subprocess.check_output(['tailscale','serve','status','--json']))['TCP']['5433']
print('CATALOG_ROUTE',json.dumps(route))
PY
'''.replace("NATIVESQL", encoded)
    result = transport().ssh(DATA, script, timeout=55)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-2500:])
        raise SystemExit(result.returncode)


def mirror_and_probe():
    """Project the protected plain DSN to APP and verify real authenticated catalog connectivity.

    Inputs: existing DATA secret and authorized APP host. Output: nonsecret route/authentication results.
    Side effects: identical 0600 APP credential projection and a small verification receipt; no catalog writes.
    Choose before the Go client consumes its existing plain-DSN configuration contract.
    """
    ops = transport()
    secret = ops.ssh(DATA, "sudo cat /data/probata/secrets/casebible-catalog/writer.dsn\n", timeout=20)
    if secret.returncode:
        raise RuntimeError("Cannot read protected DATA writer projection")
    encoded = base64.b64encode(secret.stdout).decode("ascii")
    script = r'''set -e
sudo python3 - <<'PY'
import base64,json,os,pathlib,shutil,socket,subprocess,urllib.parse
secret=pathlib.Path('/data/probata/secrets/casebible-catalog');secret.mkdir(mode=0o700,parents=True,exist_ok=True);secret.chmod(0o700)
plain=secret/'writer.dsn'; data=base64.b64decode('SECRETBYTES')
if plain.exists():assert plain.read_bytes()==data,'existing APP credential differs'
else:plain.write_bytes(data)
plain.chmod(0o600)
uri=urllib.parse.urlsplit(data.decode().strip())
with socket.create_connection((uri.hostname,uri.port),timeout=10):pass
result={'host':'100.84.202.29','secret_file':str(plain),'mode':'0600','catalog_tcp_reachable':True,'authenticated_sql':False}
if shutil.which('psql'):
    env=os.environ.copy();env.update(PGHOST=uri.hostname,PGPORT=str(uri.port),PGUSER=uri.username,PGPASSWORD=urllib.parse.unquote(uri.password),PGDATABASE=uri.path.lstrip('/'),PGCONNECT_TIMEOUT='10')
    r=subprocess.run(['psql','-X','-A','-t','-v','ON_ERROR_STOP=1','-c',"SELECT current_user,current_database(),has_function_privilege(current_user,'source_occurrence_api.register_ai_context_package(text,text)','EXECUTE');"],env=env,capture_output=True,text=True,timeout=20)
    assert r.returncode==0,'APP authenticated SQL failed'
    result.update(authenticated_sql=True,sql_readback=r.stdout.strip())
print(json.dumps(result))
PY
'''.replace("SECRETBYTES", encoded)
    result = ops.ssh("100.84.202.29", script, timeout=45)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1000:])
        raise SystemExit(result.returncode)
    probe_connection()


def verify_schema():
    """Independently compare the live recovered columns and constraints against the published schema.

    Inputs: staged generation and running catalog. Output: exact schema/readback receipt and health.
    Side effects: one small server receipt; reads only database metadata and preserved table counts.
    Choose before reporting restoration as compatible with the existing catalog writers.
    """
    script = r'''set -e
sudo python3 - <<'PY'
import json,pathlib,subprocess
p=pathlib.Path('/data/probata/volumes/catalog-recovery-20261008')
schema=json.loads((p/'schema.json').read_text())
names=['catalog_registry','atomic_units','atomic_unit_members','casevault_placement','export_units_v1','source_occurrences']
container='casebible-pg-h6wtbmvvqibfreutqtjlgxum'
def sql(q):
    return subprocess.check_output(['docker','exec',container,'psql','-X','-A','-t','-v','ON_ERROR_STOP=1','-U','postgres','-d','casebible','-c',q],text=True).strip()
checks=[]
for t in schema['tables']:
    name=t['table']
    if name not in names:continue
    actual=json.loads(sql("SELECT json_agg(json_build_object('name',column_name,'type',data_type,'nullable',is_nullable='YES','default',column_default) ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='raw_duck' AND table_name='"+name+"';"))
    expected=[{'name':c['name'],'type':c['pg_data_type'],'nullable':c['nullable'],'default':c['pg_default']} for c in t['columns']]
    assert actual==expected,name+' live column mismatch'
    keys=json.loads(sql("SELECT coalesce(json_agg(x ORDER BY x.name),'[]'::json) FROM (SELECT tc.constraint_name AS name,tc.constraint_type AS type,json_agg(kcu.column_name ORDER BY kcu.ordinal_position) AS columns FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON kcu.constraint_schema=tc.constraint_schema AND kcu.constraint_name=tc.constraint_name AND kcu.table_name=tc.table_name WHERE tc.table_schema='raw_duck' AND tc.table_name='"+name+"' AND tc.constraint_type IN ('PRIMARY KEY','UNIQUE') GROUP BY tc.constraint_name,tc.constraint_type) x;"))
    assert sorted(keys,key=lambda x:x['name'])==sorted(t['keys'],key=lambda x:x['name']),name+' live constraint mismatch'
    checks.append({'table':name,'columns_match':True,'keys_match':True,'live_rows':int(sql('SELECT count(*) FROM raw_duck."'+name+'";')),'published_rows':t['rows']})
state=json.loads(subprocess.check_output(['docker','inspect',container]))[0]['State']
receipt={'live_schema_matches_published':True,'checks':checks,'running':state['Running'],'health':state.get('Health',{}).get('Status'),'full_catalog_restore':False}
(p/'schema-independent-readback.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(receipt))
PY
'''
    result = transport().ssh(DATA, script, timeout=45)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1500:])
        raise SystemExit(result.returncode)


def probe_connection():
    """Authenticate the protected writer through the recovered PostgreSQL TCP listener.

    Inputs: existing plain DSN. Output: current role/database and function privilege readback.
    Side effects: one small receipt only; choose to distinguish password authentication from reachability.
    """
    probe = r'''set -e
sudo python3 - <<'PY'
import json,os,pathlib,shutil,subprocess,urllib.parse
uri=urllib.parse.urlsplit(pathlib.Path('/data/probata/secrets/casebible-catalog/writer.dsn').read_text().strip())
env=os.environ.copy();env['PGPASSWORD']=urllib.parse.unquote(uri.password)
client=['psql'] if shutil.which('psql') else ['docker','exec','-e','PGPASSWORD','casebible-pg-h6wtbmvvqibfreutqtjlgxum','psql']
# Docker-to-own-tailnet Serve is refused; the host loopback listener is the same target.
host,port=(uri.hostname,uri.port) if shutil.which('psql') else ('127.0.0.1',5432)
cmd=client+['-X','-A','-t','-v','ON_ERROR_STOP=1','-h',host,'-p',str(port),'-U',uri.username,'-d',uri.path.lstrip('/'),'-c',"SELECT current_user,current_database(),has_function_privilege(current_user,'source_occurrence_api.register_ai_context_package(text,text)','EXECUTE');"]
r=subprocess.run(cmd,env=env,capture_output=True,text=True,timeout=20)
if r.returncode:raise RuntimeError('DATA authenticated tailnet SQL failed: '+r.stderr[:1000])
result={'host':'100.108.135.88','authenticated_sql':True,'authenticated_via_tailnet':bool(shutil.which('psql')),'sql_readback':r.stdout.strip()}
pathlib.Path('/data/probata/volumes/catalog-recovery-20261008/connection-receipt.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result))
PY
'''
    result = transport().ssh(DATA, probe, timeout=35)
    print(result.stdout.decode("utf-8", errors="replace"))
    if result.returncode:
        print(result.stderr.decode("utf-8", errors="replace")[-1000:])
        raise SystemExit(result.returncode)


def source_commit():
    """Create a reviewable two-file Git commit with an isolated index and no branch movement.

    Inputs: this helper and adjacent SQL. Output: commit SHA for parent integration.
    Side effects: Git objects and a retained alternate index only; shared staging and dirty files stay intact.
    Choose in this shared checkout instead of staging concurrent sessions' work.
    """
    root = Path(__file__).resolve().parents[4]
    actual = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], cwd=root, text=True).strip()
    if Path(actual).resolve() != root:
        raise RuntimeError("Unexpected Git boundary")
    git_dir = Path(subprocess.check_output(["git", "rev-parse", "--absolute-git-dir"], cwd=root, text=True).strip())
    env = os.environ.copy()
    env["GIT_INDEX_FILE"] = str(git_dir / ("catalog-recovery-20261008-" + uuid.uuid4().hex + ".index"))
    paths = ["modules/Consignatio/casebible/tools/ai_context_package_catalog_api.sql", "modules/Consignatio/casebible/tools/catalog_trial_recovery_20261008.py"]
    subprocess.run(["git", "read-tree", "HEAD"], cwd=root, env=env, check=True)
    subprocess.run(["git", "add", "--", *paths], cwd=root, env=env, check=True)
    subprocess.run(["git", "diff", "--cached", "--check"], cwd=root, env=env, check=True)
    tree = subprocess.check_output(["git", "write-tree"], cwd=root, env=env, text=True).strip()
    commit = subprocess.check_output(["git", "commit-tree", tree, "-p", "HEAD", "-m", "Recover trial Case Bible catalog runtime and native package SQL admission"], cwd=root, env=env, text=True).strip()
    print(json.dumps({"commit": commit, "paths": paths, "branch_moved": False, "shared_index_changed": False}))


def main():
    """Dispatch a named recovery operation without starting unrelated work.

    Inputs: CLI operation. Output: operation result. Side effects: selected operation only.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=["inspect", "metadata", "prepare", "restore", "status", "install-api", "mirror-and-probe", "verify-schema", "source-commit"])
    args = parser.parse_args()
    if args.operation == "inspect":
        inspect()
    elif args.operation == "metadata":
        metadata()
    elif args.operation == "prepare":
        prepare()
    elif args.operation == "restore":
        restore()
    elif args.operation == "status":
        status()
    elif args.operation == "install-api":
        install_api()
    elif args.operation == "mirror-and-probe":
        mirror_and_probe()
    elif args.operation == "verify-schema":
        verify_schema()
    elif args.operation == "source-commit":
        source_commit()


if __name__ == "__main__":
    main()
