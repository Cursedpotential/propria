"""Preserve changed Markdown on its registered source mount without deleting bytes.

Inputs are registered source directories and validated projection destinations.
Outputs are preflighted quarantine paths and hash/locator receipts. Only preserve()
moves bytes; choose this shared helper for source sync and ADR materialization.
"""
from __future__ import annotations

import hashlib
from pathlib import Path
import re
import sys


def validate_destination(directory: Path, target: Path) -> None:
    """Reject unsafe paths and non-directory ancestors before creating source files.

    Inputs: the registered source directory and destination. Output: None.
    Effects: filesystem reads only; raises ValueError on unsafe paths. Use for
    incoming writes as well as quarantine paths, before any source mutation.
    """
    if not target.is_relative_to(directory) or target == directory:
        raise ValueError('Source destination escapes docs root')
    for path in (target, *target.parents):
        if path.is_symlink():
            raise ValueError('Source destination or quarantine ancestor is symlinked')
        if path != target and path.exists() and not path.is_dir():
            raise ValueError('Source destination or quarantine ancestor is not a directory')
        if path == directory:
            break
    if not target.resolve().is_relative_to(directory.resolve()):
        raise ValueError('Source destination escapes docs root')


def _existing_parent(path: Path) -> Path:
    """Find the existing ancestor used to check a not-yet-created path's device.

    Input: a destination or existing file. Output: its nearest existing path.
    Effects: filesystem reads only; choose when preflighting future directories.
    """
    while not path.exists():
        path = path.parent
    return path


def _mount_id(path: Path) -> str | None:
    """Identify a Linux mount, including same-device bind mounts, without writes.

    Input: an absolute path. Output: the longest matching mount's ID, or None
    outside Linux. Effects: reads mountinfo; failure on Linux is explicit. Use
    alongside st_dev because bind mounts can share a device but reject renames.
    """
    if sys.platform != 'linux':
        return None
    selected = None
    depth = -1
    for line in Path('/proc/self/mountinfo').read_text().splitlines():
        fields = line.split()
        mount = Path(re.sub(r'\\([0-7]{3})', lambda match: chr(int(match[1], 8)), fields[4]))
        if path.is_relative_to(mount) and len(mount.parts) > depth:
            selected, depth = fields[0], len(mount.parts)
    if selected is None:
        raise ValueError(f'Cannot identify source mount for {path}')
    return selected


def validate_same_mount(directory: Path, source: Path, backup: Path) -> None:
    """Refuse nested mounts or device differences before preserving source bytes.

    Inputs: source root, current source and proposed backup. Output: None.
    Effects: read-only checks; raises ValueError before rename. Pick this over
    copy/delete fallbacks, which cannot provide the required atomic preservation.
    """
    for path in (source, backup):
        validate_destination(directory, path)
    paths = [directory, _existing_parent(source), _existing_parent(backup)]
    if len({path.stat().st_dev for path in paths}) != 1:
        raise ValueError('Source quarantine crosses devices; source unchanged')
    if len({_mount_id(path.resolve()) for path in paths}) != 1:
        raise ValueError('Source quarantine crosses nested mounts; source unchanged')
    for path in (source, backup):
        for ancestor in (path, *path.parents):
            if ancestor == directory:
                break
            if ancestor.exists() and ancestor.is_mount():
                raise ValueError('Source quarantine crosses nested mounts; source unchanged')


def quarantine_path(directory: Path, source: Path, generation: str, project: str) -> Path:
    """Preflight a unique backup below the registered source root's quarantine.

    Inputs: source root/path, operation generation and approved project ID.
    Output: root/to_be_deleted/generation/project/relative-path. Effects: reads
    only; refuses existing backup paths. Use before the first write of a batch.
    """
    if not re.fullmatch(r'(?:sync|adr)-[0-9a-f]{32}', generation):
        raise ValueError('Invalid source quarantine generation')
    if not re.fullmatch(r'[a-z][a-z0-9-]*', project):
        raise ValueError('Invalid source quarantine project')
    validate_destination(directory, source)
    backup = directory / 'to_be_deleted' / generation / project / source.relative_to(directory)
    validate_same_mount(directory, source, backup)
    if backup.exists():
        raise ValueError('Source quarantine destination already exists')
    return backup


def preserve(directory: Path, source: Path, backup: Path, expected_hash: str | None = None) -> dict:
    """Atomically rename an original to its preflighted backup and return provenance.

    Inputs: source root, source, backup and optional expected SHA-256. Output:
    source/backup locators and original SHA-256. Effects: creates quarantine
    parents and renames once, never copies or deletes; rechecks safety immediately
    before mutation. Use after whole-batch preflight in source sync or ADR writes.
    """
    validate_same_mount(directory, source, backup)
    if backup.exists():
        raise ValueError('Source quarantine destination already exists')
    sha = hashlib.sha256(source.read_bytes()).hexdigest()
    if expected_hash is not None and sha != expected_hash:
        raise ValueError('Source changed before quarantine; re-plan before apply')
    backup.parent.mkdir(parents=True, exist_ok=True)
    validate_same_mount(directory, source, backup)
    source.replace(backup)
    return {'source_path': str(source), 'backup_path': str(backup), 'sha256': sha}
