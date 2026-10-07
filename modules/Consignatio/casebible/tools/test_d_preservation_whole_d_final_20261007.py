"""Exercise pinned whole-D verification boundaries with retained synthetic fixtures.

Inputs: synthetic metadata only. Outputs: unittest assertions. Effects: retained
fixture files; no source-drive reads, transport, catalog writes or deletion.
Choose before positive-size reviewed non-Backup D verification.
Byline: Codex GPT-6 2026-10-07.
"""
import hashlib
import json
import unittest
import uuid

import d_preservation_verify_20261006 as tool
import test_d_preservation_verify_20261006 as fixtures


class WholeDTests(unittest.TestCase):
    """Check explicit D allowlist and root gates; fixtures in, assertions out, retained writes; choose boundary regression. Byline: Codex GPT-6 2026-10-07."""

    def setUp(self):
        """Prepare retained whole-D metadata; no inputs, fixture config output, file creation; choose independent tests. Byline: Codex GPT-6 2026-10-07."""
        self.fixture = fixtures.PreservationVerificationTests()
        self.fixture.setUp()
        self.config = dict(self.fixture.config, whole_d=True)
        self.row = json.loads(self.fixture.manifest.read_text())
        self.row['rel_path'] = 'Owner documents/evidence.db'
        self.bind()

    def bind(self, allowlist=None):
        """Pin unique synthetic manifest and allowlist bytes; optional lines input, config output, retained writes; choose guard variants. Byline: Codex GPT-6 2026-10-07."""
        manifest = self.fixture.work / (uuid.uuid4().hex + '.jsonl')
        raw = (json.dumps(self.row) + '\n').encode()
        manifest.write_bytes(raw)
        allowed = self.fixture.work / (uuid.uuid4().hex + '.files')
        lines = self.row['rel_path'] + '\n' if allowlist is None else allowlist
        allowed.write_bytes(lines.encode())
        self.config.update(manifest=str(manifest), manifest_sha256=hashlib.sha256(raw).hexdigest(),
            source_path_allowlist=str(allowed), source_path_allowlist_sha256=hashlib.sha256(lines.encode()).hexdigest())

    def test_explicit_positive_scope(self):
        """Accept pinned positive-size D evidence and leave history unassessed; config input, assertions output, no effects; choose positive guard. Byline: Codex GPT-6 2026-10-07."""
        self.assertEqual(tool.load_manifest(self.config)[0]['relative_path'], self.row['rel_path'])
        self.assertEqual(tool.historical_sources(tool.load_manifest(self.config), True), [])

    def test_unsafe_names(self):
        """Reject traversal, roots, aliases and control characters; paths input, assertions output, no effects; choose confinement guard. Byline: Codex GPT-6 2026-10-07."""
        for path in ('../secret', '/root/a', 'F:/case/a', 'D:/a', 'a/../b', 'a//b', 'a/CON.txt', 'a/trail.', 'a\nb', ''):
            with self.subTest(path=path), self.assertRaises(ValueError):
                tool.safe_relative(path, True)

    def test_allowlist_and_root_rejections(self):
        """Reject altered pins, extra paths, missing mode and F roots; fixture metadata input, assertions output, retained writes; choose authorization guard. Byline: Codex GPT-6 2026-10-07."""
        for changes in ({'whole_d': False}, {'source_path_allowlist_sha256': '0'*64}, {'source_path_allowlist': None}):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                tool.load_manifest(dict(self.config, **changes))
        self.bind(self.row['rel_path'] + '\nUnreviewed/file\n')
        with self.assertRaises(ValueError):
            tool.load_manifest(self.config)
        self.row['source_root'] = 'F:/'
        self.bind()
        with self.assertRaises(ValueError):
            tool.load_manifest(self.config)

    def test_no_empty_extension(self):
        """Reject size-zero objects in positive-size verification; fixture input, assertion output, retained write; choose scope guard. Byline: Codex GPT-6 2026-10-07."""
        self.row['size_bytes'] = 0
        self.bind()
        with self.assertRaises(ValueError):
            tool.load_manifest(self.config)

    def test_upload_scope_receipt(self):
        """Require matching native whole-D receipt pins; synthetic receipt input, assertions output, retained writes; choose scope-chain proof. Byline: Codex GPT-6 2026-10-07."""
        self.row['rel_path'] = self.fixture.relative
        self.bind()
        rows = tool.load_manifest(self.config)
        with self.assertRaises(ValueError):
            tool.validate_upload(self.config, rows)
        self.fixture.receipt.update(whole_d=True, empty_files_only=False,
            source_path_allowlist_sha256=self.config['source_path_allowlist_sha256'])
        self.fixture.bind_receipt()
        self.config.update(upload_receipt=self.fixture.config['upload_receipt'],
            upload_sha256=self.fixture.config['upload_sha256'])
        # The fixture manifest differs only in name; preserve the manifest pin in the receipt.
        self.fixture.receipt['manifest_sha256'] = self.config['manifest_sha256']
        self.fixture.bind_receipt()
        self.config.update(upload_receipt=self.fixture.config['upload_receipt'],
            upload_sha256=self.fixture.config['upload_sha256'])
        tool.validate_upload(self.config, rows)
        self.fixture.receipt['source_path_allowlist_sha256'] = '0'*64
        self.fixture.bind_receipt()
        self.config.update(upload_receipt=self.fixture.config['upload_receipt'],
            upload_sha256=self.fixture.config['upload_sha256'])
        with self.assertRaises(ValueError):
            tool.validate_upload(self.config, rows)


if __name__ == '__main__':
    unittest.main()
