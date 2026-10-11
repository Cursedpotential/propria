package humancontext

import (
	"context"
	"errors"
	"fmt"
)

// ValidateSourcePins compares registered identity and attribution with an observed source.
// Inputs: expected pins and source metadata from an opener or derivative manifest. Output: error or nil.
// Effects: none. Choose before downstream work; only unknown provider version, length and ETag may become observed.
func ValidateSourcePins(expected, observed Source) error {
	if expected.SourceRef == "" || expected.SourceVersionRef == "" || expected.RegistrationReceiptRef == "" {
		return errors.New("human source requires registered identity references")
	}
	for _, field := range []struct{ name, expected, observed string }{
		{"source_ref", expected.SourceRef, observed.SourceRef},
		{"source_version_ref", expected.SourceVersionRef, observed.SourceVersionRef},
		{"registration_receipt_ref", expected.RegistrationReceiptRef, observed.RegistrationReceiptRef},
		{"format", expected.Format, observed.Format},
		{"owner_person_ref", expected.OwnerPersonRef, observed.OwnerPersonRef},
		{"perspective_person_ref", expected.PerspectivePersonRef, observed.PerspectivePersonRef},
		{"perspective", expected.Perspective, observed.Perspective},
	} {
		if field.expected != field.observed {
			return fmt.Errorf("human source %s pin differs", field.name)
		}
	}
	if expected.ProviderVersionID != "" && expected.ProviderVersionID != observed.ProviderVersionID {
		return errors.New("human source provider version pin differs")
	}
	if expected.ByteLength < 0 || observed.ByteLength < 0 || (expected.ByteLength > 0 && expected.ByteLength != observed.ByteLength) {
		return errors.New("human source byte length pin differs")
	}
	if expected.ETag != "" && !SameETag(expected.ETag, observed.ETag) {
		return errors.New("human source ETag pin differs")
	}
	return nil
}

// validateStageManifest checks a derivative's contract, workflow, stage and exact source pins.
// Inputs: registered request, loaded manifest and expected predecessor stage. Output: error or nil.
// Effects: none. Choose before cached receipt reuse, DuckDB extraction or search operations.
func validateStageManifest(req Request, manifest Manifest, stage string) error {
	if req.ContractVersion != Contract || req.WorkflowID == "" || manifest.ContractVersion != Contract ||
		manifest.WorkflowID != req.WorkflowID || manifest.Stage != stage {
		return errors.New("human manifest contract, workflow or stage differs")
	}
	return ValidateSourcePins(req.Source, manifest.Source)
}

// cachedStage validates a persisted result's manifest before returning an idempotent retry.
// Inputs: request, output stage and source pins already checked on its predecessor. Outputs: cached result/found/error.
// Effects: metadata and manifest reads only. Choose over direct LookupStage so cache hits cannot bypass source validation.
func (b *Bundles) cachedStage(ctx context.Context, req Request, stage string, predecessor Source) (Result, bool, error) {
	if b.Metadata == nil {
		return Result{}, false, nil
	}
	result, found, err := b.Metadata.LookupStage(ctx, req, stage)
	if err != nil || !found {
		return result, found, err
	}
	manifest, err := b.ReadManifest(result.ManifestRef)
	if err != nil {
		return Result{}, false, err
	}
	if err := validateStageManifest(req, manifest, stage); err != nil {
		return Result{}, false, err
	}
	if err := ValidateSourcePins(predecessor, manifest.Source); err != nil {
		return Result{}, false, err
	}
	return result, true, nil
}
