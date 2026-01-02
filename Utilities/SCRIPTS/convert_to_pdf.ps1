$ErrorActionPreference = 'Stop'

$docxDir = Join-Path $PSScriptRoot "perplexity_chats_docx"
$pdfDir = Join-Path $PSScriptRoot "perplexity_chats_pdf"

New-Item -ItemType Directory -Force -Path $pdfDir | Out-Null

Write-Host ""
Write-Host "Converting Word documents to PDF" -ForegroundColor Cyan
Write-Host "Input: $docxDir" -ForegroundColor Gray
Write-Host "Output: $pdfDir" -ForegroundColor Gray
Write-Host ""
Write-Host "============================================================"
Write-Host ""

$word = New-Object -ComObject Word.Application
$word.Visible = $false

$count = 0
$files = Get-ChildItem -Path $docxDir -Filter *.docx | Where-Object { -not $_.Name.StartsWith('~$') }

foreach ($file in $files) {
    Write-Host "Converting $($file.Name)..." -NoNewline

    try {
        $docxPath = $file.FullName
        $pdfPath = Join-Path $pdfDir "$($file.BaseName).pdf"

        $doc = $word.Documents.Open($docxPath)
        $doc.SaveAs2($pdfPath, 17)
        $doc.Close()

        $pdfSize = (Get-Item $pdfPath).Length / 1KB
        Write-Host " OK ($([math]::Round($pdfSize, 1)) KB)" -ForegroundColor Green
        $count++
    }
    catch {
        Write-Host " ERROR: $_" -ForegroundColor Red
    }
}

$word.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($word) | Out-Null

Write-Host ""
Write-Host "============================================================"
Write-Host "Conversion complete! Created $count PDF files" -ForegroundColor Green
Write-Host "Location: $pdfDir"
Write-Host "============================================================"
Write-Host ""
