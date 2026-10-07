"""Exercise D preservation proof gates using tiny retained fixtures and mocked external services.

Inputs: synthetic bytes and manifests; outputs unittest assertions; effects new retained
fixtures on E: or the server only. No source drive, B2, catalog writes or deletions.
Choose before the explicitly authorized one-PNG Temporal verification.
Byline: Codex / GPT-6 / 2026-10-06.
"""
import asyncio
import ast
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import unittest
from unittest.mock import AsyncMock, patch
import uuid

import d_preservation_verify_20261006 as tool


class PreservationVerificationTests(unittest.TestCase):
    """Check fail-closed frozen proof and catalog contracts; synthetic inputs, assertions, retained fixtures; choose bounded regression. Byline: Codex 2026-10-06."""
    def setUp(self):
        """Create unique tiny fixtures without cleanup; no inputs, test state output, retained writes; choose isolated tests. Byline: Codex 2026-10-06."""
        base = Path('E:/AI_Workspace/Projects/Propria/modules/Consignatio/docs/receipts/d-preservation-verify-synthetic') if os.name == 'nt' else Path('/data/consignatio/d-preservation-verify-synthetic')
        self.work = base / uuid.uuid4().hex
        self.work.mkdir(parents=True)
        self.relative = 'Backup/Projects/_TO_BE_DELETED/fixture.png'
        self.data = b'bounded synthetic destination\x00\xff'
        self.manifest = self.work / 'manifest.jsonl'
        self.manifest.write_text(json.dumps(dict(rel_path=self.relative, source_root='D:/',
            size_bytes=len(self.data), source_mtime='2025-12-13T21:12:48+00:00', status='source_hash_missing')) + '\n', encoding='utf-8')
        self.config = dict(manifest=str(self.manifest), manifest_sha256=hashlib.sha256(self.manifest.read_bytes()).hexdigest(),
            expected_count=1, expected_bytes=len(self.data), upload_workflow='synthetic-upload', upload_run='synthetic-run',
            workflow_id='synthetic-verify', run_id='synthetic-verify-run', work=str(self.work))
        self.rows = tool.load_manifest(self.config)
        self.object = dict(Path=self.relative, Size=len(self.data), IsDir=False, ID='version-one',
                           Hashes={'SHA-1': hashlib.sha1(self.data).hexdigest()}, Metadata={}, ModTime='2025-12-13T21:12:48Z')
        observation = dict(relative_path=self.relative, size=len(self.data), mtime_ns=self.rows[0]['mtime_ns'], matches_manifest=True)
        tool.save(self.work / 'before.json', [observation])
        tool.save(self.work / 'after.json', [observation])
        tool.save(self.work / 'api.json', [self.object])
        self.log = self.work / 'rclone.jsonl'
        self.log.write_text(json.dumps(dict(object=self.relative, msg='Copied (new)', level='info')) + '\n', encoding='utf-8')
        self.receipt = dict(stage='upload', status='completed', manifest_sha256=self.config['manifest_sha256'],
            count=1, bytes=len(self.data), metadata_unchanged=True, rclone_exit_code=0, api_exit_code=0,
            api_missing_count=0, api_size_mismatch_count=0, destination='b2:salem-data/' + tool.PREFIX,
            temporal=dict(workflow_id='synthetic-upload', run_id='synthetic-run'), before='E:\\receipt\\before.json',
            after='E:\\receipt\\after.json', api_observation='E:\\receipt\\api.json', rclone_json_log='E:\\receipt\\rclone.jsonl',
            rclone_observed_counts=dict(copied_log_count=1, existing_unchanged_log_count=0, unclassified_manifest_count=0,
                                       error_log_count=0, malformed_json_lines=0))
        self.receipt_path = self.work / 'upload-receipt.json'
        self.bind_receipt()

    def bind_receipt(self):
        """Pin a newly named synthetic receipt; input fixture state, output config, retained writes; choose contract variants. Byline: Codex 2026-10-06."""
        self.receipt_path = self.work / ('upload-' + uuid.uuid4().hex + '.json')
        self.config.update(upload_receipt=str(self.receipt_path), upload_sha256=tool.save(self.receipt_path, self.receipt))

    def cache(self, data=None):
        """Retain a tiny cache member; input optional bytes, output root, fixture writes; choose full-stream checks. Byline: Codex 2026-10-06."""
        root = self.work / 'cache'
        target = root / self.relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(self.data if data is None else data)
        return root

    def test_successful_upload_is_metadata_only(self):
        """Accept exact completed transport without fresh-source claims; synthetic receipt input, assertions, no external effects; choose positive gate. Byline: Codex 2026-10-06."""
        result = tool.validate_upload(self.config, self.rows)
        self.assertEqual(result['copied_log_count'], 1)
        self.assertIn('no fresh D source hash', result['limits'])
        self.assertEqual(set(result['artifact_sha256']), {'before', 'after', 'api_observation', 'rclone_json_log'})

    def test_missing_final_receipt(self):
        """Reject unfinished transport with no final receipt; missing path input, assertion output, no effects; choose primary-upload guard. Byline: Codex 2026-10-06."""
        self.config['upload_receipt'] = str(self.work / 'unfinished.json')
        with self.assertRaises(FileNotFoundError):
            tool.validate_upload(self.config, self.rows)

    def test_failed_or_changed_upload(self):
        """Reject failed upload and changed stats flag; synthetic variants, assertions, retained writes; choose completion gate. Byline: Codex 2026-10-06."""
        for key, value in [('status', 'failed'), ('metadata_unchanged', False), ('count', 2), ('rclone_exit_code', 1)]:
            with self.subTest(key=key):
                previous = self.receipt[key]
                self.receipt[key] = value
                self.bind_receipt()
                with self.assertRaisesRegex(ValueError, 'successful upload'):
                    tool.validate_upload(self.config, self.rows)
                self.receipt[key] = previous

    def test_wrong_upload_run(self):
        """Reject receipt attribution to a different run; synthetic identity input, assertion output, no effects; choose run binding. Byline: Codex 2026-10-06."""
        self.config['upload_run'] = 'different-run'
        with self.assertRaisesRegex(ValueError, 'workflow/run'):
            tool.validate_upload(self.config, self.rows)

    def test_unaccounted_log_outcome(self):
        """Reject successful exit with missing per-file outcomes; empty retained log, assertion, fixture write; choose exact accounting. Byline: Codex 2026-10-06."""
        self.log.write_text('', encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'accounting'):
            tool.validate_upload(self.config, self.rows)

    def test_size_only_skip_remains_limited(self):
        """Accept accounted skips while retaining source-proof limits; synthetic skip input, assertions, fixture writes; choose transport distinction. Byline: Codex 2026-10-06."""
        self.log.write_text(json.dumps(dict(object=self.relative, msg='Unchanged skipping', level='debug')), encoding='utf-8')
        self.receipt['rclone_observed_counts'].update(copied_log_count=0, existing_unchanged_log_count=1)
        self.bind_receipt()
        self.assertEqual(tool.validate_upload(self.config, self.rows)['size_only_skip_count'], 1)

    def test_frozen_manifest_pin_and_counts(self):
        """Reject wrong manifest pins and totals; synthetic config variants, assertions, no external effects; choose scope gate. Byline: Codex 2026-10-06."""
        for update in [dict(manifest_sha256='0' * 64), dict(expected_count=2), dict(expected_bytes=1)]:
            with self.subTest(update=update), self.assertRaises(ValueError):
                tool.load_manifest({**self.config, **update})

    def test_unsafe_paths(self):
        """Reject traversal, line breaks and Windows aliases; path inputs, assertions, no effects; choose download confinement. Byline: Codex 2026-10-06."""
        for path in ['Backup/../a', '/Backup/a', 'Backup/a\nb', 'Backup/NUL.png', 'Backup/a:b', 'Backup/a\\b', 'Backup/x.']:
            with self.subTest(path=path), self.assertRaises(ValueError):
                tool.safe_relative(path)
        self.assertEqual(tool.safe_relative(self.relative), self.relative)

    def test_exact_listing_missing_extra_duplicate_or_size(self):
        """Reject incomplete or ambiguous destination snapshots; synthetic listings, assertions, no effects; choose both version snapshots. Byline: Codex 2026-10-06."""
        variants = [[], [self.object, self.object], [{**self.object, 'Size': 1}], [{**self.object, 'ID': ''}],
                    [self.object, {**self.object, 'Path': 'Backup/extra.png'}]]
        for objects in variants:
            with self.subTest(objects=objects), self.assertRaises(ValueError):
                tool.validate_listing(objects, self.rows)

    def test_changed_version_same_size(self):
        """Reject replacement versions even at equal size/hash; snapshot inputs, assertion output, no effects; choose mutation race guard. Byline: Codex 2026-10-06."""
        before = tool.validate_listing([self.object], self.rows)
        after = tool.validate_listing([{**self.object, 'ID': 'version-two'}], self.rows)
        with self.assertRaisesRegex(ValueError, 'version'):
            tool.compare_versions(before, after)

    def test_full_three_hashes(self):
        """Hash all fixture bytes with three algorithms and size; tiny cache input, assertions, retained writes; choose independent byte proof. Byline: Codex 2026-10-06."""
        observed = tool.hash_file(self.cache(), self.rows[0])
        self.assertEqual(observed, dict(size=len(self.data), **{k: hashlib.new(k, self.data).hexdigest() for k in ('sha1', 'sha256', 'md5')}))

    def test_missing_cache_file(self):
        """Reject missing retained files instead of fabricating completion; empty cache input, assertion, retained directory; choose missing-file regression. Byline: Codex 2026-10-06."""
        root = self.work / 'empty-cache'
        root.mkdir()
        with self.assertRaises(FileNotFoundError):
            tool.hash_file(root, self.rows[0])

    def test_truncated_cache(self):
        """Reject short downloads despite existing file names; truncated bytes input, assertion, retained fixture; choose size regression. Byline: Codex 2026-10-06."""
        with self.assertRaisesRegex(ValueError, 'size mismatch'):
            tool.hash_file(self.cache(b'x'), self.rows[0])

    def test_hash_conflicts_and_historical_attribution(self):
        """Reject provider/historical hash conflicts and label historical matches; synthetic hashes input, assertions, no external effects; choose provenance regression. Byline: Codex 2026-10-06."""
        observed = tool.hash_file(self.cache(), self.rows[0])
        historical = dict(source='local/D-Backup', path=self.relative[7:], source_id='', scope='', size=len(self.data),
                          native_hash_kind='sha256', native_hash=observed['sha256'], md5=observed['md5'])
        result = tool.compare_hashes(observed, self.object, [historical])
        self.assertFalse(result['fresh_d_source_hash_verified'])
        self.assertEqual(result['historical_source_matches'][0]['matched_algorithms'], ['md5', 'sha256'])
        for provider, sources in [({**self.object, 'Hashes': {'SHA-1': '0' * 40}}, []),
                                   (self.object, [{**historical, 'native_hash': '0' * 64}])]:
            with self.subTest(provider=provider), self.assertRaisesRegex(ValueError, 'hash'):
                tool.compare_hashes(observed, provider, sources)
        self.assertEqual(tool.compare_hashes(observed, self.object, [])['historical_source_matches'], [])

    def test_no_overwrite_of_retained_artifacts(self):
        """Retain conflicting receipts instead of replacing them; existing fixture path, assertions, no replacement; choose preservation guard. Byline: Codex 2026-10-06."""
        path = self.work / 'retain.json'
        digest = tool.save(path, {'old': True})
        self.assertEqual(tool.save(path, {'old': True}), digest)
        with self.assertRaisesRegex(ValueError, 'artifact conflict'):
            tool.save(path, {'old': False})

    def test_catalog_exact_payload_readback_conflict(self):
        """Reject ledger conflicts after insert-on-conflict; mocked SQL input, assertions, no database effects; choose independent catalog gate. Byline: Codex 2026-10-06."""
        row = dict(relative_path=self.relative, dest_key=tool.PREFIX + self.relative, b2_object_version_id='version-one')
        with self.assertRaisesRegex(ValueError, 'readback conflict'):
            tool.record_rows(self.config, [row], pg_execute=lambda sql: [])

    def test_existing_catalog_writer_contract(self):
        """Preserve sibling action and exact destination-only payload; mocked ledger input, assertions, retained readback; choose reader compatibility. Byline: Codex 2026-10-06."""
        calls, stored = [], []
        def execute(sql):
            """Emulate exact existing ledger insert/readback; SQL input, rows output, in-memory effects; choose writer contract fixture. Byline: Codex 2026-10-06."""
            calls.append(sql)
            if 'INSERT INTO' in sql:
                match = re.search(r"jsonb_to_recordset\('((?:[^']|'')*)'::jsonb\)", sql)
                stored.extend(json.loads(match[1].replace("''", "'")))
                return []
            return copy.deepcopy(stored)
        row = dict(relative_path=self.relative, dest_key=tool.PREFIX + self.relative, b2_object_version_id='version-one',
                   hash_observation_kind='full_stream_b2_version_bound', fresh_d_source_hash_verified=False,
                   verified=dict(size=len(self.data), sha1=hashlib.sha1(self.data).hexdigest()))
        result = tool.record_rows(self.config, [row], pg_execute=execute)
        self.assertEqual(result['catalog_verified'], 1)
        self.assertEqual(stored[0]['source_bucket'], 'local/D-Backup')
        self.assertEqual(stored[0]['source_path'], self.relative[7:])
        self.assertEqual(stored[0]['preservation_action'], 'preserved_immutable_verified')
        self.assertFalse(stored[0]['payload']['fresh_d_source_hash_verified'])
        self.assertTrue(all('CREATE TABLE' not in sql and 'UPDATE ' not in sql for sql in calls))
        self.assertEqual(tool.read_json(self.work / 'catalog-readback-000000.json'), stored)

    def test_metadata_actual_values_even_when_flags_claim_success(self):
        """Reject wrong stat values despite success flags; synthetic companion input, assertion, retained fixture; choose independent metadata reconciliation. Byline: Codex 2026-10-06."""
        values = [dict(relative_path=self.relative, size=len(self.data), mtime_ns=0, matches_manifest=True)]
        tool.save(self.work / 'wrong-before.json', values)
        tool.save(self.work / 'wrong-after.json', values)
        self.receipt.update(before='E:\\receipt\\wrong-before.json', after='E:\\receipt\\wrong-after.json')
        self.bind_receipt()
        with self.assertRaisesRegex(ValueError, 'metadata does not match'):
            tool.validate_upload(self.config, self.rows)

    def test_historical_hashless_occurrence_stays_unverified(self):
        """Keep hashless history separate from source-hash matches; synthetic catalog input, assertions, no external effects; choose held-PNG regression. Byline: Codex 2026-10-06."""
        observed = tool.hash_file(self.cache(), self.rows[0])
        result = tool.compare_hashes(observed, self.object, [dict(size=len(self.data), native_hash_kind='sha256', native_hash=None, md5=None)])
        self.assertEqual(result['historical_source_matches'], [])
        self.assertEqual(len(result['historical_without_supported_hash']), 1)
        self.assertFalse(result['fresh_d_source_hash_verified'])

    def test_no_heavy_work_on_desktop(self):
        """Reject desktop heavy execution independently of fixtures; mocked host input, assertion, no effects; choose machine boundary. Byline: Codex 2026-10-06."""
        with patch.object(tool.socket, 'gethostname', return_value='desktop'), self.assertRaisesRegex(ValueError, 'ovh-files'):
            tool.require_server(self.work)

    def test_unfinished_temporal_upload(self):
        """Reject running upload before awaiting results; mocked client input, assertion output, no network; choose live completion gate. Byline: Codex 2026-10-06."""
        try:
            from temporalio.client import WorkflowExecutionStatus
        except ImportError:
            self.skipTest('Temporal SDK tested on VPS')
        handle = AsyncMock()
        handle.describe.return_value.status = WorkflowExecutionStatus.RUNNING
        client = AsyncMock()
        from unittest.mock import Mock
        client.get_workflow_handle = Mock(return_value=handle)
        with patch('temporalio.client.Client.connect', AsyncMock(return_value=client)):
            with self.assertRaisesRegex(ValueError, 'not completed'):
                asyncio.run(tool.completed_upload({**self.config, 'address': 'synthetic'}))
        handle.result.assert_not_awaited()

    def test_completed_run_binds_receipt_digest_despite_bundle_rename(self):
        """Accept a renamed server bundle only by exact completed-run digest; mocked Temporal input, assertions, no network; choose portable receipt provenance. Byline: Codex 2026-10-06."""
        try:
            from temporalio.client import WorkflowExecutionStatus
        except ImportError:
            self.skipTest('Temporal SDK tested on VPS')
        from unittest.mock import Mock
        handle = AsyncMock()
        handle.describe.return_value.status = WorkflowExecutionStatus.COMPLETED
        handle.result.return_value = [dict(stage='upload', sha256=self.config['upload_sha256'], receipt='E:\\native\\original-receipt.json')]
        client = AsyncMock()
        client.get_workflow_handle = Mock(return_value=handle)
        with patch('temporalio.client.Client.connect', AsyncMock(return_value=client)):
            result = asyncio.run(tool.completed_upload({**self.config, 'address': 'synthetic'}))
            self.assertEqual(result['native_receipt_ref'], 'E:\\native\\original-receipt.json')
            handle.result.return_value[0]['sha256'] = '0' * 64
            with self.assertRaisesRegex(ValueError, 'does not bind'):
                asyncio.run(tool.completed_upload({**self.config, 'address': 'synthetic'}))

    def test_separately_named_activity_sequence(self):
        """Track seven independent bounded-reference operations; mocked SDK input, assertions, no worker/network; choose orchestration contract. Byline: Codex 2026-10-06."""
        try:
            from temporalio import workflow
        except ImportError:
            self.skipTest('Temporal SDK tested on VPS')
        execute = AsyncMock(return_value=dict(receipt='server-artifact', sha256='synthetic'))
        with patch.object(workflow, 'execute_activity', execute):
            results = asyncio.run(tool.VerificationWorkflow().run('/data/consignatio/synthetic/config.json'))
        self.assertEqual(len(results), 7)
        self.assertEqual([c.args[0] for c in execute.call_args_list], ['d_preservation_verify_' + name + '_20261006' for name in tool.STAGES])
        self.assertTrue(all(set(c.args[1]) == {'stage', 'config'} for c in execute.call_args_list))

    def test_literal_registrations_have_canonical_structured_docs(self):
        """Require seven static registrations and structured Activity/workflow docs; source input, assertions, no effects; choose generated-catalog contract. Byline: Codex 2026-10-06."""
        tree = ast.parse(Path(tool.__file__).read_text(encoding='utf-8'))
        start = next(n for n in tree.body if isinstance(n, ast.AsyncFunctionDef) and n.name == 'start')
        assignment = next(n for n in start.body if isinstance(n, ast.Assign) and any(isinstance(t, ast.Name) and t.id == 'activities' for t in n.targets))
        names = []
        for call in assignment.value.elts:
            self.assertIsInstance(call, ast.Call)
            registration = call.func
            self.assertIsInstance(registration, ast.Call)
            literal = next(k.value for k in registration.keywords if k.arg == 'name')
            self.assertIsInstance(literal, ast.Constant)
            names.append(literal.value)
            function = getattr(tool, call.args[0].id)
            doc = function.__doc__.strip()
            for label in ('Inputs:', 'Outputs:', 'Effects:', 'Choose '):
                self.assertTrue(any(line.strip().startswith(label) for line in doc.splitlines()[1:]))
            self.assertNotIn('Inputs:', doc.splitlines()[0])
        self.assertEqual(names, ['d_preservation_verify_' + stage + '_20261006' for stage in tool.STAGES])
        for unit in (tool.VerificationWorkflow, tool.VerificationWorkflow.run):
            for label in ('Inputs:', 'Outputs:', 'Effects:', 'Choose '):
                self.assertTrue(any(line.strip().startswith(label) for line in unit.__doc__.strip().splitlines()[1:]))


if __name__ == '__main__':
    unittest.main()
