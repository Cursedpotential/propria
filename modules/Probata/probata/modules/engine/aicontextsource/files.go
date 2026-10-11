// Package aicontextsource reads bounded native context bundles and verifies the copied original.
package aicontextsource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxBytes = 32 << 20

// Identity is the unaltered registered source and provider coordinates carried by Python bundles.
// Inputs: native producer source identity. Outputs: typed metadata. Effects: none.
// Pick for equality checks against context.source registration, not as a custody object.
type Identity struct {
	SourceRef         string  `json:"source_ref"`
	ProviderVersionID *string `json:"provider_version_id"`
	PackageRef        *string `json:"package_ref"`
	SourceFormat      string  `json:"source_format"`
	SourceSHA256      string  `json:"source_sha256,omitempty"`
}

// Prepared identifies a copied exact original and its first-party source hash.
// Inputs: prepared.json. Outputs: original locator, digest and source identity. Effects: none.
// Pick for native context proof before staging or owner decision.
type Prepared struct {
	ContractVersion string   `json:"contract_version"`
	Stage           string   `json:"stage"`
	Source          Identity `json:"source"`
	OriginalRef     string   `json:"original_ref"`
	OriginalSHA256  string   `json:"original_sha256"`
}

// Root returns the existing Python ai-context output root on the shared mounted volume.
// Inputs: AI_CONTEXT_ROOT or its existing default. Outputs: absolute root. Effects: none.
// Pick for local prepared/original reads; it is never a source enrollment path.
func Root() string {
	if value := os.Getenv("AI_CONTEXT_ROOT"); value != "" {
		return value
	}
	return "/data/proffer/derive-scratch/ai-content/context"
}

// ReadPrepared verifies the bounded prepared bundle and exact original bytes below the configured root.
// Inputs: prepared file URI. Outputs: typed bundle and original bytes. Effects: read-only filesystem access.
// Pick before native candidate staging, review and graph projection to recheck the original hash.
func ReadPrepared(ref string) (Prepared, []byte, error) {
	raw, path, err := ReadFile(ref, "prepared.json")
	if err != nil {
		return Prepared{}, nil, err
	}
	var prepared Prepared
	if err := json.Unmarshal(raw, &prepared); err != nil {
		return Prepared{}, nil, fmt.Errorf("native context prepared bundle: %w", err)
	}
	if prepared.ContractVersion != "ai-context-v1" || prepared.Stage != "prepared" || prepared.Source.SourceRef == "" || prepared.OriginalRef == "" || !digest(prepared.OriginalSHA256) {
		return Prepared{}, nil, errors.New("native context prepared bundle is incomplete")
	}
	original, originalPath, err := ReadFile(prepared.OriginalRef, "")
	if err != nil {
		return Prepared{}, nil, err
	}
	if filepath.Dir(originalPath) != filepath.Dir(path) || (filepath.Base(originalPath) != "original.md" && filepath.Base(originalPath) != "original.json") {
		return Prepared{}, nil, errors.New("native context original differs from prepared directory")
	}
	hash := sha256.Sum256(original)
	if hex.EncodeToString(hash[:]) != prepared.OriginalSHA256 || (prepared.Source.SourceSHA256 != "" && prepared.Source.SourceSHA256 != prepared.OriginalSHA256) {
		return Prepared{}, nil, errors.New("native context original SHA256 differs")
	}
	return prepared, original, nil
}

// ReadFile reads one exact regular file URI below the existing context root without following links.
// Inputs: file URI and optional required base name. Outputs: bounded bytes and canonical path.
// Effects: read-only filesystem access. Pick for prepared, candidates and original bundle files.
func ReadFile(ref, base string) ([]byte, string, error) {
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "file" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, "", errors.New("native context needs an exact local file URI")
	}
	uriPath := u.Path
	if len(uriPath) >= 3 && uriPath[0] == '/' && uriPath[2] == ':' {
		uriPath = uriPath[1:]
	}
	path := filepath.Clean(filepath.FromSlash(uriPath))
	root := filepath.Clean(Root())
	if !filepath.IsAbs(root) || !filepath.IsAbs(path) || (base != "" && filepath.Base(path) != base) {
		return nil, "", errors.New("native context file path is invalid")
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, "", errors.New("native context file escaped mounted root")
	}
	current := root
	for _, part := range append([]string{""}, strings.Split(rel, string(os.PathSeparator))...) {
		if part != "" {
			current = filepath.Join(current, part)
		}
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, "", errors.New("native context file path is unavailable or linked")
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxBytes {
		return nil, "", errors.New("native context file is not a bounded regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) < 1 || len(raw) > maxBytes {
		return nil, "", errors.New("native context file exceeds byte bound")
	}
	return raw, path, nil
}

// Hash computes the lowercase SHA256 of an already bounded bundle file.
// Inputs: bytes. Outputs: digest. Effects: none. Pick for optional caller hash comparison.
func Hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// ValidDigest checks a complete lowercase SHA256 without opening a source.
// Inputs: digest text. Outputs: validity. Effects: none. Pick for original/candidate pins.
func ValidDigest(value string) bool { return digest(value) }

// VerifyQuote checks an absolute Unicode span against the copied native original.
// Inputs: declared format, original bytes, native pointer, span and exact quote. Outputs: span SHA256 or error.
// Effects: none. Pick before staging any model assertion; never derive assertion text from the quote.
func VerifyQuote(format string, original []byte, pointer string, start, end int, quote string) (string, error) {
	if start < 0 || end <= start || quote == "" {
		return "", errors.New("native context source span is invalid")
	}
	var value string
	if format == "gemini_markdown" || format == "chatgpt_markdown" || format == "claude_markdown" {
		if pointer != "" {
			return "", errors.New("native Markdown cannot have a JSON pointer")
		}
		if !utf8.Valid(original) {
			return "", errors.New("native Markdown is not UTF-8")
		}
		value = string(original)
	} else {
		switch format {
		case "chatgpt", "claude", "claude_ai_export_json", "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json", "claude_conversations_json":
		default:
			return "", errors.New("native context source format is unsupported")
		}
		if pointer == "" || !strings.HasPrefix(pointer, "/") {
			return "", errors.New("native JSON requires a pointer")
		}
		var document any
		if err := json.Unmarshal(original, &document); err != nil {
			return "", fmt.Errorf("native JSON malformed: %w", err)
		}
		var node any = document
		for _, encoded := range strings.Split(pointer[1:], "/") {
			for i := 0; i < len(encoded); i++ {
				if encoded[i] == '~' {
					if i+1 >= len(encoded) || encoded[i+1] != '0' && encoded[i+1] != '1' {
						return "", errors.New("native JSON pointer escape invalid")
					}
					i++
				}
			}
			part := strings.ReplaceAll(strings.ReplaceAll(encoded, "~1", "/"), "~0", "~")
			switch typed := node.(type) {
			case map[string]any:
				node = typed[part]
			case []any:
				index, err := strconv.Atoi(part)
				if err != nil || index < 0 || index >= len(typed) || strconv.Itoa(index) != part {
					return "", errors.New("native JSON index invalid")
				}
				node = typed[index]
			default:
				return "", errors.New("native JSON pointer absent")
			}
		}
		parsed, ok := node.(string)
		if !ok {
			return "", errors.New("native JSON pointer is not text")
		}
		value = parsed
	}
	points := []rune(value)
	if end > len(points) || string(points[start:end]) != quote {
		return "", errors.New("native context quote differs from exact original span")
	}
	return Hash([]byte(quote)), nil
}

// digest rejects missing, nonhex and mixed-case hashes.
// Inputs: digest text. Outputs: validity. Effects: none. Pick inside bounded source validation.
func digest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
