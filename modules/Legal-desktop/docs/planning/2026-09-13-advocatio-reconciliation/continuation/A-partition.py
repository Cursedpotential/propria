"""Pin and partition the existing inventory; never overwrite completed parts."""
import hashlib
import json
from pathlib import Path

HERE = Path(__file__).resolve().parent
INPUT = HERE.parent / 'inputs/stack/toolkit-capabilities.json'
COLUMNS = ['source_id', 'input_index', 'capability', 'kind', 'origin_path',
           'origin_version', 'source_path', 'source_sha256',
           'observed_implementation', 'observed_wiring', 'destination',
           'integration_approach', 'proposed_route', 'human_access', 'agent_access',
           'dependencies', 'missing_information', 'required_verification', 'next_action']
KINDS = {
    'callable': {'command', 'hook', 'maintenance_script', 'mcp_server', 'mcp_tool',
                 'mcp_widget', 'runtime_adapter', 'sidecar_route', 'utility_script'},
    'guidance': {'agent', 'skill', 'procedure', 'mcp_prompt'},
    'references': {'event_context_pack', 'external_service_assessment',
                   'external_source_record', 'mcp_resource', 'resource_pack'},
}

def main():
    raw = INPUT.read_bytes()
    data = json.loads(raw)
    buckets = {k: [] for k in KINDS}
    seen = set()
    for index, entry in enumerate(data['entries']):
        lanes = [k for k, kinds in KINDS.items() if entry['kind'] in kinds]
        assert len(lanes) == 1, entry['kind']
        identity = {k: entry.get(k, '') for k in
                    ('origin_path', 'origin_version', 'kind', 'name', 'source_path')}
        key = 'cap-' + hashlib.sha256(json.dumps(identity, sort_keys=True,
                                    ensure_ascii=False).encode()).hexdigest()[:20]
        assert key not in seen, identity
        seen.add(key)
        buckets[lanes[0]].append({'source_id': key, 'input_index': index, **entry})
    parts = HERE / 'parts'
    parts.mkdir(exist_ok=True)
    manifest = {'input': str(INPUT), 'input_sha256': hashlib.sha256(raw).hexdigest(),
                'total': len(seen), 'columns': COLUMNS,
                'sections': {k: {'count': len(v), 'source_ids': [e['source_id'] for e in v]}
                             for k, v in buckets.items()}}
    for name, payload in [('assignment', manifest), *buckets.items()]:
        path = parts / (name + '.json')
        text = json.dumps(payload, indent=2, ensure_ascii=False) + '\n'
        if path.exists():
            assert path.read_text(encoding='utf-8') == text, f'Existing assignment differs: {path}'
        else:
            path.write_text(text, encoding='utf-8')
    print(json.dumps({'total': len(seen), 'sections': {k: len(v) for k,v in buckets.items()}}))

if __name__ == '__main__':
    main()
