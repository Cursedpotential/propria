// Byline: Claude Code · Sonnet · 2026-10-02
package activities

import (
	"context"
	"testing"

	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// aiChatStore implements only what the refusal paths touch; any other call panics on the nil embed.
type aiChatStore struct {
	FirstPartyContextStore
	format string
}

func (s aiChatStore) LoadFirstPartyContext(context.Context, proffer.StageRequest, proffer.Ref, proffer.Ref) (FirstPartyContextInput, error) {
	return FirstPartyContextInput{
		Source:           firstparty.Source{DeclaredFormat: s.format},
		Messages:         []firstparty.SourceMessage{{}},
		PlatformResolved: true,
	}, nil
}

func aiChatRequest() proffer.StageRequest {
	return proffer.StageRequest{
		RequestID: "r1", SourceVersionRef: "sv1",
		Refs: map[string]proffer.Ref{"normalized_generation": "g1", "normalized_verification": "v1"},
	}
}

func requireNonRetryableAIChat(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.True(t, appErr.NonRetryable())
	require.Contains(t, appErr.Error(), "search-only")
}

func TestFirstPartyProposeRefusesAIChatFormats(t *testing.T) {
	for _, format := range []string{"chatgpt_official_json", "chatgpt_json_array"} {
		acts := FirstPartyContextActivities{Store: aiChatStore{format: format}}
		_, err := acts.ProposeFirstPartyContext(context.Background(), aiChatRequest())
		requireNonRetryableAIChat(t, err)
	}
}

func TestFirstPartyRebuildBackstopRefusesAIChatBeforeAnyCommit(t *testing.T) {
	acts := FirstPartyContextActivities{Store: aiChatStore{format: "chatgpt_official_json"}}
	_, err := acts.rebuild(context.Background(), aiChatRequest(), FirstPartyReceipt{SourceVersionRef: "sv1", NormalizedGenerationRef: "g1"}, "v1")
	require.Error(t, err)
	requireNonRetryableAIChat(t, stopRetryingPermanent(err))
}
