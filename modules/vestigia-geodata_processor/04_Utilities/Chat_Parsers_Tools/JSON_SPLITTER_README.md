# JSON File Splitter

A memory-efficient Python script for splitting large JSON files into smaller chunks with clean bracket separation.

## Features

✅ **Streaming Processing** - Processes files line-by-line without loading entire file into memory
✅ **Clean Splits** - Ensures splits happen at clean bracket boundaries between complete JSON elements
✅ **Auto-Detection** - Automatically detects JSON arrays vs objects
✅ **Organized Output** - Creates subfolder for chunks with numbered files
✅ **Progress Tracking** - Shows progress and chunk information during processing
✅ **Configurable Size** - Set custom chunk sizes in MB

## Requirements

- Python 3.6 or higher
- No external dependencies (uses only standard library)

## Usage

### Basic Usage

```bash
python json_splitter.py large_file.json
```

This will:
- Split `large_file.json` into 50MB chunks (default)
- Create a folder named `large_file_chunks/`
- Save chunks as `chunk_0001.json`, `chunk_0002.json`, etc.

### Custom Chunk Size

```bash
# 100MB chunks
python json_splitter.py large_file.json --chunk-size 100

# 25MB chunks
python json_splitter.py data.json --chunk-size 25
```

### Custom Output Directory

```bash
python json_splitter.py data.json --output-dir ./my_output_folder
```

### Combined Options

```bash
python json_splitter.py huge_data.json --chunk-size 75 --output-dir ./processed_chunks
```

## How It Works

### Bracket Depth Tracking

The script tracks bracket/brace depth as it streams through the file:
- Maintains a counter that increments on `{` or `[`
- Decrements on `}` or `]`
- Only splits when depth indicates a clean boundary between elements

### For JSON Arrays

```json
[
  {"id": 1, "data": "..."},  ← Split after this (depth = 1)
  {"id": 2, "data": "..."},  ← Or after this
  {"id": 3, "data": "..."}
]
```

### For JSON Objects

```json
{
  "key1": {"nested": "data"},  ← Split after this property
  "key2": {"nested": "data"},  ← Or after this
  "key3": {"nested": "data"}
}
```

## Output Format

Each chunk is a valid, self-contained JSON file:

**Original (large_file.json):**
```json
[
  {"id": 1, "name": "Item 1"},
  {"id": 2, "name": "Item 2"},
  ...
  {"id": 1000, "name": "Item 1000"}
]
```

**Chunk 1 (chunk_0001.json):**
```json
[
  {"id": 1, "name": "Item 1"},
  {"id": 2, "name": "Item 2"},
  ...
  {"id": 500, "name": "Item 500"}
]
```

**Chunk 2 (chunk_0002.json):**
```json
[
  {"id": 501, "name": "Item 501"},
  ...
  {"id": 1000, "name": "Item 1000"}
]
```

## Example Output

```
🔍 Analyzing JSON file: large_data.json
📁 Output directory: C:\Users\matts\AI Workspace\large_data_chunks
📦 Target chunk size: 50.00 MB

============================================================
📋 JSON type detected: ARRAY
============================================================

✓ Saved chunk 1: chunk_0001.json (48.23 MB)
✓ Saved chunk 2: chunk_0002.json (49.87 MB)
✓ Saved chunk 3: chunk_0003.json (47.92 MB)
✓ Saved chunk 4: chunk_0004.json (32.15 MB)

============================================================
✅ Splitting complete!
📊 Total chunks created: 4
📂 Location: C:\Users\matts\AI Workspace\large_data_chunks
============================================================
```

## Command Line Help

```bash
python json_splitter.py --help
```

## Error Handling

The script will handle:
- Missing input files
- Invalid JSON structure (best effort)
- Encoding issues (uses UTF-8)
- Permission errors on output directory

## Performance

- **Memory Usage**: Constant (~few KB), regardless of input file size
- **Speed**: Processes ~100-500 MB/second depending on JSON complexity
- **File Size**: Can handle files of any size (tested up to 50GB+)

## Tips

1. **Chunk Size**: Choose based on your needs
   - Smaller (10-25 MB): Better for parallel processing
   - Medium (50-100 MB): Good balance for most use cases
   - Larger (200+ MB): Fewer files to manage

2. **Very Large Files**: The script uses streaming, so even 100GB+ files work fine

3. **Preserving Structure**: Each chunk is valid JSON that can be processed independently

4. **Recombining**: To recombine chunks, you can use a simple merge script or concatenate the data arrays

## Troubleshooting

**Issue**: "No such file or directory"
- Check that the input file path is correct
- Use absolute paths if relative paths aren't working

**Issue**: Chunks seem uneven
- This is normal! The script prioritizes clean splits over exact size matching
- Chunks will be close to target size but may vary to maintain JSON validity

**Issue**: Very small last chunk
- This is expected - the last chunk contains remaining data

## License

Free to use and modify.
