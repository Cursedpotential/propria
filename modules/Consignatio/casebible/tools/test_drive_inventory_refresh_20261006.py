"""Test native metadata production and receipt-governed loading with retained tiny fixtures.

Inputs: synthetic source trees and frozen metadata; outputs unittest assertions and
optional rollback-only disposable PostgreSQL schema proof. Effects: tiny derived writes
on E: or ovh-files; never scans D, loads raw_duck, deletes fixtures or reads source payloads.
Choose before parent reviews the concrete native scan command.
Byline: Codex / GPT-6 / 2026-10-06.
"""
import ast
import asyncio
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import threading
import unittest
from unittest.mock import AsyncMock, Mock, patch
import uuid

import local_drive_listing as walker
import drive_inventory_load_20261006 as loader
import drive_inventory_refresh_20261006 as refresh


class DriveInventoryTests(unittest.TestCase):
    """Exercise scan safety, frozen coverage and catalog selection; synthetic inputs, assertions, retained fixtures; choose bounded regression. Byline: Codex 2026-10-06."""
    def setUp(self):
        """Create a tiny retained fixture outside D; no inputs, fixture state output, derived writes; choose test isolation. Byline: Codex 2026-10-06."""
        base = Path(__file__).resolve().parents[2] / 'docs/receipts/drive-inventory-synthetic' if os.name == 'nt' else Path('/data/consignatio/drive-inventory-synthetic')
        self.work = base / uuid.uuid4().hex
        self.root = self.work / 'source'
        self.root.mkdir(parents=True)
        (self.root / 'one.txt').write_bytes(b'tiny synthetic fixture')
        (self.root / 'empty.txt').write_bytes(b'')
        self.generation = uuid.uuid4().hex
        self.temporal = dict(workflow_id='synthetic-drive-scan', run_id='synthetic-run', activity_id='1', attempt=1)

    def scan(self, root=None):
        """Observe only the tiny fixture; input optional root, result refs output, retained writes; choose native walker checks. Byline: Codex 2026-10-06."""
        return walker.scan(root or self.root, self.work / uuid.uuid4().hex, self.generation, self.temporal)

    def bundle(self, status=None):
        """Build explicitly synthetic portable native metadata; optional status input, bundle/pin output, retained writes; choose VPS parser fixtures. Byline: Codex 2026-10-06."""
        observed = self.scan()
        source = Path(observed['receipt']).parent
        native = json.loads(Path(observed['receipt']).read_text())
        native.update(root='D:/', platform='nt')
        if status:
            native.update(status=status, enumeration_finished=False, interruption='synthetic interruption')
        bundle = self.work / uuid.uuid4().hex
        bundle.mkdir()
        for name in ('listing.jsonl', 'walker-source.py'):
            with (bundle / name).open('xb') as output:
                output.write((source / name).read_bytes())
        pin = walker.save(bundle / 'listing-receipt.json', native)
        return bundle, pin

    def change_bundle(self, bundle, edit):
        """Retain a separately named mutated synthetic receipt bundle; inputs bundle/edit, new bundle/pin output, derived writes; choose negative fixtures. Byline: Codex 2026-10-06."""
        target = self.work / uuid.uuid4().hex
        target.mkdir()
        native = json.loads((bundle / 'listing-receipt.json').read_text())
        edit(native)
        for name in ('listing.jsonl', 'walker-source.py'):
            (target / name).write_bytes((bundle / name).read_bytes())
        pin = walker.save(target / 'listing-receipt.json', native)
        return target, pin

    def test_native_scan_no_payload_reads_and_three_counts(self):
        """Prove the scanner never opens fixture payloads; synthetic tree input, metadata assertions, retained outputs; choose source-read boundary. Byline: Codex 2026-10-06."""
        original = Path.open
        root = self.root
        def guarded(path, *args, **kwargs):
            """Reject source payload opens in the fixture; path inputs, file output elsewhere, no source effects; choose metadata-only assertion. Byline: Codex 2026-10-06."""
            if path.is_relative_to(root):
                raise AssertionError('source payload opened')
            return original(path, *args, **kwargs)
        with patch.object(Path, 'open', guarded):
            result = self.scan()
        receipt = json.loads(Path(result['receipt']).read_text())
        self.assertEqual(receipt['status'], 'complete')
        self.assertEqual(receipt['counts']['file_count'], 2)
        self.assertEqual(receipt['counts']['error_count'], 0)
        self.assertEqual(receipt['counts']['skip_count'], 0)
        self.assertEqual(loader.digest(result['listing']), (receipt['raw_sha256'], receipt['raw_bytes']))

    def test_explicit_root_exclusions_do_not_exclude_nested_names(self):
        """Record root policy skips while keeping same-named nested folders; fixture tree input, assertions, retained outputs; choose exact scope. Byline: Codex 2026-10-06."""
        for name in walker.POLICY['root_exclusions']:
            directory = self.root / name
            directory.mkdir()
            (directory / 'hidden.txt').write_bytes(b'fixture')
        nested = self.root / 'nested/$RECYCLE.BIN'
        nested.mkdir(parents=True)
        (nested / 'keep.txt').write_bytes(b'fixture')
        result = self.scan()
        rows = [json.loads(line) for line in Path(result['listing']).read_text().splitlines()]
        self.assertEqual(sum(r['kind'] == 'skip' for r in rows), 2)
        self.assertIn('nested/$RECYCLE.BIN/keep.txt', [r['rel_path'] for r in rows if r['kind'] == 'file'])

    def test_reparse_entry_never_followed(self):
        """Record a synthetic reparse entry without opening its target; mocked entry input, assertions, retained output; choose junction safety. Byline: Codex 2026-10-06."""
        entry = Mock(name='reparse')
        entry.name, entry.path = 'junction', str(self.root / 'junction')
        entry.stat.return_value = Mock(st_mode=0o040755, st_file_attributes=0x410, st_reparse_tag=0xA0000003)
        iterator = Mock()
        iterator.__enter__ = Mock(return_value=iter([entry]))
        iterator.__exit__ = Mock(return_value=False)
        with patch.object(walker.os, 'scandir', return_value=iterator) as scandir:
            result = self.scan()
        rows = [json.loads(line) for line in Path(result['listing']).read_text().splitlines()]
        self.assertEqual(rows[0]['reason'], 'reparse_point')
        self.assertEqual(scandir.call_count, 1)
        entry.stat.assert_called_once_with(follow_symlinks=False)

    def test_listdir_failure_retained_and_not_complete(self):
        """Retain inaccessible-root evidence rather than claiming empty; mocked scandir input, assertions, retained receipt; choose LISTDIR regression. Byline: Codex 2026-10-06."""
        with patch.object(walker.os, 'scandir', side_effect=PermissionError('synthetic access denied')):
            result = self.scan()
        receipt = json.loads(Path(result['receipt']).read_text())
        self.assertEqual(receipt['status'], 'failed')
        self.assertEqual(receipt['counts']['error_count'], 1)
        self.assertIn('LISTDIR', Path(result['listing']).read_text())

    def test_stat_failure_retained(self):
        """Retain a STAT error in exact row counts; mocked metadata input, assertions, retained receipts; choose per-entry failure coverage. Byline: Codex 2026-10-06."""
        entry = Mock()
        entry.name, entry.path = 'missing.txt', str(self.root / 'missing.txt')
        entry.stat.side_effect = FileNotFoundError('synthetic race')
        iterator = Mock()
        iterator.__enter__ = Mock(return_value=iter([entry]))
        iterator.__exit__ = Mock(return_value=False)
        with patch.object(walker.os, 'scandir', return_value=iterator):
            result = self.scan()
        receipt = json.loads(Path(result['receipt']).read_text())
        self.assertEqual(receipt['status'], 'partial')
        self.assertEqual(receipt['object_count'], 1)
        self.assertIn('STAT', Path(result['listing']).read_text())

    def test_interruption_keeps_raw_and_failure_receipt(self):
        """Retain cancellation evidence without deleting partial artifacts; cancelled fixture input, assertions, retained writes; choose interruption handling. Byline: Codex 2026-10-06."""
        work = self.work / 'interrupted'
        with self.assertRaisesRegex(RuntimeError, 'retained'):
            walker.scan(self.root, work, self.generation, self.temporal, cancelled=lambda: True)
        receipt = json.loads((work / 'listing-receipt.json').read_text())
        self.assertFalse(receipt['enumeration_finished'])
        self.assertIn(receipt['status'], ('failed', 'partial'))
        self.assertTrue((work / 'listing.jsonl').exists())

    def test_exclusive_artifacts_and_outside_source(self):
        """Reject artifact overwrite and source-contained output; synthetic paths input, assertions, no source writes; choose retention guard. Byline: Codex 2026-10-06."""
        with self.assertRaisesRegex(ValueError, 'outside'):
            walker.scan(self.root, self.root / 'output', self.generation, self.temporal)
        result = self.scan()
        with self.assertRaises(FileExistsError):
            walker.scan(self.root, Path(result['receipt']).parent, self.generation, self.temporal)

    def test_frozen_bundle_validation_and_stable_catalog_payload(self):
        """Validate exact metadata and operation-independent receipt identity; synthetic bundle input, assertions, retained TSV; choose portable admission. Byline: Codex 2026-10-06."""
        bundle, pin = self.bundle()
        first = loader.prepare(bundle, pin, self.work / 'validation1')
        second = loader.prepare(bundle, pin, self.work / 'validation2')
        self.assertEqual(loader.catalog_payload(first), loader.catalog_payload(second))
        self.assertEqual(first['object_count'], 2)
        self.assertEqual(first['status'], 'complete')

    def test_hash_count_policy_and_completion_conflicts(self):
        """Reject altered pins, false totals, exclusions and completion; synthetic variants, assertions, retained fixtures; choose fail-closed validation. Byline: Codex 2026-10-06."""
        bundle, pin = self.bundle()
        with self.assertRaisesRegex(ValueError, 'SHA256'):
            loader.prepare(bundle, '0' * 64, self.work / 'wrongpin')
        for edit in [lambda r: r.update(raw_sha256='0' * 64), lambda r: r['counts'].update(file_count=99),
                     lambda r: r.update(enumeration_finished=False), lambda r: r['policy'].update(other_exclusions=['Backup'])]:
            changed, changed_pin = self.change_bundle(bundle, edit)
            with self.assertRaises(ValueError):
                loader.prepare(changed, changed_pin, self.work / uuid.uuid4().hex)

    def test_partial_bundle_preserved_without_completion_promotion(self):
        """Admit a valid interrupted observation only as partial; synthetic receipt input, assertions, retained TSV; choose non-current history. Byline: Codex 2026-10-06."""
        bundle, pin = self.bundle('partial')
        self.assertEqual(loader.prepare(bundle, pin, self.work / 'partial-validation')['status'], 'partial')

    def test_native_guard_rejects_wrong_drive_artifacts(self):
        """Reject non-E production work without scanning; wrong path input, assertions, no effects; choose desktop boundary. Byline: Codex 2026-10-06."""
        with self.assertRaises(ValueError):
            refresh.require_native('D:/not-allowed')

    def test_legacy_sql_cannot_recreate_view(self):
        """Require the historical entry point to reject all unreceipted writes; source input, assertions, no SQL effects; choose legacy regression. Byline: Codex 2026-10-06."""
        sql = Path(loader.__file__).with_name('drive_objects_load.sql').read_text()
        self.assertIn('RAISE EXCEPTION', sql)
        self.assertNotIn('CREATE OR REPLACE VIEW', sql.upper())
        self.assertNotIn('INSERT INTO', sql.upper())

    def test_copy_escaping_and_schema_allowlist(self):
        """Preserve COPY delimiters and reject arbitrary schemas; scalar inputs, assertions, no effects; choose load encoding. Byline: Codex 2026-10-06."""
        self.assertEqual(loader.pg_text('a\tb\nc\\d'), r'a\tb\nc\\d')
        self.assertEqual(loader.pg_text(None), r'\N')
        for value in ['public', 'raw_duck;DROP SCHEMA raw_duck', 'drive_inventory_test_x']:
            with self.assertRaises(ValueError):
                loader.schema_name(value)

    def test_production_load_gate_precedes_any_database_mutation(self):
        """Keep DDL behind the production-release check; CLI AST input, assertions, no effects; choose lake fingerprint protection. Byline: Codex 2026-10-06."""
        source = Path(loader.__file__).read_text()
        self.assertLess(source.index('asyncio.run(production_release('), source.index('sql = load_sql(receipt)'))
        self.assertIn('publication_exclusive_until_load_finishes', source)

    def test_workflow_scan_ship_load_are_separate(self):
        """Prevent scans from implicitly loading production; mocked Temporal input, assertions, no network; choose launch boundaries. Byline: Codex 2026-10-06."""
        try:
            from temporalio import workflow
        except ImportError:
            self.skipTest('Temporal SDK exercised in cached uv/VPS tests')
        for operation, stages in [('scan', ['scan']), ('ship', ['transfer', 'validate']), ('load', ['load', 'readback'])]:
            execute = AsyncMock(return_value={'receipt': 'synthetic'})
            with patch.object(workflow, 'execute_activity', execute):
                asyncio.run(refresh.DriveInventoryRefresh().run(dict(config='fixture', operation=operation, reference={})))
            self.assertEqual([c.args[0] for c in execute.call_args_list], ['drive_inventory_' + s + '_20261006' for s in stages])

    def test_activity_and_workflow_docs_are_structured(self):
        """Require canonical structured docs for every drive Activity/workflow; module input, assertions, no effects; choose generated reference contract. Byline: Codex 2026-10-06."""
        for unit in (refresh.scan_activity, refresh.transfer_activity, refresh.validate_activity, refresh.load_activity,
                     refresh.readback_activity, refresh.DriveInventoryRefresh, refresh.DriveInventoryRefresh.run):
            for label in ('Inputs:', 'Outputs:', 'Effects:', 'Choose '):
                self.assertTrue(any(line.strip().startswith(label) for line in unit.__doc__.strip().splitlines()[1:]))

    def test_registry_descriptions_are_canonical_and_targets_bounded(self):
        """Derive four owned descriptions from functions rather than parallel registry prose; synthetic receipt input, assertions, retained fixtures; choose generated registry contract. Byline: Codex 2026-10-06."""
        bundle, pin = self.bundle()
        receipt = loader.prepare(bundle, pin, self.work / 'registry-validation')
        sql = loader.registry_sql(receipt)
        self.assertEqual([x[0] for x in loader.registry_units()], ['drive_objects', 'drive_objects_current', 'drive_listing_receipts', 'drive_inventory_current_receipts'])
        self.assertEqual(sql.count('INSERT INTO raw_duck.catalog_registry('), 4)
        for name, object_type, kind, unit in loader.registry_units():
            self.assertIn(loader.literal(loader.summary(unit)), sql)
            self.assertIn(unit.__module__ + '.' + unit.__name__, sql)
        with patch.object(walker.scan, '__doc__', 'Synthetic revised canonical summary.\n\nInputs: fixture.'):
            self.assertIn('Synthetic revised canonical summary.', loader.registry_sql(receipt))
        self.assertIn(receipt['receipt_sha256'], sql)
        self.assertIn(receipt['loader_sha256'], sql)
        self.assertNotIn('catalog_registry_build.py', sql)
        self.assertIn("'exact_registry'", loader.readback_sql(receipt))

    def pg_fixture_sql(self, schema):
        """Create only rollback-contained catalog contract fixtures; input schema, output SQL, no execution; choose producer integration tests. Byline: Codex 2026-10-06."""
        return f"""\\set ON_ERROR_STOP 1
BEGIN; CREATE SCHEMA {schema};
CREATE TABLE {schema}.drive_objects(drive text NOT NULL,rel_path text NOT NULL,size bigint,mtime timestamptz,error text,listed_at timestamptz NOT NULL,PRIMARY KEY(drive,listed_at,rel_path));
CREATE TABLE {schema}.catalog_registry(object_schema text NOT NULL,object_name text NOT NULL,object_type text NOT NULL,kind text,covers text,as_of text,status text NOT NULL,superseded_by text,made_by text,evidence text,est_rows bigint,size text,moved_from text,registered_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(object_schema,object_name));
INSERT INTO {schema}.catalog_registry(object_schema,object_name,object_type,kind,covers,as_of,status,evidence)
 SELECT 'unowned','classification_'||i,'table','analysis','Retained fixture','2026-10-02','current','unchanged-'||i FROM generate_series(1,33) i;
INSERT INTO {schema}.catalog_registry(object_schema,object_name,object_type,kind,covers,status,evidence,est_rows,moved_from)
 VALUES('{schema}','drive_objects','table','listing','Old description','current','retained prior evidence',123,'retained origin');
INSERT INTO {schema}.drive_objects(drive,rel_path,size,listed_at) VALUES('D:','legacy.txt',7,'2026-10-03T01:04:48Z');
"""

    def run_pg_fixture(self, sql, label):
        """Execute and retain one disposable SQL attempt; input SQL/label, process result output, rollback-only database effects; choose live contract evidence. Byline: Codex 2026-10-06."""
        result = subprocess.run(['docker', 'exec', '-i', loader.PG_CONTAINER, 'psql', '-U', 'postgres', '-d', 'casebible', '-X', '-At', '-v', 'ON_ERROR_STOP=1'], input=sql, text=True, capture_output=True, timeout=90)
        (self.work / (label + '.sql')).write_text(sql)
        (self.work / (label + '.log')).write_text(result.stdout + result.stderr)
        return result

    @unittest.skipUnless(os.environ.get('DRIVE_INVENTORY_PG_TEST') == '1', 'rollback-only VPS PostgreSQL integration opt-in')
    def test_postgresql_registry_conflicts_reject_entire_load(self):
        """Reject unknown classifications and conflicting drive ownership in rolled-back schemas; synthetic SQL input, assertions, retained logs; choose scope rejection proof. Byline: Codex 2026-10-06."""
        bundle, pin = self.bundle()
        receipt = loader.prepare(bundle, pin, self.work / 'registry-rejection-validation')
        for mode, message in [('unknown', 'Unknown registry classification'), ('ownership', 'Drive registry ownership/classification conflicts')]:
            schema = 'drive_inventory_test_' + uuid.uuid4().hex
            sql = self.pg_fixture_sql(schema)
            if mode == 'unknown':
                sql += f"UPDATE {schema}.catalog_registry SET status='unknown' WHERE object_schema='unowned' AND object_name='classification_1';\n"
            else:
                sql += f"UPDATE {schema}.catalog_registry SET kind='analysis' WHERE object_name='drive_objects';\n"
            sql += loader.load_sql(receipt, schema).replace('\\set ON_ERROR_STOP 1\nBEGIN;', '').replace('COMMIT;', '')
            sql += '\nROLLBACK;\n'
            result = self.run_pg_fixture(sql, mode + '-rejected')
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(message, result.stderr)

    @unittest.skipUnless(os.environ.get('DRIVE_INVENTORY_PG_TEST') == '1', 'rollback-only VPS PostgreSQL integration opt-in')
    def test_postgresql_receipt_selection_and_exact_rows(self):
        """Prove complete/empty precedence and partial/failed exclusion in a rolled-back schema; synthetic SQL input, assertions, no production objects; choose catalog integration. Byline: Codex 2026-10-06."""
        schema = 'drive_inventory_test_' + uuid.uuid4().hex
        bundle, pin = self.bundle()
        receipt = loader.prepare(bundle, pin, self.work / 'pg-validation')
        table = f'{schema}.drive_objects'
        sql = self.pg_fixture_sql(schema) + loader.ddl(schema)
        sql += f"DO $$ BEGIN IF (SELECT status FROM {schema}.drive_inventory_current_receipts)<>'legacy_unverified' THEN RAISE EXCEPTION 'legacy incorrectly verified'; END IF; END $$;\n"
        partial = dict(receipt, status='partial')
        sql += 'SAVEPOINT partial_admission;\n' + loader.load_sql(partial, schema).replace('\\set ON_ERROR_STOP 1\nBEGIN;', '').replace('COMMIT;', '')
        sql += f"DO $$ BEGIN IF (SELECT status FROM {schema}.drive_inventory_current_receipts)<>'legacy_unverified' OR EXISTS(SELECT 1 FROM {schema}.catalog_registry WHERE as_of LIKE '%{receipt['listed_at'][:10]}%') THEN RAISE EXCEPTION 'partial observation promoted freshness'; END IF; END $$;\n"
        sql += 'DO $$ BEGIN IF ' + loader.registry_equality(partial, schema) + " IS NOT TRUE THEN RAISE EXCEPTION 'partial registry binding failed'; END IF; END $$;\nROLLBACK TO partial_admission;\n"
        load = loader.load_sql(receipt, schema).replace('\\set ON_ERROR_STOP 1\nBEGIN;', '').replace('COMMIT;', '')
        sql += load
        sql += 'DO $$ BEGIN IF NOT (' + loader.equality(receipt, schema) + ") THEN RAISE EXCEPTION 'exact row mismatch'; END IF; END $$;\n"
        sql += 'DO $$ BEGIN IF ' + loader.registry_equality(receipt, schema) + " IS NOT TRUE THEN RAISE EXCEPTION 'registry exact readback mismatch'; END IF; END $$;\n"
        sql += f"DO $$ BEGIN IF (SELECT count(*) FROM {schema}.catalog_registry WHERE object_schema='unowned' AND evidence='unchanged-'||substring(object_name FROM '[0-9]+'))<>33 OR (SELECT moved_from FROM {schema}.catalog_registry WHERE object_name='drive_objects')<>'retained origin' OR (SELECT evidence FROM {schema}.catalog_registry WHERE object_name='drive_objects') NOT LIKE 'retained prior evidence%' OR EXISTS(SELECT 1 FROM {schema}.catalog_registry WHERE status='unknown') THEN RAISE EXCEPTION 'registry protection failed'; END IF; END $$;\n"
        sql += f"CREATE TEMP TABLE di_registry_expected AS SELECT * FROM {schema}.catalog_registry;\nALTER TABLE di_registry_before RENAME TO di_registry_before_first;\nALTER TABLE di_registry_context RENAME TO di_registry_context_first;\n"
        sql += loader.registry_sql(receipt, schema)
        sql += f"DO $$ BEGIN IF EXISTS(SELECT to_jsonb(g) FROM {schema}.catalog_registry g EXCEPT SELECT to_jsonb(x) FROM di_registry_expected x) OR EXISTS(SELECT to_jsonb(x) FROM di_registry_expected x EXCEPT SELECT to_jsonb(g) FROM {schema}.catalog_registry g) THEN RAISE EXCEPTION 'registry retry changed stable evidence'; END IF; END $$;\n"
        for column, damage in [('covers', 'wrong generated description'), ('evidence', 'missing proof')]:
            sql += f"SAVEPOINT damaged_registry; UPDATE {schema}.catalog_registry SET {column}={loader.literal(damage)} WHERE object_schema='{schema}' AND object_name='drive_objects_current';\n"
            sql += 'DO $$ BEGIN IF ' + loader.registry_equality(receipt, schema) + " IS NOT FALSE THEN RAISE EXCEPTION 'damaged registry escaped readback'; END IF; END $$;\nROLLBACK TO damaged_registry;\n"
        for status, date in [('partial', '2026-11-01'), ('failed', '2026-11-02'), ('legacy_unverified', '2026-11-03')]:
            sql += f"INSERT INTO {schema}.drive_listing_receipts(drive,listed_at,generation_id,status,scope,object_count,object_bytes,receipt) VALUES('D:','{date}','test-{status}','{status}','whole_drive_metadata',0,0,'{{}}');\n"
        sql += f"DO $$ BEGIN IF (SELECT generation_id FROM {schema}.drive_inventory_current_receipts)<>{loader.literal(receipt['generation_id'])} THEN RAISE EXCEPTION 'partial/failed/newer legacy replaced complete'; END IF; END $$;\n"
        sql += f"INSERT INTO {schema}.drive_listing_receipts(drive,listed_at,generation_id,status,scope,object_count,object_bytes,receipt) VALUES('D:','2026-12-01','verified-empty','empty','whole_drive_metadata',0,0,'{{}}');\n"
        sql += f"DO $$ BEGIN IF (SELECT generation_id FROM {schema}.drive_inventory_current_receipts)<>'verified-empty' OR EXISTS(SELECT 1 FROM {schema}.drive_objects_current) THEN RAISE EXCEPTION 'empty receipt selection failed'; END IF; END $$;\n"
        sql += f"SELECT json_build_object('legacy_bootstrap',true,'complete_precedence',true,'partial_failed_excluded',true,'empty_selected_without_rows',true,'four_registry_entries',true,'unowned_33_unchanged',true,'unknown_zero',true,'retry_stable',true,'damaged_registry_rejected',true,'six_view_columns',(SELECT count(*) FROM information_schema.columns WHERE table_schema='{schema}' AND table_name='drive_objects_current'));\nROLLBACK;\n"
        result = self.run_pg_fixture(sql, 'disposable-schema')
        self.assertEqual(result.returncode, 0, result.stderr)
        evidence = [json.loads(line) for line in result.stdout.splitlines() if line.startswith('{')][0]
        self.assertEqual(evidence['six_view_columns'], 6)


if __name__ == '__main__':
    unittest.main()
