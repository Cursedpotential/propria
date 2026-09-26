package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileAnalysis contains detailed analysis of a file
type FileAnalysis struct {
	// Basic info
	Path      string `json:"path"`
	Name      string `json:"name"`
	Extension string `json:"extension"`
	Size      int64  `json:"size"`

	// Timestamps
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Accessed time.Time `json:"accessed"`

	// Content preview
	HeadLines  []string `json:"headLines"`
	TailLines  []string `json:"tailLines"`
	LineCount  int64    `json:"lineCount"`
	IsTextFile bool     `json:"isTextFile"`
	Encoding   string   `json:"encoding"`

	// Hashes for duplicate detection
	MD5Hash    string `json:"md5Hash"`
	SHA256Hash string `json:"sha256Hash"`
	QuickHash  string `json:"quickHash"` // First 1MB hash for quick comparison
	// General-purpose hash (MD5 by default when available)
	Hash string `json:"hash,omitempty"`

	// Metadata
	MimeType      string `json:"mimeType"`
	IsCloud       bool   `json:"isCloud"`
	CloudProvider string `json:"cloudProvider"`
	IsSymlink     bool   `json:"isSymlink"`
	TargetPath    string `json:"targetPath,omitempty"`

	// Folder info
	FolderPath   string `json:"folderPath"`
	RelativePath string `json:"relativePath"`
	Depth        int    `json:"depth"`
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Hash        string         `json:"hash"`
	Size        int64          `json:"size"`
	Files       []FileAnalysis `json:"files"`
	TotalWasted int64          `json:"totalWasted"` // Size that could be saved
}

// FileAnalyzer performs deep file analysis
type FileAnalyzer struct {
	mutex         sync.RWMutex
	hashCache     map[string]string
	duplicates    map[string][]FileAnalysis
	quickHashMap  map[string][]string // Quick hash to file paths for fast pre-filtering
	cloudDetector *CloudDriveDetector
}

// NewFileAnalyzer creates a new file analyzer
func NewFileAnalyzer() *FileAnalyzer {
	return &FileAnalyzer{
		hashCache:     make(map[string]string),
		duplicates:    make(map[string][]FileAnalysis),
		quickHashMap:  make(map[string][]string),
		cloudDetector: NewCloudDriveDetector(),
	}
}

// AnalyzeFile performs comprehensive analysis of a single file
func (fa *FileAnalyzer) AnalyzeFile(path string, calculateFullHash bool) (*FileAnalysis, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	analysis := &FileAnalysis{
		Path:         path,
		Name:         info.Name(),
		Extension:    strings.TrimPrefix(filepath.Ext(path), "."),
		Size:         info.Size(),
		Modified:     info.ModTime(),
		FolderPath:   filepath.Dir(path),
		RelativePath: filepath.Base(path),
		Depth:        strings.Count(path, string(os.PathSeparator)),
	}

	// Get additional timestamps (Windows-specific)
	fa.getFileTimestamps(path, analysis)

	// Check if symlink
	if info.Mode()&os.ModeSymlink != 0 {
		analysis.IsSymlink = true
		if target, err := os.Readlink(path); err == nil {
			analysis.TargetPath = target
		}
	}

	// Detect cloud status
	analysis.IsCloud, analysis.CloudProvider = fa.detectCloudStatus(path)

	// Determine MIME type
	analysis.MimeType = fa.detectMimeType(path)

	// Check if text file and extract preview
	if fa.isTextFile(path) {
		analysis.IsTextFile = true
		analysis.Encoding = "UTF-8" // Could be enhanced with actual encoding detection

		// Extract head and tail lines
		if err := fa.extractFilePreview(path, analysis); err != nil {
			// Log error but don't fail the entire analysis
			fmt.Printf("Warning: failed to extract preview for %s: %v\n", path, err)
		}
	}

	// Calculate hashes for duplicate detection
	if calculateFullHash && info.Size() > 0 {
		// Quick hash (first 1MB)
		if quickHash, err := fa.calculateQuickHash(path); err == nil {
			analysis.QuickHash = quickHash

			// Store in quick hash map for fast duplicate pre-filtering
			fa.mutex.Lock()
			fa.quickHashMap[quickHash] = append(fa.quickHashMap[quickHash], path)
			fa.mutex.Unlock()
		}

		// Full hashes (for files that might be duplicates)
		if fa.shouldCalculateFullHash(analysis.QuickHash, info.Size()) {
			if sha256Hash, err := fa.calculateSHA256(path); err == nil {
				analysis.SHA256Hash = sha256Hash
				analysis.MD5Hash = sha256Hash
			}
		}

		// Populate a general-purpose Hash field for compatibility
		if analysis.SHA256Hash != "" {
			analysis.Hash = analysis.SHA256Hash
		} else if analysis.QuickHash != "" {
			analysis.Hash = analysis.QuickHash
		}
	}

	return analysis, nil
}

// extractFilePreview extracts head and tail lines from a text file
func (fa *FileAnalyzer) extractFilePreview(path string, analysis *FileAnalysis) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	const maxPreviewLines = 10
	analysis.HeadLines = make([]string, 0, maxPreviewLines)

	// Read head lines
	lineCount := int64(0)
	allLines := make([]string, 0)

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount <= maxPreviewLines {
			analysis.HeadLines = append(analysis.HeadLines, line)
		}

		// Store all lines if file is small
		if lineCount <= maxPreviewLines*2 {
			allLines = append(allLines, line)
		}
	}

	analysis.LineCount = lineCount

	// Extract tail lines if file has more than 2*maxPreviewLines
	if lineCount > maxPreviewLines*2 {
		// Re-read file for tail (more efficient for large files)
		analysis.TailLines = fa.readTailLines(path, maxPreviewLines)
	} else if lineCount > maxPreviewLines {
		// Use stored lines for tail
		startIdx := len(allLines) - maxPreviewLines
		if startIdx < 0 {
			startIdx = 0
		}
		analysis.TailLines = allLines[startIdx:]
	}

	return scanner.Err()
}

// readTailLines reads the last n lines from a file efficiently
func (fa *FileAnalyzer) readTailLines(path string, n int) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	// Use a ring buffer for efficient tail reading
	lines := make([]string, 0, n)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if len(lines) >= n {
			lines = lines[1:]
		}
		lines = append(lines, scanner.Text())
	}

	return lines
}

// isTextFile checks if a file is likely a text file
func (fa *FileAnalyzer) isTextFile(path string) bool {
	// Check by extension first
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".log": true, ".csv": true,
		".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".ini": true, ".cfg": true, ".conf": true, ".config": true,
		".py": true, ".js": true, ".ts": true, ".go": true,
		".java": true, ".c": true, ".cpp": true, ".h": true,
		".cs": true, ".php": true, ".rb": true, ".rs": true,
		".sql": true, ".sh": true, ".bat": true, ".ps1": true,
		".html": true, ".css": true, ".scss": true, ".vue": true,
		".jsx": true, ".tsx": true, ".svelte": true,
	}

	if textExtensions[ext] {
		return true
	}

	// Sample file content for binary check
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 512 bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (binary indicator)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}

	return true
}

// calculateQuickHash calculates a hash of the first 1MB of the file
func (fa *FileAnalyzer) calculateQuickHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()

	// Read first 1MB
	buffer := make([]byte, 1024*1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	hasher.Write(buffer[:n])
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// calculateMD5 calculates the SHA256 hash of the entire file (kept for compatibility)
func (fa *FileAnalyzer) calculateMD5(path string) (string, error) {
	return fa.calculateSHA256(path)
}

// calculateSHA256 calculates the SHA256 hash of the entire file
func (fa *FileAnalyzer) calculateSHA256(path string) (string, error) {
	fa.mutex.RLock()
	if cached, ok := fa.hashCache[path+"_sha256"]; ok {
		fa.mutex.RUnlock()
		return cached, nil
	}
	fa.mutex.RUnlock()

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	fa.mutex.Lock()
	fa.hashCache[path+"_sha256"] = hash
	fa.mutex.Unlock()

	return hash, nil
}

// shouldCalculateFullHash determines if we should calculate full hash based on quick hash
func (fa *FileAnalyzer) shouldCalculateFullHash(quickHash string, size int64) bool {
	if quickHash == "" {
		return false
	}

	fa.mutex.RLock()
	defer fa.mutex.RUnlock()

	// If multiple files share the same quick hash, they might be duplicates
	if paths, ok := fa.quickHashMap[quickHash]; ok && len(paths) > 1 {
		return true
	}

	// Always hash small files completely
	if size < 1024*1024 { // Less than 1MB
		return true
	}

	return false
}

// detectCloudStatus checks if file is in a cloud storage location
func (fa *FileAnalyzer) detectCloudStatus(path string) (bool, string) {
	lowPath := strings.ToLower(path)

	// Check common cloud storage patterns
	cloudPatterns := map[string]string{
		"onedrive":     "OneDrive",
		"googledrive":  "Google Drive",
		"google drive": "Google Drive",
		"my drive":     "Google Drive",
		"dropbox":      "Dropbox",
		"icloud":       "iCloud",
		"box sync":     "Box",
		"box":          "Box",
	}

	for pattern, provider := range cloudPatterns {
		if strings.Contains(lowPath, pattern) {
			return true, provider
		}
	}

	// Check if file has cloud attributes (Windows)
	if fa.hasCloudAttributes(path) {
		// Try to determine provider from path
		for pattern, provider := range cloudPatterns {
			if strings.Contains(lowPath, pattern) {
				return true, provider
			}
		}
		return true, "Unknown"
	}

	return false, ""
}

// hasCloudAttributes checks Windows file attributes for cloud status
func (fa *FileAnalyzer) hasCloudAttributes(path string) bool {
	// Windows cloud file attributes
	const (
		FILE_ATTRIBUTE_RECALL_ON_OPEN        = 0x00040000
		FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS = 0x00400000
		FILE_ATTRIBUTE_PINNED                = 0x00080000
		FILE_ATTRIBUTE_UNPINNED              = 0x00100000
	)

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check for cloud attributes using syscall
	if sysInfo, ok := info.Sys().(interface{ FileAttributes() uint32 }); ok {
		attrs := sysInfo.FileAttributes()
		return (attrs&FILE_ATTRIBUTE_RECALL_ON_OPEN != 0) ||
			(attrs&FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS != 0) ||
			(attrs&FILE_ATTRIBUTE_PINNED != 0) ||
			(attrs&FILE_ATTRIBUTE_UNPINNED != 0)
	}

	return false
}

// detectMimeType detects the MIME type of a file
func (fa *FileAnalyzer) detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".mov":  "video/quicktime",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	return "application/octet-stream"
}

// getFileTimestamps gets creation, modification, and access times
func (fa *FileAnalyzer) getFileTimestamps(path string, analysis *FileAnalysis) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	analysis.Modified = info.ModTime()

	// Windows-specific timestamp extraction
	if sysInfo, ok := info.Sys().(interface {
		CreationTime() time.Time
		LastAccessTime() time.Time
	}); ok {
		analysis.Created = sysInfo.CreationTime()
		analysis.Accessed = sysInfo.LastAccessTime()
	} else {
		// Fallback for non-Windows or if syscall fails
		analysis.Created = analysis.Modified
		analysis.Accessed = analysis.Modified
	}
}

// FindDuplicates finds duplicate files in the analyzed set
func (fa *FileAnalyzer) FindDuplicates(files []FileAnalysis) []DuplicateGroup {
	duplicateMap := make(map[string][]FileAnalysis)

	// Group files by hash
	for _, file := range files {
		if file.SHA256Hash != "" {
			duplicateMap[file.SHA256Hash] = append(duplicateMap[file.SHA256Hash], file)
		} else if file.MD5Hash != "" {
			duplicateMap[file.MD5Hash] = append(duplicateMap[file.MD5Hash], file)
		}
	}

	// Create duplicate groups
	var groups []DuplicateGroup
	for hash, files := range duplicateMap {
		if len(files) > 1 {
			group := DuplicateGroup{
				Hash:        hash,
				Size:        files[0].Size,
				Files:       files,
				TotalWasted: files[0].Size * int64(len(files)-1),
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// AnalyzeDirectory performs deep analysis on all files in a directory
func (fa *FileAnalyzer) AnalyzeDirectory(rootPath string, options ScanOptions) ([]FileAnalysis, []DuplicateGroup, error) {
	var analyses []FileAnalysis
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Thread pool for parallel analysis
	semaphore := make(chan struct{}, options.ThreadCount)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Apply filters
		if options.MinSize > 0 && info.Size() < options.MinSize {
			return nil
		}
		if options.MaxSize > 0 && info.Size() > options.MaxSize {
			return nil
		}

		// Check extensions filter
		if len(options.Extensions) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			found := false
			for _, allowedExt := range options.Extensions {
				if strings.EqualFold(ext, strings.TrimPrefix(allowedExt, ".")) {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			analysis, err := fa.AnalyzeFile(p, options.CalculateHash)
			if err == nil {
				mu.Lock()
				analyses = append(analyses, *analysis)
				mu.Unlock()
			}
		}(path)

		return nil
	})

	wg.Wait()

	// Find duplicates if hashing was enabled
	var duplicates []DuplicateGroup
	if options.CalculateHash {
		duplicates = fa.FindDuplicates(analyses)
	}

	return analyses, duplicates, err
}
