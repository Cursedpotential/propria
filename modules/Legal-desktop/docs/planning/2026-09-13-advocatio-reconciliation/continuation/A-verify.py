"""Merge disjoint completed sections and prove identity and coverage."""
import csv
import hashlib
import json
from collections import Counter
from pathlib import Path

HERE = Path(__file__).resolve().parent
PARTS = HERE / 'parts'
DESTINATIONS = {'resource','method','mcp_tool','shared_code','operational_support','unresolved'}
APPROACHES = {'reuse_connection','expose_implementation','import_reconcile','adapt','port','investigate'}

def main():
    assignment = json.loads((PARTS/'assignment.json').read_text(encoding='utf-8'))
    raw = (HERE.parent/'inputs/stack/toolkit-capabilities.json').read_bytes()
    assert hashlib.sha256(raw).hexdigest() == assignment['input_sha256'], 'Inventory changed'
    entries = json.loads(raw)['entries']
    columns = assignment['columns']
    merged, seen, proofs = [], set(), {}
    for section, expected in assignment['sections'].items():
        path = PARTS/(section+'.csv')
        with path.open(encoding='utf-8-sig', newline='') as stream:
            reader = csv.DictReader(stream)
            assert reader.fieldnames == columns, (section, reader.fieldnames)
            rows = list(reader)
        assert len(rows) == expected['count'], section
        assert {r['source_id'] for r in rows} == set(expected['source_ids']), section
        for row in rows:
            assert None not in row and all(v is not None for v in row.values()), row['source_id']
            assert row['source_id'] not in seen, row['source_id']
            seen.add(row['source_id'])
            index = int(row['input_index'])
            entry = entries[index]
            identity = {k: entry.get(k, '') for k in
                        ('origin_path','origin_version','kind','name','source_path')}
            expected_id = 'cap-' + hashlib.sha256(json.dumps(identity, sort_keys=True,
                                     ensure_ascii=False).encode()).hexdigest()[:20]
            assert row['source_id'] == expected_id, (section,index)
            fields = {'capability':'name','kind':'kind','origin_path':'origin_path',
                      'origin_version':'origin_version','source_path':'source_path',
                      'source_sha256':'source_sha256','observed_implementation':'implementation_status',
                      'observed_wiring':'wired_status'}
            for target, source in fields.items():
                value = entry.get(source, '')
                assert row[target] == str(value), (section, index, target)
            assert row['destination'] in DESTINATIONS, row['source_id']
            assert row['integration_approach'] in APPROACHES, row['source_id']
            for col in columns[10:]:
                assert row[col].strip(), (section,index,col)
        merged.extend(rows)
        proofs[section] = {'rows':len(rows),'sha256':hashlib.sha256(path.read_bytes()).hexdigest()}
    assert len(merged) == len(entries) == len(seen)
    assert sorted(int(r['input_index']) for r in merged) == list(range(len(entries)))
    merged.sort(key=lambda r:int(r['input_index']))
    output = HERE/'A-capability-port-map.csv'
    with output.open('w',encoding='utf-8',newline='') as stream:
        writer = csv.DictWriter(stream,fieldnames=columns)
        writer.writeheader()
        writer.writerows(merged)
    with output.open(encoding='utf-8',newline='') as stream:
        assert list(csv.DictReader(stream)) == merged
    result = {'input_sha256':assignment['input_sha256'], 'input_count':len(entries),
              'output_count':len(merged),'unique_ids':len(seen),'omissions':0,'duplicate_ids':0,
              'identity_and_observed_fields_preserved':True,'csv_roundtrip':True,
              'sections':proofs,'destinations':dict(Counter(r['destination'] for r in merged)),
              'approaches':dict(Counter(r['integration_approach'] for r in merged)),
              'output_sha256':hashlib.sha256(output.read_bytes()).hexdigest(),
              'scope':'Static inventory mapping; no execution or legal-source validation claimed'}
    (HERE/'A-verification.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
    print(json.dumps(result))

if __name__ == '__main__':
    main()
