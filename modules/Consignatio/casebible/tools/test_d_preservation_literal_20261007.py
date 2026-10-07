"""Test receipt-bound literal Unicode verification with retained synthetic cache files.

Inputs: synthetic upload references and tiny filenames, never D source payloads.
Outputs: assertions for mode/mask mismatches, reader parity and full cache hashes.
Effects: retained fixture metadata and tiny server-only rclone local copies.
Choose before whole-D literal-name verification; no B2 or catalog writes occur.
Byline: Codex / GPT-6.1-sol high / 2026-10-07; sibling: whole-D guard tests.
"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import unittest
from unittest.mock import patch

import d_preservation_verify_20261006 as tool
import test_d_preservation_verify_20261006 as fixtures


class LiteralVerificationTests(unittest.TestCase):
    """Check literal receipt and cache contracts using synthetic retained fixtures.

    Inputs: fixture metadata. Outputs: assertions. Effects: tiny retained files only.
    Choose for filename identity guards over inference from successful transport.
    Byline: Codex / GPT-6.1-sol high / 2026-10-07.
    """
    def setUp(self):
        """Initialize isolated metadata; no inputs, fixture output, retained writes; choose per-test setup. Byline: Codex 2026-10-07."""
        self.fixture = fixtures.PreservationVerificationTests()
        self.fixture.setUp()

    def test_literal_receipt_mode_and_mask_are_bound(self):
        """Reject missing mode, altered mask and mismatched reader; synthetic receipt input, assertions, retained files; choose upload binding regression. Byline: Codex 2026-10-07."""
        f = self.fixture
        f.config['literal_path_identity'] = True
        with self.assertRaisesRegex(ValueError, 'mode differs'):
            tool.validate_upload(f.config, f.rows)
        f.receipt.update(literal_path_identity=True, backend_encoding='Raw')
        f.bind_receipt()
        with self.assertRaisesRegex(ValueError, 'mask differs'):
            tool.validate_upload(f.config, f.rows)
        f.receipt['backend_encoding'] = 'Slash,Del,Ctl,Dot'
        f.bind_receipt()
        proof = tool.validate_upload(f.config, f.rows)
        self.assertTrue(proof['literal_path_identity'])
        with self.assertRaisesRegex(ValueError, 'mode differs'):
            tool.validate_upload({**f.config, 'literal_path_identity': False}, f.rows)
        for value in (1, 'Raw', None):
            with self.assertRaisesRegex(ValueError, 'Boolean'):
                tool.literal_backend_flags({'literal_path_identity': value})
        self.assertEqual(tool.literal_backend_flags({}), [])

    def test_listing_and_download_share_exact_flags(self):
        """Bind both live rclone readers to the receipt encoder; mocked readers input, assertions, retained metadata; choose listing/download parity regression. Byline: Codex 2026-10-07."""
        f = self.fixture
        config = {**f.config, 'literal_path_identity': True, 'activity_id': 'synthetic', 'attempt': 1}
        calls = []
        def fake_run(args, **kwargs):
            """Write a synthetic listing only; argv input, no output, retained JSON; choose reader command inspection. Byline: Codex 2026-10-07."""
            calls.append(args)
            if args[1] == 'lsjson':
                kwargs['stdout'].write(json.dumps([f.object]).encode())
        with patch.object(tool, 'require_server'), patch.object(tool.subprocess, 'run', side_effect=fake_run):
            tool.run_stage('listing_before', config)
            tool.run_stage('listing_after', config)
            tool.run_stage('download', config)
        self.assertEqual(len(calls), 3)
        for argv in calls:
            for flag in ('--local-encoding', '--b2-encoding'):
                self.assertEqual(argv[argv.index(flag)+1], 'Slash,Del,Ctl,Dot')

    @unittest.skipUnless(os.name != 'nt' and shutil.which('rclone'), 'server-only installed rclone cache proof')
    def test_linux_literal_cache_names_and_three_hashes(self):
        """Copy tiny literal Unicode fixtures and hash their intended cache paths on Linux.

        Inputs: synthetic full-width punctuation, control pictures and trailing-space
        symbols. Outputs: exact literal filenames and full SHA1/SHA256/MD5 assertions.
        Effects: retained server fixtures only; no D, B2 or catalog access.
        Choose to prove installed-rclone cache compatibility before real readback.
        Byline: Codex / GPT-6.1-sol high / 2026-10-07.
        """
        base = self.fixture.work
        source = base/'source'
        source.mkdir()
        cache = base/'literal-cache'
        names = ['Rclone/literal\uff1acolon.txt', 'Rclone/newline\u240a.txt',
                 'Rclone/end\u2420', 'Rclone/quote\uff02.txt', 'Rclone/slash\uff0f.txt',
                 'Rclone/escape\u201b\u240a.txt', 'Rclone/del\u2421.txt']
        data = b'synthetic cache readback only\x00\xff'
        for name in names:
            path = source/name
            path.parent.mkdir(exist_ok=True)
            with path.open('xb') as output:
                output.write(data)
        files = base/'literal.files'
        with files.open('x', encoding='utf-8', newline='\n') as output:
            output.write(''.join(name+'\n' for name in names))
        argv = ['rclone', 'copy', str(source), str(cache), '--files-from-raw', str(files),
                '--immutable', '--transfers', '2', *tool.literal_backend_flags({'literal_path_identity': True})]
        with (base/'fixture-copy.log').open('xb') as log:
            subprocess.run(argv, stdout=log, stderr=log, check=True, timeout=60)
        self.assertEqual({p.relative_to(cache).as_posix() for p in cache.rglob('*') if p.is_file()}, set(names))
        for name in names:
            row = {'relative_path': name, 'size': len(data)}
            result = tool.hash_file(cache, row)
            self.assertEqual(result, {'size': len(data), **{k: hashlib.new(k, data).hexdigest() for k in ('sha1', 'sha256', 'md5')}})


if __name__ == '__main__':
    unittest.main()
