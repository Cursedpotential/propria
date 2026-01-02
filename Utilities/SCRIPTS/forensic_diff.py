import sys
import os
import json
import difflib
import yaml
import argparse
from datetime import datetime

# Try to import DeepDiff, fall back to custom implementation if missing
try:
    from deepdiff import DeepDiff
    HAS_DEEPDIFF = True
except ImportError:
    HAS_DEEPDIFF = False

def log(msg):
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    # print(f"[{timestamp}] {msg}", file=sys.stderr)

def load_file(filepath):
    """Smart loader that handles JSON, YAML, and Text."""
    if not os.path.exists(filepath):
        return None, "missing"
    
    ext = os.path.splitext(filepath)[1].lower()
    
    try:
        with open(filepath, 'r', encoding='utf-8', errors='replace') as f:
            content = f.read()
            
        if ext in ['.json']:
            return json.loads(content), 'json'
        elif ext in ['.yml', '.yaml']:
            return yaml.safe_load(content), 'yaml'
        else:
            return content, 'text'
    except Exception as e:
        return None, str(e)

def custom_deep_diff(d1, d2, path=""):
    """
    A lightweight recursive diff for dicts/lists if DeepDiff isn't installed.
    Returns a list of changes.
    """
    changes = []
    
    if isinstance(d1, dict) and isinstance(d2, dict):
        all_keys = set(d1.keys()) | set(d2.keys())
        for k in all_keys:
            new_path = f"{path}.{k}" if path else k
            if k not in d1:
                changes.append({"type": "added", "path": new_path, "value": d2[k]})
            elif k not in d2:
                changes.append({"type": "removed", "path": new_path, "old_value": d1[k]})
            else:
                changes.extend(custom_deep_diff(d1[k], d2[k], new_path))
                
    elif isinstance(d1, list) and isinstance(d2, list):
        # Naive list comparison (by index) - adequate for configs, less so for ordered data
        len1, len2 = len(d1), len(d2)
        for i in range(max(len1, len2)):
            new_path = f"{path}[{i}]"
            if i >= len1:
                changes.append({"type": "added", "path": new_path, "value": d2[i]})
            elif i >= len2:
                changes.append({"type": "removed", "path": new_path, "old_value": d1[i]})
            else:
                changes.extend(custom_deep_diff(d1[i], d2[i], new_path))
                
    elif d1 != d2:
        changes.append({
            "type": "changed", 
            "path": path, 
            "old_value": d1, 
            "new_value": d2
        })
        
    return changes

def diff_text(t1, t2):
    """
    Semantic text diff using difflib SequenceMatcher.
    Returns a summary of added/removed lines and a similarity ratio.
    """
    d = difflib.SequenceMatcher(None, t1, t2)
    similarity = d.ratio()
    
    # Generate delta
    delta = list(difflib.unified_diff(
        t1.splitlines(), 
        t2.splitlines(), 
        lineterm=''
    ))
    
    # Analyze delta
    added = len([line for line in delta if line.startswith('+') and not line.startswith('+++')])
    removed = len([line for line in delta if line.startswith('-') and not line.startswith('---')])
    
    return {
        "similarity_score": round(similarity, 4),
        "lines_added": added,
        "lines_removed": removed,
        "diff_preview": delta[:20]  # First 20 lines of diff
    }

def main():
    parser = argparse.ArgumentParser(description="Forensic File Differentiator")
    parser.add_argument("file1", help="Path to base file")
    parser.add_argument("file2", help="Path to new file")
    parser.add_argument("--format", choices=['json', 'text'], default='json', help="Output format")
    
    args = parser.parse_args()
    
    data1, type1 = load_file(args.file1)
    data2, type2 = load_file(args.file2)
    
    if data1 is None or data2 is None:
        print(json.dumps({"error": "Failed to load files"}))
        return

    result = {
        "file1": args.file1,
        "file2": args.file2,
        "type": type1 if type1 == type2 else "mixed",
        "timestamp": datetime.now().isoformat()
    }

    # STRUCTURED DATA DIFF
    if type1 in ['json', 'yaml'] and type2 in ['json', 'yaml']:
        if HAS_DEEPDIFF:
            # Use the real deal if available
            dd = DeepDiff(data1, data2, ignore_order=True)
            result["diff"] = json.loads(dd.to_json())
            result["method"] = "DeepDiff"
        else:
            # Fallback
            result["diff"] = custom_deep_diff(data1, data2)
            result["method"] = "Native_Recursive_Walk"
            
    # TEXT/CODE DIFF
    else:
        # Ensure strings
        t1 = data1 if isinstance(data1, str) else json.dumps(data1)
        t2 = data2 if isinstance(data2, str) else json.dumps(data2)
        
        result["diff"] = diff_text(t1, t2)
        result["method"] = "difflib.SequenceMatcher"

    # Output
    print(json.dumps(result, indent=2))

if __name__ == "__main__":
    main()
