package profferworker

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/sourcepreview"
)

// newSourcePreviewActivity builds the optional read-only raw-source viewer using existing object-store configuration.
// Inputs: stores plus SOURCE_PREVIEW_CALLBACK_URL and mounted PROFFER_SERVICE_TOKEN_FILE. Outputs: optional Activity.
// Effects: reads config only; choose for raw inspection with no changes to ingestion or the normalized preview path.
func newSourcePreviewActivity(stores objectstores.Stores) (*sourcepreview.Activity, error) {
	callback := strings.TrimSpace(os.Getenv("SOURCE_PREVIEW_CALLBACK_URL"))
	if callback == "" {
		return nil, nil
	}
	u, err := url.Parse(callback)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Scheme != "https" && u.Scheme != "http" || u.Path != "" && u.Path != "/" {
		return nil, sourcepreview.ErrInvalid
	}
	provider, err := sourcepreview.NewProvider(stores, map[string]string{"r2": os.Getenv("CASEBIBLE_R2_CONFIG_PATH"), "b2": os.Getenv("B2_CONFIG_PATH")})
	if err != nil {
		return nil, err
	}
	tokenFile := strings.TrimSpace(os.Getenv("PROFFER_SERVICE_TOKEN_FILE"))
	if tokenFile == "" {
		tokenFile = "/run/secrets/proffer-service-token"
	}
	return &sourcepreview.Activity{Provider: provider, CallbackURL: strings.TrimRight(callback, "/"), TokenFile: tokenFile, Client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
