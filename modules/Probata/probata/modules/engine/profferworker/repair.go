// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Worker wiring for the repair workflow builder: the five repair-plan
// Activities RepairPlanWorkflow schedules. Storage placement is the same
// configuration the derive route reads (OBJECT_STORES_JSON, SOURCE_ROOTS_JSON,
// DERIVED_ROOTS_JSON, DERIVE_SCRATCH_DIR); no provider, bucket or vault path
// is named here.

package profferworker

import (
	"fmt"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/repairplan"
)

func buildRepairPlanActivities(
	db platformpostgres.DB,
	cfg Config,
	stores objectstores.Stores,
	derivedRoots smsthreads.DerivedRoots,
	objectStores func(string) (smsthreads.ObjectStore, error),
	catalog activities.CatalogVersionFinder,
) (activities.RepairPlanActivities, error) {
	planStore, err := platformpostgres.NewRepairPlanStore(db)
	if err != nil {
		return activities.RepairPlanActivities{}, err
	}
	roots, err := objectstores.RootsFromEnv()
	if err != nil {
		return activities.RepairPlanActivities{}, err
	}
	find := activities.RepairFindOtherVersionActivity{Stores: objectStores, Roots: roots}
	if catalog != nil {
		if !stores.Has(cfg.Catalog.ObjectScheme) {
			return activities.RepairPlanActivities{}, fmt.Errorf(
				"proffer worker: INTAKE_DISCOVERY_OBJECT_STORE scheme %q is not a configured object store (configured: %v)",
				cfg.Catalog.ObjectScheme, stores.Schemes())
		}
		find.Catalog = catalog
		find.Store = activities.CatalogStore{Scheme: cfg.Catalog.ObjectScheme, Bucket: cfg.Catalog.ObjectBucket}
	}
	return activities.NewRepairPlanActivities(activities.RepairPlanActivities{
		Validate: activities.RepairPlanValidateActivity{Environment: repairplan.Environment{
			Registry: repairplan.DefaultRegistry(), Anchors: planStore, DerivedRoots: derivedRoots,
			Stores: stores, SourceRoots: roots, IdentityAdmitted: platformpostgres.AdmittedCaseIdentity,
		}},
		Receipts: activities.RepairStepReceiptActivity{Store: planStore},
		Find:     find,
		Salvage: activities.RepairSalvageTruncatedXMLActivity{
			Stores: objectStores, DerivedRoots: derivedRoots, Roots: roots, ScratchRoot: cfg.DeriveScratchDir,
		},
		Lenient: activities.RepairLenientDecodeActivity{
			Stores: objectStores, DerivedRoots: derivedRoots, Roots: roots,
			ScratchRoot: cfg.DeriveScratchDir, MaxChunk: cfg.DeriveMaxChunkBytes,
		},
	}), nil
}
