# Conversion Options for Ongoing Use

## Best Lightweight Solutions (No Docker)

### Option 1: wkhtmltopdf + Pandoc ⭐ RECOMMENDED
**Already have pandoc, just need wkhtmltopdf**

```bash
# Install:
choco install wkhtmltopdf

# Use:
pandoc input.md -o output.pdf --pdf-engine=wkhtmltopdf
```

Or use the Python wrapper: `python simple_md_to_pdf.py ./markdown_folder`

**Pros:**
- Fast, lightweight
- Battle-tested
- Works offline
- No services needed

---

### Option 2: Stirling PDF MCP (Turn on when needed)
**Already configured at:** `stirling-pdf-mcp/`

```bash
# Start Stirling PDF:
docker start stirling-pdf  # or podman

# Use via Claude/Gemini:
"Use stirling-pdf to convert markdown to PDF"

# Stop when done:
docker stop stirling-pdf
```

**Pros:**
- Handles ANY format
- Professional quality
- Batch processing
- Turn on/off as needed

---

### Option 3: Use DOCX as Intermediate
**Current files:** `perplexity_chats_docx/`

DOCX → PDF options:
1. **Word:** File → Save As → PDF
2. **LibreOffice:** If installed
3. **Stirling PDF:** When running

**Pros:**
- DOCX already created
- No conversion needed for storage
- Convert to PDF only when ingesting

---

## For Ongoing Workflow

### Recommended Setup:
1. **Install wkhtmltopdf** (one-time, ~50MB)
2. **Keep Stirling PDF MCP** for complex jobs
3. **Store as DOCX** for archives

### Daily Use:
```bash
# Simple conversions:
python simple_md_to_pdf.py ./markdown

# Complex/batch jobs:
docker start stirling-pdf
# Use MCP via Claude/Gemini
docker stop stirling-pdf
```

---

## What You Have Now

✅ **Clean markdown converter** - No escape sequences
✅ **DOCX converter** - Professional formatting
✅ **Stirling PDF MCP** - Full-featured when needed
✅ **Simple pandoc wrapper** - Ready for wkhtmltopdf

All in: `C:\Users\matts\AI Workspace\`

---

## Next Steps for Ongoing Solution

1. **Install wkhtmltopdf:**
   ```
   choco install wkhtmltopdf
   ```

2. **Test:**
   ```
   python simple_md_to_pdf.py ./perplexity_chats_clean_markdown
   ```

3. **Done!** Lightweight, offline, always available.
