# Manual Installation Instructions

## wkhtmltopdf

### Option 1: Download from GitHub
https://github.com/wkhtmltopdf/wkhtmltopdf/releases

Download: `wkhtmltox-0.12.6-1.msvc2015-win64.exe`

Install to default location, then add to PATH:
```
C:\Program Files\wkhtmltopdf\bin
```

### Option 2: Direct Download
https://wkhtmltopdf.org/downloads.html

---

## Ghostscript

### Option 1: Download from Official Site
https://ghostscript.com/releases/gsdnld.html

Download: `gs10064.exe` (64-bit)

### Option 2: GitHub Releases
https://github.com/ArtifexSoftware/ghostpdl-downloads/releases

### Option 3: .NET Version (Ghostscript.NET)
NuGet Package: https://www.nuget.org/packages/Ghostscript.NET/

Create a .NET project first:
```bash
mkdir ghostscript-tool
cd ghostscript-tool
dotnet new console
dotnet add package Ghostscript.NET
```

---

## Markdown Libraries Installed

### Node.js Libraries ✅
- **markdown-it** - Fast, spec-compliant
- **showdown** - For JavaScript/Java apps
- All markdown-it extensions

### Python Libraries ✅
- **markdown** - Standard Python markdown

---

## After Installation

### Verify wkhtmltopdf:
```bash
wkhtmltopdf --version
```

### Verify Ghostscript:
```bash
gswin64c --version  # 64-bit
# OR
gswin32c --version  # 32-bit
```

### Test Conversion:
```bash
cd "AI Workspace"
python simple_md_to_pdf.py ./perplexity_chats_clean_markdown
```

---

## Quick Setup (All-in-One)

**Run PowerShell as Administrator:**

```powershell
# Install both tools
choco install wkhtmltopdf ghostscript -y

# Verify
wkhtmltopdf --version
gswin64c --version

# Test
cd "C:\Users\matts\AI Workspace"
python simple_md_to_pdf.py ./perplexity_chats_clean_markdown
```

Done! You'll have lightweight PDF conversion ready.
