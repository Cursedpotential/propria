package previewmodel

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnknownArtifactCoordinatesNeverInventZeroUUID(t *testing.T) {
	for _, value := range []any{OperationResults{Status: "missing"}, SourceFingerprint{Status: "unavailable"}} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "00000000-0000-0000-0000-000000000000") || !strings.Contains(string(data), `"source_version_id":null`) {
			t.Fatalf("unknown coordinate became invented identity: %s", data)
		}
	}
}
