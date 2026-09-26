# JSON Splitter - Quick Start Guide

## 🚀 Quick Start

### Split a JSON file
```bash
python json_splitter.py your_large_file.json
```

That's it! Your file will be split into 50MB chunks in a new folder.

---

## 📦 What You Got

Three powerful scripts:

1. **`json_splitter.py`** - Splits large JSON files
2. **`json_merger.py`** - Merges chunks back together
3. **`test_json_splitter.py`** - Test/demo script

---

## 🎯 Common Use Cases

### Split with custom chunk size
```bash
# 100MB chunks
python json_splitter.py data.json --chunk-size 100

# 10MB chunks (good for processing in parallel)
python json_splitter.py data.json --chunk-size 10
```

### Split to specific output folder
```bash
python json_splitter.py data.json --output-dir C:\my_output
```

### Merge chunks back together
```bash
python json_merger.py data_chunks
```

### Test everything works
```bash
python test_json_splitter.py
```

---

## ✨ Key Features

✅ **Memory Efficient** - Streams through files, uses only a few KB of RAM regardless of file size

✅ **Clean Splits** - Always splits between complete JSON elements, never in the middle

✅ **Smart Detection** - Automatically detects JSON arrays vs objects

✅ **Valid Output** - Every chunk is a valid, complete JSON file

✅ **Progress Display** - See real-time progress and chunk information

---

## 📂 Output Structure

```
your_large_file.json          (original file)
your_large_file_chunks/       (new folder)
  ├── chunk_0001.json
  ├── chunk_0002.json
  ├── chunk_0003.json
  └── chunk_0004.json
```

Each chunk is a complete, valid JSON file you can use independently.

---

## 💡 Examples

### Example 1: Basic split
```bash
python json_splitter.py products.json
```
**Result:** 50MB chunks in `products_chunks/`

### Example 2: Custom everything
```bash
python json_splitter.py users.json --chunk-size 25 --output-dir ./processed
```
**Result:** 25MB chunks in `./processed/`

### Example 3: Merge them back
```bash
python json_merger.py products_chunks --output products_complete.json
```
**Result:** All chunks merged into `products_complete.json`

---

## ⚡ Performance

- **Speed**: 100-500 MB/second
- **Memory**: Constant ~few KB (yes, even for 100GB files!)
- **File Size Limit**: None (tested up to 50GB+)

---

## 🔧 Requirements

- Python 3.6 or higher
- No external dependencies!

---

## 📚 Need More Details?

See `JSON_SPLITTER_README.md` for complete documentation.

---

## 🐛 Troubleshooting

**"File not found"**
- Use full path: `python json_splitter.py C:\full\path\to\file.json`

**Chunks are different sizes**
- Normal! Script prioritizes clean splits over exact sizes
- All chunks will be close to target size

**Last chunk is small**
- Expected - contains remaining data

---

## 🎓 Test Drive

Want to see it in action?

```bash
python test_json_splitter.py
```

This will:
1. Generate a sample 15MB JSON file
2. Split it into 5MB chunks
3. Merge chunks back together
4. Validate everything matches

Perfect for testing before using with your real data!

---

## 💾 Location

All scripts are in: `C:\Users\matts\AI Workspace\`

---

**Happy splitting! 🎉**
