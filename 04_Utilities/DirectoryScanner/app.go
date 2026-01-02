package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// =====================
// Core data structures
// =====================

// FileNode represents a file or directory in the scanned tree.
type FileNode struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	FormattedSize string    `json:"formattedSize"`
	Created       time.Time `json:"created"`
	Modified      time.Time `json:"modified"`
	Accessed      time.Time `json:"accessed"`
	IsDir         bool      `json:"isDir"`
	IsSymlink     bool      `json:"isSymlink"`
	IsHidden      bool      `json:"isHidden"`
	IsSystem      bool      `json:"isSystem"`
	IsReadOnly    bool      `json:"isReadOnly"`
	IsCloud       bool      `json:"isCloud"`
	CloudType     string    `json:"cloudType,omitempty"`
	Extension     string    `json:"extension"`
	MimeType      string    `json:"mimeType"`
	Permissions   string    `json:"permissions"`
	Owner         string    `json:"owner"`
	Hash          string    `json:"hash,omitempty"`
	Attributes    uint32    `json:"attributes"`
	ChildCount    int       `json:"childCount"`

	ChildrenLoaded bool        `json:"childrenLoaded"`
	Children       []*FileNode `json:"children,omitempty"`

	ParentID string `json:"parentId,omitempty"`
}

// ScanOptions configures how a scan should behave.
type ScanOptions struct {
	Path            string   `json:"path"`
	MaxDepth        int      `json:"maxDepth"`
	IncludeHidden   bool     `json:"includeHidden"`
	IncludeSystem   bool     `json:"includeSystem"`
	FollowSymlinks  bool     `json:"followSymlinks"`
	MinSize         int64    `json:"minSize"`
	MaxSize         int64    `json:"maxSize"`
	Extensions      []string `json:"extensions"`
	ExcludeDirs     []string `json:"excludeDirs"`
	ExcludePatterns []string `json:"excludePatterns"`
	ThreadCount     int      `json:"threadCount"`
	CalculateHash   bool     `json:"calculateHash"`
	CacheResults    bool     `json:"cacheResults"`
	PageSize        int      `json:"pageSize"`
}

// ScanProgress is emitted to the frontend for live status.
type ScanProgress struct {
	SessionID      string        `json:"sessionId"`
	CurrentPath    string        `json:"currentPath"`
	FilesScanned   int64         `json:"filesScanned"`
	DirsScanned    int64         `json:"dirsScanned"`
	TotalSize      int64         `json:"totalSize"`
	FormattedSize  string        `json:"formattedSize"`
	Errors         []string      `json:"errors"`
	Percentage     float64       `json:"percentage"`
	Speed          float64       `json:"speed"`
	TimeRemaining  time.Duration `json:"timeRemaining"`
	CurrentDepth   int           `json:"currentDepth"`
	EstimatedTotal int64         `json:"estimatedTotal"`
	Status         string        `json:"status"`
}

// ScanSession tracks a single scan run.
type ScanSession struct {
	ID        string
	StartTime time.Time
	EndTime   *time.Time
	Options   ScanOptions

	Root       *FileNode
	TotalFiles int64
	TotalDirs  int64
	TotalSize  int64

	Duplicates  map[string][]*FileNode
	Cache       *sql.DB
	Errors      []string
	ErrorsMutex sync.RWMutex
}

// Settings persisted under %USERPROFILE%\.directoryscanner\settings.json
type Settings struct {
	ThreadCount         int    `json:"threadCount"`
	CacheResults        bool   `json:"cacheResults"`
	DefaultExportFormat string `json:"defaultExportFormat"`
}

// MemoryPool is used by the scanner for reusable buffers.
type MemoryPool struct {
	buffers    chan []byte
	pageSize   int64
	maxBuffers int
}

// logWriter captures log output into the App logs slice.
type logWriter struct {
	app *App
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.app.logsMutex.Lock()
	w.app.logs = append(w.app.logs, string(p))
	w.app.logsMutex.Unlock()
	return len(p), nil
}

// =====================
// App struct & startup
// =====================

type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	scanner  *Scanner
	analyzer *FileAnalyzer

	currentSession *ScanSession
	sessionMutex   sync.RWMutex

	progressCallback func(ScanProgress)

	cloudDrives      []CloudDrive
	cloudDrivesMutex sync.RWMutex

	db         *sql.DB
	memoryPool *MemoryPool
	logs       []string
	logsMutex  sync.Mutex
	exporter   *Exporter
}

// NewApp constructs the backend and wires logging.
func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		ctx:      ctx,
		cancel:   cancel,
		scanner:  NewScanner(),
		analyzer: NewFileAnalyzer(),
		memoryPool: &MemoryPool{
			buffers:    make(chan []byte, 100),
			pageSize:   1024 * 1024,
			maxBuffers: 100,
		},
		logs:     make([]string, 0),
		exporter: NewExporter(),
	}

	log.SetOutput(&logWriter{app: app})
	log.Println("App instance created")
	return app
}

// GetCurrentTime is a dummy method to force binding generator to recognize time.Time
func (a *App) GetCurrentTime() time.Time {
	return time.Now()
}

// Startup is called by Wails when the app launches.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("=== App Startup ===")

	if err := a.initializeCache(); err != nil {
		log.Printf("failed to initialize cache: %v", err)
	}

	if err := a.DetectCloudDrives(); err != nil {
		log.Printf("DetectCloudDrives error: %v", err)
	} else {
		drives := a.GetCloudDrives()
		if len(drives) == 0 {
			log.Println("No cloud drives detected; user can still browse any folder manually.")
		} else {
			log.Printf("Detected %d cloud drive(s):", len(drives))
			for _, d := range drives {
				log.Printf("  [%s] %s at %s (mounted=%v)", d.Type, d.DisplayName, d.MountPoint, d.Mounted)
			}
		}
	}

	log.Println("=== Startup Complete ===")
}

// Shutdown flushes logs and closes resources.
func (a *App) Shutdown(ctx context.Context) {
	a.cancel()

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}

	logFile, err := os.Create(fmt.Sprintf("log-%s.txt", time.Now().Format("2006-01-02-15-04-05")))
	if err != nil {
		log.Printf("failed to create log file: %v", err)
		return
	}
	defer logFile.Close()

	a.logsMutex.Lock()
	for _, line := range a.logs {
		if _, err := logFile.WriteString(line); err != nil {
			break
		}
	}
	a.logsMutex.Unlock()
	log.Println("Shutdown complete; logs written to disk")
}

// GetLogs returns all captured log lines for the UI.
func (a *App) GetLogs() []string {
	log.Println("GetLogs called")
	a.logsMutex.Lock()
	defer a.logsMutex.Unlock()
	return append([]string{}, a.logs...)
}

// =====================
// Scan control & status
// =====================

// SubscribeScanProgress wires a Wails event emitter for scan updates.
func (a *App) SubscribeScanProgress() error {
	a.progressCallback = func(p ScanProgress) {
		if a.ctx != nil {
			// fire-and-forget to avoid blocking scanner
			go runtime.EventsEmit(a.ctx, "scanProgress", p)
		}
	}
	log.Println("Progress callback subscribed")
	return nil
}

// StartScan validates the path, creates a session, and kicks off the scanner.
func (a *App) StartScan(options ScanOptions) (*FileNode, error) {
	log.Printf("StartScan called with path=%q maxDepth=%d threadCount=%d",
		options.Path, options.MaxDepth, options.ThreadCount)

	if options.Path == "" {
		err := fmt.Errorf("scan path is empty")
		log.Println(err)
		return nil, err
	}

	info, err := os.Stat(options.Path)
	if err != nil {
		msg := fmt.Errorf("scan path %q does not exist or is not accessible: %w", options.Path, err)
		log.Println(msg)
		return nil, msg
	}
	if !info.IsDir() {
		msg := fmt.Errorf("scan path %q is not a directory", options.Path)
		log.Println(msg)
		return nil, msg
	}

	if a.progressCallback == nil {
		log.Println("WARNING: progressCallback is nil; frontend should call SubscribeScanProgress() first")
	}

	a.sessionMutex.Lock()
	sessionID := uuid.New().String()
	a.currentSession = &ScanSession{
		ID:        sessionID,
		StartTime: time.Now(),
		Options:   options,
		Errors:    make([]string, 0),
	}
	a.sessionMutex.Unlock()

	log.Printf("Session %s starting scan of %q", sessionID, options.Path)

	go func(sid string, opts ScanOptions) {
		root, err := a.scanner.ScanWithProgress(
			a.ctx,
			opts,
			a.progressCallback,
			a.currentSession,
		)
		if err != nil {
			a.sessionMutex.Lock()
			if a.currentSession != nil && a.currentSession.ID == sid {
				a.currentSession.Errors = append(a.currentSession.Errors, err.Error())
			}
			a.sessionMutex.Unlock()
			log.Printf("ScanWithProgress failed: %v", err)
			return
		}

		a.sessionMutex.Lock()
		defer a.sessionMutex.Unlock()
		if a.currentSession != nil && a.currentSession.ID == sid {
			a.currentSession.Root = root
			end := time.Now()
			a.currentSession.EndTime = &end
			log.Printf("Scan completed for %s", opts.Path)
		}
	}(sessionID, options)

	return nil, nil
}

// StopScan asks the scanner to cancel work (implementation in scanner.go).
func (a *App) StopScan() error {
	log.Println("StopScan requested")
	StopScan()
	return nil
}

// PauseScan currently just clears the active session (stub for future).
func (a *App) PauseScan() error {
	a.sessionMutex.Lock()
	if a.currentSession != nil {
		log.Printf("Pausing session %s (clearing reference only)", a.currentSession.ID)
		a.currentSession = nil
	}
	a.sessionMutex.Unlock()
	return nil
}

// ResumeScan is a placeholder; actual resume would require scanner support.
func (a *App) ResumeScan() error {
	log.Println("ResumeScan is not implemented")
	return nil
}

// GetScanStatus proxies to global scanner status.
func (a *App) GetScanStatus() ScanStatus {
	return GetScanStatus()
}

// GetCurrentProgress builds a summary snapshot from the current session.
func (a *App) GetCurrentProgress() *ScanProgress {
	a.sessionMutex.RLock()
	defer a.sessionMutex.RUnlock()

	if a.currentSession == nil {
		return nil
	}

	status := "running"
	if a.currentSession.EndTime != nil {
		status = "completed"
	}

	return &ScanProgress{
		SessionID:    a.currentSession.ID,
		FilesScanned: a.currentSession.TotalFiles,
		DirsScanned:  a.currentSession.TotalDirs,
		TotalSize:    a.currentSession.TotalSize,
		Status:       status,
	}
}

// =====================
// Cloud drives & tree
// =====================

// GetCloudDrives returns the last-detected cloud drives.
func (a *App) GetCloudDrives() []CloudDrive {
	a.cloudDrivesMutex.RLock()
	defer a.cloudDrivesMutex.RUnlock()
	return a.cloudDrives
}

// DetectCloudDrives refreshes the cached list using cloud_detector.go.
func (a *App) DetectCloudDrives() error {
	a.cloudDrivesMutex.Lock()
	defer a.cloudDrivesMutex.Unlock()

	detector := NewCloudDriveDetector()
	drives, err := detector.DetectAll()
	if err != nil {
		return err
	}

	a.cloudDrives = drives
	return nil
}

// LoadChildrenLazy returns children of a node by ID (no pagination for now).
func (a *App) LoadChildrenLazy(nodeID string, pageNum int) ([]*FileNode, error) {
	a.sessionMutex.RLock()
	defer a.sessionMutex.RUnlock()

	if a.currentSession == nil || a.currentSession.Root == nil {
		return nil, fmt.Errorf("no active session")
	}

	node := a.findNodeByID(a.currentSession.Root, nodeID)
	if node == nil {
		return nil, fmt.Errorf("node not found")
	}

	return node.Children, nil
}

func (a *App) findNodeByID(node *FileNode, targetID string) *FileNode {
	if node.ID == targetID {
		return node
	}
	for _, child := range node.Children {
		if found := a.findNodeByID(child, targetID); found != nil {
			return found
		}
	}
	return nil
}

// =====================
// Browse & export UX
// =====================

// Browse opens a directory chooser and returns the selected path.
func (a *App) Browse() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{})
}

// BrowseForExport opens a save dialog with filters based on format.
func (a *App) BrowseForExport(format string) (string, error) {
	var filters []runtime.FileFilter

	switch strings.ToLower(format) {
	case "json":
		filters = []runtime.FileFilter{{DisplayName: "JSON Files", Pattern: "*.json"}}
	case "csv":
		filters = []runtime.FileFilter{{DisplayName: "CSV Files", Pattern: "*.csv"}}
	case "excel", "xlsx":
		filters = []runtime.FileFilter{{DisplayName: "Excel Files", Pattern: "*.xlsx"}}
	case "html":
		filters = []runtime.FileFilter{{DisplayName: "HTML Files", Pattern: "*.html"}}
	case "pdf":
		filters = []runtime.FileFilter{{DisplayName: "PDF Files", Pattern: "*.pdf"}}
	case "markdown", "md":
		filters = []runtime.FileFilter{{DisplayName: "Markdown Files", Pattern: "*.md"}}
	case "xml":
		filters = []runtime.FileFilter{{DisplayName: "XML Files", Pattern: "*.xml"}}
	case "tree", "txt":
		filters = []runtime.FileFilter{{DisplayName: "Text Files", Pattern: "*.txt"}}
	}

	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Filters: filters,
	})
}

// ExportResults exports the current scan using exporter.go (no deep analysis).
func (a *App) ExportResults(format string, outputPath string) error {
	a.sessionMutex.RLock()
	defer a.sessionMutex.RUnlock()

	if a.currentSession == nil || a.currentSession.Root == nil {
		return fmt.Errorf("no scan results to export")
	}
	if outputPath == "" {
		return fmt.Errorf("output path is empty")
	}

	if a.exporter == nil {
		a.exporter = NewExporter()
	}

	opts := ExportOptions{
		OutputPath:      outputPath,
		IncludeHidden:   a.currentSession.Options.IncludeHidden,
		MaxDepth:        a.currentSession.Options.MaxDepth,
		Format:          format,
		IncludeAnalysis: false,
	}
	log.Printf("ExportResults: format=%s path=%s", format, outputPath)
	return a.exporter.Export(a.currentSession.Root, opts)
}

// =====================
// Settings & cache
// =====================

func (a *App) initializeCache() error {
	// Placeholder for future DB/cache setup.
	return nil
}

func (a *App) getSettingsPath() string {
	return filepath.Join(os.Getenv("USERPROFILE"), ".directoryscanner", "settings.json")
}

// GetSettings loads user settings or returns defaults.
func (a *App) GetSettings() (Settings, error) {
	log.Println("Getting settings...")
	settingsPath := a.getSettingsPath()

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		log.Printf("Failed to read settings file, returning defaults: %v", err)
		return Settings{
			ThreadCount:         4,
			CacheResults:        true,
			DefaultExportFormat: "json",
		}, nil
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Printf("Failed to unmarshal settings, returning error: %v", err)
		return Settings{}, err
	}

	log.Printf("Returning settings: %+v", settings)
	return settings, nil
}

// SaveSettings persists settings to disk.
func (a *App) SaveSettings(settings Settings) error {
	log.Printf("Saving settings: %+v", settings)
	settingsPath := a.getSettingsPath()

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		log.Printf("Failed to create settings directory: %v", err)
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal settings: %v", err)
		return err
	}

	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		log.Printf("Failed to write settings file: %v", err)
		return err
	}

	log.Println("Settings saved successfully.")
	return nil
}

// ExportSettings writes the current settings to a chosen path.
func (a *App) ExportSettings(path string) error {
	settings, err := a.GetSettings()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ImportSettings loads settings from a file and saves them.
func (a *App) ImportSettings(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	return a.SaveSettings(settings)
}
