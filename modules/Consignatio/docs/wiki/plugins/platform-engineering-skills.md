---
title: "platform-engineering-skills"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# platform-engineering-skills

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

User-owned engineering skills adapted for the Agno-MCP-Platform custody, data, analysis, and operations contracts.

Source: `E:/AI_Workspace/plugins/plugins/platform-engineering-skills`. Version: `0.1.0`.
Registered: `True`. Installed manifests: not found in inspected manifests.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `platform-data-integrity`

Audit data quality and integrity for the Agno-MCP-Platform PostgreSQL custody and control plane. Use for live-versus-schema validation, lineage and referential-integrity checks, invariant reports, and proposed constraints; do not use for generic database advice or unauthorized data cleanup.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-data-integrity/SKILL.md:1>) · SHA-256 `658db722e08658a5b082c21170ceab7abfdecbde1c70c4464e81701cd2d58761`

### `platform-design-diagrams`

Create evidence-backed architecture, sequence, custody, delta, or drift diagrams for Agno-MCP-Platform. Use when a Platform relationship or workflow is materially clearer visually; do not use for decorative graphics or diagrams based only on stale planning documents.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-design-diagrams/SKILL.md:1>) · SHA-256 `7376ac71e343531f6abe0d5a96ec8bb84d569e54f234309789238697c65d2b3d`

### `platform-feature-engineering`

Design reproducible derived features for Agno-MCP-Platform behavioral, forensic, retrieval, or model evaluation workflows. Use for feature definitions, leakage analysis, temporal splits, provenance, and feature validation in this project; do not use for generic ML tutorials.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-feature-engineering/SKILL.md:1>) · SHA-256 `18b09dfae99426763c8afb4efeef61dc3365dd2c0af1e7842584c16074f370e8`

### `platform-infrastructure-drift`

Detect source-to-Coolify-to-runtime drift for Agno-MCP-Platform infrastructure. Use for compose, Dockerfile, image, environment-contract, route, service, and deployed-version comparisons; do not use for Terraform-first workflows or unauthorized remediation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-infrastructure-drift/SKILL.md:1>) · SHA-256 `a492536d83d3bde8facee042cc7dde8be2d91612698393b18b9ee177d72156b8`

### `platform-postgres-migrations`

Design, review, or execute PostgreSQL migrations for Agno-MCP-Platform. Use for numbered SQL migrations, schema evolution, backfills, live migration verification, and forward-repair planning in this repository; do not use for MySQL, MongoDB, or unapproved production changes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-postgres-migrations/SKILL.md:1>) · SHA-256 `0d0d28cdb08d902c05f23c28e2d8fe31cf94212754d4d9a13921c9472a4df4c4`

### `platform-text-analysis`

Extract and analyze text for Agno-MCP-Platform evidence and knowledge workflows with custody, attribution, horizon, and uncertainty controls. Use for entities, claims, themes, sentiment signals, topic candidates, or NLP-tool design in this project; do not use for generic document summarization.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-text-analysis/SKILL.md:1>) · SHA-256 `7936368fd197e399388b48f05146940fdf80c2a929b3de799e65a018508f79b6`

### `platform-vision-evidence`

Inspect or design computer-vision processing for images and video in Agno-MCP-Platform evidence workflows. Use for media observations, OCR or object-detection outputs, visual derivatives, and vision-tool integration in this project; do not use for ordinary image editing or unrelated computer-vision projects.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-vision-evidence/SKILL.md:1>) · SHA-256 `22323692ef3393f356df0504f0f2f9264f8e4a0e640e5a7231972971ce7bcbab`

### `platform-zero-tech-debt`

Refactor an Agno-MCP-Platform subsystem toward its accepted architecture instead of preserving obsolete compatibility layers. Use only when the user explicitly asks for a proper redesign, modernization, legacy removal, or technical-debt cleanup; do not use for incident containment, security backports, or surgical fixes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/skills/platform-zero-tech-debt/SKILL.md:1>) · SHA-256 `3c35cc709045e34aec17fc25eaf00beb6bf55234174c22dc20e0290bba242b67`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `upstream/computer-vision-processor/skills/processing-computer-vision-tasks/scripts/image_analyzer.py`

computer-vision-processor - Analysis Script
Script to perform various image analysis tasks (object detection, classification, segmentation) based on user input and specified models.
Generated: 2025-12-10 03:48:17

```text
python "E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/computer-vision-processor/skills/processing-computer-vision-tasks/scripts/image_analyzer.py" --help
```

Declared arguments: `--json`, `--output`, `-o`, `target`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| target | — | positional | — | Target directory to analyze |
| --output, -o | — | False | — | Output report file |
| --json | store_true | False | — | Output as JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [image_analyzer.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/computer-vision-processor/skills/processing-computer-vision-tasks/scripts/image_analyzer.py:1>) · SHA-256 `db418ea9bd230fd9aca63ea1d2bb42787302906030429530f0e15987d75d9a36`

### `upstream/data-validation-engine/skills/validating-database-integrity/scripts/configure_validation_rules.py`

Interactively configure data validation rules for a database table.

This script allows users to define and customize validation rules for database tables,
which are then saved in a configuration file for use by other validation scripts.

```text
python "E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/data-validation-engine/skills/validating-database-integrity/scripts/configure_validation_rules.py" --help
```

Declared arguments: `--database`, `--load`, `--not-null`, `--output`, `--range`, `--table`, `--unique`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --table | — | False | — | Table name for non-interactive mode |
| --database | — | False | — | Database name |
| --not-null | — | False | — | Comma-separated columns that must not be NULL |
| --unique | — | False | — | Comma-separated columns that must be unique |
| --range | — | False | — | Range validations in format: col:min:max,col2:min2:max2 |
| --load | — | False | — | Load existing configuration file |
| --output | — | False | — | Output file for configuration (JSON) |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [configure_validation_rules.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/data-validation-engine/skills/validating-database-integrity/scripts/configure_validation_rules.py:1>) · SHA-256 `f59947bd94032daf9bedf6c4d13be12c23e4c3f71bb0c6b4dbcd05f39ed49396`

### `upstream/data-validation-engine/skills/validating-database-integrity/scripts/generate_validation_report.py`

Generate comprehensive report of data validation results.

This script creates detailed HTML, JSON, or Markdown reports from validation results,
including statistics, identified issues, trend analysis, and recommendations.

```text
python "E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/data-validation-engine/skills/validating-database-integrity/scripts/generate_validation_report.py" --help
```

Declared arguments: `--format`, `--output`, `--results`, `--verbose`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --results | — | True | — | Path to JSON file containing validation results |
| --format | — | False | ['json', 'markdown', 'html'] | Report format |
| --output | — | False | — | Output file for report |
| --verbose | store_true | False | — | Print detailed output |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [generate_validation_report.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/data-validation-engine/skills/validating-database-integrity/scripts/generate_validation_report.py:1>) · SHA-256 `25ae535d87c4f3554dcd3dfd4c2b1eab13084d157dfec6e2afae76fba9f87175`

### `upstream/engineer-design-diagram/skills/engineer-design-diagram/scripts/fingerprint.py`

fingerprint.py — write, read, and diff structural fingerprints for engineer-design-diagram.

Usage:
    fingerprint.py write --input graph.json          # write new state
    fingerprint.py read                                # dump current state
    fingerprint.py diff --input graph.json            # diff current state against provided graph
    fingerprint.py path                                # print the resolved state file path

State file location (fallback chain):
    1. ${CLAUDE_PLUGIN_DATA}/arch-state.json
    2. ${XDG_STATE_HOME}/claude/arch/arch-state.json
    3. ~/.claude-state/arch/arch-state.json

Schema documented in references/fingerprint-spec.md. Current schema_version: "1".

Required packages: Python 3.9+ standard library only (hashlib, json, pathlib, os, sys, argparse).

```text
python "E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/engineer-design-diagram/skills/engineer-design-diagram/scripts/fingerprint.py" --help
```

Declared arguments: `--input`

Declared subcommands:

| Command | Source help |
|---|---|
| `write` | Write new state from a graph JSON input |
| `read` | Dump current state file to stdout |
| `diff` | Diff current state against a new graph JSON |
| `path` | Print resolved state file path |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --input | — | True | — | Path to graph JSON |
| --input | — | True | — | Path to graph JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [fingerprint.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/engineer-design-diagram/skills/engineer-design-diagram/scripts/fingerprint.py:1>) · SHA-256 `292849a2e886ea31fc02b1aba2e93edff956f31fb0e4f39a053ac3dda3fb11a4`

### `upstream/engineer-design-diagram/skills/engineer-design-diagram/scripts/validate_html.py`

validate_html.py — accessibility + no-external-deps check for engineer-design-diagram output.

Usage:
    validate_html.py path/to/diagram.html

Checks:
    1. SVG root has role="img" and aria-labelledby
    2. <title> element is present in SVG
    3. At least one <rect> has a <title> child (component labeling)
    4. prefers-reduced-motion media query exists in <style>
    5. No external script src (Google Fonts links are allowed)

Exits 0 on pass, 1 on fail with details printed to stderr.
Required packages: Python 3.9+ standard library only (re, sys, pathlib).

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate_html.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/engineer-design-diagram/skills/engineer-design-diagram/scripts/validate_html.py:1>) · SHA-256 `b3620532fe9a7457c2a9c9a06a0ce6220645fffff75f81644a3134a184563904`

### `upstream/feature-engineering-toolkit/skills/engineering-features-for-machine-learning/scripts/feature_importance_analyzer.py`

feature-engineering-toolkit - Analysis Script
Analyzes feature importance using various techniques (e.g., permutation importance, SHAP values) and provides insights into which features are most influential.
Generated: 2025-12-10 03:48:17

```text
python "E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/feature-engineering-toolkit/skills/engineering-features-for-machine-learning/scripts/feature_importance_analyzer.py" --help
```

Declared arguments: `--json`, `--output`, `-o`, `target`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| target | — | positional | — | Target directory to analyze |
| --output, -o | — | False | — | Output report file |
| --json | store_true | False | — | Output as JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [feature_importance_analyzer.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/feature-engineering-toolkit/skills/engineering-features-for-machine-learning/scripts/feature_importance_analyzer.py:1>) · SHA-256 `d54442233a30456d2c045016c8e4cde4ae75358cc2aba9f927e7922d966dd376`

### `upstream/nlp-text-analyzer/skills/analyzing-text-with-nlp/scripts/analyze_text.py`

nlp-text-analyzer - Analysis Script
Script to perform text analysis tasks (sentiment, keywords, topics) based on user input and specified parameters.
Generated: 2025-12-10 03:48:17

```text
python "E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/nlp-text-analyzer/skills/analyzing-text-with-nlp/scripts/analyze_text.py" --help
```

Declared arguments: `--json`, `--output`, `-o`, `target`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| target | — | positional | — | Target directory to analyze |
| --output, -o | — | False | — | Output report file |
| --json | store_true | False | — | Output as JSON |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [analyze_text.py:1](<E:/AI_Workspace/plugins/plugins/platform-engineering-skills/upstream/nlp-text-analyzer/skills/analyzing-text-with-nlp/scripts/analyze_text.py:1>) · SHA-256 `07cfccf34e47ee4b6bb72acb2acadd79e0b176c855af10ecb0faecdf3b48a2a4`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
