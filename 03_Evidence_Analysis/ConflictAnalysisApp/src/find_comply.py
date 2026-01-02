#!/usr/bin/env python3
"""
Quick search script to find "comply" and related coercive control statements.
Run this against your SMS Backup & Restore XML files.

Usage:
    python find_comply.py <xml_file>
    python find_comply.py <directory_of_xml_files>
"""
import sys
from pathlib import Path
from sms_backup_parser import search_for_pattern, stream_sms_records, stream_call_log


# Critical patterns to find
COERCIVE_PATTERNS = [
    r"comply",
    r"if you want (to see|us|her)",
    r"add me on facebook",
    r"tag (me|you)",
    r"you('ll| will) do (what|as)",
    r"obey",
    r"my way or",
    r"end of discussion",
]


def find_critical_evidence(xml_path: Path):
    """Search for critical coercive control evidence."""
    print(f"\n{'='*60}")
    print(f"Searching: {xml_path.name}")
    print('='*60)
    
    for pattern in COERCIVE_PATTERNS:
        matches = list(search_for_pattern(xml_path, pattern))
        if matches:
            print(f"\n🎯 FOUND '{pattern}' - {len(matches)} matches:")
            for m in matches:
                dt = m['datetime'].strftime('%Y-%m-%d %H:%M:%S') if m['datetime'] else 'Unknown'
                print(f"\n  [{dt}] {m['direction']} from {m.get('sender', 'Unknown')}")
                print(f"  MESSAGE: {m['message']}")
                print(f"  Source: {m['source']}")


def find_blocking_timeline(xml_path: Path, target_name: str = None):
    """Find blocking evidence timeline."""
    print(f"\n{'='*60}")
    print(f"Call Blocking Evidence: {xml_path.name}")
    print('='*60)
    
    blocked_calls = []
    for rec in stream_call_log(xml_path):
        if rec['is_blocked_indicator']:
            if target_name is None or target_name.lower() in rec.get('contact_name', '').lower():
                blocked_calls.append(rec)
    
    if blocked_calls:
        print(f"\n🚫 Found {len(blocked_calls)} blocking indicators:")
        for call in sorted(blocked_calls, key=lambda x: x['datetime'] or ''):
            dt = call['datetime'].strftime('%Y-%m-%d %H:%M:%S') if call['datetime'] else 'Unknown'
            print(f"  [{dt}] {call['contact_name']} ({call['phone']})")
            print(f"    Type: {call['call_type_name']} | Duration: {call['duration']}s")
            print(f"    Evidence: {', '.join(call['block_evidence'])}")
    else:
        print("  No blocking indicators found")


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    
    path = Path(sys.argv[1])
    
    if path.is_dir():
        xml_files = list(path.glob("*.xml"))
        if not xml_files:
            print(f"No XML files found in {path}")
            sys.exit(1)
        for xml_file in xml_files:
            find_critical_evidence(xml_file)
            find_blocking_timeline(xml_file)
    elif path.is_file():
        find_critical_evidence(path)
        find_blocking_timeline(path)
    else:
        print(f"Path not found: {path}")
        sys.exit(1)


if __name__ == "__main__":
    main()
