// Byline: Codex · GPT-5 · 2026-10-05
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/contextreview"
	"github.com/Cursedpotential/probata/engine/sourcecontext"
	"github.com/Cursedpotential/probata/engine/sourcemeta"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var errAdmissionBegin = errors.New("admitted transaction sentinel")

// A nil embedded DB makes any unintended statement panic.
type previewAdmissionDB struct {
	DB
	detail        string
	matter, court *uuid.UUID
	reads, begins int
}

func (d *previewAdmissionDB) QueryRow(context.Context, string, ...any) pgx.Row {
	d.reads++
	return repairPlanRow{values: []any{d.detail, d.matter, d.court}}
}
func (d *previewAdmissionDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	d.begins++
	return nil, errAdmissionBegin
}

func TestOverlayStoresRequireDurableAdmissionBeforeTransaction(t *testing.T) {
	matter, court, foreign := uuid.MustParse(authoritativeMatterID), uuid.MustParse(authoritativeCourtCaseID), uuid.MustParse("11111111-1111-1111-1111-111111111111")
	valid := `{"operating_mode":"LIVE","matter_id":"01a0f751-e07b-75cc-9ad5-63ad9449a8ba","court_case_id":"01a0f751-e07b-76a1-a738-eb3e3aa3e68c"}`
	for _, tc := range []struct {
		name, detail string
		court        *uuid.UUID
		admitted     bool
	}{
		{"unknown", "", &court, false},
		{"DEV", strings.Replace(valid, "LIVE", "DEV", 1), &court, false},
		{"mode-only", `{"operating_mode":"LIVE"}`, &court, false},
		{"foreign-receipt", strings.Replace(valid, authoritativeMatterID, foreign.String(), 1), &court, false},
		{"foreign-source-court", valid, &foreign, false},
		{"LIVE", valid, &court, true},
	} {
		for _, family := range []string{"review-and-foreshadowing", "source-correction"} {
			t.Run(tc.name+"/"+family, func(t *testing.T) {
				db := &previewAdmissionDB{detail: tc.detail, matter: &matter, court: tc.court}
				var err error
				if family == "review-and-foreshadowing" {
					store := &ContextReviewStore{db: db}
					_, err = store.persist(context.Background(), overlayWrite{subject: contextreview.Subject{PreviewHandle: testPreviewHandle, MessageID: "11111111-1111-1111-1111-111111111111"}})
				} else {
					store := &SourceMetadataStore{db: db}
					_, err = store.PersistCorrection(context.Background(), sourcemeta.CorrectionSpec{PreviewHandle: testPreviewHandle, SubjectSHA256: strings.Repeat("ab", 32), FieldKey: "embedded:time", Action: sourcemeta.ActionCorrect, CorrectedValue: json.RawMessage(`"fixed"`), ChangeReason: "owner correction", ActorSubjectUID: "uid", ActorUsername: "owner", IdempotencyKey: "key"})
				}
				if err == nil || db.reads != 1 {
					t.Fatalf("err=%v db=%+v", err, db)
				}
				if tc.admitted {
					if !errors.Is(err, errAdmissionBegin) || db.begins != 1 {
						t.Fatalf("LIVE denied: err=%v db=%+v", err, db)
					}
				} else if db.begins != 0 {
					t.Fatalf("unverified write reached transaction: %+v", db)
				}
			})
		}
	}
}

func TestPreImportSourceContextRejectsUnknownDevOrForeignScopeBeforeDatabase(t *testing.T) {
	store := &SourceContextStore{}
	for _, tc := range []sourcecontext.Spec{
		{OperatingMode: ""}, {OperatingMode: "DEV"}, {OperatingMode: "REAL"},
		{OperatingMode: "LIVE", MatterID: authoritativeMatterID, CourtCaseID: "11111111-1111-1111-1111-111111111111"},
	} {
		if _, err := store.PersistSourceContext(context.Background(), tc); err == nil {
			t.Fatal("unverified bootstrap admitted", tc)
		}
	}
}
