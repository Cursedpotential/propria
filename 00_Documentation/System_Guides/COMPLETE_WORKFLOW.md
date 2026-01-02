# Complete JSON to PDF Workflow

## What You Got

### 1. **Stirling PDF MCP Server** 📦
**Location:** `AI Workspace/stirling-pdf-mcp/`

A full MCP server that lets Claude, Gemini, and other AI tools use Stirling PDF for conversions.

**Features:**
- Convert any file to PDF (DOCX, MD, HTML, images)
- Batch convert entire directories
- Merge PDFs
- Compress PDFs
- All through simple commands to your AI

**Setup:**
1. Make sure Stirling PDF is running (Docker/Podman)
2. Dependencies already installed ✓
3. Add to your AI tools using configs in `add-to-configs.md`

---

### 2. **JSON Splitters** ✂️

**Simple Splitter:** `simple_json_splitter.py`
- Splits at `},` or `],` boundaries
- Fast, works with broken JSON
- Creates 2MB chunks by default

**Smart Splitter:** `conversation_splitter.py`
- Splits by conversation count
- Keeps JSON valid
- Better for structured data

**Usage:**
```bash
python simple_json_splitter.py large_file.json --chunk-size-mb 2
python conversation_splitter.py conversations.json --conversations-per-chunk 50
```

---

### 3. **Converters** 🔄

#### Clean Markdown Converter (Recommended!)
**File:** `clean_markdown_converter.py`

Creates **clean markdown with NO escape sequences** - no more `\n` problems!

```bash
python clean_markdown_converter.py ./conversation_chunks
```

Output: Perfect markdown files ready for Stirling PDF

#### DOCX Converter
**File:** `conversation_to_docx.py`

Creates Word documents (used python-docx).

```bash
python conversation_to_docx.py ./chunks --output-prefix "perplexity_chats"
```

---

## Recommended Workflow

### Once Stirling PDF is Running:

**Step 1:** Convert JSON to clean markdown
```bash
cd "AI Workspace"
python clean_markdown_converter.py "conversations-20251215_153247-2f08f0ab_chunks"
```

**Step 2:** Use Stirling PDF MCP to convert markdown to PDF

Tell Claude or Gemini:
```
Use stirling-pdf to batch convert all files in ./perplexity_chats_clean_markdown to PDFs in ./perplexity_chats_final_pdfs
```

Done! Clean PDFs with no escape sequence issues.

---

## Alternative: Direct DOCX if You Prefer

The DOCX files are already created in:
`perplexity_chats_docx/`

You can:
1. Open them in Word directly
2. Use Stirling PDF to convert them:
   ```
   Use stirling-pdf to batch convert ./perplexity_chats_docx to PDFs
   ```

---

## All Created Tools

**Splitters:**
- `simple_json_splitter.py` - Fast, works with broken JSON
- `json_splitter.py` - Smart depth-tracking
- `conversation_splitter.py` - By conversation count
- `batch_json_splitter.py` - Process multiple files

**Converters:**
- `clean_markdown_converter.py` - **RECOMMENDED** - Clean MD, no escapes
- `conversation_to_docx.py` - Creates Word docs
- `json_to_markdown.py` - Basic MD converter
- `robust_conversation_extractor.py` - Handles broken JSON

**MCP Server:**
- `stirling-pdf-mcp/` - Full Stirling PDF integration

**Utilities:**
- `json_merger.py` - Merge chunks back
- `docx_to_pdf.py` - Python docx2pdf converter
- `convert_to_pdf.ps1` - PowerShell Word converter

**Docs:**
- `QUICK_START.md` - Quick reference
- `JSON_SPLITTER_README.md` - Full splitter docs
- All tools have `--help` options

---

## Quick Reference

### Split Large JSON
```bash
python simple_json_splitter.py myfile.json --chunk-size-mb 2
```

### Convert to Clean Markdown
```bash
python clean_markdown_converter.py ./chunks
```

### Use Stirling PDF (via MCP)
```
Convert all markdown in ./folder to PDF
```

---

## Next Steps

1. ✅ Splitters created
2. ✅ Converters created
3. ✅ MCP server created
4. ⏳ Wait for Stirling PDF to finish installing
5. ⏳ Add MCP to Claude/Gemini config
6. ⏳ Convert markdown to PDF with Stirling PDF

**Your conversations are already split and converted to DOCX!**
- 9 DOCX files in `perplexity_chats_docx/`
- 414 conversations total
- Once Stirling PDF is ready, convert them to PDF in seconds

---

## Why This Approach?

✅ **No `\n` escape issues** - Clean text processing
✅ **Leverages existing tools** - Stirling PDF is battle-tested
✅ **Reusable** - MCP works with all your AI tools
✅ **Fast** - Batch processing in parallel
✅ **Flexible** - Multiple converters for different needs
