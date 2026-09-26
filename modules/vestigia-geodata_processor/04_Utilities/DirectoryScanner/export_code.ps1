# export-codebase.ps1
param(
    [string]$RootPath = (Get-Location),
    [string]$OutputFile = "codebase.txt"
)

$Root = (Resolve-Path $RootPath).ProviderPath
$Output = Join-Path $Root $OutputFile

# Remove old output if it exists
if (Test-Path $Output) {
    Remove-Item $Output -Force
}

# File patterns to include
$includePatterns = @(
    "*.go",
    "go.mod", "go.sum",
    "*.html", "*.htm",
    "*.gohtml", "*.tmpl", "*.tpl",
    "*.css",
    "*.js",
    "*.ts", "*.tsx",
    "*.json",
    "*.yaml", "*.yml",
    "*.env",
    "Dockerfile",
    "Makefile",
    "*.md"
)

# Directories to exclude anywhere in the path
$excludeDirs = @(
    "\.git\",
    "\vendor\",
    "\node_modules\",
    "\.idea\",
    "\.vscode\",
    "\dist\",
    "\build\"
)

# Get all candidate files recursively matching patterns
$files = Get-ChildItem -Path $Root -Recurse -File -Include $includePatterns |
    Where-Object {
        $full = $_.FullName
        foreach ($ex in $excludeDirs) {
            if ($full -like "*$ex*") { return $false }
        }
        return $true
    } |
    Sort-Object FullName  # deterministic order
# [web:72][web:78][web:80]

foreach ($file in $files) {
    # Compute a repo-relative path with forward slashes for sanity
    $relativePath = $file.FullName.Substring($Root.Length).TrimStart('\','/')
    $relativePath = $relativePath -replace '\\','/'

    Add-Content -Path $Output -Value ""
    Add-Content -Path $Output -Value "===== $relativePath ====="
    Add-Content -Path $Output -Value ""

    # -Raw to avoid breaking long lines; -Encoding utf8 for consistency
    Get-Content -Path $file.FullName -Raw -ErrorAction SilentlyContinue |
        Out-File -FilePath $Output -Append -Encoding utf8
}
# [web:73][web:79][web:82]
