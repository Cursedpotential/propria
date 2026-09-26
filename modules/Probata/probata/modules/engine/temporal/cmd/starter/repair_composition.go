// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Starter wiring for the repair workflow builder's HTTP routes. The starter
// validates plans for the Workbench against the same storage configuration
// the worker writes with (OBJECT_STORES_JSON, SOURCE_ROOTS_JSON,
// DERIVED_ROOTS_JSON); the worker validates again before any step runs, so a
// difference between the two can refuse a plan but never misplace a write.

package main

import (
	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/repairplan"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
)

func newRepairPlanService(db platformpostgres.DB, c client.Client, taskQueue string) (repairplan.Service, error) {
	anchors, err := platformpostgres.NewRepairPlanStore(db)
	if err != nil {
		return repairplan.Service{}, err
	}
	stores, err := objectstores.StoresFromEnv()
	if err != nil {
		return repairplan.Service{}, err
	}
	roots, err := objectstores.RootsFromEnv()
	if err != nil {
		return repairplan.Service{}, err
	}
	derivedRoots, err := smsthreads.DerivedRootsFromEnv()
	if err != nil {
		return repairplan.Service{}, err
	}
	if err := derivedRoots.RequireConfiguredSchemes(stores.Schemes()); err != nil {
		return repairplan.Service{}, err
	}
	runs, err := platformtemporal.NewRepairPlanStarter(c, taskQueue)
	if err != nil {
		return repairplan.Service{}, err
	}
	return repairplan.Service{
		Env: repairplan.Environment{
			Registry: repairplan.DefaultRegistry(), Anchors: anchors, DerivedRoots: derivedRoots,
			Stores: stores, SourceRoots: roots, MatterMode: platformpostgres.MatterModeForIdentity,
		},
		Runs: runs,
	}, nil
}
