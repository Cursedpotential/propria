package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Files retains immutable bounded JSON in the worker's existing shared derive root.
// Inputs: configured absolute root. Outputs: references/decoded values. Effects: derived files only; choose outside history.
type Files struct{ root string }

// NewFiles verifies an existing non-symlink root without creating runtime infrastructure.
func NewFiles(root string) (*Files, error) {
	if !filepath.IsAbs(root) {
		return nil, ErrInvalid
	}
	clean := filepath.Clean(root)
	resolved, e := filepath.EvalSymlinks(clean)
	if e != nil || resolved != clean {
		return nil, ErrInvalid
	}
	info, e := os.Stat(clean)
	if e != nil || !info.IsDir() {
		return nil, ErrInvalid
	}
	return &Files{clean}, nil
}

// Put atomically retains exact derived JSON and fsyncs its directory before returning a reference.
// Inputs: canonical run UUID, logical revision key and bounded value. Outputs: immutable ref; effects derived file creation only.
func (f *Files) Put(runID, key string, v any) (Ref, error) {
	if f == nil || !ID(runID) || !bounded(key, 200, true) || strings.ContainsAny(key, "/\\.") {
		return Ref{}, ErrInvalid
	}
	raw, e := json.Marshal(v)
	if e != nil || len(raw) == 0 || len(raw) > MaxBytes {
		return Ref{}, ErrInvalid
	}
	digest := DigestBytes(raw)
	dir := filepath.Join(f.root, runID)
	if e = os.Mkdir(dir, 0700); e != nil && !os.IsExist(e) {
		return Ref{}, e
	}
	resolved, e := filepath.EvalSymlinks(dir)
	if e != nil || resolved != dir {
		return Ref{}, ErrInvalid
	}
	target := filepath.Join(dir, key+"-"+digest+".json")
	if prior, e := os.ReadFile(target); e == nil {
		if !bytes.Equal(prior, raw) {
			return Ref{}, ErrConflict
		}
	} else {
		temp, e := os.CreateTemp(dir, ".pending-")
		if e != nil {
			return Ref{}, e
		}
		name := temp.Name()
		defer os.Remove(name)
		if e = temp.Chmod(0400); e == nil {
			_, e = temp.Write(raw)
		}
		if e == nil {
			e = temp.Sync()
		}
		closeErr := temp.Close()
		if e == nil {
			e = closeErr
		}
		if e != nil {
			return Ref{}, e
		}
		if e = os.Link(name, target); e != nil {
			if prior, readErr := os.ReadFile(target); readErr != nil || !bytes.Equal(prior, raw) {
				return Ref{}, ErrConflict
			}
		}
		directory, e := os.Open(dir)
		if e != nil {
			return Ref{}, e
		}
		e = directory.Sync()
		directory.Close()
		if e != nil {
			return Ref{}, e
		}
		root, e := os.Open(f.root)
		if e != nil {
			return Ref{}, e
		}
		e = root.Sync()
		root.Close()
		if e != nil {
			return Ref{}, e
		}
	}
	u := url.URL{Scheme: "file", Path: target}
	return Ref{URI: u.String(), SHA256: digest, Bytes: int64(len(raw))}, nil
}

// Read validates an exact bounded derivative reference inside the configured root before decoding.
// Inputs: immutable ref and destination. Outputs: decoded value. Effects: derived file read only; no source mounts.
func (f *Files) Read(ref Ref, v any) error {
	if f == nil || ref.Bytes < 1 || ref.Bytes > MaxBytes || len(ref.SHA256) != 64 {
		return ErrInvalid
	}
	u, e := url.Parse(ref.URI)
	if e != nil || u.Scheme != "file" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return ErrInvalid
	}
	p := filepath.Clean(u.Path)
	rel, e := filepath.Rel(f.root, p)
	if e != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return ErrScope
	}
	resolved, e := filepath.EvalSymlinks(p)
	if e != nil || resolved != p {
		return ErrInvalid
	}
	stream, e := os.Open(p)
	if e != nil {
		return e
	}
	defer stream.Close()
	raw, e := io.ReadAll(io.LimitReader(stream, ref.Bytes+1))
	if e != nil || int64(len(raw)) != ref.Bytes || DigestBytes(raw) != ref.SHA256 {
		return fmt.Errorf("%w: derivative reference identity differs", ErrInvalid)
	}
	return json.Unmarshal(raw, v)
}
