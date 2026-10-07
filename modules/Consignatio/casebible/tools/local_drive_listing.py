#!/usr/bin/env python3
"""Observe native drive metadata with retained completion and exclusion evidence.

Inputs: explicit root and new artifact directory; outputs JSONL and a pinned completion
receipt. Effects: derived artifact writes only; no source payload reads or source writes.
Choose through drive_inventory_refresh_20261006, replacing the unreceipted Oct 2 TSV
walker. Source: reviewed MAIN local_drive_listing.py, Claude Code / Opus 5.5, 2026-10-02.
The observation spans an interval and is not an atomic filesystem snapshot.
Byline: Codex / GPT-6 / 2026-10-06.
"""
import contextlib
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import socket
import stat

BYLINE = 'Codex GPT-6 2026-10-06'
SCOPE = 'whole_drive_metadata'
POLICY = {'version': 1, 'root_exclusions': ['$RECYCLE.BIN', 'System Volume Information'],
          'reparse_points': 'record_and_never_follow', 'other_exclusions': []}
POLICY_SHA256 = hashlib.sha256(json.dumps(POLICY, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def now():
    """Return UTC receipt time; no inputs, ISO output, no effects; choose observation boundaries. Byline: Codex 2026-10-06."""
    return dt.datetime.now(dt.timezone.utc).isoformat()


def save(path, value):
    """Retain JSON exclusively; inputs path/value, output SHA256, file creation only; choose immutable receipts. Byline: Codex 2026-10-06."""
    raw = json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2).encode('utf-8')
    with Path(path).open('xb') as output:
        output.write(raw)
        output.flush()
        os.fsync(output.fileno())
    return hashlib.sha256(raw).hexdigest()


def timestamp(ns):
    """Normalize metadata nanoseconds to PG microseconds; input integer, ISO output, no effects; choose exact load representation. Byline: Codex 2026-10-06."""
    instant = dt.datetime(1970, 1, 1, tzinfo=dt.timezone.utc) + dt.timedelta(microseconds=ns // 1000)
    return instant.isoformat(timespec='microseconds')


@contextlib.contextmanager
def directory_guard(path):
    """Hold a directory without following reparse points; input path, output context, metadata handles only; choose race-resistant native traversal. Byline: Codex 2026-10-06."""
    info = Path(path).lstat()
    if stat.S_ISLNK(info.st_mode) or getattr(info, 'st_file_attributes', 0) & 0x400:
        raise OSError('reparse directory rejected')
    if not stat.S_ISDIR(info.st_mode):
        raise OSError('not a directory')
    if os.name != 'nt':
        yield  # synthetic POSIX fixtures only; production wrapper requires Windows
        return
    import ctypes
    from ctypes import wintypes
    kernel = ctypes.WinDLL('kernel32', use_last_error=True)
    kernel.CreateFileW.argtypes = (wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD, wintypes.LPVOID,
                                  wintypes.DWORD, wintypes.DWORD, wintypes.HANDLE)
    kernel.CreateFileW.restype = wintypes.HANDLE
    kernel.CloseHandle.argtypes = (wintypes.HANDLE,)
    kernel.GetFileInformationByHandleEx.argtypes = (wintypes.HANDLE, ctypes.c_int, wintypes.LPVOID, wintypes.DWORD)
    kernel.GetFileInformationByHandleEx.restype = wintypes.BOOL
    class AttributeTag(ctypes.Structure):
        """Describe native handle attributes; input Win32 metadata, output fields, no effects; choose reparse validation. Byline: Codex 2026-10-06."""
        _fields_ = [('attributes', wintypes.DWORD), ('tag', wintypes.DWORD)]
    # No FILE_SHARE_DELETE: retain ancestor handles during DFS so directory paths
    # cannot be renamed/replaced between validation and scandir. READ_ATTRIBUTES only.
    handle = kernel.CreateFileW(str(Path(path).absolute()), 0x80, 0x3, None, 3, 0x02200000, None)
    if handle == wintypes.HANDLE(-1).value:
        raise ctypes.WinError(ctypes.get_last_error())
    try:
        tag = AttributeTag()
        if not kernel.GetFileInformationByHandleEx(handle, 9, ctypes.byref(tag), ctypes.sizeof(tag)):
            raise ctypes.WinError(ctypes.get_last_error())
        if tag.attributes & 0x400 or not tag.attributes & 0x10:
            raise OSError('reparse or non-directory handle rejected')
        yield
    finally:
        kernel.CloseHandle(handle)


def entries(root, counters, cancelled=lambda: False):
    """Enumerate metadata with explicit errors/skips and held ancestors; inputs root/counters/cancel, output events, no payload access; choose bounded DFS. Byline: Codex 2026-10-06."""
    stack = []
    def push(path, relative):
        """Open one guarded directory frame; inputs path/relative, output none, metadata handles; choose nonrecursive DFS. Byline: Codex 2026-10-06."""
        context = contextlib.ExitStack()
        try:
            context.enter_context(directory_guard(path))
            iterator = context.enter_context(os.scandir(path))
        except BaseException:
            context.close()
            raise
        stack.append((relative, iterator, context))
        counters['directories_opened'] += 1
    try:
        try:
            push(root, '')
        except OSError as error:
            yield dict(kind='error', rel_path='.', error='LISTDIR ' + str(error))
            return
        while stack:
            if cancelled():
                raise InterruptedError('metadata observation cancelled')
            relative, iterator, context = stack[-1]
            try:
                entry = next(iterator)
            except StopIteration:
                counters['directories_enumerated'] += 1
                context.close()
                stack.pop()
                continue
            except OSError as error:
                yield dict(kind='error', rel_path=relative or '.', error='LISTDIR ' + str(error))
                context.close()
                stack.pop()
                continue
            counters['entries_seen'] += 1
            path = (relative + '/' if relative else '') + entry.name
            if not relative and entry.name.casefold() in {name.casefold() for name in POLICY['root_exclusions']}:
                yield dict(kind='skip', rel_path=path, reason='declared_root_exclusion')
                continue
            try:
                info = entry.stat(follow_symlinks=False)
            except OSError as error:
                yield dict(kind='error', rel_path=path, error='STAT ' + str(error))
                continue
            attributes = getattr(info, 'st_file_attributes', 0)
            if stat.S_ISLNK(info.st_mode) or attributes & 0x400:
                yield dict(kind='skip', rel_path=path, reason='reparse_point', attributes=attributes,
                           reparse_tag=getattr(info, 'st_reparse_tag', None))
            elif stat.S_ISDIR(info.st_mode):
                try:
                    push(entry.path, path)
                except OSError as error:
                    yield dict(kind='error', rel_path=path, error='LISTDIR ' + str(error))
            elif stat.S_ISREG(info.st_mode):
                yield dict(kind='file', rel_path=path, size=info.st_size,
                           mtime=timestamp(info.st_mtime_ns), mtime_ns=info.st_mtime_ns)
            else:
                yield dict(kind='error', rel_path=path, error='STAT unsupported file type')
    finally:
        for _, _, context in reversed(stack):
            context.close()


def scan(root, work, generation_id, temporal=None, cancelled=lambda: False, pulse=lambda value: None):
    """Retain drive-relative file metadata and traversal errors by observation interval, with explicit exclusions and completion evidence.

    Inputs: source root, unique artifact directory, generation ID and Temporal context.
    Outputs: frozen JSONL, source snapshot and complete, empty, partial or failed receipt references.
    Effects: reads directory metadata without payload reads or source writes; retains derived artifacts outside the source.
    Choose for native metadata inventory; drive_inventory_refresh_20261006 tracks execution and the VPS loader admits receipts.
    Byline: Codex / GPT-6 / 2026-10-06.
    """
    root, work = Path(root), Path(work)
    if work.absolute().is_relative_to(root.absolute()):
        raise ValueError('artifacts must remain outside observed source root')
    work.mkdir(parents=True, exist_ok=False)
    source = Path(__file__).read_bytes()
    with (work / 'walker-source.py').open('xb') as output:
        output.write(source)
    started = now()
    counters = dict(file_count=0, file_bytes=0, error_count=0, skip_count=0,
                    directories_opened=0, directories_enumerated=0, entries_seen=0)
    digest, raw_bytes, finished, failure = hashlib.sha256(), 0, False, None
    listing = work / 'listing.jsonl'
    with listing.open('xb') as output:
        try:
            for event in entries(root, counters, cancelled):
                if event['kind'] == 'file':
                    counters['file_count'] += 1
                    counters['file_bytes'] += event['size']
                else:
                    counters[event['kind'] + '_count'] += 1
                raw = (json.dumps(event, ensure_ascii=False, sort_keys=True, separators=(',', ':')) + '\n').encode('utf-8')
                output.write(raw)
                digest.update(raw)
                raw_bytes += len(raw)
                if (counters['file_count'] + counters['error_count'] + counters['skip_count']) % 1000 == 0:
                    pulse(dict(counters))
            finished = True
        except BaseException as error:
            failure = error
        finally:
            output.flush()
            os.fsync(output.fileno())
    status = ('empty' if counters['file_count'] == 0 else 'complete') if finished and not counters['error_count'] else (
        'partial' if counters['file_count'] or counters['directories_enumerated'] else 'failed')
    receipt = dict(schema_version=1, drive='D:', root='D:/' if str(root).replace('\\', '/').rstrip('/') == 'D:' else str(root),
                   generation_id=generation_id, scope=SCOPE, policy=POLICY, policy_sha256=POLICY_SHA256,
                   started_at=started, ended_at=now(), listed_at=started, enumeration_finished=finished,
                   status=status, raw_sha256=digest.hexdigest(), raw_bytes=raw_bytes,
                   object_count=counters['file_count'] + counters['error_count'], object_bytes=counters['file_bytes'],
                   counts=counters, walker_sha256=hashlib.sha256(source).hexdigest(), execution_host=socket.gethostname(),
                   platform=os.name, temporal=temporal, interruption=str(failure) if failure else None,
                   limits='Metadata observation interval; not an atomic snapshot or source payload hash.', byline=BYLINE)
    receipt_path = work / 'listing-receipt.json'
    receipt_sha = save(receipt_path, receipt)
    result = dict(listing=str(listing), receipt=str(receipt_path), receipt_sha256=receipt_sha,
                  walker_source=str(work / 'walker-source.py'), generation_id=generation_id, status=status)
    if failure:
        raise RuntimeError('interrupted observation retained: ' + str(receipt_path)) from failure
    return result


if __name__ == '__main__':
    raise SystemExit('Use drive_inventory_refresh_20261006.py scan for a separately tracked native observation.')
