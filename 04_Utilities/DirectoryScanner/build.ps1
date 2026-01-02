#!/usr/bin/env pwsh
# Directory Scanner Pro - Complete Build & Launch Script

param(
    [switch]$NoRun = $false
)

Write-Host "+--------------------------------------------------------------+"
Write-Host "|    Directory Scanner Pro - Build & Deploy System          |"
Write-Host "+--------------------------------------------------------------+`n"

# Set Go path
$goPath = "C:\Program Files\Go\bin\go.exe"
$wailsPath = "$env:USERPROFILE\go\bin\wails.exe"

# Step 1: Verify Go Installation
Write-Host "[1/6] Checking Go installation..." -ForegroundColor Cyan
if (Test-Path $goPath) {
    $goVersion = & $goPath version
    Write-Host "V Go found: $goVersion" -ForegroundColor Green
} else {
    Write-Host "X Go not found at $goPath" -ForegroundColor Red
    Write-Host "Attempting alternative paths..." -ForegroundColor Yellow
    
    $altPaths = @(
        "C:\Go\bin\go.exe",
        "$env:USERPROFILE\go\bin\go.exe",
        "$(where.exe go)"
    )
    
    $found = $false
    foreach ($path in $altPaths) {
        if (Test-Path $path) {
            $goPath = $path
            Write-Host "V Go found at: $goPath" -ForegroundColor Green
            $found = $true
            break
        }
    }
    
    if (-not $found) {
        Write-Host "X Go installation not found anywhere on system" -ForegroundColor Red
        Write-Host "Please install Go from: https://golang.org/dl/" -ForegroundColor Yellow
        Write-Host "Then run this script again." -ForegroundColor Yellow
        exit 1
    }
}

# Step 2: Check Wails
Write-Host "`n[2/6] Checking Wails CLI..." -ForegroundColor Cyan
if (-not (Test-Path $wailsPath)) {
    Write-Host "Installing Wails CLI..." -ForegroundColor Yellow
    & $goPath install github.com/wailsapp/wails/v2/cmd/wails@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "X Failed to install Wails" -ForegroundColor Red
        exit 1
    }
}

if (Test-Path $wailsPath) {
    Write-Host "V Wails CLI ready" -ForegroundColor Green
} else {
    Write-Host "X Wails installation failed" -ForegroundColor Red
    exit 1
}

# Step 3: Download Dependencies
Write-Host "`n[3/6] Downloading Go dependencies..." -ForegroundColor Cyan
Push-Location "C:\Users\matts\DirectoryScanner"
& $goPath mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "X Failed to download dependencies" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "V Dependencies downloaded" -ForegroundColor Green

# Step 4: Tidy Go modules
Write-Host "`n[4/6] Tidying Go modules..." -ForegroundColor Cyan
& $goPath mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "X Failed to tidy modules" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "V Modules tidied" -ForegroundColor Green

# Step 5: Build with Wails
Write-Host "`n[5/6] Building application with Wails..." -ForegroundColor Cyan
Write-Host "This may take 2-5 minutes on first build..." -ForegroundColor Yellow

if (-not (Test-Path "build\bin")) {
    New-Item -ItemType Directory -Path "build\bin" -Force | Out-Null
}

& $wailsPath build -platform windows/amd64 -ldflags "-H windowsgui -s -w" -o build/bin/DirectoryScanner.exe

if ($LASTEXITCODE -ne 0) {
    Write-Host "X Build failed" -ForegroundColor Red
    Pop-Location
    exit 1
}

Write-Host "V Build successful!" -ForegroundColor Green

# Step 6: Verify executable
Write-Host "`n[6/6] Verifying executable..." -ForegroundColor Cyan
if (Test-Path "build\bin\DirectoryScanner.exe") {
    $size = (Get-Item "build\bin\DirectoryScanner.exe").Length
    $sizeMB = [math]::Round($size / 1MB, 2)
    Write-Host "V Executable created: build\bin\DirectoryScanner.exe (${sizeMB}MB)" -ForegroundColor Green
} else {
    Write-Host "X Executable not found" -ForegroundColor Red
    Pop-Location
    exit 1
}

Pop-Location

# Summary
Write-Host "`n+--------------------------------------------------------------+"
Write-Host "|                    BUILD COMPLETE! V                       |"
Write-Host "+--------------------------------------------------------------+`n"

Write-Host "Application Location:"
Write-Host "   C:\Users\matts\DirectoryScanner\build\bin\DirectoryScanner.exe`n"

Write-Host "To launch:"
Write-Host "   C:\Users\matts\DirectoryScanner\build\bin\DirectoryScanner.exe`n"

