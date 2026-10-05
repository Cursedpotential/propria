// Byline: Codex · GPT-6.1 · 2026-10-04.
package activities

import (
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
)

type libraryActivityRegistrar struct{ names []string }

func (r *libraryActivityRegistrar) RegisterActivityWithOptions(_ any, options activity.RegisterOptions) {
	r.names = append(r.names, options.Name)
}

func TestToolkitLibraryValidationRegistrationAndMissingConfiguration(t *testing.T) {
	registrar := &libraryActivityRegistrar{}
	acts := ToolkitLibraryValidationActivities{}
	RegisterToolkitLibraryValidationActivities(registrar, acts)
	require.Equal(t, []string{libraryvalidation.PrepareActivity, libraryvalidation.SnapshotActivity, libraryvalidation.VerifyActivity, libraryvalidation.ReceiptActivity}, registrar.names)
	_, err := acts.Prepare(t.Context(), libraryvalidation.LibraryValidationInput{})
	require.ErrorContains(t, err, "not configured")
	_, err = acts.SourceSnapshot(t.Context(), libraryvalidation.ClaimInput{})
	require.ErrorContains(t, err, "not configured")
	_, err = acts.VerifyClaim(t.Context(), libraryvalidation.ClaimInput{})
	require.ErrorContains(t, err, "not configured")
	_, err = acts.CommitReceipt(t.Context(), libraryvalidation.FinishInput{})
	require.ErrorContains(t, err, "not configured")
}
