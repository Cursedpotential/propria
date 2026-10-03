// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Conversation-level extraction and the Surreal send, wired into the proffer
// worker. Kept beside extraction.go: these workflows are standalone and their
// Activities are none of the 26 Proffer stages.
package profferworker

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// RegisterConversationExtraction installs the three conversation workflows and their Activities under their exact names.
func RegisterConversationExtraction(registrar ExtractionRegistrar, acts activities.ConversationActivities) {
	flow.RegisterConversationWorkflows(registrar)
	activities.RegisterConversationActivities(registrar, acts)
}

// buildConversationActivities constructs the conversation Activities. An absent Surreal connection
// disables only the send (it reports why); a present but invalid one is a loud boot failure.
func buildConversationActivities(db platformpostgres.DB, store *platformpostgres.EntityExtractionStore) (activities.ConversationActivities, error) {
	if store == nil {
		var err error
		if store, err = platformpostgres.NewEntityExtractionStore(db); err != nil {
			return activities.ConversationActivities{}, err
		}
	}
	cfg, err := surrealsink.ConfigFromEnv()
	switch {
	case errors.Is(err, surrealsink.ErrNotConfigured):
		slog.Info("send to surreal is disabled on this worker", "reason", err.Error())
		return activities.NewConversationActivities(store, store, nil, err.Error()), nil
	case err != nil:
		return activities.ConversationActivities{}, fmt.Errorf("proffer worker: surreal-case configuration: %w", err)
	}
	client, err := surrealsink.New(cfg, nil)
	if err != nil {
		return activities.ConversationActivities{}, fmt.Errorf("proffer worker: surreal-case client: %w", err)
	}
	slog.Info("send to surreal is enabled", "namespace", cfg.Namespace, "database", cfg.Database)
	return activities.NewConversationActivities(store, store, client, ""), nil
}
