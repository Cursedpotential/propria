package approvedgraphai

import "errors"

// ErrSupersededDecision identifies an old owner decision whose current receipt has changed or been withdrawn.
var ErrSupersededDecision = errors.New("approved native decision is superseded or no longer approved")
