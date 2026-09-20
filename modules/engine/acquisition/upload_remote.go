// Byline: Claude Code · Fable 5.1 · 2026-09-20 (cross-host upload:// resolution, D-132)
package acquisition

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// UploadObjectPathPrefix is where the host that owns the upload seal root
// serves sealed upload objects by digest: GET <prefix>{sha256}.
const UploadObjectPathPrefix = "/acquisition/upload/"

// NewUploadObjectHandler serves one sealed upload object per request to
// tailnet peers, addressed only by its SHA-256. It exists because upload://
// objects live on the host that accepted them, while D-132 requires the tool
// gateway to resolve locators from whichever host it runs on: found live
// 2026-09-20, every upload:// source died in assess_source_repair_activity with
// `no acquisition resolver registered for scheme "upload"`. Mount it at
// "GET "+UploadObjectPathPrefix+"{sha256}". The peer rule is the ingress's own.
func NewUploadObjectHandler(root string) (http.Handler, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("acquisition: upload object handler root is required")
	}
	if _, err := prepareSealRoot(root); err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorizedTailnetPeer(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		digestHex, err := parseUploadRef(proffer.Ref(uploadRefScheme + "://" + r.PathValue("sha256")))
		if err != nil {
			http.Error(w, "malformed upload digest", http.StatusBadRequest)
			return
		}
		objectPath, err := digestObjectPath(root, digestHex)
		if err != nil {
			http.Error(w, "malformed upload digest", http.StatusBadRequest)
			return
		}
		object, err := os.Open(objectPath)
		if err != nil {
			http.Error(w, "upload object not found", http.StatusNotFound)
			return
		}
		defer object.Close()
		info, err := object.Stat()
		if err != nil || !info.Mode().IsRegular() {
			http.Error(w, "upload object not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, object)
	}), nil
}

// ObjectHandler serves the objects this ingress sealed; see NewUploadObjectHandler.
func (u *UploadIngress) ObjectHandler() (http.Handler, error) {
	return NewUploadObjectHandler(u.cfg.Root)
}

// NewRemoteUploadResolver resolves "upload://<sha256-hex>" on a host that does
// not hold the upload seal root: it fetches the object from originURL, seals it
// into sealRoot exactly like every other provider, and fails closed unless the
// sealed digest equals the digest in the reference. maxBytes is the same bound
// the ingress enforces. A rejected download can leave an object under its own
// (different) digest in sealRoot; content addressing makes that inert.
func NewRemoteUploadResolver(sealRoot, originURL string, maxBytes int64, client *http.Client) (platformpostgres.ImmutableAcquisitionResolver, error) {
	if strings.TrimSpace(sealRoot) == "" {
		return nil, errors.New("acquisition: remote upload resolver seal root is required")
	}
	origin, err := url.Parse(strings.TrimSpace(originURL))
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" {
		return nil, fmt.Errorf("acquisition: remote upload origin %q must be an http(s) URL", originURL)
	}
	if maxBytes <= 0 {
		return nil, errors.New("acquisition: remote upload resolver max bytes must be positive")
	}
	if client == nil {
		return nil, errors.New("acquisition: remote upload resolver requires an HTTP client")
	}
	if _, err := prepareSealRoot(sealRoot); err != nil {
		return nil, err
	}
	base := strings.TrimRight(origin.String(), "/") + UploadObjectPathPrefix
	return func(ctx context.Context, ref proffer.Ref) (platformpostgres.ImmutableAcquisition, error) {
		digestHex, err := parseUploadRef(ref)
		if err != nil {
			return platformpostgres.ImmutableAcquisition{}, err
		}
		wantDigest, err := hex.DecodeString(digestHex)
		if err != nil {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: decode uploaded object digest: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+digestHex, nil)
		if err != nil {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: build upload origin request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: fetch uploaded object %s: %w", digestHex, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: upload origin returned %d for %s", resp.StatusCode, digestHex)
		}
		sealed, err := sealStream(ctx, sealRoot, io.LimitReader(resp.Body, maxBytes+1))
		if err != nil {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: seal uploaded object %s: %w", digestHex, err)
		}
		if sealed.ByteLength > maxBytes {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: uploaded object %s exceeds the %d byte bound", digestHex, maxBytes)
		}
		if !bytes.Equal(sealed.ContentSHA256, wantDigest) {
			return platformpostgres.ImmutableAcquisition{}, fmt.Errorf("acquisition: upload origin returned bytes that do not hash to %s", digestHex)
		}
		return sealed, nil
	}, nil
}
