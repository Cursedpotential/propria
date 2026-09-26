#!/usr/bin/env python3
"""
Fix malformed JSON in place_id_db.json
"""
import json
import re
from pathlib import Path

def fix_place_id_cache():
    cache_path = Path(__file__).parent.parent / "import" / "raw_api_responses" / "place_id_db.json"
    
    if not cache_path.exists():
        print(f"Cache file not found: {cache_path}")
        return
    
    print(f"Reading {cache_path}")
    with open(cache_path, 'r') as f:
        content = f.read()
    
    # Fix the malformed entry at line 902
    # The issue is a broken address field
    fixed_content = re.sub(
        r'"11110 N Genesee Rd\, Clio\, MI 48420\s+"address":\s*",([^"]*)"',
        r'"address": "11110 N Genesee Rd, Clio, MI 48420\1"',
        content
    )
    
    # Try to parse and validate
    try:
        data = json.loads(fixed_content)
        print(f"Successfully parsed JSON with {len(data)} entries")
        
        # Write back the fixed content
        with open(cache_path, 'w') as f:
            json.dump(data, f, indent=4)
        
        print("Fixed and saved place_id_db.json")
        
    except json.JSONDecodeError as e:
        print(f"JSON still invalid: {e}")
        
        # Save original as backup
        backup_path = cache_path.with_suffix('.json.backup')
        with open(backup_path, 'w') as f:
            f.write(content)
        print(f"Saved backup to {backup_path}")

if __name__ == '__main__':
    fix_place_id_cache()
