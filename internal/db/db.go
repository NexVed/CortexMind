package db

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// LogActivity writes an entry to the activity_log collection. Failures are
// logged but never propagated — activity logging must never break a request.
func LogActivity(app core.App, projectID, ownerID, action, subject string, metadata map[string]any) {
	coll, err := app.FindCollectionByNameOrId(CollActivityLog)
	if err != nil {
		log.Error().Err(err).Msg("activity_log collection missing")
		return
	}
	rec := core.NewRecord(coll)
	if projectID != "" {
		rec.Set("project", projectID)
	}
	if ownerID != "" {
		rec.Set("owner", ownerID)
	}
	rec.Set("action", action)
	rec.Set("subject", subject)
	if metadata != nil {
		rec.Set("metadata", metadata)
	}
	if err := app.Save(rec); err != nil {
		log.Error().Err(err).Str("action", action).Msg("failed to write activity log")
	}
}
