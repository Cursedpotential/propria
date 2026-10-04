# Family Court ledger and citation review — 2026-10-04

Byline: Codex, 2026-10-04.

This is bounded review evidence for the owner-approved Family Court convergence, not a release of the full library.

## Source identity and surface drift

The shared Surreal record reference:custody-guide-verification-ledger-md currently carries source_path content/custody-guide/verification_ledger.md and SHA-256 cb471a29e3ad43fe38b173f3a77b5ddfb3beb67a3063fb16f9b01cce51e05efc. Its body contains the August 12 Hayes holding update and Duperon reporter cross-check closure. This digest matches selected-text snapshot member 13 exactly.

Snapshot provenance: workflow toolkit-selected-text-20261004-v1, run 01a1089f-3d63-7691-b050-e8b9bce70588; snapshot digest 591f8c58ab4dc7535391a38e435e16c7e9ae27673eca53a8535dbfe5b1266dab. Member 13 is Projects/custody-guide/verification_ledger.md within custodyguide_v1complete_20260812.zip. The canonical plugin's corresponding ledger contains those final updates; the hosted bundled copy lacks them. No original text was altered.

The phone's working Library now reads source/reference records, with packaged files explicitly separate. This avoids treating the older bundled copy as the current working ledger. Matching this one record does not establish complete library parity.

## Mechanical ledger comparison

Workflow toolkit-ledger-comparison-20261004-v1, run 01a108b4-f70f-772c-99a2-b8a8da1b353c, completed 2026-10-04T20:57:20Z. It compared pinned snapshot members 14, 15 and 16 as unique-ID JSON arrays.

Read-back comparison receipt: /data/probata/volumes/proffer/derive-scratch/toolkit-inventory-20261004/receipts/ledger-comparison-v1.json on ovh-files; 84,469 bytes; SHA-256 67f89bbefc97f5a560037d8bff6ab0280c0851c309cdf39703ef2c62eff1de97.

Each candidate has 191 rows. Pair 14→15 has no added/removed IDs and 174 rows with differing fields. Pair 14→16 has 173 differing shared rows, one added family-court-plan and one removed genesee-family-court-plan. Pair 15→16 has 50 differing shared rows, exclusively audit_provenance, and the same one-ID replacement. No legal URL or citation field differs in these comparisons. A bounded read of caselaw-custody-standards shows the provenance filename prefix changes from genesee-family-court-source-ledger.json to family-court-source-ledger.json. No inference that every difference is harmless, no survivor selection, and no comparison against all 193 current store sources is claimed.

## Judicial text reviewed today

Hayes v Hayes, 209 Mich App 385; 532 NW2d 190 (1995): the reproduced opinion was read in full. Page 388 places the custodial-environment inquiry on actual care circumstances rather than the custody order or how the arrangement arose; the discussion continues onto page 389 for the temporary-order error. This supports preserving the final ledger's substantive correction. The exact pinpoint for any sentence must follow its own reporter page marker; the entire passage must not be labelled page 387. Source: [reproduced Hayes opinion](https://law.justia.com/cases/michigan/court-of-appeals/1995/209-mich-app-385.html), accessed 2026-10-04.

Duperon v Duperon, 175 Mich App 77; 437 NW2d 318 (1989): the reproduced opinion and footnotes were read in full. Page 79 distinguishes admission of the FOC report by both parties' agreement, background/context consideration, and findings based on competent hearing evidence. Pages 79–80 discuss the absence of a blanket duty to consider every report. These are text/pinpoint confirmations. Source: [reproduced Duperon opinion](https://law.justia.com/cases/michigan/court-of-appeals/1989/175-mich-app-77.html), accessed 2026-10-04.

The current [Michigan Judicial Institute precedent discussion](https://www.courts.michigan.gov/4a52e7/siteassets/publications/benchbooks/appeals-opinions/appealsopinionsresponsivehtml5.zip/Appeals_Opinions/Ch_1_General_Appellate_Issues/Precedent.htm) distinguishes published pre-November-1990 precedent under MCR 7.215(C)(2) from the subsequent-panel rule under MCR 7.215(J)(1). Duperon should not be relabelled unpublished or merely persuasive because of its date; any generalized binding-status wording needs that distinction. Accessed 2026-10-04.

Still open: case-specific negative-treatment/citator checks, a current official-reporter image for Hayes, and claim-level revalidation of the remaining library. Today's web reads are review observations; immutable server-fetched authority snapshots and validation receipts bound to proposed updates remain to be created. No authority/source record was updated or marked globally validated by this review.

