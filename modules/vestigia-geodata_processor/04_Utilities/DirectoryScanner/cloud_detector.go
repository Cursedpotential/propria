package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// CloudDrive represents a detected cloud sync location.
type CloudDrive struct {
	Type        string `json:"type"`         // short provider key (OneDrive, GoogleDrive, etc.)
	Provider    string `json:"provider"`     // human-friendly provider name
	MountPoint  string `json:"mount_point"`  // root mount path if available
	Path        string `json:"path"`         // normalized path (usually same as MountPoint)
	Email       string `json:"email"`        // account email if discovered
	Mounted     bool   `json:"mounted"`      // whether path exists on disk
	DisplayName string `json:"display_name"` // label for UI
	Detected    bool   `json:"detected"`     // true if this location was found
	Notes       string `json:"notes"`        // free-form notes
}

// CloudDriveDetector is a thin wrapper used by other components.
type CloudDriveDetector struct{}

// NewCloudDriveDetector constructs a detector.
func NewCloudDriveDetector() *CloudDriveDetector {
	return &CloudDriveDetector{}
}

// DetectAll discovers cloud storage locations using registry + common paths.
func (d *CloudDriveDetector) DetectAll() ([]CloudDrive, error) {
	return DetectCloudDrives(), nil
}

// DetectCloudDrives aggregates per-provider checks and deduplicates results.
func DetectCloudDrives() []CloudDrive {
	var drives []CloudDrive

	// OneDrive
	drives = append(drives, detectOneDrive()...)

	// Google Drive (Drive for desktop / streaming)
	drives = append(drives, detectGoogleDrive()...)

	// Dropbox
	drives = append(drives, detectDropbox()...)

	// iCloud
	drives = append(drives, detectICloud()...)

	// Box
	drives = append(drives, detectBox()...)

	// Deduplicate by mount point or display/provider combo.
	seen := make(map[string]bool)
	out := make([]CloudDrive, 0, len(drives))

	for _, dr := range drives {
		key := strings.ToLower(strings.TrimSpace(dr.MountPoint))
		if key == "" {
			key = strings.ToLower(dr.DisplayName + "|" + dr.Provider)
		}
		if key == "" {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, dr)
	}

	if len(out) == 0 {
		log.Println("DetectCloudDrives: no cloud providers found")
	} else {
		log.Printf("DetectCloudDrives: found %d provider mount(s)", len(out))
		for _, d := range out {
			log.Printf("  [%s] %s at %s (mounted=%v)", d.Type, d.DisplayName, d.MountPoint, d.Mounted)
		}
	}

	return out
}

// -----------------
// Provider helpers
// -----------------

func detectOneDrive() []CloudDrive {
	var drives []CloudDrive

	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\OneDrive\Accounts`,
		registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS,
	)
	if err != nil {
		return drives
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(0)
	if err != nil {
		return drives
	}

	for _, sub := range subkeys {
		accountKey, err := registry.OpenKey(
			registry.CURRENT_USER,
			fmt.Sprintf(`Software\Microsoft\OneDrive\Accounts\%s`, sub),
			registry.QUERY_VALUE,
		)
		if err != nil {
			continue
		}

		email, _, _ := accountKey.GetStringValue("UserEmail")
		root, _, _ := accountKey.GetStringValue("UserFolder")
		accountKey.Close()

		if root == "" {
			continue
		}

		mounted := checkPathExists(root)
		display := "OneDrive"
		if email != "" {
			display = fmt.Sprintf("OneDrive - %s", email)
		}

		drives = append(drives, CloudDrive{
			Type:        "OneDrive",
			Provider:    "OneDrive",
			MountPoint:  root,
			Path:        root,
			Email:       email,
			Mounted:     mounted,
			DisplayName: display,
			Detected:    mounted,
			Notes:       "found via registry",
		})
	}

	return drives
}

func detectGoogleDrive() []CloudDrive {
	var drives []CloudDrive

	// Primary: DriveFS registry entries (Google Drive for Desktop / streaming).
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Google\DriveFS`,
		registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS,
	)
	if err == nil {
		subkeys, err := key.ReadSubKeyNames(0)
		if err == nil {
			for _, sub := range subkeys {
				if sub == "Share" {
					continue
				}
				accountKey, err := registry.OpenKey(
					registry.CURRENT_USER,
					fmt.Sprintf(`Software\Google\DriveFS\%s`, sub),
					registry.QUERY_VALUE,
				)
				if err != nil {
					continue
				}

				mountPoint, _, _ := accountKey.GetStringValue("MountPoint")
				accountKey.Close()

				if mountPoint == "" {
					continue
				}

				mounted := checkPathExists(mountPoint)
				email := extractEmailFromPath(mountPoint)
				display := "Google Drive"
				if email != "" {
					display = fmt.Sprintf("Google Drive - %s", email)
				}

				drives = append(drives, CloudDrive{
					Type:        "GoogleDrive",
					Provider:    "Google Drive",
					MountPoint:  mountPoint,
					Path:        mountPoint,
					Email:       email,
					Mounted:     mounted,
					DisplayName: display,
					Detected:    mounted,
					Notes:       "found via registry",
				})
			}
		}
		key.Close()
	}

	// Fallback: legacy / common paths in user profile.
	userProfile := os.Getenv("USERPROFILE")
	common := []string{
		filepath.Join(userProfile, "Google Drive"),
		filepath.Join(userProfile, "My Drive"),
		// Google Drive Stream/File Stream paths
		filepath.Join(userProfile, "Google Drive Streaming"),
		filepath.Join(userProfile, "GoogleDriveStreaming"),
		filepath.Join(userProfile, "Google Drive - Streaming"),
	}

	// Also scan for any folder that starts with "Google" in user profile
	files, err := os.ReadDir(userProfile)
	if err == nil {
		for _, file := range files {
			if file.IsDir() {
				name := strings.ToLower(file.Name())
				if strings.Contains(name, "google") && (strings.Contains(name, "drive") || strings.Contains(name, "stream")) {
					googlePath := filepath.Join(userProfile, file.Name())
					common = append(common, googlePath)
				}
			}
		}
	}

	for _, p := range common {
		if !checkPathExists(p) {
			continue
		}
		drives = append(drives, CloudDrive{
			Type:        "GoogleDrive",
			Provider:    "Google Drive",
			MountPoint:  p,
			Path:        p,
			Mounted:     true,
			DisplayName: "Google Drive",
			Detected:    true,
			Notes:       "found via common path",
		})
	}

	return drives
}

func detectDropbox() []CloudDrive {
	var drives []CloudDrive

	// Registry hint.
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Dropbox\ks`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		if path, _, err := key.GetStringValue("Client"); err == nil && path != "" {
			drives = append(drives, CloudDrive{
				Type:        "Dropbox",
				Provider:    "Dropbox",
				MountPoint:  path,
				Path:        path,
				Mounted:     checkPathExists(path),
				DisplayName: "Dropbox",
				Detected:    checkPathExists(path),
				Notes:       "found via registry",
			})
		}
		key.Close()
	}

	// Fallback to USERPROFILE\Dropbox.
	userProfile := os.Getenv("USERPROFILE")
	common := filepath.Join(userProfile, "Dropbox")
	if checkPathExists(common) {
		drives = append(drives, CloudDrive{
			Type:        "Dropbox",
			Provider:    "Dropbox",
			MountPoint:  common,
			Path:        common,
			Mounted:     true,
			DisplayName: "Dropbox",
			Detected:    true,
			Notes:       "found via common path",
		})
	}

	return drives
}

func detectICloud() []CloudDrive {
	var drives []CloudDrive

	userProfile := os.Getenv("USERPROFILE")
	root := filepath.Join(userProfile, "iCloudDrive")
	if checkPathExists(root) {
		drives = append(drives, CloudDrive{
			Type:        "iCloud",
			Provider:    "iCloud Drive",
			MountPoint:  root,
			Path:        root,
			Mounted:     true,
			DisplayName: "iCloud Drive",
			Detected:    true,
			Notes:       "found via common path",
		})
	}

	return drives
}

func detectBox() []CloudDrive {
	var drives []CloudDrive

	userProfile := os.Getenv("USERPROFILE")
	root := filepath.Join(userProfile, "Box")
	if checkPathExists(root) {
		drives = append(drives, CloudDrive{
			Type:        "Box",
			Provider:    "Box",
			MountPoint:  root,
			Path:        root,
			Mounted:     true,
			DisplayName: "Box",
			Detected:    true,
			Notes:       "found via common path",
		})
	}

	return drives
}

// -------------
// Misc helpers
// -------------

func checkPathExists(path string) bool {
	if path == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	_, err := os.Stat(cleanPath)
	return err == nil
}

func extractEmailFromPath(path string) string {
	cleanPath := filepath.Clean(path)
	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, p := range parts {
		if strings.Contains(p, "@") {
			return p
		}
	}
	return ""
}

// InitializeCloudRanges is a no-op kept for compatibility.
func InitializeCloudRanges() error {
	return nil
}

// GetCloudProvider is a conservative stub (IP-based detection not used here).
func GetCloudProvider(ip string) string {
	return "Unknown"
}
