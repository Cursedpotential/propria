// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime"
	"reflect"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

// Service supplies independent sync units using existing B2 artifacts/extractor and the protected parent backend.
// Inputs: bounded dependencies and server signing key; outputs: refs/status only. Effects: documented on each unit.
// Choose alongside the existing validator. No legal claim, currency decision, direct source DB write or automatic publication occurs here.
type Service struct {
	Scope      Scope
	Storage    Storage
	Backend    Backend
	Artifacts  libraryvalidation.Artifacts
	Extractor  libraryvalidation.Extractor
	SigningKey []byte
	Now        func() time.Time
}

// NewService checks composition without making network calls; inputs: dependencies; outputs: ready service; effects: none.
func NewService(scope Scope, store Storage, backend Backend, artifacts libraryvalidation.Artifacts, extractor libraryvalidation.Extractor, key []byte) (*Service, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if store == nil || backend == nil || artifacts == nil || len(key) < 32 {
		return nil, errors.New("sync dependencies or signing key missing")
	}
	return &Service{Scope: scope, Storage: store, Backend: backend, Artifacts: artifacts, Extractor: extractor, SigningKey: append([]byte(nil), key...), Now: time.Now}, nil
}

type envelope struct {
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

func (s *Service) seal(ctx context.Context, kind, key string, value any) (libraryvalidation.ArtifactRef, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return libraryvalidation.ArtifactRef{}, err
	}
	e := envelope{Kind: kind, Payload: raw}
	mac := hmac.New(sha256.New, s.SigningKey)
	mac.Write([]byte(kind))
	mac.Write([]byte{0})
	mac.Write(raw)
	e.Signature = hex.EncodeToString(mac.Sum(nil))
	body, _ := json.Marshal(e)
	if int64(len(body)) > metadataBudget {
		return libraryvalidation.ArtifactRef{}, errors.New("sync evidence metadata budget exceeded")
	}
	return s.Artifacts.Put(ctx, "library-sync/"+key+".json", body, "application/json")
}
func (s *Service) unseal(ctx context.Context, ref libraryvalidation.ArtifactRef, kind string, out any) error {
	raw, err := s.Artifacts.Read(ctx, ref, metadataBudget)
	if err != nil {
		return err
	}
	var e envelope
	if json.Unmarshal(raw, &e) != nil || e.Kind != kind {
		return errors.New("sync descriptor kind invalid")
	}
	mac := hmac.New(sha256.New, s.SigningKey)
	mac.Write([]byte(e.Kind))
	mac.Write([]byte{0})
	mac.Write(e.Payload)
	sig, err := hex.DecodeString(e.Signature)
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return errors.New("sync descriptor authentication failed")
	}
	return json.Unmarshal(e.Payload, out)
}

func (s *Service) observation(obj Object) (Observation, error) {
	id, err := s.Scope.BindingID(obj.Key)
	if err != nil || obj.Bucket != s.Scope.Bucket || !validVersion(obj.VersionID) {
		return Observation{}, errors.New("invalid observation coordinates")
	}
	raw, _ := json.Marshal([]string{id, obj.VersionID})
	return Observation{Contract: ContractVersion, ObservationID: digest(raw), BindingID: id, Object: obj, Status: Pending}, nil
}
func (s *Service) clock() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func (s *Service) loadClaim(ctx context.Context, h Handle) (Claim, error) {
	var c Claim
	if err := s.unseal(ctx, h.Ref, "claim", &c); err != nil {
		return c, err
	}
	if c.Operation.OperationID != h.OperationID {
		return c, errors.New("claim identity mismatch")
	}
	if err := c.Operation.validate(s.Scope); err != nil {
		return c, err
	}
	if c.Status != Pending || c.LeaseID == "" || c.Fence <= 0 || len(c.WriteIntentID) > 200 || strings.ContainsAny(c.WriteIntentID, "\r\n\x00") {
		return c, errors.New("sync claim invalid")
	}
	return c, nil
}
func (s *Service) claim(ctx context.Context, h Handle) (Claim, error) {
	c, err := s.loadClaim(ctx, h)
	if err == nil && !c.ExpiresAt.After(s.clock()) {
		err = errors.New("sync lease expired")
	}
	return c, err
}

// ListPage observes one bounded provider metadata page; inputs: child/cursor; outputs: metadata refs; effects: listing only.
func (s *Service) ListPage(ctx context.Context, in ListInput) (Page, error) {
	return s.Storage.List(ctx, in.Child, in.Cursor)
}

// CheckObservation avoids rereading already processed versions; inputs: object metadata; outputs: seen; effects: backend read only.
func (s *Service) CheckObservation(ctx context.Context, obj Object) (bool, error) {
	o, err := s.observation(obj)
	if err != nil {
		return false, err
	}
	return s.Backend.Seen(ctx, o.ObservationID)
}

// HashSource hashes one exact original independently of extraction and persists authenticated metadata.
// Inputs: listed version; outputs: evidence handle. Effects: pinned streaming read and derivative descriptor retention.
// Hidden, oversized or unavailable bytes remain explicit blocked observations, never accepted source pointers.
func (s *Service) HashSource(ctx context.Context, obj Object) (Handle, error) {
	o, err := s.observation(obj)
	if err != nil {
		return Handle{}, err
	}
	if obj.Hidden {
		o.Status = Blocked
		o.Code = "FILE_HIDDEN"
	} else if obj.Size > MaxOriginalBytes {
		o.Status = Blocked
		o.Code = "ORIGINAL_BODY_BUDGET"
	} else if obj.Size == 0 {
		o.Status = Blocked
		o.Code = "EMPTY_ORIGINAL"
	} else {
		hashed, e := s.Storage.Hash(ctx, obj)
		if e != nil {
			o.Status = Blocked
			o.Code = "PINNED_HASH_UNAVAILABLE"
		} else if hashed.Bucket != obj.Bucket || hashed.Key != obj.Key || hashed.VersionID != obj.VersionID || hashed.Size != obj.Size || !rawHash.MatchString(hashed.SHA256) {
			o.Status = Blocked
			o.Code = "PINNED_HASH_DESCRIPTOR_MISMATCH"
		} else {
			o.Object = hashed
		}
	}
	r, err := s.seal(ctx, "observation", o.ObservationID+"/hash", o)
	return Handle{Ref: r, Status: o.Status, Code: o.Code}, err
}

// RetainSource copies independently hashed exact bytes into the existing derivative store for supported extraction.
// Inputs: authenticated hash handle; outputs: retained raw descriptor. Effects: exact GET and derivative retention only.
// Files above the existing 8 MiB adapter limit retain their original B2 version and hash, with explicit extraction blocking.
func (s *Service) RetainSource(ctx context.Context, h Handle) (Handle, error) {
	var o Observation
	if err := s.unseal(ctx, h.Ref, "observation", &o); err != nil {
		return Handle{}, err
	}
	if o.Status == Pending {
		if o.Object.Size > libraryvalidation.MaxSourceBytes {
			o.Status = Blocked
			o.Code = "EXTRACTOR_INPUT_BUDGET"
		} else {
			raw, actual, err := s.Storage.Read(ctx, o.Object, libraryvalidation.MaxSourceBytes)
			if err != nil || digest(raw) != o.Object.SHA256 || actual.VersionID != o.Object.VersionID {
				o.Status = Blocked
				o.Code = "PINNED_RETENTION_MISMATCH"
			} else {
				ref, err := s.Artifacts.Put(ctx, "library-sync/"+o.ObservationID+"/original", raw, o.Object.ContentType)
				if err != nil {
					return Handle{}, err
				}
				if strings.TrimPrefix(ref.SHA256, "sha256:") != o.Object.SHA256 || ref.Bytes != o.Object.Size {
					return Handle{}, errors.New("derivative retention mismatch")
				}
				o.RawRef = &ref
			}
		}
	}
	r, err := s.seal(ctx, "observation", o.ObservationID+"/retained", o)
	return Handle{Ref: r, Status: o.Status, Code: o.Code}, err
}

// ExtractSource invokes only the existing pinned PDF/HTML extractor and retains its full input-bound evidence outside history.
// Inputs: retained source handle; outputs: descriptor. Effects: existing parser and derivative retention; no NIM or currency inference.
func (s *Service) ExtractSource(ctx context.Context, h Handle) (Handle, error) {
	var o Observation
	if err := s.unseal(ctx, h.Ref, "observation", &o); err != nil {
		return Handle{}, err
	}
	if o.Status == Pending && o.RawRef != nil {
		media, _, mediaErr := mime.ParseMediaType(o.Object.ContentType)
		if mediaErr != nil {
			media = ""
		}
		switch media {
		case "application/json", "text/markdown", "text/plain":
			o.Status = CitationRequired
		case "application/pdf", "text/html", "application/xhtml+xml":
			if s.Extractor == nil {
				o.Status = Blocked
				o.Code = "EXTRACTOR_NOT_CONFIGURED"
				break
			}
			x, err := s.Extractor.Extract(ctx, libraryvalidation.Snapshot{Raw: *o.RawRef, MediaType: media, FetchedAt: s.clock()})
			textBytes := 0
			for _, page := range x.Pages {
				textBytes += len(page)
			}
			if err != nil || x.InputSHA256 != o.Object.SHA256 || x.VersionID != o.RawRef.VersionID || x.Extractor == "" || x.ExtractorVersion == "" || x.LowConfidence || len(x.Pages) == 0 || len(x.Pages) > libraryvalidation.MaxPages || textBytes > libraryvalidation.MaxTextBytes {
				o.Status = Blocked
				o.Code = "EXTRACTOR_INPUT_BINDING_FAILED"
				break
			}
			r, err := s.seal(ctx, "extraction", o.ObservationID+"/extraction", struct {
				Original  Object                        `json:"original"`
				Raw       libraryvalidation.ArtifactRef `json:"raw"`
				Extracted libraryvalidation.Extracted   `json:"extracted"`
			}{o.Object, *o.RawRef, x})
			if err != nil {
				return Handle{}, err
			}
			o.ExtractionRef = &r
			o.Status = CitationRequired
		default:
			o.Status = Blocked
			o.Code = "UNSUPPORTED_EXTRACTOR_FORMAT"
		}
	}
	r, err := s.seal(ctx, "observation", o.ObservationID+"/extracted", o)
	return Handle{Ref: r, Status: o.Status, Code: o.Code}, err
}

// StageObservation submits only retained evidence references to existing guarded backend import/proposal gates.
// Inputs: authenticated observation; outputs: explicit durable IDs/status. Effects: backend transaction; never auto-publishes.
func (s *Service) StageObservation(ctx context.Context, h Handle) (Outcome, error) {
	var o Observation
	if err := s.unseal(ctx, h.Ref, "observation", &o); err != nil {
		return Outcome{}, err
	}
	expected, err := s.observation(o.Object)
	if err != nil || expected.ObservationID != o.ObservationID || expected.BindingID != o.BindingID {
		return Outcome{}, errors.New("observation identity mismatch")
	}
	if o.Status != Blocked && (!rawHash.MatchString(o.Object.SHA256) || o.RawRef == nil) {
		return Outcome{}, errors.New("observation incomplete")
	}
	out, err := s.Backend.Observe(ctx, o)
	if err == nil && (out.Status == Synced || out.Status == "published" || out.Status == libraryvalidation.Verified) {
		return Outcome{}, errors.New("observation backend attempted unsupported automatic publication")
	}
	return out, err
}

// ClaimOperation obtains a binding lease and retains its immutable descriptor outside history.
// Inputs: operation/attempt IDs; outputs: handle. Effects: backend lease transaction and authenticated derivative retention.
func (s *Service) ClaimOperation(ctx context.Context, in OperationInput) (Handle, error) {
	if !uuidID.MatchString(in.OperationID) || in.AttemptID == "" || len(in.AttemptID) > 200 {
		return Handle{}, errors.New("invalid operation attempt")
	}
	c, err := s.Backend.Claim(ctx, in)
	if err != nil {
		return Handle{}, err
	}
	if c.Operation.OperationID != in.OperationID {
		return Handle{}, errors.New("backend claimed different operation")
	}
	if c.Status == Synced {
		return Handle{OperationID: in.OperationID, Status: Synced}, nil
	}
	if err = c.Operation.validate(s.Scope); err != nil {
		return Handle{}, err
	}
	if c.Status != Pending || c.LeaseID == "" || c.Fence <= 0 || !c.ExpiresAt.After(s.clock()) || len(c.WriteIntentID) > 200 || strings.ContainsAny(c.WriteIntentID, "\r\n\x00") {
		return Handle{}, errors.New("backend lease invalid")
	}
	r, err := s.seal(ctx, "claim", in.OperationID+"/claim-"+digest([]byte(in.AttemptID)), c)
	return Handle{OperationID: in.OperationID, Ref: r, Status: Pending}, err
}

// RecoveryInput names the prior immutable descriptor and same workflow attempt for intent-safe lease recovery.
// Inputs/outputs: reference and attempt ID only. Effects: none in this type.
type RecoveryInput struct {
	Previous  Handle `json:"previous"`
	AttemptID string `json:"attempt_id"`
}

// RefreshOperation reloads durable write-intent state without changing immutable operation/base pointers.
// Inputs: previous claim and workflow attempt. Outputs: fresh lease descriptor or terminal synced result.
// Effects: backend lease/recovery transaction and derivative retention. Expired writing leases must reconcile their prior intent.
func (s *Service) RefreshOperation(ctx context.Context, in RecoveryInput) (Handle, error) {
	old, err := s.loadClaim(ctx, in.Previous)
	if err != nil {
		return Handle{}, err
	}
	fresh, err := s.ClaimOperation(ctx, OperationInput{OperationID: old.Operation.OperationID, AttemptID: in.AttemptID})
	if err != nil || fresh.Status == Synced {
		return fresh, err
	}
	c, err := s.claim(ctx, fresh)
	if err != nil {
		return Handle{}, err
	}
	if !reflect.DeepEqual(old.Operation, c.Operation) || c.Fence < old.Fence || (old.WriteIntentID != "" && old.WriteIntentID != c.WriteIntentID) {
		return Handle{}, errors.New("backend changed immutable operation/base or lost consumed intent")
	}
	return fresh, nil
}

type prepared struct {
	Claim   Handle                        `json:"claim"`
	Payload libraryvalidation.ArtifactRef `json:"payload"`
}

// PreparePayload verifies exact immutable backend payload bytes and retains them before the writing phase.
// Inputs: leased claim handle; outputs: prepared handle. Effects: payload HTTP read/hash and derivative write; no original-key PUT.
func (s *Service) PreparePayload(ctx context.Context, h Handle) (Handle, error) {
	c, err := s.claim(ctx, h)
	if err != nil {
		return Handle{}, err
	}
	raw, err := s.Backend.Payload(ctx, c)
	if err != nil {
		return Handle{}, err
	}
	if err = ValidatePayload(c.Operation, raw); err != nil {
		return Handle{}, err
	}
	r, err := s.Artifacts.Put(ctx, "library-sync/"+h.OperationID+"/payload", raw, c.Operation.ContentType)
	if err != nil {
		return Handle{}, err
	}
	if strings.TrimPrefix(r.SHA256, "sha256:") != c.Operation.PayloadSHA256 || r.Bytes != c.Operation.PayloadSize {
		return Handle{}, errors.New("retained payload descriptor mismatch")
	}
	ref, err := s.seal(ctx, "prepared", h.OperationID+"/prepared", prepared{h, r})
	return Handle{OperationID: h.OperationID, Ref: ref, Status: Pending}, err
}

// WriteVersion consumes one backend intent and issues at most one retained PUT after a fresh base-version check.
// Inputs: prepared ref and workflow attempt ID; outputs: reference/status. Effects: backend intent and one optional B2 PUT.
// Temporal retries cannot blindly repeat PUT: consumed or uncertain intents return write_unknown for separate reconciliation.
func (s *Service) WriteVersion(ctx context.Context, h Handle, attempt string) (Handle, error) {
	var p prepared
	if err := s.unseal(ctx, h.Ref, "prepared", &p); err != nil {
		return Handle{}, err
	}
	c, err := s.claim(ctx, p.Claim)
	if err != nil {
		return Handle{}, err
	}
	if c.WriteIntentID != "" {
		return Handle{OperationID: h.OperationID, Status: WriteUnknown, Code: "PRIOR_INTENT_REQUIRES_RECONCILIATION"}, nil
	}
	raw, err := s.Artifacts.Read(ctx, p.Payload, MaxPayloadBytes)
	if err != nil {
		return Handle{}, err
	}
	if err = ValidatePayload(c.Operation, raw); err != nil {
		return Handle{}, err
	}
	head, err := s.Storage.Head(ctx, c.Operation.Key)
	if err != nil {
		return Handle{}, err
	}
	if (head == nil) != (c.Operation.BaseB2Pointer == nil) || (head != nil && head.VersionID != c.Operation.BaseB2Pointer.VersionID) {
		return Handle{OperationID: h.OperationID, Status: Conflicted, Code: "B2_BASE_CHANGED"}, nil
	}
	intent, err := s.Backend.BeginWrite(ctx, c, attempt)
	if err != nil || !intent.MayWrite || intent.IntentID == "" {
		return Handle{OperationID: h.OperationID, Status: WriteUnknown, Code: "WRITE_INTENT_UNCERTAIN"}, nil
	}
	written, err := s.Storage.Put(ctx, c.Operation, raw, intent.IntentID)
	if err != nil {
		return Handle{OperationID: h.OperationID, Status: WriteUnknown, Code: "PUT_OUTCOME_UNKNOWN"}, nil
	}
	ref, err := s.seal(ctx, "written", h.OperationID+"/written", written)
	if err != nil {
		return Handle{OperationID: h.OperationID, Status: WriteUnknown, Code: "WRITE_EVIDENCE_UNAVAILABLE"}, nil
	}
	return Handle{OperationID: h.OperationID, Ref: ref, Status: "written"}, nil
}

// ReconcilePlan names independently hashable retained versions from bounded provider history.
// Inputs: claim and history; outputs: refs and count; effects: none in this type.
type ReconcilePlan struct {
	Claim     Handle    `json:"claim"`
	Objects   []Object  `json:"objects"`
	Coverage  string    `json:"coverage"`
	Code      string    `json:"error_code,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// HistoryResult carries only a plan reference and bounded number of hash tasks.
type HistoryResult struct {
	Handle
	Count int `json:"count"`
}

// PlanReconciliation lists exact-key versions without hashing and retains the bounded candidate plan.
// Inputs: claim; outputs: plan/count. Effects: bounded listing and descriptor retention. Missing base or partial history prevents success.
func (s *Service) PlanReconciliation(ctx context.Context, h Handle) (HistoryResult, error) {
	c, err := s.claim(ctx, h)
	if err != nil {
		return HistoryResult{}, err
	}
	history, err := s.Storage.History(ctx, c.Operation.Key)
	if err != nil {
		return HistoryResult{}, err
	}
	p := ReconcilePlan{Claim: h, Objects: []Object{}, Coverage: "complete", CreatedAt: s.clock()}
	if !history.Complete {
		p.Coverage = "partial"
		p.Code = "VERSION_HISTORY_BUDGET"
	}
	seen := map[string]bool{}
	for _, obj := range history.Objects {
		if obj.Bucket != c.Operation.Bucket || obj.Key != c.Operation.Key || !validVersion(obj.VersionID) {
			return HistoryResult{}, errors.New("history returned different object coordinates")
		}
		if seen[obj.VersionID] || obj.UploadedAt.IsZero() || obj.UploadedAt.After(s.clock()) {
			p.Coverage = "partial"
			p.Code = "VERSION_HISTORY_IDENTITY_AMBIGUOUS"
		}
		seen[obj.VersionID] = true
	}
	var base *Object
	if c.Operation.BaseB2Pointer != nil {
		for i := range history.Objects {
			if history.Objects[i].VersionID == c.Operation.BaseB2Pointer.VersionID {
				base = &history.Objects[i]
				break
			}
		}
		if base == nil {
			p.Coverage = "partial"
			p.Code = "BASE_VERSION_NOT_OBSERVED"
		}
	}
	for _, obj := range history.Objects {
		if base == nil || !obj.UploadedAt.Before(base.UploadedAt) || obj.Latest {
			p.Objects = append(p.Objects, obj)
		}
	}
	if len(p.Objects) > MaxPage {
		p.Objects = p.Objects[:MaxPage]
		p.Coverage = "partial"
		p.Code = "RECONCILIATION_TASK_BUDGET"
	}
	r, err := s.seal(ctx, "reconcile", h.OperationID+"/reconcile", p)
	return HistoryResult{Handle: Handle{OperationID: h.OperationID, Ref: r, Status: Pending}, Count: len(p.Objects)}, err
}

// HashInput names one version in a retained reconciliation plan; inputs/outputs: references and index only.
type HashInput struct {
	Plan  Handle `json:"plan"`
	Index int    `json:"index"`
}

// HashReconciliation independently reads/hashes one exact retained candidate; inputs: plan/index; outputs: evidence ref.
// Effects: bounded pinned GET/hash and descriptor retention; hide markers preserve identity without fake content hashes.
func (s *Service) HashReconciliation(ctx context.Context, in HashInput) (Handle, error) {
	var p ReconcilePlan
	if err := s.unseal(ctx, in.Plan.Ref, "reconcile", &p); err != nil {
		return Handle{}, err
	}
	if in.Index < 0 || in.Index >= len(p.Objects) {
		return Handle{}, errors.New("invalid reconciliation index")
	}
	obj := p.Objects[in.Index]
	status := Pending
	code := ""
	if !obj.Hidden {
		hashed, err := s.Storage.Hash(ctx, obj)
		if err != nil {
			status = Blocked
			code = "RECONCILE_HASH_UNAVAILABLE"
		} else if hashed.VersionID != obj.VersionID || hashed.Key != obj.Key || hashed.Bucket != obj.Bucket || hashed.Size != obj.Size || !rawHash.MatchString(hashed.SHA256) {
			status = Blocked
			code = "RECONCILE_HASH_MISMATCH"
		} else {
			obj = hashed
		}
	}
	ref, err := s.seal(ctx, "version-check", in.Plan.OperationID+"/version-"+digest([]byte(obj.VersionID)), obj)
	return Handle{OperationID: in.Plan.OperationID, Ref: ref, Status: status, Code: code}, err
}

// AckInput carries all completed hash references; inputs: retained plan/checks; outputs: durable Outcome; effects: none in this type.
type AckInput struct {
	Plan    Handle   `json:"plan"`
	Checks  []Handle `json:"checks"`
	Current Handle   `json:"current"`
}

type currentEvidence struct {
	Plan      libraryvalidation.ArtifactRef `json:"plan"`
	Object    *Object                       `json:"object"`
	CheckedAt time.Time                     `json:"checked_at"`
}

// CheckCurrent performs a fresh current-version observation after separately completed byte hashes.
// Inputs: reconciliation handle; outputs: authenticated HEAD/absence descriptor. Effects: one HEAD and derivative retention.
// Choose immediately before backend CAS; changes after this bounded observation are retained by the next observer cycle, not hidden by an atomic-B2 claim.
func (s *Service) CheckCurrent(ctx context.Context, h Handle) (Handle, error) {
	var p ReconcilePlan
	if err := s.unseal(ctx, h.Ref, "reconcile", &p); err != nil {
		return Handle{}, err
	}
	c, err := s.claim(ctx, p.Claim)
	if err != nil {
		return Handle{}, err
	}
	obj, err := s.Storage.Head(ctx, c.Operation.Key)
	if err != nil {
		return Handle{}, err
	}
	if obj != nil && (obj.Bucket != c.Operation.Bucket || obj.Key != c.Operation.Key || !validVersion(obj.VersionID)) {
		return Handle{}, errors.New("current object coordinates mismatch")
	}
	ref, err := s.seal(ctx, "current", h.OperationID+"/current", currentEvidence{Plan: h.Ref, Object: obj, CheckedAt: s.clock()})
	return Handle{OperationID: h.OperationID, Ref: ref, Status: Pending}, err
}

// Acknowledge compares every retained version check and submits a fail-closed backend pointer CAS.
// Inputs: plan plus all hash handles; outputs: durable synced/conflict/unknown/retry outcome. Effects: descriptor reads and backend transaction.
// A racing upload/hide, duplicate own version, missing hash, stale record/pointer or partial coverage can never silently win.
func (s *Service) Acknowledge(ctx context.Context, in AckInput) (Outcome, error) {
	var p ReconcilePlan
	if err := s.unseal(ctx, in.Plan.Ref, "reconcile", &p); err != nil {
		return Outcome{}, err
	}
	c, err := s.claim(ctx, p.Claim)
	if err != nil {
		return Outcome{}, err
	}
	completion := Completion{IntentID: c.WriteIntentID, LeaseID: c.LeaseID, Fence: c.Fence, BasePointerRevision: c.Operation.BasePointerRevision, Observed: []Object{}, Coverage: p.Coverage, Status: WriteUnknown, Code: p.Code}
	checked := map[string]Object{}
	failed := len(in.Checks) != len(p.Objects)
	var current currentEvidence
	if in.Current.OperationID != c.Operation.OperationID || s.unseal(ctx, in.Current.Ref, "current", &current) != nil || current.Plan != in.Plan.Ref || current.CheckedAt.Before(p.CreatedAt) || current.CheckedAt.After(s.clock()) || s.clock().Sub(current.CheckedAt) > IOTimeout {
		failed = true
		completion.Code = "CURRENT_VERSION_CHECK_MISSING_OR_STALE"
	}
	for _, h := range in.Checks {
		var obj Object
		if h.OperationID != c.Operation.OperationID || s.unseal(ctx, h.Ref, "version-check", &obj) != nil {
			return Outcome{}, errors.New("reconciliation check invalid")
		}
		if _, exists := checked[obj.VersionID]; exists {
			failed = true
		}
		checked[obj.VersionID] = obj
		if h.Status == Blocked {
			failed = true
		}
	}
	own := []Object{}
	competing := false
	latestOwn := false
	for _, expected := range p.Objects {
		obj, ok := checked[expected.VersionID]
		if !ok || obj.Key != expected.Key || obj.Bucket != expected.Bucket || obj.Size != expected.Size || obj.Hidden != expected.Hidden || obj.Latest != expected.Latest || obj.UploadedAt != expected.UploadedAt {
			failed = true
			completion.Observed = append(completion.Observed, expected)
			continue
		}
		if !obj.Hidden && !rawHash.MatchString(obj.SHA256) {
			failed = true
		}
		completion.Observed = append(completion.Observed, obj)
		if c.Operation.BaseB2Pointer != nil && obj.VersionID == c.Operation.BaseB2Pointer.VersionID {
			if obj.Hidden || obj.SHA256 != c.Operation.BaseB2Pointer.SHA256 || obj.Size != c.Operation.BaseB2Pointer.Size {
				competing = true
			}
			continue
		}
		if !obj.Hidden && c.WriteIntentID != "" && obj.IntentID == c.WriteIntentID && obj.OperationID == c.Operation.OperationID && obj.SHA256 == c.Operation.PayloadSHA256 && obj.Size == c.Operation.PayloadSize && obj.ContentType == c.Operation.ContentType {
			own = append(own, obj)
			if obj.Latest {
				latestOwn = true
			}
		} else {
			competing = true
		}
	}
	if !failed {
		var expectedCurrent *Object
		for i := range completion.Observed {
			if completion.Observed[i].Latest {
				if expectedCurrent != nil {
					failed = true
					completion.Code = "MULTIPLE_LATEST_VERSIONS"
				}
				expectedCurrent = &completion.Observed[i]
			}
		}
		if expectedCurrent != nil && !expectedCurrent.Hidden {
			if current.Object == nil || current.Object.VersionID != expectedCurrent.VersionID || current.Object.Size != expectedCurrent.Size || current.Object.OperationID != expectedCurrent.OperationID || current.Object.IntentID != expectedCurrent.IntentID {
				competing = true
				completion.Code = "CURRENT_VERSION_CHANGED"
				if current.Object != nil {
					completion.Observed = append(completion.Observed, *current.Object)
				}
			}
		} else if current.Object != nil {
			competing = true
			completion.Code = "CURRENT_VERSION_CHANGED"
			completion.Observed = append(completion.Observed, *current.Object)
		}
	}
	if competing {
		completion.Status = Conflicted
		completion.Code = "COMPETING_RETAINED_VERSION"
	} else if failed || p.Coverage != "complete" {
		completion.Status = WriteUnknown
		if completion.Code == "" {
			completion.Code = "INCOMPLETE_VERSION_CHECKS"
		}
	} else if len(own) > 1 {
		completion.Status = Conflicted
		completion.Code = "MULTIPLE_OPERATION_VERSIONS"
	} else if len(own) == 1 && latestOwn {
		completion.Status = Synced
		completion.Written = &Pointer{VersionID: own[0].VersionID, SHA256: own[0].SHA256, Size: own[0].Size, ContentType: own[0].ContentType, ObservedAt: s.clock()}
	} else if len(own) == 0 {
		completion.Status = RetryWait
		completion.Code = "NO_WRITE_VERSION_OBSERVED"
	} else {
		completion.Status = WriteUnknown
		completion.Code = "WRITTEN_VERSION_NOT_CURRENT"
	}
	out, err := s.Backend.Complete(ctx, c, completion)
	if err == nil && out.Status == Synced && completion.Status != Synced {
		return Outcome{}, errors.New("backend accepted uncleared sync evidence")
	}
	return out, err
}

// RecordFailure persists a safe terminal/retry state for a leased operation; inputs: claim/status/code; outputs: Outcome.
// Effects: backend failure transaction. Choose when an Activity fails before reconciliation can produce complete evidence.
func (s *Service) RecordFailure(ctx context.Context, h Handle, status, code string) (Outcome, error) {
	c, err := s.claim(ctx, h)
	if err != nil {
		return Outcome{}, err
	}
	return s.Backend.Failure(ctx, c, status, code)
}
