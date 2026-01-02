#!/usr/bin/env python3
"""
Find 100% duplicate files by hashing.
Excludes node_modules and vendor directories.
"""
import os
import hashlib
from pathlib import Path
from collections import defaultdict
import json

def hash_file(filepath):
    """Generate SHA256 hash of file."""
    try:
        hasher = hashlib.sha256()
        with open(filepath, 'rb') as f:
            for chunk in iter(lambda: f.read(65536), b''):
                hasher.update(chunk)
        return hasher.hexdigest()
    except Exception as e:
        print(f"Error hashing {filepath}: {e}")
        return None

def find_duplicates(root_dir):
    """Find duplicate files by hash."""
    hash_to_files = defaultdict(list)
    excluded_dirs = {'node_modules', 'vendor', '.git', '__pycache__', 'dist', 'build'}

    print(f"Scanning {root_dir}...")
    file_count = 0

    for dirpath, dirnames, filenames in os.walk(root_dir):
        # Skip excluded directories
        dirnames[:] = [d for d in dirnames if d not in excluded_dirs]

        for filename in filenames:
            if filename == 'find_duplicates.py':
                continue

            filepath = os.path.join(dirpath, filename)
            file_count += 1
            if file_count % 100 == 0:
                print(f"Processed {file_count} files...")

            file_hash = hash_file(filepath)
            if file_hash:
                hash_to_files[file_hash].append(filepath)

    # Filter to only duplicates
    duplicates = {h: files for h, files in hash_to_files.items() if len(files) > 1}

    return duplicates

def main():
    root = Path(__file__).parent
    print(f"Finding duplicates in: {root}")

    duplicates = find_duplicates(root)

    print(f"\n{'='*60}")
    print(f"FOUND {len(duplicates)} SETS OF DUPLICATES")
    print(f"{'='*60}\n")

    duplicate_report = []

    for i, (file_hash, files) in enumerate(duplicates.items(), 1):
        size = os.path.getsize(files[0])
        print(f"\n{i}. Duplicate Set (Hash: {file_hash[:16]}...)")
        print(f"   Size: {size:,} bytes")
        print(f"   Count: {len(files)} files")

        for filepath in sorted(files):
            rel_path = os.path.relpath(filepath, root)
            print(f"   - {rel_path}")

        duplicate_report.append({
            'hash': file_hash,
            'size': size,
            'count': len(files),
            'files': [os.path.relpath(f, root) for f in sorted(files)]
        })

    # Save report
    report_path = root / 'DUPLICATE_REPORT.json'
    with open(report_path, 'w') as f:
        json.dump(duplicate_report, f, indent=2)

    print(f"\n{'='*60}")
    print(f"Report saved to: {report_path}")
    print(f"{'='*60}")

if __name__ == '__main__':
    main()
