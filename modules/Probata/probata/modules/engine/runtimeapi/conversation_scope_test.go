package runtimeapi

import (
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestConversationExplicitCasePairRejectsMismatch checks current owner scope reaches RequestInput without silently replacing disagreement.
// Inputs: explicit valid/mismatched court IDs and LIVE. Output: admitted exact pair or 422.
// Effects: HTTP recorder only; choose for BFF full-pair forwarding regression.
func TestConversationExplicitCasePairRejectsMismatch(t *testing.T) {
	for _, court := range []string{caseidentity.AuthoritativeCourtCaseID, "55555555-5555-4555-8555-555555555555"} {
		raw, _ := json.Marshal(map[string]any{"operating_mode": "LIVE", "matter_id": caseidentity.AuthoritativeMatterID, "court_case_id": court})
		h := &ConversationExtractionHTTPHandler{}
		response := httptest.NewRecorder()
		body, ok := h.decode(response, httptest.NewRequest("POST", "/", strings.NewReader(string(raw))))
		if court == caseidentity.AuthoritativeCourtCaseID {
			if !ok || body.CourtCaseID != court {
				t.Fatal("explicit case lost")
			}
		} else if ok || response.Code != 422 {
			t.Fatal("mismatched case admitted")
		}
	}
}
