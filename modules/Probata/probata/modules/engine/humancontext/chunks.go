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

// NativeChunk carries one bounded externally planned conversation slice with original locators.
// Inputs: Python Neural plan JSONL. Outputs: native references only; never legacy normalized IDs.
type NativeChunk struct {
	ThreadID        string              `json:"thread_id"`
	ChunkIndex      int64               `json:"chunk_index"`
	WindowIndex     int64               `json:"window_index"`
	Text            string              `json:"text"`
	MessageRefs     []string            `json:"message_refs"`
	Members         []NativeChunkMember `json:"members"`
	Chunker         string              `json:"chunker"`
	ChunkerVersion  string              `json:"chunker_version"`
	BoundaryVersion string              `json:"boundary_version"`
	Overlap         int64               `json:"overlap"`
}

// NativeChunkMember retains the decoded body segment and original XML record coordinate.
// Inputs: bounded chunk plan. Outputs: exact viewer locations, not invented message identities.
type NativeChunkMember struct {
	Ordinal   int64 `json:"ordinal"`
	Segment   int64 `json:"segment"`
	BodyStart int64 `json:"body_start"`
	BodyEnd   int64 `json:"body_end"`
	Locator   struct {
		SourceRef        string `json:"source_ref"`
		SourceVersionRef string `json:"source_version_ref"`
		Ordinal          int64  `json:"record_ordinal"`
		Start            int64  `json:"start"`
		End              int64  `json:"end"`
	} `json:"locator"`
	At           *string  `json:"at"`
	Participants []string `json:"participants"`
}

// nativeChunkObject maps a source-pinned Neural plan to existing ProfferChunks properties.
// Inputs: registered source and bounded native chunk. Output: deterministic technical index object.
// Effects: none; no schema changes, normalized IDs, fabricated clocks or evidence decisions.
func nativeChunkObject(source Source, manifestRef string, chunk NativeChunk) (SearchObject, error) {
	if chunk.ThreadID == "" || chunk.Chunker != "neural_distilbert" || chunk.ChunkerVersion == "" || chunk.BoundaryVersion == "" || len(chunk.Text) == 0 || len(chunk.Text) > 7500 || len(chunk.MessageRefs) == 0 || len(chunk.Members) != len(chunk.MessageRefs) {
		return SearchObject{}, errors.New("invalid native conversation chunk")
	}
	var first, last *string
	participants := []string{}
	seen := map[string]bool{}
	ordinals := map[int64]bool{}
	for j, m := range chunk.Members {
		expected := "native-record:" + source.SourceVersionRef + ":" + strconv.FormatInt(m.Ordinal, 10) + ":" + strconv.FormatInt(m.BodyStart, 10) + ":" + strconv.FormatInt(m.BodyEnd, 10)
		if m.Ordinal < 1 || m.BodyStart < 0 || m.BodyEnd < m.BodyStart || m.Locator.SourceVersionRef != source.SourceVersionRef || m.Locator.SourceRef != source.SourceRef || m.Locator.Ordinal != m.Ordinal || chunk.MessageRefs[j] != expected {
			return SearchObject{}, errors.New("native chunk member source differs")
		}
		ordinals[m.Ordinal] = true
		if m.At != nil {
			if first == nil || *m.At < *first {
				first = m.At
			}
			if last == nil || *m.At > *last {
				last = m.At
			}
		}
		for _, p := range m.Participants {
			if strings.TrimSpace(p) != "" && !seen[p] {
				participants = append(participants, p)
				seen[p] = true
			}
		}
	}
	digest := sha256.Sum256([]byte(chunk.ThreadID))
	content := sha256.Sum256([]byte(chunk.Text))
	coordinate := IndexKind + ":" + source.SourceVersionRef + ":" + chunk.ThreadID + ":" + strconv.FormatInt(chunk.ChunkIndex, 10) + ":" + chunk.ChunkerVersion + ":" + hex.EncodeToString(content[:])
	props := map[string]any{"text": chunk.Text, "record_kind": "conversation_chunk", "corpus": "native_human_context", "origin_system": "probata-native-context", "thread_id": chunk.ThreadID, "thread_digest": hex.EncodeToString(digest[:]), "content_hash": hex.EncodeToString(content[:]), "first_message_id": chunk.MessageRefs[0], "last_message_id": chunk.MessageRefs[len(chunk.MessageRefs)-1], "message_ids": chunk.MessageRefs, "source_version_ids": []string{source.SourceVersionRef}, "source_version_id": source.SourceVersionRef, "chunker": chunk.Chunker, "chunker_version": chunk.ChunkerVersion, "overlap": chunk.Overlap, "chunk_index": chunk.ChunkIndex, "message_count": len(ordinals), "ingest_run_id": source.SourceVersionRef, "object_id_construction": "uuid5-native-source-conversation-window-chunk-v1"}
	if len(participants) > 0 {
		props["participant_names"] = participants
	}
	if first != nil {
		props["start_at"] = *first
	}
	if last != nil {
		props["end_at"] = *last
	}
	// The immutable plan manifest is discovered through source metadata; no unsupported schema property is added.
	_ = manifestRef
	return SearchObject{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte(coordinate)).String(), Properties: props}, nil
}

// EncodeNativeChunk is the bounded fixture/adapter serialization contract used outside Temporal payloads.
// Input: one chunk. Output: JSON bytes. Effects: none.
func EncodeNativeChunk(chunk NativeChunk) ([]byte, error) { return json.Marshal(chunk) }
