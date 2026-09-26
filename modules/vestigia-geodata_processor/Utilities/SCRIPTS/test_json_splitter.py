"""
Test script for JSON splitter - creates sample data and demonstrates usage.

This script:
1. Generates a sample large JSON file
2. Splits it using json_splitter.py
3. Merges it back using json_merger.py
4. Validates the result
"""

import json
import os
import subprocess
import sys
from pathlib import Path


def generate_sample_json(output_file: str, num_records: int = 10000):
    """Generate a sample JSON file with many records."""
    print(f"📝 Generating sample JSON with {num_records} records...")

    data = []
    for i in range(num_records):
        record = {
            "id": i + 1,
            "name": f"Record {i + 1}",
            "description": f"This is a test record with ID {i + 1}. " * 10,
            "metadata": {
                "created": "2025-12-16",
                "type": "test",
                "tags": ["sample", "test", f"tag{i % 10}"],
                "score": (i * 7) % 100,
                "active": i % 2 == 0
            },
            "nested_data": {
                "level1": {
                    "level2": {
                        "level3": {
                            "value": f"Deep value {i}",
                            "numbers": list(range(i % 10))
                        }
                    }
                }
            }
        }
        data.append(record)

    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, indent=2)

    file_size = Path(output_file).stat().st_size
    print(f"✓ Created {output_file} ({file_size / (1024*1024):.2f} MB)")


def run_command(cmd: list, description: str):
    """Run a command and print its output."""
    print(f"\n{'='*60}")
    print(f"🚀 {description}")
    print(f"{'='*60}")
    print(f"Command: {' '.join(cmd)}\n")

    result = subprocess.run(cmd, capture_output=False, text=True)

    if result.returncode != 0:
        print(f"❌ Command failed with exit code {result.returncode}")
        return False

    return True


def validate_json(file1: str, file2: str):
    """Validate that two JSON files contain the same data."""
    print(f"\n🔍 Validating merged file matches original...")

    with open(file1, 'r', encoding='utf-8') as f1:
        data1 = json.load(f1)

    with open(file2, 'r', encoding='utf-8') as f2:
        data2 = json.load(f2)

    if data1 == data2:
        print("✅ Validation passed! Merged file matches original.")
        return True
    else:
        print("❌ Validation failed! Files don't match.")
        return False


def main():
    # Setup paths
    workspace = Path(__file__).parent
    sample_file = workspace / "sample_large.json"
    chunks_dir = workspace / "sample_large_chunks"
    merged_file = workspace / "sample_large_merged.json"

    print("\n" + "="*60)
    print("JSON Splitter/Merger Test Suite")
    print("="*60)

    # Step 1: Generate sample data
    if not sample_file.exists():
        generate_sample_json(str(sample_file), num_records=10000)
    else:
        print(f"📄 Using existing sample file: {sample_file.name}")

    # Step 2: Split the file
    if not chunks_dir.exists():
        success = run_command(
            [sys.executable, str(workspace / "json_splitter.py"),
             str(sample_file), "--chunk-size", "5"],
            "Splitting JSON file into 5MB chunks"
        )

        if not success:
            print("❌ Splitting failed!")
            return 1
    else:
        print(f"📁 Using existing chunks directory: {chunks_dir.name}")

    # Step 3: List chunks
    if chunks_dir.exists():
        chunks = sorted(chunks_dir.glob("chunk_*.json"))
        print(f"\n📊 Chunks created: {len(chunks)}")
        for chunk in chunks:
            size = chunk.stat().st_size / (1024 * 1024)
            print(f"   - {chunk.name}: {size:.2f} MB")

    # Step 4: Merge chunks back
    if not merged_file.exists():
        success = run_command(
            [sys.executable, str(workspace / "json_merger.py"),
             str(chunks_dir), "--output", str(merged_file)],
            "Merging chunks back into single file"
        )

        if not success:
            print("❌ Merging failed!")
            return 1
    else:
        print(f"📝 Using existing merged file: {merged_file.name}")

    # Step 5: Validate
    if validate_json(str(sample_file), str(merged_file)):
        print("\n" + "="*60)
        print("✅ ALL TESTS PASSED!")
        print("="*60)

        # Cleanup prompt
        print("\n💡 Test files created:")
        print(f"   - {sample_file.name} (original)")
        print(f"   - {chunks_dir.name}/ (chunks)")
        print(f"   - {merged_file.name} (merged)")
        print("\nTo clean up, run:")
        print(f"   rm {sample_file.name}")
        print(f"   rm -r {chunks_dir.name}")
        print(f"   rm {merged_file.name}")

        return 0
    else:
        return 1


if __name__ == '__main__':
    sys.exit(main())
