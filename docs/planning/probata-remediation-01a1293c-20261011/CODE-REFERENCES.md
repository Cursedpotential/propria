# Verified immutable code references

Repository: Cursedpotential/propria. Each path and line interval below was checked against its exact Git blob. A code reference proves the inspected implementation, not live deployment, table existence, corpus completeness or test success. Full 40-character commits and URLs are also in SOURCE-EVIDENCE.json. Product names are Probata Sources/Activity/Read; workbench in paths is the existing internal module name.

## E01

Local Process uploads then starts Proffer; folder processing uses current locator; no final placement receipt.

[modules/Probata/probata/modules/workbench/web/src/components/sources/sources-screen.tsx @ 6a9b1626e2fd:328–401](https://github.com/Cursedpotential/propria/blob/6a9b1626e2fd14878fcaee7473cb0f5c8c43a2ba/modules/Probata/probata/modules/workbench/web/src/components/sources/sources-screen.tsx#L328-L401)

## E02

Process gate checks busy/target/matter/court, not verified permanent placement.

[modules/Probata/probata/modules/workbench/web/src/components/sources/sources-screen.tsx @ 6a9b1626e2fd:447–462](https://github.com/Cursedpotential/propria/blob/6a9b1626e2fd14878fcaee7473cb0f5c8c43a2ba/modules/Probata/probata/modules/workbench/web/src/components/sources/sources-screen.tsx#L447-L462)

## E03

Unit-kind marks and canonical-home explanatory text are not classification or placement.

[modules/Probata/probata/modules/workbench/web/src/components/sources/source-metadata-panel.tsx @ 9399c24824e3:210–245](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/workbench/web/src/components/sources/source-metadata-panel.tsx#L210-L245)

## E04

Read-modify-write local JSON unit marks.

[modules/Probata/probata/modules/workbench/api/app/service/source_unit_marks.py @ 9399c24824e3:43–60](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/workbench/api/app/service/source_unit_marks.py#L43-L60)

## E05

Atomic file replacement without shared catalog membership transaction.

[modules/Probata/probata/modules/workbench/api/app/repo/source_unit_marks.py @ 9399c24824e3:34–54](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/workbench/api/app/repo/source_unit_marks.py#L34-L54)

## E06

Upload seal is upload://sha256 digest+size, not final B2 placement.

[modules/Probata/probata/modules/engine/acquisition/upload.go @ 9399c24824e3:79–123](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/engine/acquisition/upload.go#L79-L123)

## E07

Local immutable content seal.

[modules/Probata/probata/modules/engine/acquisition/seal.go @ 9399c24824e3:118–164](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/engine/acquisition/seal.go#L118-L164)

## E08

Existing add-only B2 copy and size/SHA1 readback machinery; not shared application placement.

[modules/Consignatio/casebible/tools/casevault_placement.sh @ 9399c24824e3:3–65](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/casebible/tools/casevault_placement.sh#L3-L65)

## E09

Existing raw_duck placement mapping upserts latest verification status; no immutable attempt history.

[modules/Consignatio/casebible/tools/casevault_placement_load.sql @ 9399c24824e3:6–48](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/casebible/tools/casevault_placement_load.sql#L6-L48)

## E10

Registration keyed by AcquisitionRef and request replay rejects changed reference.

[modules/Probata/probata/modules/engine/postgres/source_lifecycle_repository.go @ 9399c24824e3:99–145](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/engine/postgres/source_lifecycle_repository.go#L99-L145)

## E11

State inferred from source_ref and completed run/manifest; independent states collapsed.

[modules/Probata/probata/modules/workbench/web/src/components/sources/source-state.ts @ 9399c24824e3:4–68](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/workbench/web/src/components/sources/source-state.ts#L4-L68)

## E12

Local filesystem rename; skips missing source/existing target; not shared B2 service.

[modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot/apps/src-tauri/src/file_organizer.rs @ 9399c24824e3:884–921](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot/apps/src-tauri/src/file_organizer.rs#L884-L921)

## E13

Dated catalog membership joins; requires current generation/lineage reconciliation.

[modules/Probata/probata/modules/workbench/api/app/repo/intake_discovery.py @ 9399c24824e3:113–210](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/workbench/api/app/repo/intake_discovery.py#L113-L210)

## E14

Provider filename listing/search is a different search surface, bounded at 25000 keys.

[modules/Probata/probata/modules/workbench/api/app/service/proffer_sources.py @ 9399c24824e3:179–234](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/workbench/api/app/service/proffer_sources.py#L179-L234)

## E15

Current raw_duck.bucket_objects reader plus historical relocation mapping; old b2_objects complaint stale.

[modules/Probata/probata/modules/engine/postgres/catalog_versions.go @ e1cb96cbd09d:61–101](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/postgres/catalog_versions.go#L61-L101)

## E16

Native AI finish retains/catalogs package after preparation/extraction/publication; partial candidate outcomes.

[modules/Probata/probata/modules/engine/proffer/context_first.go @ 059c8aa5f943:207–356](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/proffer/context_first.go#L207-L356)

## E17

Fixed manifest-derived _Incoming destination; exact completed replay; uncertain attempt cannot auto-resume.

[modules/Probata/probata/modules/engine/activities/ai_context_package.go @ 059c8aa5f943:247–307](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_context_package.go#L247-L307)

## E18

Exact ai-context-catalog/v1 UTF8 payload, root and member admission contract.

[modules/Consignatio/casebible/tools/ai_context_package_catalog_api.sql @ 059c8aa5f943:16–80](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Consignatio/casebible/tools/ai_context_package_catalog_api.sql#L16-L80)

## E19

Exact catalog reconstruction and serializable atomic registration in existing raw_duck relations.

[modules/Consignatio/casebible/tools/ai_context_package_catalog_api.sql @ 059c8aa5f943:89–208](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Consignatio/casebible/tools/ai_context_package_catalog_api.sql#L89-L208)

## E20

Independent registration/readback transactions and complete receipt comparison.

[modules/Probata/probata/modules/engine/activities/ai_context_package_catalog.go @ 059c8aa5f943:77–160](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_context_package_catalog.go#L77-L160)

## E21

Baseline BFF omits matter/court IDs and exposes ChatGPT/Claude only.

[modules/Probata/probata/modules/workbench/api/app/runtime/context_sources.py @ 059c8aa5f943:25–37](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/workbench/api/app/runtime/context_sources.py#L25-L37)

## E22

Divergent restored BFF adds Gemini but still omits scope.

[modules/Probata/probata/modules/workbench/api/app/runtime/context_sources.py @ f9b4bd6478b0:26–38](https://github.com/Cursedpotential/propria/blob/f9b4bd6478b017162aba3d561c0bc311d0b07ff0/modules/Probata/probata/modules/workbench/api/app/runtime/context_sources.py#L26-L38)

## E23

Native staging requires LIVE scope and returns existing local PreparedRef.

[modules/Probata/probata/modules/engine/activities/ai_context_candidate_staging.go @ 059c8aa5f943:49–66](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_context_candidate_staging.go#L49-L66)

## E24

Restored candidate BFF requires b2 prepared_ref; native file ref fails validation as502.

[modules/Probata/probata/modules/workbench/api/app/runtime/ai_candidate_review.py @ f9b4bd6478b0:19–55](https://github.com/Cursedpotential/propria/blob/f9b4bd6478b017162aba3d561c0bc311d0b07ff0/modules/Probata/probata/modules/workbench/api/app/runtime/ai_candidate_review.py#L19-L55)

## E25

Native bundle refs are worker-local file URIs with exact run/stage path checks.

[modules/Probata/probata/server/analysis/ai_context.py @ 059c8aa5f943:85–156](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/analysis/ai_context.py#L85-L156)

## E26

Gemini decoder accepts label-only full lines, rejecting inline role-label form.

[modules/Probata/probata/server/analysis/ai_context.py @ 059c8aa5f943:353–376](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/analysis/ai_context.py#L353-L376)

## E27

Detector test admits inline Gemini role labels.

[modules/Probata/probata/modules/engine/sourceformat/ai_gemini_test.go @ 059c8aa5f943:5–13](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/sourceformat/ai_gemini_test.go#L5-L13)

## E28

Detector format IDs include ChatGPT/Claude Markdown beyond context workflow support.

[modules/Probata/probata/modules/engine/sourceformat/ai.go @ 059c8aa5f943:14–151](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/sourceformat/ai.go#L14-L151)

## E29

Created works retain native role/time/pointer/span provenance and source-local bundle references.

[modules/Probata/probata/server/analysis/ai_context.py @ 059c8aa5f943:475–503](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/analysis/ai_context.py#L475-L503)

## E30

Separate legacy AI work-product fixed _derived/misc destination.

[modules/Probata/probata/modules/engine/activities/ai_workproduct_placement.go @ 059c8aa5f943:26–31](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_workproduct_placement.go#L26-L31)

## E31

Fixed KnowledgeBase/legal content root.

[modules/Probata/probata/modules/engine/activities/toolkit_content_placement.go @ e1cb96cbd09d:28–60](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/activities/toolkit_content_placement.go#L28-L60)

## E32

Exact legal root and admissible subprefix contract.

[modules/Probata/probata/modules/engine/extraction/librarysync/contract.go @ e1cb96cbd09d:42–72](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/extraction/librarysync/contract.go#L42-L72)

## E33

Wiki default and exact equality guard at Code/wiki.

[modules/Consignatio/casebible/tools/publish_wiki.py @ e1cb96cbd09d:43–46](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Consignatio/casebible/tools/publish_wiki.py#L43-L46)

## E34

Configurable library-validation DerivedKnowledge default and derivative-root configuration.

[modules/Probata/probata/deploy/proffer-worker.yaml @ e1cb96cbd09d:125–135](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/deploy/proffer-worker.yaml#L125-L135)

## E35

Existing versioned B2 derivative artifact writer and independent exact-version readback.

[modules/Probata/probata/modules/engine/extraction/libraryvalidation/artifacts.go @ e1cb96cbd09d:22–84](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/extraction/libraryvalidation/artifacts.go#L22-L84)

## E36

Wrong lowercase recovery/library-sources destination persists.

[modules/Probata/probata/modules/engine/activities/toolkit_package_preservation.go @ e1cb96cbd09d:26–36](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/activities/toolkit_package_preservation.go#L26-L36)

## E37

Recovered correction was a plan; historical counts not current live proof.

[modules/Consignatio/docs/receipts/recovered-placement-correction-20261004.md @ e1cb96cbd09d:5–15](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Consignatio/docs/receipts/recovered-placement-correction-20261004.md#L5-L15)

## E38

Legacy catalog_reconcile/vault defaults; evidence promotion intentionally unbound.

[modules/Legal-desktop/api/legal_workspace/services/consignatio_catalog.py @ e1cb96cbd09d:30–76](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Legal-desktop/api/legal_workspace/services/consignatio_catalog.py#L30-L76)

## E39

JSON bindings override merges sections if configured.

[modules/Legal-desktop/api/legal_workspace/services/consignatio_catalog.py @ e1cb96cbd09d:122–132](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Legal-desktop/api/legal_workspace/services/consignatio_catalog.py#L122-L132)

## E40

Catalog URL and bindings env settings.

[modules/Legal-desktop/api/legal_workspace/config.py @ e1cb96cbd09d:57–58](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Legal-desktop/api/legal_workspace/config.py#L57-L58)

## E41

Acquired-third-party availability joins evidence.acquisition.

[modules/Probata/probata/sql/bootstrap/schema_snapshot_20260907.sql @ e1cb96cbd09d:2894–2912](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/sql/bootstrap/schema_snapshot_20260907.sql#L2894-L2912)

## E42

ADR extraction lifecycle/no per-row approval and candidate families left unreconciled.

[modules/Probata/probata/docs/adr/0062-claim-candidates-temporal-edges-and-the-assertion-layer.md @ e1cb96cbd09d:57–88](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/docs/adr/0062-claim-candidates-temporal-edges-and-the-assertion-layer.md#L57-L88)

## E43

Claim candidate review_state pending differs from ADR lifecycle.

[modules/Probata/probata/sql/bootstrap/schema_snapshot_20260907.sql @ e1cb96cbd09d:11787–11837](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/sql/bootstrap/schema_snapshot_20260907.sql#L11787-L11837)

## E44

AI pending candidate_fact, undated event and strategy/history fact-account mapping.

[modules/Probata/probata/modules/engine/postgres/ai_content_review_store.go @ e1cb96cbd09d:199–239](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/postgres/ai_content_review_store.go#L199-L239)

## E45

Deterministic stage-local JSON durability does not establish portable DerivedKnowledge envelope.

[modules/Probata/probata/server/analysis/ai_content.py @ e1cb96cbd09d:102–146](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/server/analysis/ai_content.py#L102-L146)

## E46

Direct AI Weaviate publish and independent vector/property readback already exist.

[modules/Probata/probata/server/analysis/ai_content.py @ e1cb96cbd09d:1799–1862](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/server/analysis/ai_content.py#L1799-L1862)

## E47

Approved entity/event graph claims retain exact provenance; scope is limited.

[modules/Probata/probata/modules/engine/approvedgraph/project.go @ e1cb96cbd09d:17–101](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/approvedgraph/project.go#L17-L101)

## E48

Existing typed source-pinned graph bundle.

[modules/Probata/probata/modules/engine/surrealsink/context_graph_contract.go @ e1cb96cbd09d:32–84](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/surrealsink/context_graph_contract.go#L32-L84)

## E49

Mapped derivatives plus source-local fallback/legacy lookup; rule requires reconciliation.

[modules/Probata/probata/modules/engine/derive/smsthreads/derivedroot.go @ e1cb96cbd09d:5–28](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/derive/smsthreads/derivedroot.go#L5-L28)

## E50

Env mapping rejects configured destination inside source root.

[modules/Probata/probata/modules/engine/derive/smsthreads/derivedroot.go @ e1cb96cbd09d:81–115](https://github.com/Cursedpotential/propria/blob/e1cb96cbd09ddd07df8cad4dc44c843631e44e59/modules/Probata/probata/modules/engine/derive/smsthreads/derivedroot.go#L81-L115)

## E51

Advanced evidence promotion disabled.

[modules/Probata/probata/server/case_management/repository.py @ 059c8aa5f943:112–115](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/case_management/repository.py#L112-L115)

## E52

Production LegalSourcePackage issuer/route not wired.

[modules/Probata/probata/server/contracts/legal_source_package.py @ 059c8aa5f943:1–5](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/contracts/legal_source_package.py#L1-L5)

## E53

Evidence implementation contract adds no executor/request code.

[modules/Probata/probata/docs/pending-review/2026-10-06-evidence-promotion-implementation-contract.md @ 059c8aa5f943:50–61](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/docs/pending-review/2026-10-06-evidence-promotion-implementation-contract.md#L50-L61)

## E54

Current all-bucket listing view with dated occurrence join; historical counts only.

[modules/Consignatio/casebible/tools/superindex_source_current_20261003.sql @ 059c8aa5f943:1–40](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Consignatio/casebible/tools/superindex_source_current_20261003.sql#L1-L40)

## E55

Search locator resolution and explicit unknown coverage; optional image lane.

[modules/Consignatio/Intake/backend/src/casebible_index/filesystem_search.py @ 059c8aa5f943:50–75](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Consignatio/Intake/backend/src/casebible_index/filesystem_search.py#L50-L75)

## E56

Independent OCR literal and visual lanes exist with screenshot/photo/scan filtering.

[modules/Consignatio/Intake/backend/src/casebible_index/image_search_api.py @ 059c8aa5f943:1–20](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Consignatio/Intake/backend/src/casebible_index/image_search_api.py#L1-L20)

## E57

Run counts explicitly do not prove corpus coverage.

[modules/Consignatio/Intake/backend/src/casebible_index/run_status.py @ 059c8aa5f943:13–41](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Consignatio/Intake/backend/src/casebible_index/run_status.py#L13-L41)

## E58

Go Temporal text/image stages; image stage disabled unless configured.

[modules/Probata/probata/modules/engine/superindex/workflow.go @ 059c8aa5f943:9–32](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/superindex/workflow.go#L9-L32)

## E59

Old helper generates deletion candidates from name+size; unsuitable for placement service.

[modules/Consignatio/casebible/tools/vault_move_rest.py @ 9399c24824e3:6–89](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/casebible/tools/vault_move_rest.py#L6-L89)

## E60

Metadata-only large file preview on restored branch; preserve current trial preview behavior.

[modules/Probata/probata/modules/workbench/api/app/service/source_inspection.py @ f9b4bd6478b0:218–248](https://github.com/Cursedpotential/propria/blob/f9b4bd6478b017162aba3d561c0bc311d0b07ff0/modules/Probata/probata/modules/workbench/api/app/service/source_inspection.py#L218-L248)

## E61

Source-unit placement also targets AI _Incoming.

[modules/Probata/probata/modules/engine/activities/ai_source_unit_placement.go @ 059c8aa5f943:26–34](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_source_unit_placement.go#L26-L34)

## E62

Go metadata validator independently enforces exact AI package root.

[modules/Probata/probata/modules/engine/postgres/context_package.go @ 059c8aa5f943:40–43](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/postgres/context_package.go#L40-L43)

## E63

Physical exact pinned B2 version hash/size verification precedes catalog proof.

[modules/Probata/probata/modules/engine/activities/ai_context_package.go @ 059c8aa5f943:196–228](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_context_package.go#L196-L228)

## E64

Current main unconditionally schedules OCR before image embeddings; any OCR Activity error aborts the image slice before embedding/publication.

[modules/Probata/probata/modules/engine/superindex/workflow.go @ 9399c24824e3:478–506](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/engine/superindex/workflow.go#L478-L506)

## E65

Image OCR raises non-retryable OcrUnavailable for an unavailable engine; configuration errors propagate rather than becoming an optional OCR coverage result.

[modules/Consignatio/Intake/backend/src/casebible_index/image_stage.py @ 9399c24824e3:379–416](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/Intake/backend/src/casebible_index/image_stage.py#L379-L416)

## E66

Image facts hash retained image/rendered-page bytes, inspect dimensions/format and metadata with OCR off, and classify kind with basis; device/software/GPS/time-conflict fields exist. This is not implemented model-based platform or capture-device attribution.

[modules/Consignatio/Intake/backend/src/casebible_index/image_stage.py @ 9399c24824e3:267–318](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/Intake/backend/src/casebible_index/image_stage.py#L267-L318)

## E67

Existing AI integration branch already performs optional model candidate extraction and exact quote/record matching to native Unicode-codepoint spans, pointers, roles and clocks, with per-chunk partial enrichment/cooldown. LangExtract must be benchmarked against this existing capability; code is not live delivery proof.

[modules/Probata/probata/server/analysis/ai_context.py @ 9fe9f8b31af9:538–680](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/server/analysis/ai_context.py#L538-L680)

## E68

Newer inspected native worker branch registers then prepares/extracts/enriches/publishes/verifies; its finish function only records status and omits the baseline assembled-package retention/catalog Activities. This branch difference is not proof of an integrated permanent-home lifecycle.

[modules/Probata/probata/modules/engine/proffer/context_first.go @ 9fe9f8b31af9:180–293](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/modules/engine/proffer/context_first.go#L180-L293)

## E69

The native package manifest embeds the complete ContextPackageRequest and requires its RunID; manifest-hash-derived destination therefore incorporates processing-request/run data, alongside the baseline root construction in E17.

[modules/Probata/probata/modules/engine/activities/ai_context_package.go @ 059c8aa5f943:64–110](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_context_package.go#L64-L110)

## E70

Native chunk configuration explicitly defaults to neural_distilbert, bounded 450-model-token windows, 7500-character maximum and two-message overlap for the message-thread route. This is configuration, not proof every route or live deployment uses it.

[modules/Probata/probata/server/context_chunks/config.py @ 9fe9f8b31af9:24–34](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/server/context_chunks/config.py#L24-L34)

## E71

Inspected main explicitly names mirth/chonky_distilbert_base_uncased_1, constructs WindowedNeural around Chonkie NeuralChunker, and includes library/model/window/size/overlap/text-format in chunker version. Exact resolved model/tokenizer artifact revisions and integrated live operation still require proof.

[modules/Probata/probata/server/context_chunks/chunker.py @ 9399c24824e3:35–160](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/server/context_chunks/chunker.py#L35-L160)

## E72

AI integration prepares complete-conversation neural topic chunks with Unicode-codepoint offsets, native-record segment slices and a persisted prepared bundle containing records/chunks. Existing local persistence is confirmed; durable source-local readback and exact-generation owner review/approval are not proved by this code.

[modules/Probata/probata/server/analysis/ai_context.py @ 9fe9f8b31af9:393–472](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/server/analysis/ai_context.py#L393-L472)

## E73

Inspected worker branch declares chonkie[semantic,code,table]==1.7.0 and explicitly documents omission of neural/torch extras from this dependency declaration. Actual image/runtime dependencies may differ and must be verified; this alone does not prove NeuralChunker is unavailable.

[modules/Probata/probata/pyproject.toml @ 9fe9f8b31af9:65–67](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/pyproject.toml#L65-L67)

## E74

Native v1 catalog payload has exactly 13 fields and each member has six fields. Contract changes need a distinct version rather than adding fields to historical v1 payloads.

[modules/Probata/probata/modules/engine/activities/ai_context_package_catalog.go @ 059c8aa5f943:25–46](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/activities/ai_context_package_catalog.go#L25-L46)

## E75

Native catalog adapter requires ai-context-catalog/v1 and 4–65 members, registers in a serializable transaction, then performs an independent read-only reconstruction. Unknown commit outcomes require identical pinned-payload retry rather than a new package identity.

[modules/Probata/probata/modules/engine/postgres/ai_context_package_catalog.go @ 059c8aa5f943:18–59](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/postgres/ai_context_package_catalog.go#L18-L59)

## E76

Current main Intake query derives resolution from occurrences[0].disposition; an underlying resolution column is not required. Compatibility still requires the query input columns and effective runtime bindings, not assumed view shape.

[modules/Consignatio/Intake/backend/src/casebible_index/sql/catalog_source_current.sql @ 9399c24824e3:1–24](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Consignatio/Intake/backend/src/casebible_index/sql/catalog_source_current.sql#L1-L24)

## E77

Newer native worker retains assistant-created work text with role/native time/source reference/native JSON pointer and codepoint spans; extraction is not legal adoption.

[modules/Probata/probata/server/analysis/ai_context.py @ 9fe9f8b31af9:475–504](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/server/analysis/ai_context.py#L475-L504)

## E78

Current private legal document/revision DTOs expose document/matter/revision/digest/actor/origin; no explicit AI source-generation/native-span adoption mapping in these fields.

[modules/Legal-desktop/api/legal_workspace/services/work_documents.py @ 9399c24824e3:46–66](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/services/work_documents.py#L46-L66)

## E79

Current legal import allocates a private document and stores immutable revision bytes plus audit-outbox event in one transaction; import is not a human approval.

[modules/Legal-desktop/api/legal_workspace/services/work_documents.py @ 9399c24824e3:156–227](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/services/work_documents.py#L156-L227)

## E80

Existing substantive review requires eligible authenticated human; agent, transport and tailnet identities do not establish legal adoption.

[modules/Legal-desktop/api/legal_workspace/api/auth.py @ 9399c24824e3:60–92](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/api/auth.py#L60-L92)

## E81

Existing legal work product has distinct private/review/approved/release/supersession states and edit-version semantics; source_package_id does not itself provide exact AI generation/span mapping.

[modules/Legal-desktop/api/legal_workspace/domain/work_product.py @ 9399c24824e3:14–58](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/domain/work_product.py#L14-L58)

## E82

Existing section approval binds human reviewer and current content hash/package context; changed hash invalidates latest approval. Its citation/package requirements belong to this review route.

[modules/Legal-desktop/api/legal_workspace/services/workspace.py @ 9399c24824e3:1030–1088](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/services/workspace.py#L1030-L1088)

## E83

Whole-document import/edit actor authentication and selected workspace matter are distinct from substantive human approval; explicit adoption must not reuse ordinary import as implicit approval.

[modules/Legal-desktop/api/legal_workspace/api/work_document_routes.py @ 9399c24824e3:25–39](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/api/work_document_routes.py#L25-L39)

## E84

Existing StrategyKind and StrategyNote cover theory, strategy, direction, scratch draft, idea and chat extract; source_chat alone does not supply exact native generation/span lineage.

[modules/Legal-desktop/api/legal_workspace/domain/strategy.py @ 9399c24824e3:16–36](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/domain/strategy.py#L16-L36)

## E85

Existing add/list strategy sibling persists private notes through Workspace._write; this does not prove the shared mobile/workdesk adapter or exact provenance fields.

[modules/Legal-desktop/api/legal_workspace/services/workspace.py @ 9399c24824e3:529–542](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Legal-desktop/api/legal_workspace/services/workspace.py#L529-L542)

## E86

Newer native worker BFF retains source/version/format fields without matter/court-case scope.

[modules/Probata/probata/modules/workbench/api/app/runtime/context_sources.py @ 9fe9f8b31af9:25–37](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/modules/workbench/api/app/runtime/context_sources.py#L25-L37)

## E87

Go request includes optional case IDs in workflow identity, but API validation bounds strings without fresh registry approval.

[modules/Probata/probata/modules/engine/runtimeapi/context_source_api.go @ 059c8aa5f943:54–116](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/runtimeapi/context_source_api.go#L54-L116)

## E88

Native registration accepts optional scope; incomplete or invalid UUID pairs become unscoped SQL coordinates, so registration does not prevent later missing-scope staging failure.

[modules/Probata/probata/modules/engine/postgres/context_source_registration.go @ 059c8aa5f943:181–230](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/postgres/context_source_registration.go#L181-L230)

## E89

Baseline attempts scoped review staging before embeddings, then may publish/read back semantic content despite staging failure; finishes partial with candidate_review_failed.

[modules/Probata/probata/modules/engine/proffer/context_first.go @ 059c8aa5f943:301–356](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/proffer/context_first.go#L301-L356)

## E90

Existing fresh bounded registry ReadScope verifies the approved matter, court-case and actual relationship; reuse this sibling for case-specific admission.

[modules/Probata/probata/modules/engine/postgres/case_identity_scope_store.go @ 9399c24824e3:44–78](https://github.com/Cursedpotential/propria/blob/9399c24824e3e0d8d83643f8ae69a41a9f019d56/modules/Probata/probata/modules/engine/postgres/case_identity_scope_store.go#L44-L78)

## E91

Newer native workflow extracts candidates, embeds and publishes without the baseline candidate-review staging Activity; branch reconciliation is required.

[modules/Probata/probata/modules/engine/proffer/context_first.go @ 9fe9f8b31af9:261–293](https://github.com/Cursedpotential/propria/blob/9fe9f8b31af94242dd04aff8d745e04a5b169039/modules/Probata/probata/modules/engine/proffer/context_first.go#L261-L293)

## E92

Go native prepared/original opener accepts only bounded file URIs under its configured root, verifies original bytes, and cannot read a B2 URI by changing a BFF regex. Its default root differs from the Python default; live overrides unverified.

[modules/Probata/probata/modules/engine/aicontextsource/files.go @ 059c8aa5f943:43–129](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/aicontextsource/files.go#L43-L129)

## E93

Native review verifier resolves prepared original and checks registered source/version/provider/package/format against the case-bound context registration; durable locator changes must preserve this verification.

[modules/Probata/probata/modules/engine/postgres/ai_context_source.go @ 059c8aa5f943:15–65](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/modules/engine/postgres/ai_context_source.go#L15-L65)

## E94

Restored review tests supply a synthetic B2 prepared_ref and mock Go responses; these assertions do not exercise the native producer file reference or actual Go opener.

[modules/Probata/probata/modules/workbench/api/tests/test_ai_candidate_review.py @ f9b4bd6478b0:15–54](https://github.com/Cursedpotential/propria/blob/f9b4bd6478b017162aba3d561c0bc311d0b07ff0/modules/Probata/probata/modules/workbench/api/tests/test_ai_candidate_review.py#L15-L54)

## E95

Restored review UI exposes quote/native provenance and separate decision/publication status, with refresh/read retry; reuse this review surface while adding artifact-resolution actions.

[modules/Probata/probata/modules/workbench/web/src/components/review/ai-candidate-review.tsx @ f9b4bd6478b0:74–113](https://github.com/Cursedpotential/propria/blob/f9b4bd6478b017162aba3d561c0bc311d0b07ff0/modules/Probata/probata/modules/workbench/web/src/components/review/ai-candidate-review.tsx#L74-L113)

## E96

Current main separates detection/admission from parsing; ChatGPT/Claude heading Markdown is index-only because the parser route is unvalidated. Native JSON signatures have separate IDs.

[modules/Probata/probata/modules/engine/sourceformat/admission.go @ 833ace8f9956:26–79](https://github.com/Cursedpotential/propria/blob/833ace8f99563ce7a628c40b0584c2ba765b4ef3/modules/Probata/probata/modules/engine/sourceformat/admission.go#L26-L79)

## E97

Current main already maps exact native ChatGPT/Claude JSON formats to DuckDB and pins chatgpt_json_array_v2 and claude_ai_export_json_v1; reuse this sibling rather than assume native extraction is absent.

[modules/Probata/probata/modules/engine/activities/elt_structured.go @ 833ace8f9956:187–230](https://github.com/Cursedpotential/propria/blob/833ace8f99563ce7a628c40b0584c2ba765b4ef3/modules/Probata/probata/modules/engine/activities/elt_structured.go#L187-L230)

## E98

Baseline native decoder extracts supported ChatGPT mapping/Claude text fields and rejects no-text exports; its separate Markdown decoder is specifically Gemini. Attachment recognition does not prove text extraction.

[modules/Probata/probata/server/analysis/ai_context.py @ 059c8aa5f943:282–353](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/analysis/ai_context.py#L282-L353)

## E99

Native prepare selects Markdown only for gemini_markdown; other accepted native aliases read JSON. This does not implement ChatGPT/Claude heading Markdown decoding.

[modules/Probata/probata/server/analysis/ai_context.py @ 059c8aa5f943:439–461](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/server/analysis/ai_context.py#L439-L461)

## E100

Restored Sources AI chooser adds exact Gemini Markdown beside generic ChatGPT/Claude labels; this UI choice alone proves neither all Gemini exports nor integrated worker availability.

[modules/Probata/probata/modules/workbench/web/src/components/sources/sources-screen.tsx @ f9b4bd6478b0:527–535](https://github.com/Cursedpotential/propria/blob/f9b4bd6478b017162aba3d561c0bc311d0b07ff0/modules/Probata/probata/modules/workbench/web/src/components/sources/sources-screen.tsx#L527-L535)

## E101

Current detector tests cover Claude native array/single/attachment-only and ChatGPT/Claude heading Markdown, with negative prose/fence/mixed-role fixtures; recognition assertions are not end-to-end processing tests.

[modules/Probata/probata/modules/engine/sourceformat/detect_test.go @ 833ace8f9956:30–76](https://github.com/Cursedpotential/propria/blob/833ace8f99563ce7a628c40b0584c2ba765b4ef3/modules/Probata/probata/modules/engine/sourceformat/detect_test.go#L30-L76)

## E102

Existing native Gemini tests assert full-line roles, Unicode/CRLF exact spans, absent native clocks/pointers, byte retention and retry integrity using bounded mocked preparation; no live integrated route proven.

[modules/Probata/probata/tests/test_ai_context.py @ 059c8aa5f943:18–95](https://github.com/Cursedpotential/propria/blob/059c8aa5f943f6c186658a12aad1bde5aeb6169f/modules/Probata/probata/tests/test_ai_context.py#L18-L95)

## E103

Current main has synthetic persisted-pin coverage assertions for ChatGPT v2, historical v1/unpinned receipts, Claude v1 and wrong-pin/envelope rejection. These inspected tests were not run in this planning task.

[modules/Probata/probata/modules/engine/postgres/duckdb_context_coverage_test.go @ 833ace8f9956:43–76](https://github.com/Cursedpotential/propria/blob/833ace8f99563ce7a628c40b0584c2ba765b4ef3/modules/Probata/probata/modules/engine/postgres/duckdb_context_coverage_test.go#L43-L76)
