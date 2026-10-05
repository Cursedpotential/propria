// Byline: Codex, GPT-6, 2026-10-04. Retained shared library validation starter surface.
package runtimeapi

import (
	"context"
	"crypto/hmac"
	"errors"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// ToolkitLibraryValidator starts idempotent validation for a retained shared proposal.
// Inputs: proposal record id. Outputs: workflow/run ids. Effects: Temporal start only.
// Choose for service dispatch after a shared proposal is durable; never accepts personal record bodies or validation flags.
type ToolkitLibraryValidator interface {
	StartLibraryValidation(context.Context, string) (string, string, error)
}

// NewToolkitLibraryValidationHandler mounts authenticated retained-proposal validation.
// Inputs: workflow starter and mounted service credential. Outputs: HTTP handler or configuration error.
// Effects: request authentication and bounded workflow dispatch. Choose alongside the existing starter routes.
func NewToolkitLibraryValidationHandler(starter ToolkitLibraryValidator, tokenFile string) (http.Handler, error) {
	if starter == nil {
		return nil, errors.New("toolkit validation requires workflow starter")
	}
	if _, err := loadServiceToken(tokenFile); err != nil {
		return nil, err
	}
	allow, err := toolkitValidationServiceNetworks(os.Getenv("TOOLKIT_VALIDATION_SERVICE_CIDRS"))
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /toolkit/library/validate", toolkitValidationServiceAuth(tokenFile, allow, func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ProposalID string `json:"proposal_id"`
		}
		if err := decodePreviewJSON(w, r, &input); err != nil {
			previewError(w, http.StatusBadRequest, err)
			return
		}
		if !toolkitProposalPattern.MatchString(input.ProposalID) {
			previewError(w, http.StatusUnprocessableEntity, errors.New("invalid library proposal id"))
			return
		}
		workflowID, runID, err := starter.StartLibraryValidation(r.Context(), input.ProposalID)
		if err != nil {
			previewError(w, http.StatusServiceUnavailable, errors.New("validation workflow could not start; retained proposal can be retried"))
			return
		}
		previewJSON(w, http.StatusAccepted, map[string]string{"workflow_id": workflowID, "run_id": runID})
	}))
	return mux, nil
}

var toolkitProposalPattern = regexp.MustCompile(`^library_proposal:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

// toolkitValidationServiceNetworks admits only explicit private service networks beside the existing tailnet.
// Inputs: comma-separated CIDRs. Outputs: parsed network allowlist or a configuration error.
// Effects: none. Choose for container-to-starter validation only; no other route's authentication changes.
func toolkitValidationServiceNetworks(value string) ([]*net.IPNet, error) {
	_, tailnet, _ := net.ParseCIDR("100.64.0.0/10")
	result := []*net.IPNet{tailnet}
	for _, value := range strings.Split(value, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		ip, network, err := net.ParseCIDR(value)
		if err != nil || ip.To4() == nil || !ip.IsPrivate() {
			return nil, errors.New("toolkit validation service CIDRs must be explicit private IPv4 networks")
		}
		ones, _ := network.Mask.Size()
		if ones < 16 {
			return nil, errors.New("toolkit validation private service network must be /16 or narrower")
		}
		result = append(result, network)
	}
	return result, nil
}

// toolkitValidationServiceAuth checks the actual socket peer and the dedicated mounted service token.
// Inputs: credential file and private peer allowlist. Outputs: authenticated handler.
// Effects: authorization only; never trusts forwarded identity headers. Choose solely for retained-proposal validation dispatch.
func toolkitValidationServiceAuth(tokenFile string, networks []*net.IPNet, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		allowed := false
		if err == nil && ip != nil {
			for _, network := range networks {
				if network.Contains(ip) {
					allowed = true
					break
				}
			}
		}
		credential, tokenErr := loadServiceToken(tokenFile)
		header := r.Header.Get("Authorization")
		if !allowed || tokenErr != nil || !strings.HasPrefix(header, "Bearer ") || !hmac.Equal([]byte(strings.TrimPrefix(header, "Bearer ")), credential) {
			previewError(w, http.StatusUnauthorized, errors.New("toolkit validation service authorization required"))
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next(w, r)
	}
}
