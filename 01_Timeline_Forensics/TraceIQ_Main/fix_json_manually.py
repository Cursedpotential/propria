#!/usr/bin/env python3
"""
Manually fix the malformed JSON by reading line by line
"""
from pathlib import Path

def fix_json():
    cache_path = Path(__file__).parent.parent / "import" / "raw_api_responses" / "place_id_db.json"
    
    if not cache_path.exists():
        print(f"Cache file not found: {cache_path}")
        return
    
    print(f"Reading {cache_path}")
    with open(cache_path, 'r') as f:
        lines = f.readlines()
    
    # Fix line 902 (index 901)
    if len(lines) > 901:
        print(f"Original line 902: {lines[901].strip()}")
        lines[901] = '        "address": "11110 N Genesee Rd, Clio, MI 48420, USA",\n'
        print(f"Fixed line 902: {lines[901].strip()}")
    
    # Write back
    with open(cache_path, 'w') as f:
        f.writelines(lines)
    
    print("File fixed!")
    
    # Validate by trying to parse
    import json
    try:
        with open(cache_path, 'r') as f:
            data = json.load(f)
        print(f"Successfully validated JSON with {len(data)} entries")
    except json.JSONDecodeError as e:
        print(f"JSON still invalid: {e}")

if __name__ == '__main__':
    fix_json()
