"""Test exact-run gates and metadata-only followthrough contracts.

Inputs: synthetic metadata fixtures and mocked Temporal clients. Outputs: unittest
results. Effects: retained temporary metadata fixtures only, no network or payload
access. Choose for bounded launch validation. Byline: Codex GPT-6 2026-10-06.
"""
import asyncio
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import AsyncMock, patch
import d_preservation_followthrough_20261006 as tool


class Contracts(unittest.TestCase):
    """Check binding and fixed argv; inputs synthetic refs, output assertions, retained fixture writes; choose offline regression. Byline: Codex 2026-10-06."""
    def setUp(self):
        """Create metadata-only retained fixtures; no inputs, output fixture state, temp writes; choose each isolated test. Byline: Codex 2026-10-06."""
        self.root = Path(tempfile.mkdtemp(prefix='d-followthrough-contract-'))
        self.native = self.root / 'native'
        self.native.mkdir()
        self.manifest = self.root / 'eligible.jsonl'
        self.manifest.write_text('{}\n', encoding='utf-8')
        self.data = dict(stage='upload', status='completed', manifest_sha256=tool.digest(self.manifest),
            count=115005, bytes=14915205983, metadata_unchanged=True, rclone_exit_code=0, api_exit_code=0,
            api_missing_count=0, api_size_mismatch_count=0,
            temporal=dict(workflow_id=tool.PRIMARY, run_id=tool.PRIMARY_RUN))
        for key in ('before', 'after', 'api_observation', 'rclone_json_log'):
            file = self.native / (key + '.json')
            file.write_text('[]', encoding='utf-8')
            self.data[key] = str(file)
        self.receipt = self.native / 'upload-receipt.json'

    def bind(self):
        """Bind a synthetic receipt through production guards; no inputs, output map, fixture write; choose gate tests. Byline: Codex 2026-10-06."""
        self.receipt.write_text(json.dumps(self.data), encoding='utf-8')
        result = [dict(stage='upload', receipt=str(self.receipt), sha256=tool.digest(self.receipt))]
        with patch.multiple(tool, NATIVE=self.native, MANIFEST=self.manifest, MANIFEST_SHA=tool.digest(self.manifest)):
            return tool.bind_upload(result)

    def test_valid_binding_and_argv(self):
        """Preserve basenames and exact expected run/count; no inputs, output assertions, fixture writes; choose successful handoff contract. Byline: Codex 2026-10-06."""
        bound = self.bind()
        self.assertEqual(len(bound['artifacts']), 6)
        self.assertTrue(all(a['server_ref'] == tool.INBOX + '/' + a['basename'] for a in bound['artifacts']))
        argv = tool.verifier_argv(dict(server_python='/usr/bin/python3'), bound)
        self.assertEqual(argv[argv.index('--upload-run') + 1], tool.PRIMARY_RUN)
        self.assertEqual(argv[argv.index('--expected-count') + 1], '115005')
        self.assertEqual(argv[argv.index('--expected-bytes') + 1], '14915205983')
        self.assertEqual(argv[argv.index('--workflow-id') + 1], tool.VERIFY)

    def test_failed_or_changed_receipts_block(self):
        """Reject source changes and incomplete proof; no inputs, output assertions, fixture writes; choose failure regression. Byline: Codex 2026-10-06."""
        for key, value in [('metadata_unchanged', False), ('status', 'interrupted'), ('api_missing_count', 1), ('source_changes', ['changed'])]:
            with self.subTest(key=key):
                old = self.data.get(key)
                self.data[key] = value
                with self.assertRaises(ValueError):
                    self.bind()
                if old is None:
                    self.data.pop(key)
                else:
                    self.data[key] = old

    def test_wrong_run_and_external_reference_block(self):
        """Reject replacement runs and metadata ref escapes; no inputs, output assertions, fixture writes; choose source confinement regression. Byline: Codex 2026-10-06."""
        self.data['temporal']['run_id'] = 'replacement'
        with self.assertRaises(ValueError):
            self.bind()
        self.data['temporal']['run_id'] = tool.PRIMARY_RUN
        self.data['before'] = str(self.manifest)
        with self.assertRaises(ValueError):
            self.bind()

    def test_recovered_errors_require_separate_reconciliation(self):
        """Retain recovered errors and request a distinct run; no inputs, output assertions, metadata writes; choose transparent retry regression. Byline: Codex 2026-10-06."""
        self.data['rclone_observed_counts'] = dict(error_log_count=2)
        bound = self.bind()
        self.assertTrue(bound['requires_reconciliation'])
        self.assertEqual(bound['workflow_id'], tool.PRIMARY)
        self.assertEqual(bound['run_id'], tool.PRIMARY_RUN)

    def test_actual_exit_code_schema_required(self):
        """Reject obsolete exit field aliases; no inputs, output assertion, fixture writes; choose real receipt contract regression. Byline: Codex 2026-10-06."""
        self.data['rclone_exit'] = self.data.pop('rclone_exit_code')
        with self.assertRaises(ValueError):
            self.bind()

    def test_bounded_whole_d_literal_handoff(self):
        """Bind reviewed count, allowlist and filename mode to native handoff; synthetic metadata input, assertions, retained files; choose whole-D transport contract. Byline: Codex 2026-10-07."""
        allow = self.root/'reviewed.files'
        allow.write_text('Owner/evidence.json\n', encoding='utf-8')
        config = dict(server_python='/usr/bin/python3', expected_count=4154, expected_bytes=2165867681,
                      whole_d=True, literal_path_identity=True, source_path_allowlist=str(allow),
                      source_path_allowlist_sha256=tool.digest(allow))
        self.data.update(count=4154, bytes=2165867681, whole_d=True, empty_files_only=False,
                         literal_path_identity=True, backend_encoding='Slash,Del,Ctl,Dot',
                         source_path_allowlist_sha256=tool.digest(allow))
        self.receipt.write_text(json.dumps(self.data), encoding='utf-8')
        results = [dict(stage='upload', receipt=str(self.receipt), sha256=tool.digest(self.receipt))]
        with patch.multiple(tool, NATIVE=self.native, MANIFEST=self.manifest, MANIFEST_SHA=tool.digest(self.manifest)):
            bound = tool.bind_upload(results, config=config)
            for changes in ({'expected_count':4153}, {'literal_path_identity':False},
                            {'source_path_allowlist_sha256':'0'*64}):
                with self.assertRaises(ValueError):
                    tool.bind_upload(results, config={**config, **changes})
        self.assertEqual(len(bound['artifacts']), 7)
        argv = tool.verifier_argv(config, bound)
        self.assertEqual(argv[argv.index('--expected-count')+1], '4154')
        self.assertEqual(argv[argv.index('--expected-bytes')+1], '2165867681')
        self.assertIn('--literal-path-identity', argv)
        self.assertIn('--whole-d', argv)
        self.assertEqual(argv[argv.index('--source-path-allowlist')+1], bound['source_path_allowlist'])


class AsyncContracts(unittest.IsolatedAsyncioTestCase):
    """Exercise event waiting and duplicate guards; input mocks, output assertions, no external effects; choose orchestration tests. Byline: Codex 2026-10-06."""
    async def test_event_wait_without_poll(self):
        """Resolve one future without status queries; no inputs, output assertion, mocked heartbeat; choose event wait regression. Byline: Codex 2026-10-06."""
        with patch.object(tool.activity, 'heartbeat'):
            self.assertEqual(await tool.await_event(asyncio.create_task(asyncio.sleep(0, result=42))), 42)

    async def test_completed_adoption_never_launches_second_verifier(self):
        """Adopt only one exact completed execution without claims or launches; mocked Temporal input, assertions, retained receipt; choose stalled-SSH finalization guard. Byline: Codex 2026-10-07."""
        from unittest.mock import Mock
        work = Path(tempfile.mkdtemp(prefix='d-followthrough-adopt-'))
        client = Mock()
        description = Mock()
        description.status.name, description.run_id = 'COMPLETED', 'exact-completed-run'
        handle = client.get_workflow_handle.return_value
        handle.describe = AsyncMock(return_value=description)
        handle.result = AsyncMock(return_value=[{'stage':s} for s in ['validate','historical_sources','listing_before','download','hash_cache','listing_after','record']])
        async def metadata(argv, path, label):
            """Retain only a mocked server response; argv input, no output, tiny file; choose adoption test. Byline: Codex 2026-10-07."""
            (work/'catalog-readback.stdout').write_text('{}')
        config = dict(work=str(work), server_python='/usr/bin/python3', expected_count=4154,
                      expected_bytes=2165867681, adopt_completed_verification_run='exact-completed-run')
        with patch.object(tool, 'connected', AsyncMock(return_value=client)), \
             patch.object(tool, 'existing_verifier', AsyncMock()) as duplicate, \
             patch.object(tool, 'command', AsyncMock(side_effect=metadata)) as execute:
            await tool.remote_verify(dict(config=config,bound={'upload_sha256':'reviewed'}))
        duplicate.assert_not_called()
        client.get_workflow_handle.assert_called_once_with(tool.VERIFY,run_id='exact-completed-run')
        self.assertEqual([c.args[2] for c in execute.call_args_list], ['catalog-readback'])
        saved=json.loads((work/'followthrough-completed.json').read_text())
        self.assertEqual(saved['verification_launch'],'adopted_exact_completed_run')
        self.assertIn("len(refs)==17", execute.call_args.args[0][-1])

    async def test_adoption_rejects_running_or_incomplete_result(self):
        """Reject incomplete verification without launches or completion receipts; mocked statuses input, assertions, no external effects; choose no-fake-completion regression. Byline: Codex 2026-10-07."""
        from unittest.mock import Mock
        for status, stages in [('RUNNING', []), ('COMPLETED', [{'stage':'record'}])]:
            work=Path(tempfile.mkdtemp(prefix='d-followthrough-reject-'))
            client=Mock();handle=client.get_workflow_handle.return_value
            description=Mock();description.status.name=status
            handle.describe=AsyncMock(return_value=description);handle.result=AsyncMock(return_value=stages)
            config=dict(work=str(work),server_python='/usr/bin/python3',adopt_completed_verification_run='fixed')
            with patch.object(tool,'connected',AsyncMock(return_value=client)),patch.object(tool,'command',AsyncMock()) as command:
                with self.assertRaises(ValueError):
                    await tool.remote_verify(dict(config=config,bound={'upload_sha256':'reviewed'}))
                command.assert_not_called()
            self.assertFalse((work/'followthrough-completed.json').exists())

    async def test_detached_launch_waits_for_temporal_result(self):
        """Start one detached verifier and await its event before readback; mocked services input, assertions, retained receipt; choose bounded SSH orchestration regression. Byline: Codex 2026-10-07."""
        from unittest.mock import Mock
        work=Path(tempfile.mkdtemp(prefix='d-followthrough-detached-'))
        client=Mock();handle=client.get_workflow_handle.return_value
        description=Mock();description.status.name='COMPLETED';description.run_id='new-run'
        handle.describe=AsyncMock(return_value=description)
        stages=['validate','historical_sources','listing_before','download','hash_cache','listing_after','record']
        handle.result=AsyncMock(return_value=[{'stage':s} for s in stages])
        async def metadata(argv,path,label):
            """Retain a tiny mocked catalog response; inputs argv, no output, fixture write; choose detached test. Byline: Codex 2026-10-07."""
            if label=='catalog-readback':(work/'catalog-readback.stdout').write_text('{}')
        config=dict(work=str(work),server_python='/usr/bin/python3',detached_verifier=True,
                    expected_count=4154,expected_bytes=2165867681)
        bound=dict(manifest='/server/manifest',upload_receipt='/server/receipt',upload_sha256='reviewed',
                   workflow_id='upload',run_id='fixed')
        with patch.object(tool,'connected',AsyncMock(return_value=client)), \
             patch.object(tool,'existing_verifier',AsyncMock()),patch.object(tool.activity,'heartbeat'), \
             patch.object(tool,'command',AsyncMock(side_effect=metadata)) as commands:
            await tool.remote_verify(dict(config=config,bound=bound))
        self.assertEqual([c.args[2] for c in commands.call_args_list],['claim-verifier','verifier-launch','catalog-readback'])
        self.assertIn('subprocess.Popen',commands.call_args_list[1].args[0][-1])
        self.assertIn('start_new_session=True',commands.call_args_list[1].args[0][-1])
        self.assertEqual(handle.result.await_count,2)

    async def test_reconciliation_then_serial_partitions_then_verification(self):
        """Keep reconciliation a completed child before binding and serialize follow-on copies; no inputs, output assertions, mocked orchestration; choose transparent lineage regression. Byline: Codex 2026-10-06."""
        child = asyncio.get_running_loop().create_future()
        child.first_execution_run_id = 'separate-clean-run'
        child.set_result([dict(stage='upload')])
        original = dict(requires_reconciliation=True)
        clean = dict(requires_reconciliation=False)
        config = dict(work='audit', reconciliation=dict(work='reconcile'),
                      d_other=[dict(workflow_id='nonzero', config={}), dict(workflow_id='empty', config={})])
        with patch.object(tool.workflow, 'execute_activity', AsyncMock(side_effect=[original, clean, 'handoff', 'verified'])) as activities, \
             patch.object(tool.workflow, 'start_child_workflow', AsyncMock(return_value=child)) as reconcile, \
             patch.object(tool.workflow, 'execute_child_workflow', AsyncMock(side_effect=['nonzero-copy', 'empty-copy'])) as copies:
            result = await tool.Followthrough().run(config)
        self.assertEqual(result, ['handoff', 'nonzero-copy', 'empty-copy', 'verified'])
        self.assertEqual(reconcile.call_args.kwargs['id'], tool.RECONCILE)
        self.assertEqual(activities.call_args_list[1].args[1]['run_id'], 'separate-clean-run')
        self.assertEqual([call.kwargs['id'] for call in copies.call_args_list], ['nonzero', 'empty'])

    async def test_existing_verifier_never_relaunched(self):
        """Stop on any existing verifier status; no inputs, output assertion, mocked network; choose duplicate prevention regression. Byline: Codex 2026-10-06."""
        from unittest.mock import Mock
        client = Mock()
        description = Mock()
        description.status.name = 'RUNNING'
        client.get_workflow_handle.return_value.describe = AsyncMock(return_value=description)
        with self.assertRaisesRegex(RuntimeError, 'already exists'):
            await tool.existing_verifier(client)

    async def test_failed_primary_never_binds(self):
        """Retain a stopped receipt without invoking handoff after failed result; no inputs, output assertions, fixture writes; choose failure gate regression. Byline: Codex 2026-10-06."""
        from unittest.mock import Mock
        client = Mock()
        client.get_workflow_handle.return_value.result = AsyncMock(side_effect=RuntimeError('canceled primary'))
        work = Path(tempfile.mkdtemp(prefix='d-followthrough-stopped-'))
        with patch.object(tool, 'connected', AsyncMock(return_value=client)), patch.object(tool.activity, 'heartbeat'), patch.object(tool, 'bind_upload') as bind:
            with self.assertRaisesRegex(RuntimeError, 'canceled'):
                await tool.await_primary(dict(work=str(work)))
            bind.assert_not_called()
        self.assertTrue((work / 'primary-gate-stopped.json').exists())


if __name__ == '__main__':
    unittest.main()
