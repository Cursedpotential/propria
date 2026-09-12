# Deployed runtime receipt

- Runtime version: `consignatio-r2-b2-runtime-v4-split-canonical-source-identity`
- VPS host: `ovh-files`
- Deployed path: `/data/consignatio/migrations/r2-to-b2/runtime`
- Versioned payload-runtime receipt:
  `/data/consignatio/migrations/r2-to-b2/components/payload-runtime/20260912T180000Z`
- Versioned manifest-builder receipt:
  `/data/consignatio/migrations/r2-to-b2/components/manifest-builder/20260912T180000Z`
- Deployment state: manifest builder 12/12 and payload runtime 24/24 tests passed on the VPS
- Corpus gate: absent while the SHA export and manifest dry-run are incomplete;
  the approved control file is staged separately under
  `/data/consignatio/migrations/r2-to-b2/controls/`
- Credentials: not stored in this source tree

Active follow-on services and live counts are recorded in
`../STATUS-2026-09-12.md`. Generated run state and ledgers remain VPS-local;
the frozen generation and catalog snapshot are published to the B2 lake only
after checksum validation and one successful full-verification mapped copy.
