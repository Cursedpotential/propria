# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Directory Scanner Pro** is a high-performance desktop application built with Go and Wails v2 for scanning and analyzing directory structures on Windows. It features cloud drive detection, multi-threaded scanning, and multiple export formats.

## Build Commands

### Development
```bash
# Run in development mode with hot reload
wails dev

# Download and tidy dependencies
go mod download
go mod tidy
```

### Production Build
```bash
# Build Windows executable using the build script
build.bat

# Or build directly with Wails
wails build -platform windows/amd64 -ldflags "-H windowsgui -s -w" -o DirectoryScanner.exe
```

The built executable will be in `build\bin\DirectoryScanner.exe`.

## Architecture

### Core Components

**Backend (Go)**
- `main.go` - Entry point, embeds frontend assets
- `app.go` - Main App struct, session management, settings, and Wails bindings
- `scanner.go` - Multi-threaded directory scanning engine with progress reporting
- `exporter.go` - Export functionality (JSON, CSV, Excel, HTML, PDF)
- `cloud_detector.go` - Windows-specific cloud drive detection via registry
- `wails_app.go` - Wails runtime initialization and logger setup

**Frontend (Vanilla JS/HTML)**
- `frontend/index.html` - Single-page UI with scan controls, progress display, and settings modal
- `frontend/wailsjs/` - Auto-generated Wails bindings for Go-to-JS communication

### Key Design Patterns

**Concurrency Model**
- Scanner uses worker pool pattern with configurable thread count
- Atomic counters (`atomic.Int64`) for thread-safe statistics
- Context-based cancellation for scan interruption
- Directory scanning happens recursively with goroutines spawned per directory

**Session Management**
- Single `ScanSession` tracks active scan state
- Session contains root `FileNode` with lazy-loaded children
- Progress callback function for real-time UI updates
- Mutex-protected session access for concurrent reads

**Memory Optimization**
- Lazy loading of directory children to avoid loading entire tree in memory
- `ChildrenLoaded` flag indicates if children have been scanned
- `ChildCount` pre-calculated for directories without loading children
- File hashing only for files < 100MB

**Cloud Drive Detection**
- Windows registry-based detection for OneDrive, Google Drive, Dropbox, iCloud, Box
- Supports multiple accounts per service (especially OneDrive/Google Drive)
- Registry keys queried: `HKCU\Software\Microsoft\OneDrive\Accounts`, `HKCU\Software\Google\DriveFS`, etc.

### Data Structures

**FileNode** - Core tree node representing files/directories
- Contains metadata: size, timestamps, attributes, permissions
- Cloud detection flags: `IsCloud`, `CloudType`
- Lazy loading: `ChildrenLoaded`, `Children` array
- Windows-specific: `Attributes` bitmask for hidden/system/cloud flags

**ScanOptions** - Configuration for scan behavior
- Depth limits, filters (extensions, exclude patterns)
- Thread count, hash calculation toggle
- Hidden/system file inclusion flags

**ScanProgress** - Real-time scan statistics
- Files/dirs scanned, total size, current path
- Speed calculation (files/second)
- Session ID for tracking

## Development Patterns

### Adding New Export Formats
1. Implement exporter function in `exporter.go` with signature `func exportXXX(root *FileNode, options ExportOptions) error`
2. Add case to `ExportToFormat` switch statement
3. Update `BrowseForExport` in `app.go` with file filter

### Extending Cloud Drive Support
1. Add detection function in `cloud_detector.go` following pattern `detectXXX() []CloudDrive`
2. Query relevant Windows registry keys or check filesystem paths
3. Call from `detectCloudDrives()`

### Adding Frontend Features
1. Add UI controls to `frontend/index.html`
2. Create Go method in `App` struct (auto-exposed to frontend via Wails)
3. Call from JavaScript using `window.go.main.App.MethodName()`
4. Use async/await for all Go calls

### Windows-Specific Considerations
- File attributes accessed via `golang.org/x/sys/windows` for cloud detection
- Cloud file mask: `0x00040000` indicates cloud-synced file
- Registry access requires `golang.org/x/sys/windows/registry` package
- Network drive detection checks UNC paths (`\\`) or drive letters > E

## Testing

No formal test suite currently exists. Manual testing workflow:
1. Run `wails dev` to start dev server
2. Select various directories (local, cloud, network)
3. Test scan with different options (depth limits, filters)
4. Verify export formats generate correctly
5. Check logs via "📋 Logs" button (logs saved on shutdown)

## File Structure Notes

- Frontend is embedded at compile time via `//go:embed all:frontend`
- Wails auto-generates TypeScript bindings in `frontend/wailsjs/`
- Build artifacts go to `build/bin/`
- Settings stored in `%USERPROFILE%\.directoryscanner\settings.json`
- Runtime logs written to `log-YYYY-MM-DD-HH-MM-SS.txt` on shutdown
