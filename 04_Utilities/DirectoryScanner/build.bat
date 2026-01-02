@echo off
REM Directory Scanner Build Script for Windows

setlocal enabledelayedexpansion

echo ============================================
echo Directory Scanner Pro - Build System
echo ============================================
echo.

go version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

wails version >nul 2>&1
if errorlevel 1 (
    echo WARNING: Wails CLI not found. Installing...
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if errorlevel 1 (
        echo ERROR: Failed to install Wails
        pause
        exit /b 1
    )
)

echo [1/4] Downloading dependencies...
call go mod download
if errorlevel 1 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)

echo [2/4] Tidying go.mod...
call go mod tidy
if errorlevel 1 (
    echo ERROR: Failed to tidy modules
    pause
    exit /b 1
)

echo [3/4] Cleaning previous build...
if exist build\bin rmdir /s /q build\bin

echo [4/4] Building application...
call wails build -platform windows/amd64 ^
  -ldflags "-H windowsgui -s -w" ^
  -o DirectoryScanner.exe

if errorlevel 1 (
    echo ERROR: Build failed
    pause
    exit /b 1
)

echo.
echo ============================================
echo Build Complete!
echo ============================================
echo Executable: build\DirectoryScanner.exe
echo.

for %%F in (build\DirectoryScanner.exe) do (
    set size=%%~zF
    echo File Size: !size! bytes
)

echo.
echo Next steps:
echo  1. Run: build\DirectoryScanner.exe
echo  2. Or move to any location and run directly
echo  3. No installation required (portable)
echo.

pause
