package humancontext

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

// nativeSearchObject preserves the former message projection for reuse; the live chunk publisher never calls it.
// Inputs: native message. Output: unwired search object. Effects: none.
func nativeSearchObject(source Source, manifestRef string, record Record) (SearchObject, error) {
	var native struct {
		Kind         string          `json:"record_kind"`
		Body         string          `json:"body"`
		Direction    string          `json:"direction"`
		OccurredAt   *string         `json:"occurred_at"`
		Locator      json.RawMessage `json:"source_locator"`
		Participants json.RawMessage `json:"participants"`
	}
	if err := json.Unmarshal(record.Native, &native); err != nil {
		return SearchObject{}, err
	}
	if native.Kind != "message" || len(native.Locator) == 0 || string(native.Locator) == "null" {
		return SearchObject{}, errors.New("native index needs a message with original locator")
	}
	coordinate := Contract + ":" + source.SourceVersionRef + ":" + strconv.FormatInt(record.Ordinal, 10) + ":" + string(native.Locator)
	id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(coordinate)).String()
	hash := sha256.Sum256(record.Native)
	properties := map[string]any{"origin_system": "probata-native-context", "source_version_id": source.SourceVersionRef, "record_kind": "message", "event_kind": "message", "body": native.Body, "search_text": native.Body, "direction": native.Direction, "source_format": source.Format, "parser_id": "duckdb-native-sms-lexical", "parser_version": "1.0.0", "extraction_attempt_ref": manifestRef, "content_sha256": hex.EncodeToString(hash[:]), "dedup_key": coordinate, "dedup_key_construction": "native-source-version-ordinal-locator-v1", "object_id_construction": "uuid5-native-source-version-ordinal-locator-v1", "provenance": []string{string(native.Locator)}, "ingest_run_id": source.SourceVersionRef, "n_sources": 1}
	var participants []string
	if err := json.Unmarshal(native.Participants, &participants); err != nil {
		var entries []struct {
			Address *string `json:"address"`
		}
		if err := json.Unmarshal(native.Participants, &entries); err != nil {
			return SearchObject{}, err
		}
		for _, entry := range entries {
			if entry.Address != nil {
				participants = append(participants, *entry.Address)
			}
		}
	}
	clean := participants[:0]
	for _, participant := range participants {
		if strings.TrimSpace(participant) != "" {
			clean = append(clean, participant)
		}
	}
	if len(clean) > 0 {
		properties["participants"] = clean
	}
	if native.OccurredAt != nil {
		properties["occurred_at"] = *native.OccurredAt
		properties["event_ts"] = *native.OccurredAt
		properties["sort_ts"] = *native.OccurredAt
	}
	if source.OwnerPersonRef != "" || source.PerspectivePersonRef != "" {
		attribution, _ := json.Marshal(struct {
			Owner             string `json:"owner_person_ref,omitempty"`
			PerspectivePerson string `json:"perspective_person_ref,omitempty"`
			Perspective       string `json:"perspective,omitempty"`
		}{source.OwnerPersonRef, source.PerspectivePersonRef, source.Perspective})
		properties["provenance"] = []string{string(native.Locator), string(attribution)}
	}
	if native.Body == "" {
		properties["search_text"] = "human message " + string(native.Participants)
	}
	return SearchObject{ID: id, Properties: properties}, nil
}
