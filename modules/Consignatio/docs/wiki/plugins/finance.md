---
title: "finance"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# finance

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Real-estate and finance documents and models: purchase agreements, leases, promissory notes, deeds of trust, escrow and proration, options, notices, land contracts, project-finance modeling. One progressive-disclosure entry skill (`finance:finance`) routing to 13 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/finance`. Version: `1.1.0`.
Registered: `True`. Installed manifests: not found in inspected manifests.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `fin-commitment-letter-for-financing`

(finance) Drafts a U.S. financing commitment letter memorializing a lender's binding agreement to fund under specified economic terms, conditions precedent, and fees. Covers commercial real estate acquisition, construction, business expansion, and general commercial lending. Use when drafting loan commitment letters, lender commitment letters, financing commitments, or pre-closing funding commitments.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/commitment-letter-for-financing/SKILL.md:1>) · SHA-256 `ec0525ddeace222a23d67832337f6bb23fc3a1a107d5b274caeacd121331c4b3`

### `fin-contingency-removal`

(finance) Drafts residential real estate contingency removal forms that waive buyer contingencies from a purchase agreement. Handles inspection, financing, appraisal, and HOA contingencies with jurisdiction-specific compliance and earnest money forfeiture acknowledgments. Use when drafting contingency removal notices, waiver of contingencies, or notice of removal of contingencies in residential transactions.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/contingency-removal/SKILL.md:1>) · SHA-256 `2e7c86aab8b3ddca49159acadc3bbb6bda320a60e194eed47fd54ab5375a5ead`

### `fin-escrow-instructions`

(finance) Drafts binding escrow instructions for residential real estate closings. Extracts key terms from purchase agreements, identifies gaps or conflicts, and incorporates jurisdiction-specific requirements. Use when preparing escrow agent directives, closing instructions, escrow arrangements, or residential transaction closing documents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/escrow-instructions/SKILL.md:1>) · SHA-256 `24e92b553eb39624be1c9b86327a1c2e152673082d1f03988a1e975a86e4105a`

### `finance`

(finance) Real-estate and finance documents and models: purchase agreements, leases, promissory notes, deeds of trust, escrow and proration, options, notices, land contracts, project-finance modeling. Entry point / router — read this first, then load one member from references/. Triggers: real estate, lease, purchase agreement, promissory note, deed of trust, escrow, proration, mortgage, land contract, project finance, DSCR. Members: commitment-letter-for-financing, contingency-removal, escrow-instructions, modeling-project-finance-structures, mortgage-deed-of-trust, notice-to-perform-real-estate, option-to-purchase, promissory-note-residential, proration-schedule, residential-lease, residential-purchase-agreement, secured-promissory-note, michigan-land-contract.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/finance/SKILL.md:1>) · SHA-256 `09712918e412234140f82fb4ddce293d952e798436f1f6cd20014e67c45cc660`

### `fin-michigan-land-contract`

(finance) Drafts and reviews Michigan land contracts (installment sale / contract for deed) where a seller finances a buyer over time while retaining legal title. Covers vendor/vendee roles, legal description, price/interest/payment terms, balloon, tax-and-insurance allocation, default, statutory forfeiture vs. judicial foreclosure, recording, and required residential disclosures. Use when drafting, reviewing, or summarizing a Michigan land contract, a contract-for-deed, seller-financed home sale, or an installment purchase of real property — including a purchase from an estate. Trigger on 'land contract,' 'contract for deed,' 'seller financing,' 'installment sale of real estate,' or 'vendor/vendee.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/michigan-land-contract/SKILL.md:1>) · SHA-256 `c6e67f8dcde7895897d36dc94b95e7e595ed9fa33e16e496edaa3c88cda3c5af`

### `fin-modeling-project-finance-structures`

(finance) Builds project finance models with construction period draws, operational cash flows, DSCR covenants, and sculpted debt repayment. Use when modeling project finance, calculating debt service coverage, or structuring project lending.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/modeling-project-finance-structures/SKILL.md:1>) · SHA-256 `08b80b05cdc005edbb7fb1c02b21502f9cc285e871e2a5bfe01cd4f0e1100554`

### `fin-mortgage-deed-of-trust`

(finance) Drafts recording-ready residential Mortgages or Deeds of Trust with jurisdiction-appropriate instrument selection, uniform covenants, default/foreclosure provisions, and execution formalities. Use when drafting mortgage instruments, deeds of trust, security instruments for home loans, or real estate financing documents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/mortgage-deed-of-trust/SKILL.md:1>) · SHA-256 `a2450d8062b8deef43b3559de30e13be61ffbd4835447c419d2a61d90079ddda`

### `fin-notice-to-perform-real-estate`

(finance) Drafts jurisdiction-aware residential real-estate notices to perform (cure demands) for lease, purchase, or construction agreements where a counterparty has defaulted. Trigger when the user needs a notice to perform, notice to cure, cure notice, demand to perform, residential default notice, or pre-suit notice for a U.S. residential real-estate matter.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/notice-to-perform-real-estate/SKILL.md:1>) · SHA-256 `08977fedf5f822a74bf20e1ea28a600ba41e061130f427352b16616c6619ec0b`

### `fin-option-to-purchase`

(finance) Drafts Option to Purchase Real Estate agreements granting an optionee the exclusive right to buy property within a specified timeframe. Trigger when user needs a real estate option agreement, purchase option, right-to-purchase contract, or option-to-buy instrument for residential transactions.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/option-to-purchase/SKILL.md:1>) · SHA-256 `eda1928603522f19c28b0ef235f3f8d9d28f1986b8bc5784f1a93dfbf3fc7398`

### `fin-promissory-note-residential`

(finance) Drafts enforceable residential promissory notes with party identification, principal/interest terms, payment schedules, default/acceleration provisions, and security instrument cross-references. Ensures TILA awareness and state usury compliance. Use when drafting promissory notes for residential mortgages, deeds of trust, or seller-financed home sales; trigger keywords: promissory note, residential note, mortgage note, deed of trust note, seller financing note, balloon note.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/promissory-note-residential/SKILL.md:1>) · SHA-256 `128ae3db6f87c201e142c7b778a326f5503eb42b74a448d1a98f473596086f3d`

### `fin-proration-schedule`

(finance) Drafts a legally compliant proration schedule for real estate closings, allocating taxes, HOA fees, rent, utilities, and other obligations between buyer and seller. Triggered when preparing settlement documents, closing prorations, asset purchase closings, or real estate closing statements requiring expense allocation by closing date.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/proration-schedule/SKILL.md:1>) · SHA-256 `89a6ecb44f4d056db2472041085fc099bd2b2a7e78a26fa971f77dc6634f12dc`

### `fin-residential-lease`

(finance) Drafts jurisdictionally compliant U.S. residential lease agreements with required disclosures, security deposit compliance, and Fair Housing Act conformance. Conducts state-specific landlord-tenant law research and produces execution-ready leases. Use when drafting residential leases, rental agreements, landlord-tenant contracts, or tenancy agreements.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/residential-lease/SKILL.md:1>) · SHA-256 `d29e12fa08a79c850b91f5bf7a73198a7082e27c4d61f64301702e358cb06240`

### `fin-residential-purchase-agreement`

(finance) Drafts enforceable U.S. Residential Purchase Agreements covering parties, property description, price/earnest money, financing contingencies, inspection/due diligence, seller disclosures, closing, default remedies, and dispute resolution. Use when drafting home purchase contracts, residential real estate sale agreements, or property transfer agreements. Trigger on 'purchase agreement,' 'home sale contract,' 'residential sale,' or 'property purchase.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/residential-purchase-agreement/SKILL.md:1>) · SHA-256 `48ef9b4b80087e74a15bf3be6b779fd1de1230f24f4f752edaae671d6e12f853`

### `fin-secured-promissory-note`

(finance) Drafts U.S. secured promissory notes for commercial lending with lender-protective terms, UCC Article 9 collateral grants, and state usury compliance. Trigger when the user needs a secured promissory note, lender note, collateral-backed loan note, or UCC-1 financing instrument.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/finance/skills/secured-promissory-note/SKILL.md:1>) · SHA-256 `e7dbb5fb212bc6c9c565437396efb973e5eab6a1bf2afd6bb1a1e3282ad85776`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

No entries found in the inspected declarations.

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
