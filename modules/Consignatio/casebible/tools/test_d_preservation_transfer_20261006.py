"""Test bounded preservation guards with tiny retained E: fixtures and mocked transport.

Inputs: synthetic fixtures only. Outputs: unittest results; fixture files retained.
Effects: creates small E: artifacts, never copies D: payloads or deletes files.
Choose for offline transport safety checks, not live Temporal/B2 proof.
Byline: Codex · GPT-6 · 2026-10-06.
"""
import hashlib
import asyncio
import json
import os
from pathlib import Path
import subprocess
import threading
import unittest
from unittest.mock import patch
from unittest.mock import AsyncMock, MagicMock
from types import SimpleNamespace
import uuid

import d_preservation_transfer_20261006 as tool


class PreservationTests(unittest.TestCase):
    """Exercise input, observation and process boundaries; synthetic inputs, assertions, retained fixtures; choose offline validation. Byline: Codex 2026-10-06."""
    def setUp(self):
        """Create isolated retained fixture space; no inputs, output paths, effects tiny E: files; choose each test. Byline: Codex 2026-10-06."""
        self.work = Path('E:/AI_Workspace/.uv-cache/d-preservation-tests') / uuid.uuid4().hex
        self.work.mkdir(parents=True)
        self.row = {'relative_path': 'Backup/example.txt', 'size': 3, 'mtime_ns': 123}

    def manifest(self, rows):
        """Write a small frozen manifest; input rows, output path/hash, effects retained fixture; choose guard tests. Byline: Codex 2026-10-06."""
        path = self.work / (uuid.uuid4().hex + '.json')
        raw = json.dumps({'files': rows}).encode()
        path.write_bytes(raw)
        return path, hashlib.sha256(raw).hexdigest()

    def test_frozen_hash_and_valid_input(self):
        """Accept exact manifest and reject altered hash; synthetic input, assertions, no transport; choose freeze proof. Byline: Codex 2026-10-06."""
        path, digest = self.manifest([self.row])
        self.assertEqual(tool.load_manifest(path, digest), [{**self.row, 'mtime_precision_ns': 1}])
        with self.assertRaisesRegex(ValueError, 'SHA256'):
            tool.load_manifest(path, '0' * 64)

    def test_unsafe_paths_zero_errors_duplicates(self):
        """Reject Windows escapes and ineligible rows; synthetic inputs, assertions, fixture writes; choose scope guard. Byline: Codex 2026-10-06."""
        for relative in ('../Backup/x', 'Backup/../x', 'D:/Backup/x', 'Backup/a:b',
                         'Backup/x\nother', 'Backup/CON.txt', 'Backup/a.', 'Other/x',
                         'Backup//x', 'Backup/quarantine/x', 'Backup/a\\x'):
            with self.subTest(relative=relative):
                path, digest = self.manifest([{**self.row, 'relative_path': relative}])
                with self.assertRaises(ValueError):
                    tool.load_manifest(path, digest)
        for update in ({'size': 0}, {'error': 'unreadable'}, {'excluded': True}, {'mtime_ns': '123'}):
            path, digest = self.manifest([{**self.row, **update}])
            with self.assertRaises(ValueError):
                tool.load_manifest(path, digest)
        path, digest = self.manifest([self.row, {**self.row, 'relative_path': 'Backup/EXAMPLE.txt'}])
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            tool.load_manifest(path, digest)

    def test_explorer_jsonl_and_destination_conflict(self):
        """Consume explorer rows but reject contradictory targets; synthetic JSONL, assertions, fixture writes; choose export contract. Byline: Codex 2026-10-06."""
        row = {'rel_path': self.row['relative_path'], 'size_bytes': 3, 'source_root': 'D:/',
               'source_mtime': '2025-05-02T14:50:12+00:00', 'status': 'source_hash_missing',
               'copy_target_root': tool.DEST}
        path = self.work/'explorer.jsonl'
        raw = (json.dumps(row)+'\n').encode()
        path.write_bytes(raw)
        observed = tool.load_manifest(path, hashlib.sha256(raw).hexdigest())[0]
        self.assertEqual(observed['mtime_precision_ns'], 1000000000)
        self.assertEqual(observed['relative_path'], self.row['relative_path'])
        row['copy_target_root'] = 'b2:salem-data/Recovered/_Incoming/d-preservation-20261006/'
        raw = (json.dumps(row)+'\n').encode()
        conflict = self.work/'conflict.jsonl'
        conflict.write_bytes(raw)
        with self.assertRaisesRegex(ValueError, 'contradicts'):
            tool.load_manifest(conflict, hashlib.sha256(raw).hexdigest())

    def test_explicit_quarantine_path_override_keeps_zero_error_guards(self):
        """Permit approved historical quarantine paths without inferring junk; input synthetic rows, assertions, retained fixtures; choose explicit preservation override. Byline: Codex 2026-10-06."""
        row = {**self.row, 'relative_path': 'Backup/Projects/_TO_BE_DELETED/archive/image.png'}
        path, digest = self.manifest([row])
        with self.assertRaisesRegex(ValueError, 'quarantine outside'):
            tool.load_manifest(path, digest)
        self.assertEqual(tool.load_manifest(path, digest, True)[0]['relative_path'], row['relative_path'])
        for update in ({'size': 0}, {'error': 'known corrupted'}, {'status': 'all_zero'}):
            path, digest = self.manifest([{**row, **update}])
            with self.assertRaises(ValueError):
                tool.load_manifest(path, digest, True)

    def test_whole_d_requires_exact_frozen_allowlist(self):
        """Permit only explicit non-Backup paths with frozen matching allowlist; input synthetic rows/hash, assertions, fixture writes; choose whole-D scope proof. Byline: Codex 2026-10-06."""
        row = {**self.row, 'relative_path': 'WSL/Ubuntu/ext4.vhdx'}
        path, digest = self.manifest([row])
        with self.assertRaises(ValueError):
            tool.load_manifest(path, digest)
        allow = self.work/'source-paths.files'
        raw = b'WSL/Ubuntu/ext4.vhdx\n'
        allow.write_bytes(raw)
        config = {'whole_d': True, 'source_path_allowlist': str(allow),
                  'source_path_allowlist_sha256': hashlib.sha256(raw).hexdigest()}
        paths = tool.load_allowlist(config)
        self.assertEqual(tool.load_manifest(path, digest, allowed_paths=paths)[0]['relative_path'], row['relative_path'])
        for change in ({'source_path_allowlist_sha256': '0'*64}, {'whole_d': False}):
            with self.assertRaises(ValueError):
                tool.load_allowlist({**config, **change})
        with self.assertRaises(ValueError):
            tool.load_allowlist({'whole_d': True})
        for allowed in ({'different/file'}, paths | {'extra/file'}):
            with self.assertRaises(ValueError):
                tool.load_manifest(path, digest, allowed_paths=allowed)
        for relative in ('../outside', 'WSL/../outside', 'C:/outside', 'WSL/CON', 'WSL/a:b'):
            bad, bad_hash = self.manifest([{**row, 'relative_path': relative}])
            with self.assertRaises(ValueError):
                tool.load_manifest(bad, bad_hash, allowed_paths={relative})

    def test_whole_d_observes_only_listed_paths_and_reports_missing(self):
        """Stat a tiny explicitly listed non-Backup file and report missing paths; input fixture, assertions, tiny retained bytes; choose whole-D observation. Byline: Codex 2026-10-06."""
        directory = self.work/'WSL'/'Ubuntu'
        directory.mkdir(parents=True)
        file = directory/'ext4.vhdx'
        file.write_bytes(b'abc')
        row = {**self.row, 'relative_path': 'WSL/Ubuntu/ext4.vhdx', 'mtime_ns': file.stat().st_mtime_ns}
        self.assertTrue(tool.observe([row], self.work, whole_d=True)[0]['matches_manifest'])
        self.assertFalse(tool.observe([row], self.work)[0]['matches_manifest'])
        missing = {**row, 'relative_path': 'WSL/Ubuntu/missing.vhdx'}
        result = tool.observe([row, missing], self.work, whole_d=True)
        self.assertTrue(result[0]['matches_manifest'])
        self.assertIn('error', result[1])

    def test_empty_files_have_separate_explicit_mode(self):
        """Accept size-zero files only in an isolated empty batch; synthetic manifest, assertions, retained empty fixture; choose metadata preservation without all-zero payloads. Byline: Codex 2026-10-06."""
        row = {**self.row, 'size': 0, 'status': 'zero_byte'}
        path, digest = self.manifest([row])
        with self.assertRaises(ValueError):
            tool.load_manifest(path, digest)
        self.assertEqual(tool.load_manifest(path, digest, empty_files_only=True)[0]['size'], 0)
        for rows in ([row, self.row], [{**row, 'status': 'all_zero'}], [{**row, 'error': 'corrupt'}]):
            path, digest = self.manifest(rows)
            with self.assertRaises(ValueError):
                tool.load_manifest(path, digest, empty_files_only=True)
        backup = self.work/'Backup'
        backup.mkdir()
        file = backup/'example.txt'
        file.write_bytes(b'')
        row['mtime_ns'] = file.stat().st_mtime_ns
        self.assertTrue(tool.observe([row], self.work)[0]['matches_manifest'])

    def test_observation_detects_changes_and_missing(self):
        """Detect changed and absent tiny files; inputs fixture rows, output assertions, effects fixture bytes only; choose stat validation. Byline: Codex 2026-10-06."""
        backup = self.work / 'Backup'
        backup.mkdir()
        file = backup / 'example.txt'
        file.write_bytes(b'abc')
        row = {**self.row, 'mtime_ns': file.stat().st_mtime_ns}
        self.assertTrue(tool.observe([row], self.work)[0]['matches_manifest'])
        file.write_bytes(b'changed')
        self.assertFalse(tool.observe([row], self.work)[0]['matches_manifest'])
        self.assertIn('error', tool.observe([self.row], self.work / 'Backup')[0])

    def test_reparse_rejected(self):
        """Reject a reparse ancestor through mocked stat; input synthetic attrs, assertions, no link creation; choose Windows link safety. Byline: Codex 2026-10-06."""
        class Reparse:
            """Represent a Windows reparse stat; no inputs, synthetic attrs, no effects; choose fixture. Byline: Codex 2026-10-06."""
            st_mode = 0
            st_file_attributes = 0x400
        with patch.object(Path, 'lstat', return_value=Reparse()):
            self.assertIn('reparse', tool.observe([self.row], self.work)[0]['error'])

    def test_rounded_timestamp_precision_and_exact_native_stat(self):
        """Accept rounded microsecond observations but keep native ns exact; input tiny NTFS fixture, assertions, retained file; choose timestamp regression. Byline: Codex 2026-10-06."""
        backup = self.work / 'Backup'
        backup.mkdir()
        file = backup / 'example.txt'
        file.write_bytes(b'abc')
        native_ns = 1746197412000000700
        os.utime(file, ns=(native_ns, native_ns))
        self.assertEqual(file.stat().st_mtime_ns, native_ns)
        row = {'rel_path': self.row['relative_path'], 'size_bytes': 3, 'source_root': 'D:/',
               'source_mtime': '2025-05-02T14:50:12.000001+00:00'}
        path = self.work / 'rounded.jsonl'
        raw = (json.dumps(row) + '\n').encode()
        path.write_bytes(raw)
        normalized = tool.load_manifest(path, hashlib.sha256(raw).hexdigest())[0]
        observation = tool.observe([normalized], self.work)[0]
        self.assertTrue(observation['matches_manifest'])
        self.assertEqual(observation['mtime_delta_ns'], -300)
        self.assertEqual(observation['mtime_precision_ns'], 1000)
        beyond = {**normalized, 'mtime_ns': normalized['mtime_ns'] + 1000}
        self.assertFalse(tool.observe([beyond], self.work)[0]['matches_manifest'])
        exact_path, digest = self.manifest([{**self.row, 'mtime_ns': native_ns + 100,
                                             'mtime_precision_ns': 1000000000}])
        exact = tool.load_manifest(exact_path, digest)[0]
        self.assertEqual(exact['mtime_precision_ns'], 1)
        self.assertFalse(tool.observe([exact], self.work)[0]['matches_manifest'])
        self.assertTrue(tool.observe([{**exact, 'mtime_ns': native_ns}], self.work)[0]['matches_manifest'])

    def test_command_is_bounded_and_dry(self):
        """Assert fixed transport flags and no hash command; fixture list, assertions, no effects; choose argv contract. Byline: Codex 2026-10-06."""
        args = tool.command(self.work / 'input.files', True)
        self.assertEqual(args[:4], [tool.RCLONE, 'copy', 'D:/', tool.DEST])
        for flag in ('--immutable', '--metadata', '--fast-list', '--files-from-raw', '--size-only', '--b2-disable-checksum', '--dry-run'):
            self.assertIn(flag, args)
        self.assertNotIn('--no-traverse', args)
        self.assertEqual(args[args.index('--files-from-raw') + 1], str(self.work / 'input.files'))
        self.assertNotIn('--dry-run', tool.command(self.work / 'input.files', False))
        self.assertEqual(args[args.index('--transfers') + 1], '16')
        self.assertEqual(args[args.index('--checkers') + 1], '8')
        self.assertEqual(args[args.index('--buffer-size') + 1], '1MiB')
        self.assertEqual(args[args.index('--b2-chunk-size') + 1], '16MiB')
        self.assertEqual(args[args.index('--b2-upload-concurrency') + 1], '1')
        for value in (0, -1, 17, 100, True, '16'):
            with self.assertRaises(ValueError):
                tool.command(self.work/'input.files', False, value)
        reduced = tool.command(self.work/'input.files', False, 8)
        self.assertEqual(reduced[reduced.index('--transfers') + 1], '8')

    def test_changed_upload_writes_failure_receipt(self):
        """Retain explicit failed observation before any copy; synthetic stat input, assertions, receipt writes; choose source-change guard. Byline: Codex 2026-10-06."""
        path, digest = self.manifest([self.row])
        observations = [{**self.row, 'matches_manifest': False}]
        with patch.object(tool, 'observe', return_value=observations), patch.object(tool, 'run_process') as process:
            with self.assertRaisesRegex(RuntimeError, 'stat mismatch'):
                tool.stage('upload', {'work': str(self.work), 'manifest': str(path), 'sha256': digest}, threading.Event(), lambda: None)
            process.assert_not_called()
        receipt = json.loads(next(self.work.glob('*-receipt.json')).read_text())
        self.assertEqual(receipt['status'], 'failed')
        self.assertFalse(receipt['byte_verified'])

    def test_literal_encoding_is_explicit_and_preserves_default_argv(self):
        """Bind both encoders only for the explicit literal mode; synthetic argv, assertions, no effects; choose namespace guard regression. Byline: Codex 2026-10-07."""
        legacy = tool.command(self.work/'input.files', True)
        self.assertNotIn('--local-encoding', legacy)
        self.assertNotIn('--b2-encoding', legacy)
        literal = tool.command(self.work/'input.files', True, literal_path_identity=True)
        self.assertEqual(literal[:-5], legacy[:-1])
        self.assertEqual(literal[-5:], ['--local-encoding', 'Slash,Del,Ctl,Dot',
                                      '--b2-encoding', 'Slash,Del,Ctl,Dot', '--dry-run'])
        for invalid in (None, 1, 'Raw', 'false'):
            with self.assertRaisesRegex(ValueError, 'Boolean'):
                tool.backend_encoding_flags(invalid)

    @unittest.skipUnless(os.name == 'nt' and Path(tool.RCLONE).is_file(), 'native rclone fixture proof')
    def test_installed_rclone_literal_unicode_dryrun_accounts_every_path(self):
        """Reproduce Unicode omissions and prove exact literal outcomes with retained E fixtures; input synthetic names, assertions, dry-run only; choose installed-backend regression. Byline: Codex 2026-10-07."""
        source = self.work/'source'
        source.mkdir()
        names = ['literal\uff1acolon.txt', 'newline\u240a.txt', 'end\u2420',
                 'quote\uff02.txt', 'slash\uff0f.txt', 'escape\u201b\u240a.txt',
                 'del\u2421.txt', 'zero\u2400.txt', 'ordinary.txt']
        rows = []
        for name in names:
            file = source/name
            with file.open('xb') as output:
                output.write(b'tiny synthetic data')
            rows.append({'relative_path': name, 'size': file.stat().st_size})
        files = self.work/'literal.files'
        with files.open('x', encoding='utf-8', newline='\n') as output:
            output.write(''.join(name+'\n' for name in names))
        for literal_mode in (False, True):
            args = tool.command(files, True, literal_path_identity=literal_mode)
            args[2:4] = [str(source), str(self.work/('destination-'+str(literal_mode)))]
            log = self.work/('proof-'+str(literal_mode)+'.jsonl')
            with log.open('xb') as stderr, (self.work/('stdout-'+str(literal_mode))).open('xb') as stdout:
                result = subprocess.run(args, stdout=stdout, stderr=stderr, timeout=60, check=False)
            self.assertEqual(result.returncode, 0)
            counts = tool.log_counts(log, rows)
            self.assertEqual(counts['error_log_count'], 0)
            if literal_mode:
                self.assertEqual(counts['would_copy_count'], len(names))
                self.assertEqual(counts['unclassified_manifest_count'], 0)
            else:
                self.assertGreater(counts['unclassified_manifest_count'], 0)

    def test_dryrun_literal_mode_binding_and_api_namespace(self):
        """Reject namespace changes after review and bind API listing to both encoders; synthetic transport, assertions, retained receipts; choose dry/upload parity regression. Byline: Codex 2026-10-07."""
        path, digest = self.manifest([self.row])
        config = {'work': str(self.work), 'manifest': str(path), 'sha256': digest,
                  'literal_path_identity': True}
        observations = [{**self.row, 'matches_manifest': True}]
        calls = []
        def fake(args, output, errors, cancel, heartbeat):
            """Create only tiny fake transport outputs; input argv, output success, retained fixtures; choose receipt binding test. Byline: Codex 2026-10-07."""
            calls.append(args)
            if args[1] == 'lsjson':
                output.write_text(json.dumps([{'Path': self.row['relative_path'], 'Size': 3, 'Hashes': {}}]))
            else:
                errors.write_text(json.dumps({'object': self.row['relative_path'], 'skipped': 'copy',
                                             'msg': 'Skipped copy as --dry-run is set (size 3)', 'level': 'notice'})+'\n')
            return 0
        with patch.object(tool, 'observe', return_value=observations), patch.object(tool, 'run_process', side_effect=fake):
            dry = tool.stage('dry-run', config, threading.Event(), lambda: None)
            approved = {**config, 'dry_receipt': dry['receipt'], 'reviewed_dry_run_sha256': dry['sha256']}
            with self.assertRaisesRegex(RuntimeError, 'scope differs: literal_path_identity'):
                tool.stage('upload', {**approved, 'literal_path_identity': False}, threading.Event(), lambda: None)
            self.assertEqual(len(calls), 1)
            result = tool.stage('upload', approved, threading.Event(), lambda: None)
        self.assertTrue(json.loads(Path(dry['receipt']).read_text())['literal_path_identity'])
        receipt = json.loads(Path(result['receipt']).read_text())
        self.assertEqual(receipt['backend_encoding'], tool.LITERAL_ENCODING)
        self.assertFalse(receipt['byte_verified'])
        for args in calls:
            for flag in ('--local-encoding', '--b2-encoding'):
                self.assertEqual(args[args.index(flag)+1], tool.LITERAL_ENCODING)

    def test_dry_run_receipt_and_upload_gate(self):
        """Require independently supplied dry receipt hash before upload; synthetic observations, assertions, receipts only; choose review boundary. Byline: Codex 2026-10-06."""
        path, digest = self.manifest([self.row])
        config = {'work': str(self.work), 'manifest': str(path), 'sha256': digest}
        observations = [{**self.row, 'matches_manifest': True}]
        def fake_process(args, output, errors, cancel, heartbeat):
            """Return synthetic API observations; input argv, output zero, effects tiny fixture listing; choose mocked transport. Byline: Codex 2026-10-06."""
            if args[1] == 'lsjson':
                output.write_text(json.dumps([{'Path': self.row['relative_path'], 'Size': 3, 'Hashes': {'SHA-1': 'fixture'}}]))
            else:
                errors.write_text(json.dumps({'object': self.row['relative_path'], 'skipped': 'copy',
                                             'msg': 'Skipped copy as --dry-run is set (size 3)', 'level': 'notice'})+'\n')
            return 0
        with patch.object(tool, 'observe', return_value=observations), patch.object(tool, 'run_process', side_effect=fake_process) as process:
            dry = tool.stage('dry-run', config, threading.Event(), lambda: None)
            self.assertIn('--dry-run', process.call_args.args[0])
            with self.assertRaisesRegex(RuntimeError, 'hash mismatch'):
                tool.stage('upload', {**config, 'dry_receipt': dry['receipt'], 'reviewed_dry_run_sha256': '0'*64}, threading.Event(), lambda: None)
            self.assertEqual(process.call_count, 1)
            result = tool.stage('upload', {**config, 'dry_receipt': dry['receipt'], 'reviewed_dry_run_sha256': dry['sha256']}, threading.Event(), lambda: None)
        receipt = json.loads(Path(result['receipt']).read_text())
        self.assertEqual(receipt['server_verification'], 'pending')
        self.assertFalse(receipt['byte_verified'])
        self.assertEqual(receipt['api_exit_code'], 0)

    def test_json_outcome_counts_are_observed_not_exit_inferences(self):
        """Count dry-run notices and explicit skips without interpreting totals as byte proof; input JSON events, assertions, fixture log; choose review evidence. Byline: Codex 2026-10-06."""
        rows = [self.row, {**self.row, 'relative_path': 'Backup/existing'},
                {**self.row, 'relative_path': 'Backup/unclassified'}]
        log = self.work/'rclone.jsonl'
        events = [{'object': self.row['relative_path'], 'skipped': 'copy', 'size': 3,
                   'msg': 'Skipped copy as --dry-run is set (size 3)', 'level': 'notice'},
                  {'object': 'Backup/existing', 'msg': 'Unchanged skipping', 'level': 'debug'},
                  {'object': 'outside-manifest', 'msg': 'Copied (new)', 'level': 'info'}]
        log.write_text(''.join(json.dumps(event)+'\n' for event in events))
        counts = tool.log_counts(log, rows)
        self.assertEqual(counts['would_copy_count'], 1)
        self.assertEqual(counts['existing_unchanged_log_count'], 1)
        self.assertEqual(counts['unclassified_manifest_count'], 1)
        self.assertEqual(counts['copied_log_count'], 0)

    def test_cancellation_terminates_owned_child(self):
        """Terminate only a mocked owned subprocess on cancellation; synthetic event, assertions, retained empty logs; choose lifecycle safety. Byline: Codex 2026-10-06."""
        cancel = threading.Event()
        cancel.set()
        with patch.object(subprocess, 'Popen') as create, patch.object(subprocess, 'run') as tree_stop:
            child = create.return_value
            child.pid = 12345
            child.poll.return_value = None
            with self.assertRaises(InterruptedError):
                tool.run_process(['fixture'], self.work/'out', self.work/'err', cancel, lambda: None)
            child.terminate.assert_called_once()
            child.wait.assert_called_once_with(timeout=15)
            child.kill.assert_not_called()
            tree_stop.assert_called_once()
            self.assertEqual(tree_stop.call_args.args[0], ['taskkill.exe', '/PID', '12345', '/T', '/F'])

    def test_temporal_definitions_register_offline(self):
        """Validate SDK decorators without connecting or starting work; input installed SDK, assertions, no network effects; choose registration smoke test. Byline: Codex 2026-10-06."""
        try:
            from temporalio.client import Client
        except ImportError:
            self.skipTest('SDK optional for stdlib tests; run with parent uv runtime')
        with patch.object(Client, 'connect', side_effect=RuntimeError('offline registration reached connect')):
            with self.assertRaisesRegex(RuntimeError, 'offline registration reached connect'):
                asyncio.run(tool.start_worker({}, 'fixture', 'unused'))

    def test_predecessor_gate_accepts_only_fixed_success(self):
        """Block follow-on transfers after failed primary and accept its exact completed run; mocked statuses, assertions, no network; choose serialization guard. Byline: Codex 2026-10-06."""
        try:
            from temporalio.client import Client
            from temporalio import activity
        except ImportError:
            self.skipTest('requires parent cached SDK')
        for status in ('COMPLETED', 'FAILED', 'CANCELED', 'TERMINATED', 'TIMED_OUT'):
            client = MagicMock()
            handle = client.get_workflow_handle.return_value
            handle.describe = AsyncMock(return_value=SimpleNamespace(status=SimpleNamespace(name=status)))
            with patch.object(Client, 'connect', new=AsyncMock(return_value=client)), patch.object(activity, 'heartbeat'):
                if status == 'COMPLETED':
                    result = asyncio.run(tool.primary_completion_gate())
                    self.assertEqual(result['run_id'], tool.PRIMARY_RUN)
                else:
                    with self.assertRaisesRegex(RuntimeError, status):
                        asyncio.run(tool.primary_completion_gate())
            client.get_workflow_handle.assert_called_once_with(tool.PRIMARY_ID, run_id=tool.PRIMARY_RUN)


if __name__ == '__main__':
    unittest.main()
