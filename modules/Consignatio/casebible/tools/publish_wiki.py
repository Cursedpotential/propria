#!/usr/bin/env python3
"""Publish a bounded documentation bundle into the existing Case Vault wiki and verify it.

Inputs: server-local ZIP, staging directory, and remote wiki prefix.
Outputs: JSON receipt containing per-file SHA-256 readback and publication status.
Side effects: adds documentation objects and extends the exact expected wiki INDEX.
Choose for source-document publication, never for corpus relocation or bulk indexing.
Run on the VPS; transport of these generated small source artifacts is separate.
Byline: Codex · GPT-6 · 2026-10-04.
"""
import argparse
import hashlib
import json
import subprocess
import zipfile
from datetime import datetime, timezone
from pathlib import Path


def rclone(*arguments: str) -> bytes:
    """Run one bounded rclone operation and return bytes or fail with its exit code.

    Inputs: fixed command arguments. Output: captured stdout.
    Side effects: those of the selected rclone operation; credentials stay server-local.
    Choose for publish/readback rather than constructing shell command strings.
    """
    result = subprocess.run(['rclone', *arguments], capture_output=True, timeout=300)
    if result.returncode:
        raise RuntimeError(f'rclone {arguments[0]} failed with exit {result.returncode}')
    return result.stdout


def main() -> None:
    """Validate, add, extend INDEX and independently read every published document.

    Inputs: ZIP, staging folder, remote prefix flags. Output: publication receipt.
    Side effects: source-document uploads only; no deletes, moves or index refresh.
    Use as the server publication entry point after local documentation verification.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--zip', required=True, type=Path)
    parser.add_argument('--staging', required=True, type=Path)
    parser.add_argument('--remote', default='b2native-full:salem-data/consignatio/casevault/Code/wiki')
    args = parser.parse_args()
    if args.remote != 'b2native-full:salem-data/consignatio/casevault/Code/wiki':
        raise ValueError('This publication is restricted to the existing Code/wiki prefix.')
    staging = args.staging.resolve(); staging.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(args.zip) as archive:
        for member in archive.infolist():
            destination = (staging / member.filename).resolve()
            if not destination.is_relative_to(staging):
                raise ValueError('Bundle member escaped the staging directory.')
            if member.is_dir():
                destination.mkdir(parents=True, exist_ok=True)
            else:
                destination.parent.mkdir(parents=True, exist_ok=True)
                destination.write_bytes(archive.read(member))
    expected_index = (staging / 'assets/prior-index-sha256.txt').read_text(encoding='ascii').strip()
    observed_index = hashlib.sha256(rclone('cat', args.remote + '/INDEX.md')).hexdigest()
    if observed_index != expected_index:
        raise RuntimeError('Wiki INDEX changed since the reviewed source; publication stopped before writes.')
    manifest = []
    for path in sorted(staging.rglob('*')):
        if path.is_file():
            manifest.append({'path': path.relative_to(staging).as_posix(), 'bytes': path.stat().st_size,
                             'sha256': hashlib.sha256(path.read_bytes()).hexdigest()})
    rclone('copy', str(staging), args.remote, '--immutable', '--exclude', '/INDEX.md', '--transfers', '2', '--checkers', '4')
    # Re-check immediately before the one intended overwrite of the existing scaffold.
    if hashlib.sha256(rclone('cat', args.remote + '/INDEX.md')).hexdigest() != expected_index:
        raise RuntimeError('Wiki INDEX changed during publication; extension was not applied.')
    rclone('copyto', str(staging / 'INDEX.md'), args.remote + '/INDEX.md')
    verified = []
    for entry in manifest:
        actual = hashlib.sha256(rclone('cat', args.remote + '/' + entry['path'])).hexdigest()
        if actual != entry['sha256']:
            raise RuntimeError(f"Remote hash mismatch: {entry['path']}")
        verified.append({**entry, 'readback_sha256': actual, 'status': 'match'})
    print(json.dumps({'byline': 'Codex / GPT-6 / 2026-10-04', 'status': 'verified',
                      'verified_at': datetime.now(timezone.utc).isoformat(), 'host': 'ovh-files',
                      'remote': args.remote, 'prior_index_sha256': expected_index, 'files': verified}, indent=2))


if __name__ == '__main__':
    main()
