#!/usr/bin/env python3
"""
SMS Backup & Restore XML Preprocessor for Autopsy

This script cleans large SMS/MMS backup XML files by:
1. Removing base64-encoded image data from MMS parts
2. Preserving all text content for keyword searching
3. Outputting a smaller, cleaner XML that Autopsy can process without false positives

The output file will be 90% smaller and won't trigger false matches on base64 content.

Usage:
    python preprocess_for_autopsy.py input.xml output.xml

For massive files (10GB+), run this on a VPS:
    scp preprocess_for_autopsy.py root@vps:/root/
    scp sms-backup.xml root@vps:/root/
    ssh root@vps "python3 preprocess_for_autopsy.py sms-backup.xml cleaned.xml"
    scp root@vps:/root/cleaned.xml ./
"""
import sys
import re
from pathlib import Path


def is_base64(s: str) -> bool:
    """Check if string looks like base64-encoded binary data."""
    if not s or len(s) < 50:
        return False
    # Base64 uses only these chars plus optional padding
    b64_chars = set('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=\n\r')
    ratio = sum(1 for c in s if c in b64_chars) / len(s)
    return ratio > 0.95


def process_line(line: str) -> str:
    """Process a single line, removing base64 data attributes."""
    # Pattern: data="BASE64_CONTENT"
    # We want to replace the base64 content but keep the attribute
    
    # Check if this line has a data attribute with potential base64
    if 'data="' not in line and "data='" not in line:
        return line
    
    # Handle data="..." 
    pattern = r'data="([^"]*)"'
    matches = re.findall(pattern, line)
    for match in matches:
        if is_base64(match):
            # Replace with placeholder
            line = line.replace(f'data="{match}"', 'data="[BASE64_REMOVED]"')
    
    # Handle data='...'
    pattern = r"data='([^']*)'"
    matches = re.findall(pattern, line)
    for match in matches:
        if is_base64(match):
            line = line.replace(f"data='{match}'", "data='[BASE64_REMOVED]'")
    
    return line


def stream_process(input_path: Path, output_path: Path) -> dict:
    """
    Stream process the XML file line by line.
    This keeps memory usage low for massive files.
    """
    stats = {
        "lines_processed": 0,
        "base64_removed": 0,
        "input_size": input_path.stat().st_size,
        "output_size": 0
    }
    
    with open(input_path, 'r', encoding='utf-8', errors='ignore') as infile:
        with open(output_path, 'w', encoding='utf-8') as outfile:
            for line in infile:
                stats["lines_processed"] += 1
                
                original_len = len(line)
                processed = process_line(line)
                
                if len(processed) < original_len - 100:  # Significant reduction
                    stats["base64_removed"] += 1
                
                outfile.write(processed)
                stats["output_size"] += len(processed)
                
                # Progress indicator every 100k lines
                if stats["lines_processed"] % 100000 == 0:
                    print(f"  Processed {stats['lines_processed']:,} lines...", file=sys.stderr)
    
    return stats


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        print("\nUsage: python preprocess_for_autopsy.py <input.xml> <output.xml>")
        sys.exit(1)
    
    input_path = Path(sys.argv[1])
    output_path = Path(sys.argv[2])
    
    if not input_path.exists():
        print(f"Error: {input_path} not found")
        sys.exit(1)
    
    print(f"Processing: {input_path}")
    print(f"Input size: {input_path.stat().st_size / (1024*1024):.1f} MB")
    
    stats = stream_process(input_path, output_path)
    
    print(f"\nResults:")
    print(f"  Lines processed: {stats['lines_processed']:,}")
    print(f"  Base64 blocks removed: {stats['base64_removed']:,}")
    print(f"  Input size: {stats['input_size'] / (1024*1024):.1f} MB")
    print(f"  Output size: {stats['output_size'] / (1024*1024):.1f} MB")
    print(f"  Reduction: {(1 - stats['output_size']/stats['input_size'])*100:.1f}%")
    print(f"\nOutput: {output_path}")
    print("Ready for Autopsy import!")


if __name__ == "__main__":
    main()
