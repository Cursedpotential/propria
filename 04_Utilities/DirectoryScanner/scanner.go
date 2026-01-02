package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/karrick/godirwalk"
)

// ScanStatus provides progress details for the active scan
type ScanStatus struct {
	Running        bool      `json:"running"`
	FilesScanned   int64     `json:"files_scanned"`
	DirsScanned    int64     `json:"dirs_scanned"`
	CurrentPath    string    `json:"current_path"`
	StartTime      time.Time `json:"start_time"`
	ElapsedSeconds int64     `json:"elapsed_seconds"`
	Error          string    `json:"error"`
}

var (
	scanRunning int32
	filesCount  int64
	dirsCount   int64
	currentPath atomic.Value // stores string
	scanStart   atomic.Value // stores time.Time
	scanErr     atomic.Value // stores string
	stopChan    chan struct{}
	scanMu      sync.Mutex
)

type Scanner struct {
	options        ScanOptions
	fileCount      atomic.Int64
	dirCount       atomic.Int64
	totalSize      atomic.Int64
	progressFunc   func(ScanProgress)
	startTime      time.Time
	errors         []string
	errorsMutex    sync.Mutex
	workerPool     chan struct{}
	wg             sync.WaitGroup
	ctx            context.Context
	isNetworkDrive bool
}

func NewScanner() *Scanner {
	return &Scanner{
		errors: make([]string, 0),
	}
}

func (s *Scanner) ScanWithProgress(
	ctx context.Context,
	options ScanOptions,
	progressFunc func(ScanProgress),
	session *ScanSession,
) (*FileNode, error) {
	s.ctx = ctx
	s.options = options
	s.progressFunc = progressFunc
	s.startTime = time.Now()
	s.isNetworkDrive = s.detectNetworkDrive(options.Path)

	threadCount := options.ThreadCount
	if threadCount == 0 {
		if s.isNetworkDrive {
			threadCount = 4
		} else {
			threadCount = runtime.NumCPU() * 2
		}
	}

	s.workerPool = make(chan struct{}, threadCount)

	// Report initial progress
	if s.progressFunc != nil {
		initialProgress := ScanProgress{
			SessionID:     session.ID,
			CurrentPath:   "Starting scan...",
			FilesScanned:  0,
			DirsScanned:   0,
			TotalSize:     0,
			FormattedSize: "0 B",
			Speed:         0,
			Status:        "starting",
			Percentage:    0.0,
		}
		s.progressFunc(initialProgress)
	}

	rootInfo, err := os.Stat(options.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat root path: %w", err)
	}

	root := &FileNode{
		ID:             rootInfo.Name() + "-" + time.Now().Format("20060102150405"),
		Name:           filepath.Base(options.Path),
		Path:           options.Path,
		IsDir:          rootInfo.IsDir(),
		Modified:       rootInfo.ModTime(),
		ChildrenLoaded: false,
	}

	if err := s.scanDirectory(ctx, root, options.Path, 0, session); err != nil {
		s.logError(fmt.Sprintf("scan error: %v", err))
	}

	session.TotalFiles = s.fileCount.Load()
	session.TotalDirs = s.dirCount.Load()
	session.TotalSize = s.totalSize.Load()

	// Report final completion status
	if s.progressFunc != nil {
		elapsed := time.Since(s.startTime)
		var speed float64
		if elapsed > 0 {
			speed = float64(s.fileCount.Load()) / elapsed.Seconds()
		}

		completionProgress := ScanProgress{
			SessionID:     session.ID,
			CurrentPath:   "Scan completed",
			FilesScanned:  s.fileCount.Load(),
			DirsScanned:   s.dirCount.Load(),
			TotalSize:     s.totalSize.Load(),
			FormattedSize: humanize.Bytes(uint64(s.totalSize.Load())),
			Speed:         speed,
			Status:        "completed",
			Percentage:    100.0,
		}
		s.progressFunc(completionProgress)
	}

	return root, nil
}

func (s *Scanner) scanDirectory(
	ctx context.Context,
	parent *FileNode,
	dirPath string,
	depth int,
	session *ScanSession,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if s.options.MaxDepth > 0 && depth >= s.options.MaxDepth {
		return nil
	}

	// Report progress when entering a new directory
	if s.progressFunc != nil && depth <= 3 { // Only report for shallow depths to avoid spam
		s.reportProgress(session, fmt.Sprintf("Scanning: %s", dirPath))
	}

	entries, err := godirwalk.ReadDirnames(dirPath, nil)
	if err != nil {
		s.logError(fmt.Sprintf("failed to read dir %s: %v", dirPath, err))
		return nil
	}

	children := make([]*FileNode, 0, len(entries))

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry)

		if s.shouldSkip(entry, fullPath) {
			continue
		}

		info, err := os.Lstat(fullPath)
		if err != nil {
			s.logError(fmt.Sprintf("failed to stat %s: %v", fullPath, err))
			continue
		}

		node := s.createFileNode(fullPath, entry, info)

		if s.isCloudFile(fullPath) {
			node.IsCloud = true
			node.CloudType = s.detectCloudType(fullPath)
		}

		if info.IsDir() {
			s.dirCount.Add(1)
			node.ChildCount = s.estimateChildCount(fullPath)

			if s.options.FollowSymlinks || !node.IsSymlink {
				s.wg.Add(1)
				go func(n *FileNode, p string, d int) {
					defer s.wg.Done()
					s.workerPool <- struct{}{}
					defer func() { <-s.workerPool }()

					s.scanDirectory(ctx, n, p, d+1, session)
				}(node, fullPath, depth+1)
			}
		} else {
			s.fileCount.Add(1)
			s.totalSize.Add(info.Size())

			if s.options.CalculateHash && info.Size() < 100*1024*1024 {
				go s.hashFileAsync(fullPath, node)
			}
		}

		children = append(children, node)

		// Report progress more frequently and include current path
		if s.fileCount.Load()%10 == 0 || s.dirCount.Load()%5 == 0 {
			s.reportProgress(session, dirPath)
		}
	}

	parent.Children = children
	parent.ChildrenLoaded = true

	s.wg.Wait()

	return nil
}

func (s *Scanner) createFileNode(path string, name string, info os.FileInfo) *FileNode {
	ext := filepath.Ext(name)
	if ext != "" {
		ext = ext[1:]
	}

	node := &FileNode{
		ID:            uuid.New().String(),
		Name:          name,
		Path:          path,
		Size:          info.Size(),
		FormattedSize: humanize.Bytes(uint64(info.Size())),
		Modified:      info.ModTime(),
		IsDir:         info.IsDir(),
		Extension:     strings.ToLower(ext),
		Attributes:    uint32(info.Mode()),
	}

	node.IsSymlink = (info.Mode() & os.ModeSymlink) != 0

	return node
}

func (s *Scanner) shouldSkip(name string, path string) bool {
	if !s.options.IncludeHidden && strings.HasPrefix(name, ".") {
		return true
	}

	for _, excluded := range s.options.ExcludeDirs {
		if name == excluded {
			return true
		}
	}

	if len(s.options.Extensions) > 0 && !s.isAllowedExtension(filepath.Ext(name)) {
		return true
	}

	for _, pattern := range s.options.ExcludePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

func (s *Scanner) isAllowedExtension(ext string) bool {
	if ext != "" {
		ext = ext[1:]
	}

	for _, allowed := range s.options.Extensions {
		if strings.EqualFold(ext, strings.TrimPrefix(allowed, ".")) {
			return true
		}
	}
	return false
}

func (s *Scanner) hashFileAsync(path string, node *FileNode) {
	s.wg.Add(1)
	defer s.wg.Done()

	s.workerPool <- struct{}{}
	defer func() { <-s.workerPool }()

	hash, err := s.hashFile(path)
	if err != nil {
		s.logError(fmt.Sprintf("failed to hash %s: %v", path, err))
		return
	}

	node.Hash = hash
}

func (s *Scanner) hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.CopyN(hash, file, 512*1024); err != nil && err != io.EOF {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (s *Scanner) isCloudFile(path string) bool {
	const CloudMask = 0x00040000

	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	if sysInfo, ok := stat.Sys().(interface{ Attributes() uint32 }); ok {
		return (sysInfo.Attributes() & CloudMask) != 0
	}

	return false
}

func (s *Scanner) detectCloudType(path string) string {
	path = strings.ToLower(path)

	if strings.Contains(path, "onedrive") {
		return "onedrive"
	}
	if strings.Contains(path, "googledrive") || strings.Contains(path, "google drive") {
		return "googledrive"
	}
	if strings.Contains(path, "dropbox") {
		return "dropbox"
	}
	if strings.Contains(path, "box") && strings.Contains(path, "sync") {
		return "box"
	}
	if strings.Contains(path, "iclouddrive") {
		return "icloud"
	}

	return "unknown"
}

func (s *Scanner) detectNetworkDrive(path string) bool {
	if strings.HasPrefix(path, "\\\\") {
		return true
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	if len(abs) > 1 && abs[1] == ':' {
		drive := abs[0]
		return drive > 'E'
	}

	return false
}

func (s *Scanner) estimateChildCount(dirPath string) int {
	entries, err := godirwalk.ReadDirnames(dirPath, nil)
	if err != nil {
		return 0
	}
	return len(entries)
}

func (s *Scanner) reportProgress(session *ScanSession, currentPath string) {
	if s.progressFunc == nil {
		return
	}

	elapsed := time.Since(s.startTime)
	filesScanned := s.fileCount.Load()

	var speed float64
	if elapsed > 0 {
		speed = float64(filesScanned) / elapsed.Seconds()
	}

	progress := ScanProgress{
		SessionID:     session.ID,
		CurrentPath:   currentPath,
		FilesScanned:  filesScanned,
		DirsScanned:   s.dirCount.Load(),
		TotalSize:     s.totalSize.Load(),
		FormattedSize: humanize.Bytes(uint64(s.totalSize.Load())),
		Speed:         speed,
		Status:        "scanning",
	}

	s.progressFunc(progress)
}

func (s *Scanner) logError(msg string) {
	s.errorsMutex.Lock()
	defer s.errorsMutex.Unlock()
	s.errors = append(s.errors, msg)
}

func (s *Scanner) GetErrors() []string {
	s.errorsMutex.Lock()
	defer s.errorsMutex.Unlock()

	result := make([]string, len(s.errors))
	copy(result, s.errors)
	return result
}

// StartScan starts walking the provided root path in a background goroutine and updates progress.
func StartScan(ctx context.Context, root string) error {
	scanMu.Lock()
	defer scanMu.Unlock()

	if atomic.LoadInt32(&scanRunning) == 1 {
		return nil // already running
	}

	// reset state
	atomic.StoreInt64(&filesCount, 0)
	atomic.StoreInt64(&dirsCount, 0)
	currentPath.Store("")
	scanErr.Store("")
	atomic.StoreInt32(&scanRunning, 1)
	stopChan = make(chan struct{})
	scanStart.Store(time.Now())

	go func() {
		defer atomic.StoreInt32(&scanRunning, 0)
		walkFn := func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				scanErr.Store(err.Error())
				return nil // continue walking so we can find other items
			}
			// abort if requested
			select {
			case <-stopChan:
				return filepath.SkipDir
			default:
			}
			currentPath.Store(p)
			if d.IsDir() {
				atomic.AddInt64(&dirsCount, 1)
			} else {
				atomic.AddInt64(&filesCount, 1)
			}
			return nil
		}

		// Begin walking
		if err := filepath.WalkDir(root, walkFn); err != nil {
			scanErr.Store(err.Error())
		}
	}()

	return nil
}

// StopScan requests the background scan to stop (if running)
func StopScan() {
	scanMu.Lock()
	defer scanMu.Unlock()
	if atomic.LoadInt32(&scanRunning) == 0 {
		return
	}
	close(stopChan)
}

// GetScanStatus returns the current scan progress snapshot
func GetScanStatus() ScanStatus {
	var ss ScanStatus
	ss.Running = atomic.LoadInt32(&scanRunning) == 1
	ss.FilesScanned = atomic.LoadInt64(&filesCount)
	ss.DirsScanned = atomic.LoadInt64(&dirsCount)
	if v := currentPath.Load(); v != nil {
		ss.CurrentPath = v.(string)
	}
	if t := scanStart.Load(); t != nil {
		if st, ok := t.(time.Time); ok && !st.IsZero() {
			ss.StartTime = st
			ss.ElapsedSeconds = int64(time.Since(st).Seconds())
		}
	}
	if e := scanErr.Load(); e != nil {
		ss.Error = e.(string)
	}
	return ss
}
