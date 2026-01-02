import json
from pathlib import Path
from typing import Dict

SCHEMAS_DIR = Path(__file__).parent

def load_schema(name: str) -> Dict:
    schema_path = SCHEMAS_DIR / f"{name}.json"
    if schema_path.exists():
        with open(schema_path, 'r') as f:
            return json.load(f)
    return {}

def save_schema(name: str, schema: Dict):
    schema_path = SCHEMAS_DIR / f"{name}.json"
    with open(schema_path, 'w') as f:
        json.dump(schema, f, indent=2)

def list_schemas() -> list:
    return [f.stem for f in SCHEMAS_DIR.glob("*.json")]
