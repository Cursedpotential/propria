// Byline: Codex · GPT-6.1 · 2026-10-05.
package librarysync

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

func TestWriteLostReplyReconcilesExactlyOneRetainedVersion(t *testing.T) {
	s, store, back := fixtureService(t)
	store.lostReply = true
	claim, p := prepareWrite(t, s)
	written, err := s.WriteVersion(context.Background(), p, "attempt-1")
	requireNoError(t, err)
	if written.Status != WriteUnknown {
		t.Fatalf("lost reply status %s", written.Status)
	}
	again, err := s.WriteVersion(context.Background(), p, "attempt-1")
	requireNoError(t, err)
	if again.Status == "written" || store.putCalls != 1 {
		t.Fatal("retried writer issued blind second PUT")
	}
	out := reconcile(t, s, claim, nil)
	if out.Status != Synced || back.completed.IntentID != "intent-fixture" || back.completed.Written == nil {
		t.Fatalf("reconciliation=%+v completion=%+v", out, back.completed)
	}
}

func TestConsumedIntentLostResponseDoesNotWriteAndQueuesReconciledRetry(t *testing.T) {
	s, store, back := fixtureService(t)
	back.loseIntentReply = true
	claim, p := prepareWrite(t, s)
	out, err := s.WriteVersion(context.Background(), p, "attempt-1")
	requireNoError(t, err)
	if out.Status != WriteUnknown || store.putCalls != 0 {
		t.Fatal("uncertain intent wrote")
	}
	outcome := reconcile(t, s, claim, nil)
	if outcome.Status != RetryWait || back.completed.IntentID == "" {
		t.Fatal("unwritten consumed intent not retained")
	}
}

func TestExpiredLeaseRecoversPriorIntentWithoutBlindPut(t *testing.T) {
	s, store, back := fixtureService(t)
	claim, p := prepareWrite(t, s)
	_, err := s.WriteVersion(context.Background(), p, "attempt-1")
	requireNoError(t, err)
	s.Now = func() time.Time { return fixtureNow.Add(2 * time.Hour) }
	if _, err = s.WriteVersion(context.Background(), p, "attempt-1"); err == nil {
		t.Fatal("expired lease admitted")
	}
	back.claim.Fence = 2
	back.claim.LeaseID = "recovered"
	back.claim.ExpiresAt = fixtureNow.Add(3 * time.Hour)
	fresh, err := s.RefreshOperation(context.Background(), RecoveryInput{claim, "attempt-1"})
	requireNoError(t, err)
	prepared, err := s.PreparePayload(context.Background(), fresh)
	requireNoError(t, err)
	out, err := s.WriteVersion(context.Background(), prepared, "attempt-1")
	requireNoError(t, err)
	if store.putCalls != 1 || out.Status != WriteUnknown {
		t.Fatal("recovered consumed intent generated new PUT")
	}
}

func TestRefreshRejectsImmutableBaseMutationAndLostIntent(t *testing.T) {
	for _, kind := range []string{"base", "payload", "intent", "fence"} {
		t.Run(kind, func(t *testing.T) {
			s, _, back := fixtureService(t)
			back.claim.WriteIntentID = "prior-intent"
			claim, _ := prepareWrite(t, s)
			switch kind {
			case "base":
				back.claim.Operation.BasePointerRevision = "silently-rebased"
			case "payload":
				back.claim.Operation.PayloadSHA256 = digest([]byte("other"))
			case "intent":
				back.claim.WriteIntentID = ""
			case "fence":
				back.claim.Fence = 0
			}
			if _, err := s.RefreshOperation(context.Background(), RecoveryInput{claim, "attempt-1"}); err == nil {
				t.Fatal("immutable or intent mutation accepted")
			}
		})
	}
}

func TestReconciliationNeverClearsFailedOrCompetingEvidence(t *testing.T) {
	for _, kind := range []string{"intervening-upload", "hide", "duplicate-own", "wrong-intent", "partial", "failed-hash", "missing-check", "duplicate-check", "missing-current", "stale-current", "post-hash-head-change", "record-changed"} {
		t.Run(kind, func(t *testing.T) {
			s, store, back := fixtureService(t)
			key := back.claim.Operation.Key
			baseRaw := []byte("previous Markdown body")
			baseObj := Object{Bucket: fixtureScope.Bucket, Key: key, VersionID: "base", ContentType: "text/markdown", Latest: true, UploadedAt: fixtureNow.Add(-time.Minute)}
			store.add(baseRaw, baseObj)
			back.claim.Operation.BaseB2Pointer = &Pointer{VersionID: "base", SHA256: digest(baseRaw), Size: int64(len(baseRaw)), ContentType: "text/markdown", ObservedAt: fixtureNow.Add(-time.Minute)}
			claim, p := prepareWrite(t, s)
			_, err := s.WriteVersion(context.Background(), p, "attempt-1")
			requireNoError(t, err)
			switch kind {
			case "intervening-upload":
				store.add([]byte("external writer bytes"), Object{Bucket: fixtureScope.Bucket, Key: key, VersionID: "external", ContentType: "text/markdown", UploadedAt: fixtureNow.Add(-2 * time.Second)})
			case "hide":
				store.add(nil, Object{Bucket: fixtureScope.Bucket, Key: key, VersionID: "hide", Hidden: true, Latest: true, UploadedAt: fixtureNow})
			case "duplicate-own":
				_, err = store.Put(context.Background(), back.claim.Operation, back.payload, back.claim.WriteIntentID)
				requireNoError(t, err)
			case "wrong-intent":
				store.versions[1].Object.IntentID = "wrong-intent"
			case "partial":
				store.partial = true
			case "failed-hash":
				store.hashFails["written-1"] = true
			case "record-changed":
				back.recordChanged = true
			}
			out := reconcile(t, s, claim, func(in *AckInput) {
				switch kind {
				case "missing-check":
					in.Checks = in.Checks[:len(in.Checks)-1]
				case "duplicate-check":
					in.Checks[1] = in.Checks[0]
				case "missing-current":
					in.Current = Handle{}
				case "stale-current":
					s.Now = func() time.Time { return fixtureNow.Add(2 * IOTimeout) }
				case "post-hash-head-change":
					store.headOverride = &Object{Bucket: fixtureScope.Bucket, Key: key, VersionID: "later-external", Size: 10, Latest: true, UploadedAt: fixtureNow}
					h, err := s.CheckCurrent(context.Background(), in.Plan)
					requireNoError(t, err)
					in.Current = h
				}
			})
			if out.Status == Synced {
				t.Fatalf("%s silently cleared %+v", kind, back.completed)
			}
			if len(store.versions) < 2 {
				t.Fatal("versions not retained")
			}
		})
	}
}

func TestBaseVersionChangeBeforeWritePreservesBothAndDoesNotPut(t *testing.T) {
	s, store, back := fixtureService(t)
	claim, p := prepareWrite(t, s)
	store.add([]byte("external"), Object{Bucket: fixtureScope.Bucket, Key: back.claim.Operation.Key, VersionID: "external", Latest: true, UploadedAt: fixtureNow.Add(-time.Second), ContentType: "text/markdown"})
	h, err := s.WriteVersion(context.Background(), p, "attempt-1")
	requireNoError(t, err)
	if h.Status != Conflicted || store.putCalls != 0 || back.intentCalls != 0 {
		t.Fatal("stale base overwritten")
	}
	if reconcile(t, s, claim, nil).Status != Conflicted {
		t.Fatal("base conflict not retained")
	}
}

func TestBackendCannotClearPartialWorkerEvidence(t *testing.T) {
	s, store, back := fixtureService(t)
	claim, p := prepareWrite(t, s)
	_, err := s.WriteVersion(context.Background(), p, "attempt-1")
	requireNoError(t, err)
	store.partial = true
	back.forgeSuccess = true
	fresh, err := s.RefreshOperation(context.Background(), RecoveryInput{claim, "attempt-1"})
	requireNoError(t, err)
	plan, err := s.PlanReconciliation(context.Background(), fresh)
	requireNoError(t, err)
	check, err := s.HashReconciliation(context.Background(), HashInput{plan.Handle, 0})
	requireNoError(t, err)
	current, err := s.CheckCurrent(context.Background(), plan.Handle)
	requireNoError(t, err)
	if _, err = s.Acknowledge(context.Background(), AckInput{Plan: plan.Handle, Checks: []Handle{check}, Current: current}); err == nil {
		t.Fatal("forged backend success accepted")
	}
}

func TestObservationRetainsFullOriginalAndNeverInventsCurrency(t *testing.T) {
	s, store, back := fixtureService(t)
	raw := []byte("%PDF-1.7\nSynthetic confidential text\n")
	obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "benchbooks/fixture.pdf", VersionID: "pdf-version", Latest: true, ContentType: "application/pdf", UploadedAt: fixtureNow.Add(-time.Minute)}
	store.add(raw, obj)
	obj.Size = int64(len(raw))
	h, err := s.HashSource(context.Background(), obj)
	requireNoError(t, err)
	h, err = s.RetainSource(context.Background(), h)
	requireNoError(t, err)
	h, err = s.ExtractSource(context.Background(), h)
	requireNoError(t, err)
	out, err := s.StageObservation(context.Background(), h)
	requireNoError(t, err)
	if out.Status != CitationRequired || len(back.observations) != 1 || back.observations[0].ExtractionRef == nil {
		t.Fatal("document not staged citation-required")
	}
	o := back.observations[0]
	if o.Object.SHA256 != digest(raw) || o.RawRef == nil {
		t.Fatal("original provenance absent")
	}
	retained, err := s.Artifacts.Read(context.Background(), *o.RawRef, MaxPayloadBytes)
	requireNoError(t, err)
	if string(retained) != string(raw) {
		t.Fatal("private original modified")
	}
	wire, _ := json.Marshal(o)
	if strings.Contains(string(wire), "currency_status") || strings.Contains(string(wire), "VERIFIED_PRIMARY") {
		t.Fatal("invented legal clearance")
	}
	_, err = s.StageObservation(context.Background(), h)
	requireNoError(t, err)
	if len(back.observations) != 1 {
		t.Fatal("idempotent observation duplicated")
	}
}

func TestObservationExtractionHashMismatchAndHiddenRemainBlocked(t *testing.T) {
	for _, kind := range []string{"bad-extractor", "hidden", "oversized", "missing-original", "unsupported"} {
		t.Run(kind, func(t *testing.T) {
			s, store, back := fixtureService(t)
			raw := []byte("%PDF-1.7 fixture")
			obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "case-law/fixture.pdf", VersionID: "v", UploadedAt: fixtureNow, ContentType: "application/pdf"}
			store.add(raw, obj)
			obj.Size = int64(len(raw))
			switch kind {
			case "bad-extractor":
				s.Extractor = fixtureExtractor{bad: true}
			case "hidden":
				obj.Hidden = true
			case "oversized":
				obj.Size = MaxOriginalBytes + 1
			case "missing-original":
				store.versions = nil
			case "unsupported":
				store.versions[0].Object.ContentType = "application/unknown"
			}
			h, err := s.HashSource(context.Background(), obj)
			requireNoError(t, err)
			h, err = s.RetainSource(context.Background(), h)
			requireNoError(t, err)
			h, err = s.ExtractSource(context.Background(), h)
			requireNoError(t, err)
			out, err := s.StageObservation(context.Background(), h)
			requireNoError(t, err)
			if out.Status != Blocked || back.observations[0].Code == "" {
				t.Fatalf("failed source cleared %+v", out)
			}
		})
	}
}

func TestAuthenticatedDescriptorRejectsBodyTampering(t *testing.T) {
	s, _, _ := fixtureService(t)
	claim, _ := prepareWrite(t, s)
	store := s.Artifacts.(*memoryArtifacts)
	raw := append([]byte(nil), store.data[claim.Ref.URI]...)
	var env envelope
	requireNoError(t, json.Unmarshal(raw, &env))
	var c Claim
	requireNoError(t, json.Unmarshal(env.Payload, &c))
	c.Operation.Key = fixtureScope.LegalRoot + "reference-data/evil.md"
	env.Payload, _ = json.Marshal(c)
	raw, _ = json.Marshal(env)
	store.data[claim.Ref.URI] = raw
	claim.Ref.SHA256 = libraryvalidation.Hash(raw)
	claim.Ref.Bytes = int64(len(raw))
	if _, err := s.PreparePayload(context.Background(), claim); err == nil {
		t.Fatal("authenticated descriptor tampering accepted")
	}
}

func TestObservationDoesNotAcceptAutomaticPublishResponse(t *testing.T) {
	s, store, back := fixtureService(t)
	back.observeStatus = "published"
	obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "reference-data/fixture.md", VersionID: "v", ContentType: "text/markdown", UploadedAt: fixtureNow}
	store.add([]byte("private fixture markdown"), obj)
	obj.Size = int64(len("private fixture markdown"))
	h, err := s.HashSource(context.Background(), obj)
	requireNoError(t, err)
	h, err = s.RetainSource(context.Background(), h)
	requireNoError(t, err)
	h, err = s.ExtractSource(context.Background(), h)
	requireNoError(t, err)
	if _, err = s.StageObservation(context.Background(), h); err == nil {
		t.Fatal("automatic publication accepted")
	}
}

func TestGenericProviderMIMERoutesPinnedBytesAndPreservesProvenance(t *testing.T) {
	for _, provider := range []string{"application/octet-stream", "", "binary/octet-stream"} {
		for _, source := range []struct {
			ext, body, media string
			extracted        bool
		}{
			{".md", "# Full private fixture\nUnknown personal context remains complete.\n", "text/markdown", false},
			{".PDF", "%PDF-1.7\nSynthetic pinned PDF bytes", "application/pdf", true},
			{".json", `{"record":{"private_unknown":"complete"},"types":{}}`, "application/json", false},
			{".html", "<!DOCTYPE html><html><body>Synthetic private text</body></html>", "text/html", true},
			{".xhtml", `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><body>Private fixture</body></html>`, "application/xhtml+xml", true},
		} {
			t.Run(provider+source.ext, func(t *testing.T) {
				s, store, back := fixtureService(t)
				raw := []byte(source.body)
				obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "reference-data/incoming" + source.ext, VersionID: "pinned-source-v", UploadedAt: fixtureNow, ContentType: provider, Size: int64(len(raw))}
				store.add(raw, obj)
				h, err := s.HashSource(context.Background(), obj)
				requireNoError(t, err)
				h, err = s.RetainSource(context.Background(), h)
				requireNoError(t, err)
				h, err = s.ExtractSource(context.Background(), h)
				requireNoError(t, err)
				out, err := s.StageObservation(context.Background(), h)
				requireNoError(t, err)
				if out.Status != CitationRequired || len(back.observations) != 1 {
					t.Fatal("supported generic media blocked or cleared")
				}
				o := back.observations[0]
				if o.Object.ContentType != provider || o.Object.SHA256 != digest(raw) || o.ExtractionRef == nil || o.RawRef == nil {
					t.Fatal("provider/hash provenance changed")
				}
				var evidence sourceExtractionEvidence
				requireNoError(t, s.unseal(context.Background(), *o.ExtractionRef, "extraction", &evidence))
				if evidence.ProviderContentType != provider || evidence.VerifiedMediaType != source.media || evidence.MediaCheckVersion == "" || evidence.InputSHA256 != digest(raw) || evidence.SourceVersionID != obj.VersionID || (evidence.Extracted != nil) != source.extracted {
					t.Fatal("media descriptor not bound to exact source evidence")
				}
				retained, err := s.Artifacts.Read(context.Background(), *o.RawRef, MaxPayloadBytes)
				requireNoError(t, err)
				if string(retained) != source.body || store.putCalls != 0 {
					t.Fatal("source bytes rewritten during media routing")
				}
			})
		}
	}
}

func TestSourceMediaRejectsUnsupportedAndInvalidBytes(t *testing.T) {
	for _, source := range []struct{ ext, body, mime string }{
		{".pdf", "not a PDF", "application/octet-stream"},
		{".pdf", "not a PDF", "application/pdf"},
		{".md", "bad\xffUTF8", "application/octet-stream"},
		{".md", "binary\x00body", ""},
		{".json", `{"unfinished":`, "application/octet-stream"},
		{".json", `{"unfinished":`, "application/json"},
		{".html", "plain text mislabeled HTML", "application/octet-stream"},
		{".html", "<html>bad\xff</html>", "text/html"},
		{".xhtml", `<?xml version="1.0"?><different-root/>`, ""},
		{".zip", "PK\x03\x04", "application/octet-stream"},
		{".txt", "plain text without supported generic extension", ""},
		{".md", "valid text", "application/unknown"},
		{".md", "valid text", "malformed/media;="},
	} {
		t.Run(source.ext+source.mime+source.body, func(t *testing.T) {
			s, store, back := fixtureService(t)
			raw := []byte(source.body)
			obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "reference-data/incoming" + source.ext, VersionID: "pinned-source-v", UploadedAt: fixtureNow, ContentType: source.mime, Size: int64(len(raw))}
			store.add(raw, obj)
			h, err := s.HashSource(context.Background(), obj)
			requireNoError(t, err)
			h, err = s.RetainSource(context.Background(), h)
			requireNoError(t, err)
			h, err = s.ExtractSource(context.Background(), h)
			requireNoError(t, err)
			out, err := s.StageObservation(context.Background(), h)
			requireNoError(t, err)
			if out.Status != Blocked || back.observations[0].Code != "UNSUPPORTED_OR_INVALID_SOURCE_MEDIA" || back.observations[0].Object.ContentType != source.mime {
				t.Fatal("invalid source media admitted or provenance changed")
			}
		})
	}
}

func TestSourceMediaRejectsRetainedInputHashMismatch(t *testing.T) {
	s, store, back := fixtureService(t)
	raw := []byte("# complete private fixture")
	obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "reference-data/incoming.md", VersionID: "pinned-source-v", UploadedAt: fixtureNow, ContentType: "application/octet-stream", Size: int64(len(raw))}
	store.add(raw, obj)
	h, err := s.HashSource(context.Background(), obj)
	requireNoError(t, err)
	h, err = s.RetainSource(context.Background(), h)
	requireNoError(t, err)
	var o Observation
	requireNoError(t, s.unseal(context.Background(), h.Ref, "observation", &o))
	s.Artifacts.(*memoryArtifacts).data[o.RawRef.URI] = []byte("changed after retention")
	h, err = s.ExtractSource(context.Background(), h)
	requireNoError(t, err)
	out, err := s.StageObservation(context.Background(), h)
	requireNoError(t, err)
	if out.Status != Blocked || back.observations[0].Code != "MEDIA_INPUT_BINDING_FAILED" {
		t.Fatal("changed retained input used for media proof")
	}
}
