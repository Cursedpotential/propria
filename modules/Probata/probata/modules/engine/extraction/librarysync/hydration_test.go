// Byline: Codex · GPT-6.1 · 2026-10-05. Complete private synthetic incoming fixtures; no real case adoption or provider writes.
package librarysync

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

func incomingFixtureHandle(t *testing.T, s *Service, store *memoryStorage, ext string, raw []byte) Handle {
	t.Helper()
	obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "reference-data/incoming" + ext, VersionID: "pinned-original", ContentType: "application/octet-stream", UploadedAt: fixtureNow, Size: int64(len(raw))}
	store.add(raw, obj)
	h, err := s.HashSource(context.Background(), obj)
	requireNoError(t, err)
	h, err = s.RetainSource(context.Background(), h)
	requireNoError(t, err)
	h, err = s.ExtractSource(context.Background(), h)
	requireNoError(t, err)
	return h
}

func TestHydrationUploadsFullMarkdownAndTypedCompanionBytes(t *testing.T) {
	for _, source := range []struct{ ext, body, media string }{
		{".md", "# Synthetic complete private Markdown\nUnknown context preserved in full.\n", "text/markdown"},
		{".json", `{"codec_version":"surreal-record-json/1","record_id":"note:fixture","record_version":"sha256:base","record":{"id":"note:fixture","private_unknown":{"full":"retained"},"created_at":"2026-10-05T12:00:00Z"},"types":{"/record/id":"record","/record/created_at":"datetime"}}`, "application/json"},
	} {
		t.Run(source.ext, func(t *testing.T) {
			s, store, back := fixtureService(t)
			raw := []byte(source.body)
			h := incomingFixtureHandle(t, s, store, source.ext, raw)
			hydrated, err := s.HydrateObservation(context.Background(), h)
			requireNoError(t, err)
			var receipt incomingAdmission
			requireNoError(t, s.unseal(context.Background(), hydrated.Ref, "incoming-admission", &receipt))
			in := back.incoming[receipt.Observation.ObservationID]
			if !bytes.Equal(in.Raw, raw) || in.Metadata.ContentType != source.media || in.Metadata.SourceSHA256 != digest(raw) || in.Metadata.SHA256 != digest(raw) || receipt.Encoding != "source-bytes/1" || receipt.SourceVersionID != "pinned-original" || receipt.Observation.Object.ContentType != "application/octet-stream" {
				t.Fatal("incoming full bytes/type/source provenance changed")
			}
			out, err := s.StageObservation(context.Background(), hydrated)
			requireNoError(t, err)
			if out.Status != CitationRequired || back.uploadCalls != 1 || store.putCalls != 0 || len(back.observations) != 1 {
				t.Fatal("hydration skipped upload, rewrote source or auto-cleared legal draft")
			}
			_, err = s.HydrateObservation(context.Background(), h)
			requireNoError(t, err)
			if len(back.incoming) != 1 {
				t.Fatal("idempotent incoming upload duplicated content")
			}
		})
	}
}

func TestHydrationDerivedTextPreservesOriginalAndExtractionHashes(t *testing.T) {
	s, store, back := fixtureService(t)
	raw := []byte("%PDF-1.7 synthetic private original")
	h := incomingFixtureHandle(t, s, store, ".pdf", raw)
	var o Observation
	requireNoError(t, s.unseal(context.Background(), h.Ref, "observation", &o))
	var evidence sourceExtractionEvidence
	requireNoError(t, s.unseal(context.Background(), *o.ExtractionRef, "extraction", &evidence))
	evidence.Extracted.Pages = []string{"Page one full private fixture", "Page two full unknown context"}
	ref, err := s.seal(context.Background(), "extraction", "synthetic/full-pages", evidence)
	requireNoError(t, err)
	o.ExtractionRef = &ref
	h.Ref, err = s.seal(context.Background(), "observation", "synthetic/full-source", o)
	requireNoError(t, err)
	hydrated, err := s.HydrateObservation(context.Background(), h)
	requireNoError(t, err)
	var receipt incomingAdmission
	requireNoError(t, s.unseal(context.Background(), hydrated.Ref, "incoming-admission", &receipt))
	uploaded := back.incoming[o.ObservationID]
	if string(uploaded.Raw) != strings.Join(evidence.Extracted.Pages, "\n\f\n") || uploaded.Metadata.SourceSHA256 != digest(raw) || uploaded.Metadata.SHA256 == uploaded.Metadata.SourceSHA256 || uploaded.Metadata.ContentType != "text/plain" || receipt.Encoding != "pages-formfeed/1" || receipt.Observation.RawRef == nil || *receipt.Observation.ExtractionRef != ref {
		t.Fatal("derived text conflated with original bytes or page extraction evidence")
	}
	original, err := s.Artifacts.Read(context.Background(), *receipt.Observation.RawRef, MaxPayloadBytes)
	requireNoError(t, err)
	if !bytes.Equal(original, raw) || store.putCalls != 0 {
		t.Fatal("PDF original rewritten")
	}
}

func TestHydrationPreservesBlockedAndUncertainAdmission(t *testing.T) {
	for _, kind := range []string{"blocked", "conflicted", "lost-reply", "published", "stale-source", "wrong-input-hash"} {
		t.Run(kind, func(t *testing.T) {
			s, store, back := fixtureService(t)
			h := incomingFixtureHandle(t, s, store, ".md", []byte("complete synthetic private Markdown"))
			if kind == "blocked" || kind == "conflicted" || kind == "published" {
				back.uploadStatus = kind
			}
			if kind == "lost-reply" {
				back.uploadError = true
			}
			if kind == "stale-source" || kind == "wrong-input-hash" {
				var o Observation
				requireNoError(t, s.unseal(context.Background(), h.Ref, "observation", &o))
				var evidence sourceExtractionEvidence
				requireNoError(t, s.unseal(context.Background(), *o.ExtractionRef, "extraction", &evidence))
				if kind == "stale-source" {
					evidence.SourceVersionID = "another-version"
				} else {
					evidence.InputSHA256 = digest([]byte("another-source"))
				}
				ref, err := s.seal(context.Background(), "extraction", "synthetic/mismatch", evidence)
				requireNoError(t, err)
				o.ExtractionRef = &ref
				h.Ref, err = s.seal(context.Background(), "observation", "synthetic/mismatch-source", o)
				requireNoError(t, err)
			}
			hydrated, err := s.HydrateObservation(context.Background(), h)
			if kind == "blocked" || kind == "conflicted" {
				requireNoError(t, err)
				out, err := s.StageObservation(context.Background(), hydrated)
				requireNoError(t, err)
				if out.Status != kind {
					t.Fatal("failed admission became cleared")
				}
			} else if err == nil {
				t.Fatal("uncertain/stale/unsupported admission succeeded")
			}
		})
	}
}

func TestHydratedPersonalCASResponseRequiresSignedReadyProof(t *testing.T) {
	s, store, back := fixtureService(t)
	h := incomingFixtureHandle(t, s, store, ".md", []byte("full private fixture"))
	back.observeStatus = Synced
	if _, err := s.StageObservation(context.Background(), h); err == nil {
		t.Fatal("metadata-only worker accepted personal adoption")
	}
	hydrated, err := s.HydrateObservation(context.Background(), h)
	requireNoError(t, err)
	_, err = s.StageObservation(context.Background(), hydrated)
	requireNoError(t, err)
	for _, status := range []string{"published", libraryvalidation.Verified} {
		back.observeStatus = status
		if _, err := s.StageObservation(context.Background(), hydrated); err == nil {
			t.Fatal("hydration invented publication or legal clearance")
		}
	}
	// Signed proof fields must agree even when an independently retained descriptor is presented.
	var proof incomingAdmission
	requireNoError(t, s.unseal(context.Background(), hydrated.Ref, "incoming-admission", &proof))
	proof.Metadata.SourceSHA256 = digest([]byte("wrong original"))
	hydrated.Ref, err = s.seal(context.Background(), "incoming-admission", "synthetic/wrong-proof", proof)
	requireNoError(t, err)
	back.observeStatus = Synced
	if _, err = s.StageObservation(context.Background(), hydrated); err == nil {
		t.Fatal("wrong-source ready proof adopted")
	}
	wire, _ := json.Marshal(hydrated)
	if bytes.Contains(wire, []byte("full private fixture")) {
		t.Fatal("personal body entered Activity result")
	}
}
