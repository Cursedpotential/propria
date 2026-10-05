---
title: "contextforge-tools"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# contextforge-tools

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Read from the existing registry at `2026-10-05T02:19:30.474369+00:00`. 212 entries.

Registry presence is not a successful tool invocation. Preserve the declared side effects and execution policy when selecting a tool.

Gateway tools are callable through the attached `atomic-tools` MCP server or its documented authenticated HTTP run endpoint. ContextForge tools are callable by an MCP client attached to a virtual server that exposes the named tool.

Use the exact tool name and supply the required fields shown in its schema. A placeholder is not a valid real source or workflow ID.

## `family-court-court-language-review`

Owner-requested tool 2: deterministically runs every court-language lexicon pattern (banned clinical labels, absolutes, mind-reading/motive, characterizations, emotional intensifiers, profanity/insults, threats, child-as-witness, speculation, layperson legal conclusions, recording/surveillance admissions, minor PII) plus doc-type profile checks against the text, and returns a score, stop flags, findings, profile violations, an ordered rewrite plan, and a safe phrasebank drawn from references/court-language/EXAMPLES.md. It does not rewrite the text — the calling model rewrites following references/court-language/SKILL.md + TEMPLATES.md + EXAMPLES.md, then re-reviews (mode: "review") until score >= 90 and stop_flags is empty.

id: `ed73a98a1ff84e5a8d6b27632713853f`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "text": {
      "type": "string",
      "minLength": 1,
      "maxLength": 20000
    },
    "doc_type": {
      "type": "string",
      "enum": [
        "affidavit",
        "motion_brief",
        "testimony_answer",
        "message_to_other_parent",
        "incident_log",
        "objection_to_recommendation"
      ]
    },
    "mode": {
      "type": "string",
      "enum": [
        "review",
        "rewrite_plan"
      ]
    }
  },
  "required": [
    "text",
    "doc_type"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-court-language-review`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-survival-guide`

Owner-requested tool 1: resolves a lightweight, cited context pack (sequence, applicable rules, deadlines with rule presets, traps, do-not list, phrases, safety gates, source paths) plus a writing template (full guide or one-page card) for the requested event/document type — never authors the guide itself. Optionally merges case_facts (never a child's name). The calling model writes the actual guide from these inputs; every guide is PROVISIONAL until checked against the archived MCR/MCL.

id: `1ce7d7a74fe9438ebec77c9aaa9bfd37`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "event": {
      "type": "string",
      "enum": [
        "affidavit",
        "custody-evaluation-session",
        "de-novo-hearing",
        "emergency-ex-parte-motion",
        "evidentiary-hearing",
        "foc-interview",
        "gal-lgal-interview",
        "mediation-settlement-conference",
        "motion",
        "motion-hearing",
        "objection-to-referee-recommendation",
        "ppo-hearing",
        "proof-of-service",
        "proposed-order",
        "referee-hearing",
        "response-to-motion",
        "show-cause-contempt"
      ]
    },
    "format": {
      "type": "string",
      "enum": [
        "full",
        "card",
        "json"
      ]
    },
    "include_case_facts": {
      "type": "boolean"
    }
  },
  "required": [
    "event"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-survival-guide`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-facts`

Return the county, court, judge, referee, controlling orders, next hearing, deadlines, party initials, and child count/ages from a local case-state JSON file (env CUSTODY_CASE_FILE, else ~/.config/family-court-toolkit/case.json). Never returns a child's name; any children[].name in the source file is reduced to initials. Returns { configured: false, example_path, hint } if no case file exists yet.

id: `c54d9172faba4c0b920cc05d04bdb983`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {}
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-facts`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-build-chronology`

Sort dated events while preserving source and knowledge-date fields. Does not infer truth or causation.

id: `2e69159810cd49a68af12f9c6787b8ca`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "events": {
      "minItems": 1,
      "maxItems": 500,
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "date": {
            "type": "string"
          },
          "title": {
            "type": "string",
            "minLength": 1,
            "maxLength": 1000
          },
          "source": {
            "type": "string",
            "maxLength": 1000
          },
          "knowledgeDate": {
            "type": "string"
          }
        },
        "required": [
          "date",
          "title"
        ]
      }
    }
  },
  "required": [
    "events"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-build-chronology`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-search-guide`

Search only the console's curated source registry. Stale draft prose is intentionally excluded.

id: `a1f82b0879c1461186b0ac56361acde8`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "minLength": 2,
      "maxLength": 300
    }
  },
  "required": [
    "query"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-search-guide`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-audit-sources`

Show normalized authority and currency status across the console's curated sources, the 191-record verification ledger, and the master source directory, with per-status/per-origin counts. Falls back to the 7 curated sources alone if the ledger or directory files are unavailable.

id: `643334eb926849068d3c068c9c8c1faa`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "ids": {
      "maxItems": 200,
      "type": "array",
      "items": {
        "type": "string",
        "maxLength": 80
      }
    }
  }
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-audit-sources`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-get-checklist`

Return a non-filing checklist for evidence, hearing preparation, or source review.

id: `39d77ab3d49e41ca888fd63ab0e8b1a5`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "kind": {
      "type": "string",
      "enum": [
        "evidence",
        "hearing",
        "source-review"
      ]
    }
  },
  "required": [
    "kind"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-get-checklist`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-get-packet-plan`

Create an organization plan with legal-review blocks and safety stop conditions.

id: `7cb2e24b2c334da696ea0dc8fb9c8c41`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "stage": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "goal": {
      "type": "string",
      "minLength": 1,
      "maxLength": 500
    }
  },
  "required": [
    "stage",
    "goal"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-get-packet-plan`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-calculate-planning-date`

Count calendar days before or after an anchor, adjusting unavailable dates in the safe direction (MCR 1.108-style: exclude the anchor day, roll forward over weekends/holidays). Pass either an explicit day count and direction, or a named `rule` preset (referee_objection, appeal_of_right, motion_response, mail_service_addon). This is a planning aid, not a filing deadline determination.

id: `a49adf26c7bf4408a7103f7a5a703d76`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "anchorDate": {
      "type": "string",
      "pattern": "^\\d{4}-\\d{2}-\\d{2}$"
    },
    "days": {
      "type": "integer",
      "minimum": 0,
      "maximum": 3650
    },
    "direction": {
      "type": "string",
      "enum": [
        "after",
        "before"
      ]
    },
    "includeAnchor": {
      "type": "boolean"
    },
    "holidays": {
      "maxItems": 100,
      "type": "array",
      "items": {
        "type": "string",
        "pattern": "^\\d{4}-\\d{2}-\\d{2}$"
      }
    },
    "rule": {
      "type": "string",
      "enum": [
        "referee_objection",
        "appeal_of_right",
        "motion_response",
        "mail_service_addon"
      ]
    }
  },
  "required": [
    "anchorDate"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-calculate-planning-date`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-route-issue`

Identify safety-critical route-outs before using draft material. Does not diagnose a case or give legal advice.

id: `718baa7504b242e79ee2235635a35650`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "description": {
      "type": "string",
      "minLength": 3,
      "maxLength": 4000
    }
  },
  "required": [
    "description"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-route-issue`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-open-dashboard`

Show release status, safety gates, source confidence, and the next planning steps. The underlying guide is publication-blocked.

id: `0b33e2036965402ebb81cb7181cc515d`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {}
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-open-dashboard`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-source`

Given a record ref ("table:id"), returns its `source` block ({path, sha256, r2_path, locator, url, row, extract_id}) if it carries one, whether `source.path` exists LOCALLY (no network calls), and the r2 pointer. "every record links/points to its original source document" (owner order).

id: `6dfbd6f2394344c7bdd0ddf9ccb1ab2f`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {
      "anyOf": [
        {
          "type": "string",
          "minLength": 3
        },
        {
          "type": "object",
          "properties": {
            "table": {
              "type": "string",
              "minLength": 1
            },
            "id": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "table",
            "id"
          ]
        }
      ]
    }
  },
  "required": [
    "id"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-source`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-record`

Given a record ref ("table:id"), returns { contract: "propria.legal-record.v1", id, table, version, record }. version is computed by the case store (sha256 of the record's canonical form), so the Family Law Toolkit and Advocatio show identical ids and versions for the same record; any correction yields a new version. Returns { found: false } when no such record exists.

id: `08bfab0e16a246359489ad49c4fe38f6`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {
      "anyOf": [
        {
          "type": "string",
          "minLength": 3
        },
        {
          "type": "object",
          "properties": {
            "table": {
              "type": "string",
              "minLength": 1
            },
            "id": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "table",
            "id"
          ]
        }
      ]
    }
  },
  "required": [
    "id"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-record`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-reference`

Reference data is the ruler, not the subject — loaded whole, never analyzed. action: "load" reads a JSON/JSONL file (`path`) or, with `from_plugin: true`, every .json/.jsonl file under the plugin's content/reference/ directory, upserting each row into the `reference` table (kind: behavior_pattern|ontology_term|entity_rule| lexicon|template|authority). action: "list" filters by kind/category. action: "match" runs every loaded row's `pattern` regex and `aliases[]` against `text` and returns each hit with its character span — use this to customize outputs against the behavior-detection/ontology reference set (owner order).

id: `a28fdb1e86474763acd7f35d78e933d6`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": [
        "load",
        "list",
        "match"
      ]
    },
    "path": {
      "type": "string",
      "minLength": 1
    },
    "from_plugin": {
      "type": "boolean"
    },
    "kind": {
      "type": "string",
      "maxLength": 80
    },
    "category": {
      "type": "string",
      "maxLength": 80
    },
    "text": {
      "type": "string",
      "maxLength": 20000
    }
  },
  "required": [
    "action"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-reference`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-eval`

action: "put" upserts an eval/report row (kind: eval|report|review|audit; title, subject ref or free text, score, verdict, text, path, tool_or_model). action: "list" filters by kind/subject. "a place to store evals and reports" (owner order).

id: `f3236dbdef1748339be3f4c8ff04c911`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": [
        "put",
        "list"
      ]
    },
    "id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "kind": {
      "type": "string",
      "maxLength": 80
    },
    "title": {
      "type": "string",
      "maxLength": 1000
    },
    "subject": {
      "type": "string",
      "maxLength": 500
    },
    "score": {
      "type": "number"
    },
    "verdict": {
      "type": "string",
      "maxLength": 200
    },
    "text": {
      "type": "string",
      "maxLength": 20000
    },
    "path": {
      "type": "string",
      "maxLength": 2000
    },
    "tool_or_model": {
      "type": "string",
      "maxLength": 200
    },
    "source": {
      "type": "object",
      "propertyNames": {
        "type": "string"
      },
      "additionalProperties": {}
    }
  },
  "required": [
    "action"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-eval`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-evidence-log`

action: "append" adds one entry (action: received|collected|hashed|reviewed|produced|disclosed|admitted| excluded|returned; optional exhibit ref RELATEd via the `logs` edge, by, hash, path, notes) and stamps logged_at. action: "list" filters by exhibit ref or action. A place to log evidence handling "for when we get to that point" (owner order) — independent of the platform's own H1/H2/H3 custody hashing.

id: `f695939f91874d3fb169efc2bc3fa649`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": [
        "append",
        "list"
      ]
    },
    "id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "log_action": {
      "type": "string",
      "maxLength": 80
    },
    "exhibit": {
      "anyOf": [
        {
          "type": "string",
          "minLength": 3
        },
        {
          "type": "object",
          "properties": {
            "table": {
              "type": "string",
              "minLength": 1
            },
            "id": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "table",
            "id"
          ]
        }
      ]
    },
    "by": {
      "type": "string",
      "maxLength": 200
    },
    "hash": {
      "type": "string",
      "maxLength": 200
    },
    "path": {
      "type": "string",
      "maxLength": 2000
    },
    "notes": {
      "type": "string",
      "maxLength": 4000
    },
    "source": {
      "type": "object",
      "propertyNames": {
        "type": "string"
      },
      "additionalProperties": {}
    }
  },
  "required": [
    "action"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-evidence-log`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-memo`

action: "put" upserts a memo (kind: analysis|strategy|weakness|direction|finding|risk|goal|issue|advice; title, text, status, factors[], author, supersedes). action: "list" filters by kind/status. action: "latest" returns the newest memo per kind seen. "keep a record of analysis and strategy, weaknesses and case direction ... keep memos" (owner order).

id: `ededfbabba1c41479c3f391766d909a9`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": [
        "put",
        "list",
        "latest"
      ]
    },
    "id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "kind": {
      "type": "string",
      "maxLength": 80
    },
    "title": {
      "type": "string",
      "maxLength": 1000
    },
    "text": {
      "type": "string",
      "maxLength": 20000
    },
    "status": {
      "type": "string",
      "maxLength": 80
    },
    "factors": {
      "maxItems": 12,
      "type": "array",
      "items": {
        "type": "string"
      }
    },
    "author": {
      "type": "string",
      "maxLength": 200
    },
    "supersedes": {
      "type": "string",
      "maxLength": 200
    },
    "source": {
      "type": "object",
      "propertyNames": {
        "type": "string"
      },
      "additionalProperties": {}
    }
  },
  "required": [
    "action"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-memo`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-docket`

Returns filings + drafts + orders + upcoming court_events (date >= now) in one list, sorted by date, each tagged with its table. Filter by `status` (filing/draft/court_event) `doc_type` (filing/draft) or `in_force` (order, boolean). Use case_timeline({mode:"court"}) for the full past+upcoming court-event history instead.

id: `b5646c34a6464db6bb4cdcfcb0c9c301`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "status": {
      "type": "string",
      "maxLength": 80
    },
    "doc_type": {
      "type": "string",
      "maxLength": 80
    },
    "in_force": {
      "type": "boolean"
    }
  }
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-docket`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-status`

action: "get" returns the case_status:current record (phase, posture, next_court_event ref, open_deadlines[], notes, last_updated). action: "set" MERGEs the given fields in and stamps last_updated. One record for the whole case — "keep a record of ... case status" (owner order).

id: `c16b6238cfb54434a9f47e3dde1c9862`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": [
        "get",
        "set"
      ]
    },
    "data": {
      "type": "object",
      "propertyNames": {
        "type": "string"
      },
      "additionalProperties": {}
    }
  },
  "required": [
    "action"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-status`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-summary`

Returns the same shape as core.ts's case_facts tool (county, court, judge, referee, controlling_orders, next_hearing, deadlines, parties as initials, children as count+ages) computed from the case store instead of the local case.json file. Intended so case_facts can eventually delegate here.

id: `7955227a3b0e44939b0c4a60603cbdb1`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {}
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-summary`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-import`

Loads `path` into the case store, auto-detecting the shape: a directory (or single file) of case-extract/v1 envelopes (Case Bible intake — schema "case-extract/v1"), a case_export snapshot ({tables, edges}), or a Vincent-style case schema (parties/timeline_events/evidence_matrix/...). Children are always reduced to { initials, age } and every child's real name (and its aliases, across the whole batch for a case-extract import) is redacted out of every free-text field before it is written — a child's name never enters the store, formally or in prose. Idempotent by extract_id/record id — safe to re-run. Returns per-table/edge counts and which shape was detected.

id: `1c65a02915864c7480f6f32810f3457f`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "minLength": 1
    }
  },
  "required": [
    "path"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-import`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-export`

format: "snapshot" (default) writes every table and edge to a single JSON file (default ~/.config/family-court-toolkit/exports/.json) for backup/transfer. format: "platform" writes one NDJSON file per table plus edges.ndjson and manifest.json (schema fct-platform-bundle/v1) to a new ~/.config/family-court-toolkit/exports/platform-/ directory, shaped for the probata evidence platform's import — every row keeps its provenance `source` block; children stay redacted to initials+age. Returns the written path/dir and per-table/edge record counts.

id: `98e10b750bd24be2b4203546b782caaa`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "minLength": 1
    },
    "format": {
      "type": "string",
      "enum": [
        "snapshot",
        "platform"
      ]
    }
  }
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-export`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-timeline`

Returns timeline entries tagged by `lane`: "court" = the real court-event timeline (court_event ∪ hearing ∪ deadline ∪ order); "master" = the extracted-from-the-corpora timeline (event ∪ message ∪ exhibit), each carrying its own known_at. `mode` selects which lane(s) — default "merged" (both). `known_by` filters the master lane to only what was known by that date (the two-clock discipline: occurred_at = when it happened, known_at = when the owner learned it). `upcoming` filters the court lane to future (true) or past (false) entries. These two timelines are kept deliberately separate; a merged view tags which lane each entry came from.

id: `b87f97fe9bc1414d9c8ae9f406708eb8`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "mode": {
      "type": "string",
      "enum": [
        "court",
        "master",
        "merged"
      ]
    },
    "upcoming": {
      "type": "boolean"
    },
    "from": {
      "type": "string"
    },
    "to": {
      "type": "string"
    },
    "known_by": {
      "type": "string"
    },
    "tables": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": [
          "event",
          "message",
          "exhibit"
        ]
      }
    }
  }
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-timeline`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-factor-map`

For each of the 12 MCL 722.23 best-interest factors (a-l), returns the supporting/contradicting record counts and the top 5 supporting and top 5 contradicting events/exhibits (by RELATE weight), each with a short summary.

id: `e669e8aa75564365afc8857950feeb33`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {}
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-factor-map`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-graph`

Returns the 1- or 2-hop neighbourhood of a record (id as "table:id", e.g. "event:e2") across the evidences, supports_factor, contradicts_factor, sent_by, sent_to, and filed_in edges (or a subset via `edges`), with each neighbor's edge, direction, id, and a short text summary.

id: `1f01f0e3a59e4033940f47657d0b92dd`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {
      "anyOf": [
        {
          "type": "string",
          "minLength": 3
        },
        {
          "type": "object",
          "properties": {
            "table": {
              "type": "string",
              "minLength": 1
            },
            "id": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "table",
            "id"
          ]
        }
      ]
    },
    "depth": {
      "anyOf": [
        {
          "type": "number",
          "const": 1
        },
        {
          "type": "number",
          "const": 2
        }
      ]
    },
    "edges": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": [
          "evidences",
          "supports_factor",
          "contradicts_factor",
          "sent_by",
          "sent_to",
          "filed_in",
          "drafted_as",
          "responds_to",
          "entered_at",
          "logs",
          "evaluates",
          "about",
          "matches_pattern"
        ]
      }
    }
  },
  "required": [
    "id"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-graph`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-search`

Full-text (BM25) and, when NVIDIA_API_KEY/NIM_API_KEY is configured, vector (HNSW/cosine) search over event.description, message.body, note.text, and exhibit.label, merged by reciprocal-rank fusion in hybrid mode. Without embeddings configured, hybrid/vector modes degrade to text-only search and set degraded: true. Returns id, table, snippet, score, occurred_at, known_at per hit.

id: `8bd25d44e58e4d6f86a405003dc0a2f6`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "minLength": 1,
      "maxLength": 2000
    },
    "tables": {
      "maxItems": 4,
      "type": "array",
      "items": {
        "type": "string",
        "enum": [
          "event",
          "message",
          "note",
          "exhibit",
          "memo",
          "filing",
          "draft",
          "court_event",
          "reference",
          "eval"
        ]
      }
    },
    "k": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100
    },
    "mode": {
      "type": "string",
      "enum": [
        "text",
        "vector",
        "hybrid"
      ]
    }
  },
  "required": [
    "query"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-search`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-query`

Executes SurrealQL against the embedded case store. DELETE, REMOVE, and DEFINE statements are refused unless write: true is passed. Result rows are capped at 200 per statement (a `truncated` flag notes when that cap hit). Returns { available: false, reason } if the native surrealdb module failed to load.

id: `ed7528414a7b48acbf3b2dc1d18c291a`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "surql": {
      "type": "string",
      "minLength": 1,
      "maxLength": 10000
    },
    "params": {
      "type": "object",
      "propertyNames": {
        "type": "string"
      },
      "additionalProperties": {}
    },
    "write": {
      "type": "boolean"
    }
  },
  "required": [
    "surql"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-query`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `family-court-case-put`

Upsert one record into the embedded SurrealDB case store (person, child, order, hearing, deadline, event, message, exhibit, factor, source, note, court) and optionally RELATE it to other records in the same call. A child record may only ever carry { initials, age } — any "name" field is rejected at both the application layer and the database schema. Returns { available: false, reason } if the native surrealdb module failed to load.

id: `e918c7a5639f466798cd460f4ed7b80f`

enabled: `True`

Exposed by: `family-court-console`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "table": {
      "type": "string",
      "enum": [
        "person",
        "child",
        "order",
        "hearing",
        "deadline",
        "event",
        "message",
        "exhibit",
        "factor",
        "source",
        "note",
        "court",
        "court_event",
        "filing",
        "draft",
        "memo",
        "reference",
        "evidence_log",
        "eval",
        "case_status"
      ]
    },
    "id": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "data": {
      "type": "object",
      "propertyNames": {
        "type": "string"
      },
      "additionalProperties": {}
    },
    "relations": {
      "maxItems": 50,
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "edge": {
            "type": "string",
            "enum": [
              "evidences",
              "supports_factor",
              "contradicts_factor",
              "sent_by",
              "sent_to",
              "filed_in",
              "drafted_as",
              "responds_to",
              "entered_at",
              "logs",
              "evaluates",
              "about",
              "matches_pattern"
            ]
          },
          "from": {
            "anyOf": [
              {
                "type": "string",
                "minLength": 3
              },
              {
                "type": "object",
                "properties": {
                  "table": {
                    "type": "string",
                    "minLength": 1
                  },
                  "id": {
                    "type": "string",
                    "minLength": 1
                  }
                },
                "required": [
                  "table",
                  "id"
                ]
              }
            ]
          },
          "to": {
            "anyOf": [
              {
                "type": "string",
                "minLength": 3
              },
              {
                "type": "object",
                "properties": {
                  "table": {
                    "type": "string",
                    "minLength": 1
                  },
                  "id": {
                    "type": "string",
                    "minLength": 1
                  }
                },
                "required": [
                  "table",
                  "id"
                ]
              }
            ]
          },
          "data": {
            "type": "object",
            "propertyNames": {
              "type": "string"
            },
            "additionalProperties": {}
          }
        },
        "required": [
          "edge",
          "from",
          "to"
        ]
      }
    }
  },
  "required": [
    "table",
    "data"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `family-court-case-put`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-agent-builder-reference`

Return the required reference for Agent configuration and mutate_agent operations. Read before building an Agent.

id: `9469cd691da24141b052d81765aaebc4`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-agent-builder-reference`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-update-agent-integration`

Configure or disconnect a Slack, Telegram, or Linear conversation integration. This is the only way to manage integrations; config.replace and config.patch can't change them. Configuration never publishes the Agent. If the Agent is already published, connecting starts the channel immediately. Otherwise, the channel stays inactive until publish_agent is called.

id: `c98e439676ca4652ac523550b70b3015`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "action": {
      "type": "string",
      "enum": [
        "connect",
        "disconnect"
      ]
    },
    "type": {
      "type": "string",
      "minLength": 1,
      "description": "Integration type returned by discover_agent_assets"
    },
    "credentialId": {
      "type": "string",
      "minLength": 1,
      "description": "Accessible credential for this integration"
    },
    "settings": {
      "type": "object",
      "additionalProperties": {},
      "description": "Integration settings; required for Telegram connect operations"
    },
    "replacesCredentialId": {
      "type": "string",
      "minLength": 1,
      "description": "On connect, the credential of the same type this one takes over from. Swaps both in one operation instead of a separate disconnect"
    }
  },
  "required": [
    "agentId",
    "action",
    "type",
    "credentialId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-update-agent-integration`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-verify-agent-mcp-server`

Test an MCP server with a user-accessible credential and return its available tools. Call before writing an mcpServers config entry; validate_agent performs no live MCP check.

id: `e3665f93f3a44072a909b4c48fe0f642`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "minLength": 1
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 64,
      "pattern": "^[a-zA-Z0-9_-]+$"
    },
    "url": {
      "type": "string",
      "format": "uri",
      "description": "HTTP(S) MCP server endpoint"
    },
    "transport": {
      "type": "string",
      "enum": [
        "sse",
        "streamableHttp"
      ],
      "default": "streamableHttp"
    },
    "authentication": {
      "anyOf": [
        {
          "type": "string",
          "enum": [
            "none",
            "bearerAuth",
            "headerAuth",
            "multipleHeadersAuth",
            "mcpOAuth2Api"
          ]
        },
        {
          "type": "string",
          "pattern": "McpOAuth2Api$"
        }
      ],
      "default": "none",
      "description": "Authentication method; every value other than none requires credential"
    },
    "credential": {
      "type": "string",
      "minLength": 1,
      "description": "Accessible credential ID; required when authentication is not none"
    },
    "connectionTimeoutMs": {
      "type": "integer",
      "minimum": 1,
      "maximum": 120000
    }
  },
  "required": [
    "projectId",
    "name",
    "url"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-verify-agent-mcp-server`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-discover-agent-assets`

Discover model catalogs, chat integrations, attachable workflows, published sub-agents, or MCP registry servers.

id: `ed7961ee659e4bb8890e4e1e52f9de45`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "minLength": 1
    },
    "kind": {
      "type": "string",
      "enum": [
        "models",
        "integrations",
        "workflows",
        "subagents",
        "mcpServers"
      ]
    },
    "query": {
      "type": "string",
      "minLength": 1,
      "description": "Optional filter for workflows, subagents, or MCP servers"
    },
    "provider": {
      "type": "string",
      "enum": [
        "openai",
        "anthropic",
        "google",
        "azure-openai",
        "aws-bedrock",
        "xai",
        "groq",
        "openrouter",
        "deepseek",
        "cohere",
        "mistral",
        "vercel",
        "nvidia"
      ],
      "description": "Model provider for kind=models; omit to get a provider summary without model lists"
    },
    "credentialId": {
      "type": "string",
      "minLength": 1,
      "description": "Accessible credential used to verify models for the selected provider"
    },
    "excludeAgentId": {
      "type": "string",
      "description": "Agent to omit when kind=subagents"
    }
  },
  "required": [
    "projectId",
    "kind"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-discover-agent-assets`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-delete-agent`

Permanently delete an Agent and its associated resources.

id: `2ef9d9d3dc534b9ba5d5091c8dcc3a75`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-delete-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-list-agent-versions`

List the publish history of an Agent, newest first. Pass a versionId to get_agent to inspect a version before revert_agent or publish_agent.

id: `3a10e5990f914e52835cb78cb78c6629`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "default": 20
    },
    "offset": {
      "type": "integer",
      "minimum": 0,
      "default": 0
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-list-agent-versions`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-revert-agent`

Restore an Agent draft from a published version, overwriting the draft config, skills, tasks, and custom tools. Does not publish. Inspect the version with get_agent first; the response returns the new configHash.

id: `ad7be25d4500491e8af062be5a9f69b7`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "versionId": {
      "type": "string",
      "minLength": 1,
      "description": "Published version to restore the draft from; defaults to the currently published version"
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-revert-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-unpublish-agent`

Unpublish an Agent and stop its live tasks and integrations.

id: `39ab9336fbea4cd4883457ce64c11f8c`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-unpublish-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-publish-agent`

Publish a valid Agent draft and activate its tasks and integrations. Pass versionId to republish a previously published version instead. Only call after the user explicitly requests or confirms publication; completing a build does not imply approval.

id: `ee91bc83ab1548d5a2c641da0dd387cd`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "versionId": {
      "type": "string",
      "minLength": 1,
      "description": "Republish a previously published version instead of the current draft. The draft is left untouched."
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-publish-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-call-agent`

Test an Agent draft through built-in Preview chat. Start or continue a conversation with a message request, or resume one returned approval after the human decides. This uses real tools and credentials, so external side effects are possible.

id: `74471f2700b540349fcdc3eafef45e1c`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "request": {
      "anyOf": [
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "message"
            },
            "message": {
              "type": "string",
              "minLength": 1
            },
            "sessionId": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "type",
            "message"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "approval"
            },
            "approved": {
              "type": "boolean"
            },
            "continuation": {
              "type": "object",
              "properties": {
                "runId": {
                  "type": "string"
                },
                "toolCallId": {
                  "type": "string"
                },
                "sessionId": {
                  "type": "string"
                },
                "response": {
                  "type": "string"
                }
              },
              "required": [
                "runId",
                "toolCallId",
                "sessionId",
                "response"
              ],
              "additionalProperties": false
            }
          },
          "required": [
            "type",
            "approved",
            "continuation"
          ],
          "additionalProperties": false
        }
      ]
    }
  },
  "required": [
    "agentId",
    "request"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-call-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-validate-agent`

Validate an Agent draft, sidecar references, and user-accessible credentials. Returns its n8n editor URL.

id: `f6265d2480734fc0a1d620674f4f3bea`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-validate-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-mutate-agent`

Apply one config, skill, task, or custom-tool mutation to an Agent draft. The operation fields sit directly on the operation object (no value wrapper), e.g. { "type": "config.patch", "patch": [...] }. Returns the next configHash for subsequent mutations.

id: `4f5a263b09e8407a9efe33b1a3c3bf8d`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "baseConfigHash": {
      "type": "string",
      "minLength": 1,
      "description": "Latest configHash returned by get_agent or a successful mutation"
    },
    "operation": {
      "anyOf": [
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "config.replace"
            },
            "config": {
              "type": "object",
              "additionalProperties": {}
            }
          },
          "required": [
            "type",
            "config"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "config.patch"
            },
            "patch": {
              "type": "array",
              "items": {
                "anyOf": [
                  {
                    "type": "object",
                    "properties": {
                      "op": {
                        "type": "string",
                        "const": "add"
                      },
                      "path": {
                        "type": "string"
                      },
                      "value": {
                        "anyOf": [
                          {
                            "type": "null"
                          },
                          {
                            "type": "boolean"
                          },
                          {
                            "type": "number"
                          },
                          {
                            "type": "string"
                          },
                          {
                            "type": "array",
                            "items": {}
                          },
                          {
                            "type": "object",
                            "additionalProperties": {}
                          }
                        ]
                      }
                    },
                    "required": [
                      "op",
                      "path",
                      "value"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "op": {
                        "type": "string",
                        "const": "remove"
                      },
                      "path": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "op",
                      "path"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "op": {
                        "type": "string",
                        "const": "replace"
                      },
                      "path": {
                        "type": "string"
                      },
                      "value": {
                        "$ref": "#/properties/operation/anyOf/1/properties/patch/items/anyOf/0/properties/value"
                      }
                    },
                    "required": [
                      "op",
                      "path",
                      "value"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "op": {
                        "type": "string",
                        "const": "move"
                      },
                      "from": {
                        "type": "string"
                      },
                      "path": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "op",
                      "from",
                      "path"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "op": {
                        "type": "string",
                        "const": "copy"
                      },
                      "from": {
                        "type": "string"
                      },
                      "path": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "op",
                      "from",
                      "path"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "op": {
                        "type": "string",
                        "const": "test"
                      },
                      "path": {
                        "type": "string"
                      },
                      "value": {
                        "$ref": "#/properties/operation/anyOf/1/properties/patch/items/anyOf/0/properties/value"
                      }
                    },
                    "required": [
                      "op",
                      "path",
                      "value"
                    ],
                    "additionalProperties": false
                  }
                ]
              },
              "minItems": 1
            }
          },
          "required": [
            "type",
            "patch"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "skill.upsert"
            },
            "skillId": {
              "type": "string"
            },
            "skill": {
              "type": "object",
              "properties": {
                "name": {
                  "type": "string",
                  "minLength": 1,
                  "maxLength": 128
                },
                "description": {
                  "type": "string",
                  "minLength": 1,
                  "maxLength": 512
                },
                "instructions": {
                  "type": "string",
                  "minLength": 1
                },
                "allowedTools": {
                  "type": "array",
                  "items": {
                    "type": "string",
                    "minLength": 1
                  }
                },
                "references": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "path": {
                        "type": "string",
                        "minLength": 1,
                        "maxLength": 512
                      },
                      "content": {
                        "type": "string",
                        "minLength": 1
                      }
                    },
                    "required": [
                      "path",
                      "content"
                    ],
                    "additionalProperties": false
                  },
                  "maxItems": 20
                },
                "scripts": {
                  "not": {}
                },
                "templates": {
                  "not": {}
                },
                "assets": {
                  "not": {}
                },
                "examples": {
                  "not": {}
                },
                "other": {
                  "not": {}
                }
              },
              "required": [
                "name",
                "description",
                "instructions"
              ],
              "additionalProperties": false
            }
          },
          "required": [
            "type",
            "skill"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "skill.delete"
            },
            "skillId": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "type",
            "skillId"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "task.upsert"
            },
            "taskId": {
              "type": "string"
            },
            "task": {
              "type": "object",
              "properties": {
                "name": {
                  "type": "string",
                  "minLength": 1,
                  "maxLength": 128
                },
                "objective": {
                  "type": "string",
                  "minLength": 1,
                  "maxLength": 10000
                },
                "cronExpression": {
                  "type": "string",
                  "minLength": 1,
                  "maxLength": 128,
                  "description": "Standard five-field cron expression, for example \"0 9 * * *\""
                }
              },
              "required": [
                "name",
                "objective",
                "cronExpression"
              ],
              "additionalProperties": false
            },
            "enabled": {
              "type": "boolean"
            }
          },
          "required": [
            "type",
            "task"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "task.delete"
            },
            "taskId": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "type",
            "taskId"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "customTool.upsert"
            },
            "code": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "type",
            "code"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "type": {
              "type": "string",
              "const": "customTool.delete"
            },
            "toolId": {
              "type": "string",
              "minLength": 1
            }
          },
          "required": [
            "type",
            "toolId"
          ],
          "additionalProperties": false
        }
      ]
    }
  },
  "required": [
    "agentId",
    "baseConfigHash",
    "operation"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-mutate-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-create-agent`

Create an Agent draft, optionally with its initial model, credential, instructions, and ordinary tool configuration. Returns its n8n editor URL. Use mutate_agent afterward for skills, tasks, and custom tools.

id: `39323e85a4ae48f497191a13963e1f0e`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "minLength": 1
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 128
    },
    "config": {
      "type": "object",
      "properties": {
        "model": {
          "anyOf": [
            {
              "type": "string",
              "const": ""
            },
            {
              "type": "string",
              "minLength": 1,
              "pattern": "^[a-z0-9-]+\\/(?:[a-z0-9._:-]+\\/)*[a-z0-9._:-]+$"
            }
          ]
        },
        "credential": {
          "type": "string"
        },
        "instructions": {
          "type": "string"
        },
        "personalisation": {
          "type": "object",
          "properties": {
            "icon": {
              "type": "string",
              "minLength": 1,
              "maxLength": 64
            },
            "gradient": {
              "type": "object",
              "properties": {
                "from": {
                  "type": "string",
                  "pattern": "^#[0-9A-Fa-f]{6}$"
                },
                "to": {
                  "$ref": "#/properties/config/properties/personalisation/properties/gradient/properties/from"
                },
                "angle": {
                  "type": "integer",
                  "minimum": 0,
                  "maximum": 359,
                  "default": 135
                },
                "fromStop": {
                  "type": "integer",
                  "minimum": 0,
                  "maximum": 45,
                  "default": 0
                },
                "toStop": {
                  "type": "integer",
                  "minimum": 55,
                  "maximum": 100,
                  "default": 100
                }
              },
              "required": [
                "from",
                "to"
              ],
              "additionalProperties": false,
              "default": {
                "from": "#FF1500",
                "to": "#FF6900",
                "angle": 135,
                "fromStop": 0,
                "toStop": 100
              }
            }
          },
          "required": [
            "icon"
          ],
          "additionalProperties": false
        },
        "memory": {
          "type": "object",
          "properties": {
            "enabled": {
              "type": "boolean"
            },
            "storage": {
              "type": "string",
              "enum": [
                "n8n"
              ]
            },
            "observationalMemory": {
              "type": "object",
              "properties": {
                "enabled": {
                  "type": "boolean"
                },
                "observerModel": {
                  "type": "object",
                  "properties": {
                    "model": {
                      "$ref": "#/properties/config/properties/model/anyOf/1"
                    },
                    "credential": {
                      "type": "string"
                    }
                  },
                  "required": [
                    "model",
                    "credential"
                  ],
                  "additionalProperties": false
                },
                "reflectorModel": {
                  "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel"
                },
                "observerThresholdTokens": {
                  "type": "integer",
                  "minimum": 1
                },
                "reflectorThresholdTokens": {
                  "type": "integer",
                  "minimum": 1
                },
                "renderTokenBudget": {
                  "type": "integer",
                  "minimum": 1
                },
                "observationLogTailLimit": {
                  "type": "integer",
                  "minimum": 1
                },
                "lockTtlMs": {
                  "type": "integer",
                  "minimum": 0
                }
              },
              "additionalProperties": false
            },
            "episodicMemory": {
              "anyOf": [
                {
                  "type": "object",
                  "properties": {
                    "enabled": {
                      "type": "boolean",
                      "const": false
                    }
                  },
                  "required": [
                    "enabled"
                  ],
                  "additionalProperties": false
                },
                {
                  "type": "object",
                  "properties": {
                    "enabled": {
                      "type": "boolean",
                      "const": true
                    },
                    "credential": {
                      "anyOf": [
                        {
                          "type": "string",
                          "const": "managed"
                        },
                        {
                          "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel/properties/credential"
                        }
                      ]
                    },
                    "extractorModel": {
                      "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel"
                    },
                    "reflectorModel": {
                      "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel"
                    },
                    "topK": {
                      "type": "integer",
                      "minimum": 1,
                      "maximum": 100
                    },
                    "maxEntriesPerRun": {
                      "type": "integer",
                      "minimum": 1,
                      "maximum": 50
                    }
                  },
                  "required": [
                    "enabled",
                    "credential"
                  ],
                  "additionalProperties": false
                }
              ]
            }
          },
          "required": [
            "enabled",
            "storage"
          ],
          "additionalProperties": false
        },
        "subAgents": {
          "type": "object",
          "properties": {
            "maxChildren": {
              "type": "integer",
              "minimum": 1,
              "maximum": 20,
              "description": "Maximum number of child sub-agent runs this parent agent may run in parallel. Defaults to 10 when unset."
            },
            "agents": {
              "type": "array",
              "items": {
                "type": "object",
                "properties": {
                  "agentId": {
                    "type": "string",
                    "minLength": 1
                  },
                  "useWhen": {
                    "type": "string",
                    "maxLength": 512
                  }
                },
                "required": [
                  "agentId"
                ],
                "additionalProperties": false
              }
            },
            "modelsByDifficulty": {
              "type": "object",
              "properties": {
                "low": {
                  "type": "object",
                  "properties": {
                    "model": {
                      "$ref": "#/properties/config/properties/model/anyOf/1"
                    },
                    "credential": {
                      "type": "string"
                    }
                  },
                  "required": [
                    "model",
                    "credential"
                  ],
                  "additionalProperties": false
                },
                "medium": {
                  "$ref": "#/properties/config/properties/subAgents/properties/modelsByDifficulty/properties/low"
                },
                "high": {
                  "$ref": "#/properties/config/properties/subAgents/properties/modelsByDifficulty/properties/low"
                }
              },
              "additionalProperties": false,
              "description": "Optional inline sub-agent model mappings by task difficulty. Missing mappings fall back to the parent agent model."
            }
          },
          "additionalProperties": false
        },
        "tools": {
          "type": "array",
          "items": {
            "anyOf": [
              {
                "type": "object",
                "properties": {
                  "type": {
                    "type": "string",
                    "const": "custom"
                  },
                  "id": {
                    "type": "string",
                    "minLength": 1,
                    "pattern": "^[A-Za-z0-9_]+$"
                  },
                  "requireApproval": {
                    "type": "boolean"
                  }
                },
                "required": [
                  "type",
                  "id"
                ],
                "additionalProperties": false
              },
              {
                "type": "object",
                "properties": {
                  "type": {
                    "type": "string",
                    "const": "workflow"
                  },
                  "workflowId": {
                    "type": "string",
                    "minLength": 1,
                    "description": "The workflow's stable ID."
                  },
                  "workflow": {
                    "type": "string",
                    "minLength": 1,
                    "description": "The workflow's display name and legacy lookup key."
                  },
                  "name": {
                    "type": "string"
                  },
                  "description": {
                    "type": "string"
                  },
                  "requireApproval": {
                    "type": "boolean"
                  },
                  "allOutputs": {
                    "type": "boolean",
                    "description": "Whether to return all node outputs instead of just the last node"
                  }
                },
                "required": [
                  "type",
                  "workflow"
                ],
                "additionalProperties": false
              },
              {
                "type": "object",
                "properties": {
                  "type": {
                    "type": "string",
                    "const": "node"
                  },
                  "name": {
                    "type": "string",
                    "minLength": 1
                  },
                  "description": {
                    "type": "string"
                  },
                  "inputSchema": {
                    "not": {}
                  },
                  "node": {
                    "type": "object",
                    "properties": {
                      "nodeType": {
                        "type": "string",
                        "minLength": 1
                      },
                      "nodeTypeVersion": {
                        "type": "number"
                      },
                      "nodeParameters": {
                        "type": "object",
                        "additionalProperties": {},
                        "default": {}
                      },
                      "credentials": {
                        "type": "object",
                        "additionalProperties": {
                          "anyOf": [
                            {
                              "type": "object",
                              "properties": {
                                "id": {
                                  "type": "string"
                                },
                                "name": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "id",
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "id": {
                                  "type": "null"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "__aiGatewayManaged": {
                                  "type": "boolean",
                                  "const": true
                                }
                              },
                              "required": [
                                "id",
                                "name",
                                "__aiGatewayManaged"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      }
                    },
                    "required": [
                      "nodeType",
                      "nodeTypeVersion"
                    ],
                    "additionalProperties": false
                  },
                  "requireApproval": {
                    "type": "boolean"
                  }
                },
                "required": [
                  "type",
                  "name",
                  "node"
                ],
                "additionalProperties": false
              }
            ]
          }
        },
        "providerTools": {
          "type": "object",
          "additionalProperties": {
            "type": "object",
            "additionalProperties": {}
          }
        },
        "mcpServers": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": {
                "type": "string",
                "minLength": 1,
                "maxLength": 64,
                "description": "Unique display name. The SDK normalizes it when building model-facing tool names"
              },
              "description": {
                "type": "string",
                "maxLength": 512,
                "description": "Human-readable server description"
              },
              "url": {
                "type": "string",
                "description": "MCP server endpoint URL. Empty string means setup is incomplete"
              },
              "transport": {
                "type": "string",
                "enum": [
                  "sse",
                  "streamableHttp"
                ],
                "default": "streamableHttp",
                "description": "Transport protocol"
              },
              "authentication": {
                "anyOf": [
                  {
                    "type": "string",
                    "enum": [
                      "none",
                      "bearerAuth",
                      "headerAuth",
                      "multipleHeadersAuth",
                      "mcpOAuth2Api"
                    ]
                  },
                  {
                    "type": "string",
                    "pattern": "McpOAuth2Api$"
                  }
                ],
                "default": "none",
                "description": "Auth method. Named variants or any string ending in McpOAuth2Api for registry credential types"
              },
              "credential": {
                "type": "string",
                "description": "Credential id from ask_credential. Required when authentication is not \"none\""
              },
              "metadata": {
                "type": "object",
                "properties": {
                  "nodeTypeName": {
                    "type": "string",
                    "description": "Source node type for registry servers (e.g. @n8n/mcp-registry.github). Enables correct UI form"
                  }
                },
                "additionalProperties": false,
                "description": "Server-generated metadata. Do not set this manually; only copy it from an MCP discovery result when present"
              },
              "toolFilter": {
                "anyOf": [
                  {
                    "type": "object",
                    "properties": {
                      "mode": {
                        "type": "string",
                        "const": "allow"
                      },
                      "tools": {
                        "type": "array",
                        "items": {
                          "type": "string",
                          "minLength": 1
                        },
                        "default": []
                      }
                    },
                    "required": [
                      "mode"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "mode": {
                        "type": "string",
                        "const": "exclude"
                      },
                      "tools": {
                        "type": "array",
                        "items": {
                          "type": "string",
                          "minLength": 1
                        },
                        "default": []
                      }
                    },
                    "required": [
                      "mode"
                    ],
                    "additionalProperties": false
                  }
                ],
                "description": "Restricts which tools are surfaced. Tools matched by original un-prefixed name"
              },
              "approval": {
                "anyOf": [
                  {
                    "type": "object",
                    "properties": {
                      "mode": {
                        "type": "string",
                        "const": "global"
                      }
                    },
                    "required": [
                      "mode"
                    ],
                    "additionalProperties": false
                  },
                  {
                    "type": "object",
                    "properties": {
                      "mode": {
                        "type": "string",
                        "const": "selected"
                      },
                      "tools": {
                        "type": "array",
                        "items": {
                          "type": "string",
                          "minLength": 1
                        },
                        "minItems": 1
                      }
                    },
                    "required": [
                      "mode",
                      "tools"
                    ],
                    "additionalProperties": false
                  }
                ],
                "description": "Human-in-the-loop approval. Absent = no approval required"
              },
              "connectionTimeoutMs": {
                "type": "integer",
                "minimum": 1,
                "maximum": 120000,
                "description": "Connection timeout in milliseconds"
              }
            },
            "required": [
              "name",
              "url"
            ],
            "additionalProperties": false
          },
          "maxItems": 20
        },
        "vectorStores": {
          "type": "array",
          "items": {
            "anyOf": [
              {
                "type": "object",
                "properties": {
                  "provider": {
                    "type": "string",
                    "const": "pinecone"
                  },
                  "name": {
                    "type": "string",
                    "minLength": 1,
                    "maxLength": 64,
                    "pattern": "^[a-zA-Z0-9_-]+$",
                    "description": "Unique connection name, also used as the SDK tool-name suffix: search_<name>"
                  },
                  "credential": {
                    "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel/properties/credential"
                  },
                  "useWhen": {
                    "type": "string",
                    "minLength": 1,
                    "maxLength": 512
                  },
                  "embedding": {
                    "type": "object",
                    "properties": {
                      "model": {
                        "$ref": "#/properties/config/properties/model/anyOf/1"
                      },
                      "credential": {
                        "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel/properties/credential"
                      }
                    },
                    "required": [
                      "model",
                      "credential"
                    ],
                    "additionalProperties": false
                  },
                  "indexName": {
                    "type": "string",
                    "minLength": 1
                  },
                  "namespace": {
                    "type": "string"
                  }
                },
                "required": [
                  "provider",
                  "name",
                  "credential",
                  "useWhen",
                  "embedding",
                  "indexName"
                ],
                "additionalProperties": false
              },
              {
                "type": "object",
                "properties": {
                  "provider": {
                    "type": "string",
                    "const": "qdrant"
                  },
                  "name": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/name"
                  },
                  "credential": {
                    "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel/properties/credential"
                  },
                  "useWhen": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/useWhen"
                  },
                  "embedding": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/embedding"
                  },
                  "collectionName": {
                    "type": "string",
                    "minLength": 1
                  }
                },
                "required": [
                  "provider",
                  "name",
                  "credential",
                  "useWhen",
                  "embedding",
                  "collectionName"
                ],
                "additionalProperties": false
              },
              {
                "type": "object",
                "properties": {
                  "provider": {
                    "type": "string",
                    "const": "supabase"
                  },
                  "name": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/name"
                  },
                  "credential": {
                    "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel/properties/credential"
                  },
                  "useWhen": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/useWhen"
                  },
                  "embedding": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/embedding"
                  },
                  "tableName": {
                    "type": "string",
                    "minLength": 1
                  },
                  "queryName": {
                    "type": "string"
                  }
                },
                "required": [
                  "provider",
                  "name",
                  "credential",
                  "useWhen",
                  "embedding",
                  "tableName"
                ],
                "additionalProperties": false
              },
              {
                "type": "object",
                "properties": {
                  "provider": {
                    "type": "string",
                    "const": "postgres"
                  },
                  "name": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/name"
                  },
                  "credential": {
                    "$ref": "#/properties/config/properties/memory/properties/observationalMemory/properties/observerModel/properties/credential"
                  },
                  "useWhen": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/useWhen"
                  },
                  "embedding": {
                    "$ref": "#/properties/config/properties/vectorStores/items/anyOf/0/properties/embedding"
                  },
                  "tableName": {
                    "type": "string",
                    "minLength": 1
                  }
                },
                "required": [
                  "provider",
                  "name",
                  "credential",
                  "useWhen",
                  "embedding",
                  "tableName"
                ],
                "additionalProperties": false
              }
            ]
          },
          "maxItems": 20
        },
        "config": {
          "type": "object",
          "properties": {
            "reasoning": {
              "type": "string",
              "enum": [
                "low",
                "medium",
                "high"
              ]
            },
            "promptCaching": {
              "type": "object",
              "properties": {
                "enabled": {
                  "type": "boolean"
                },
                "anthropic": {
                  "type": "object",
                  "properties": {
                    "ttl": {
                      "type": "string",
                      "enum": [
                        "5m",
                        "1h"
                      ]
                    }
                  },
                  "additionalProperties": false
                }
              },
              "required": [
                "enabled"
              ],
              "additionalProperties": false
            },
            "webSearch": {
              "type": "object",
              "properties": {
                "enabled": {
                  "type": "boolean"
                },
                "provider": {
                  "type": "string",
                  "enum": [
                    "auto",
                    "native",
                    "brave",
                    "searxng"
                  ]
                },
                "credential": {
                  "type": "string"
                }
              },
              "required": [
                "enabled"
              ],
              "additionalProperties": false
            },
            "toolCallConcurrency": {
              "type": "integer",
              "minimum": 1,
              "maximum": 100
            },
            "maxIterations": {
              "type": "integer",
              "minimum": 1,
              "maximum": 200,
              "description": "Maximum number of agent loop iterations per run. Do not set unless the user explicitly asks."
            }
          },
          "additionalProperties": false
        }
      },
      "required": [
        "model",
        "instructions"
      ],
      "additionalProperties": false,
      "description": "Optional initial Agent config without name, skills, tasks, or custom tools. The top-level name is injected into the config."
    }
  },
  "required": [
    "projectId",
    "name"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-create-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-agent`

Read an Agent draft, sidecar resources, runnable state, and configHash. Call before mutate_agent. Pass versionId to inspect a published version snapshot instead of the draft.

id: `37d21e67aa8242c59fdac707e9ab707c`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "agentId": {
      "type": "string",
      "minLength": 1,
      "description": "Agent ID"
    },
    "versionId": {
      "type": "string",
      "minLength": 1,
      "description": "Read a published version snapshot instead of the draft, e.g. the activeVersionId. Snapshots are read-only, so the response has no configHash."
    }
  },
  "required": [
    "agentId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-agent`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-agents`

Search Agents the current user can access. Use publishedOnly and excludeAgentId to discover saved sub-agents. Other agent tools only operate on agents with availableInMCP: true.

id: `5ff9fbe4234f42e8ae0fd6528b328dc1`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "minLength": 1,
      "description": "Restrict results to one project"
    },
    "query": {
      "type": "string",
      "description": "Filter by Agent name"
    },
    "publishedOnly": {
      "type": "boolean",
      "default": false
    },
    "excludeAgentId": {
      "type": "string",
      "description": "Agent ID to omit, useful for sub-agent search"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "default": 50
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-agents`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-sdk-reference`

Required reference when building a workflow, and only then. Call this BEFORE writing workflow code to learn workflow(), trigger()/node(), .add()/.to(), expr(), and credential patterns.

id: `02862d6dbea0478390e249ef444d9927`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "section": {
      "type": "string",
      "enum": [
        "patterns",
        "patterns_detailed",
        "expressions",
        "functions",
        "rules",
        "import",
        "guidelines",
        "design",
        "all"
      ],
      "description": "Optional section to retrieve. Omit this for the full reference, or use a section for targeted lookup."
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-sdk-reference`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-restore-workflow-version`

Restore a workflow to a previous version from its history. Re-applies that version as the current draft and records a new history entry. Use get_workflow_history to find the versionId.

id: `a8fc4bdf0624403aabea967177bf7eec`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to restore"
    },
    "versionId": {
      "type": "string",
      "description": "The version ID to restore, as returned by get_workflow_history"
    }
  },
  "required": [
    "workflowId",
    "versionId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-restore-workflow-version`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-update-workflow`

Atomically update an existing workflow with operation objects. Edits nodes/connections and also workflow-level settings via setWorkflowSettings — including the error workflow that runs automatically on failure to send alerts (e.g. when a user asks to "add error handling" or "notify me if this breaks"). Pass skillsUsed if n8n skills were used.

id: `41c07a68f7dc49bcb5f5bf767043600d`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to update."
    },
    "skillsUsed": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "IDs of n8n skills used to prepare this call, e.g. \"workflow-builder\". An optional plugin prefix is allowed, e.g. \"n8n-skills:workflow-builder\". Entries are normalized server-side (trimmed, lowercased, deduped); invalid identifiers are dropped."
    },
    "operations": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type": {
            "type": "string",
            "enum": [
              "updateNodeParameters",
              "setNodeParameter",
              "addNode",
              "removeNode",
              "renameNode",
              "addConnection",
              "removeConnection",
              "setNodeCredential",
              "setNodePosition",
              "setNodeDisabled",
              "setNodeSettings",
              "setWorkflowMetadata",
              "setWorkflowSettings",
              "addTags",
              "removeTags",
              "setNodeGroups"
            ],
            "description": "Operation type."
          },
          "nodeName": {
            "type": "string",
            "description": "For node-targeted ops."
          },
          "node": {
            "type": "object",
            "properties": {
              "name": {
                "type": "string",
                "description": "Unique node name."
              },
              "type": {
                "type": "string",
                "description": "Node type, e.g. \"n8n-nodes-base.set\"."
              },
              "typeVersion": {
                "type": "number"
              },
              "parameters": {
                "type": "object",
                "additionalProperties": {}
              },
              "position": {
                "type": "array",
                "items": {
                  "type": "number"
                },
                "minItems": 2,
                "maxItems": 2,
                "description": "Canvas [x, y]."
              },
              "credentials": {
                "type": "object",
                "additionalProperties": {
                  "type": "object",
                  "properties": {
                    "id": {
                      "type": "string"
                    },
                    "name": {
                      "type": "string"
                    }
                  },
                  "required": [
                    "name"
                  ],
                  "additionalProperties": false
                }
              },
              "disabled": {
                "type": "boolean"
              },
              "notes": {
                "type": "string"
              },
              "id": {
                "type": "string"
              }
            },
            "required": [
              "name",
              "type",
              "typeVersion"
            ],
            "additionalProperties": false,
            "description": "For addNode."
          },
          "parameters": {
            "type": "object",
            "additionalProperties": {},
            "description": "For updateNodeParameters."
          },
          "replace": {
            "type": "boolean",
            "description": "For updateNodeParameters; default false."
          },
          "path": {
            "type": "string",
            "minLength": 2,
            "description": "For setNodeParameter; JSON Pointer path."
          },
          "value": {
            "description": "For setNodeParameter."
          },
          "oldName": {
            "type": "string",
            "description": "For renameNode."
          },
          "newName": {
            "type": "string",
            "description": "For renameNode."
          },
          "source": {
            "type": "string",
            "description": "For connection ops."
          },
          "target": {
            "type": "string",
            "description": "For connection ops."
          },
          "sourceIndex": {
            "type": "integer",
            "minimum": 0,
            "description": "For connection ops; default 0."
          },
          "targetIndex": {
            "type": "integer",
            "minimum": 0,
            "description": "For connection ops; default 0."
          },
          "connectionType": {
            "type": "string",
            "description": "For connection ops; default \"main\"."
          },
          "credentialKey": {
            "type": "string",
            "description": "For setNodeCredential."
          },
          "credentialId": {
            "type": "string",
            "description": "For setNodeCredential."
          },
          "credentialName": {
            "type": "string",
            "description": "For setNodeCredential."
          },
          "position": {
            "$ref": "#/properties/operations/items/properties/node/properties/position",
            "description": "For setNodePosition."
          },
          "disabled": {
            "type": "boolean",
            "description": "For setNodeDisabled."
          },
          "settings": {
            "type": "object",
            "properties": {
              "onError": {
                "type": "string",
                "enum": [
                  "stopWorkflow",
                  "continueRegularOutput",
                  "continueErrorOutput"
                ],
                "description": "Error behavior."
              },
              "retryOnFail": {
                "type": "boolean"
              },
              "maxTries": {
                "type": "integer",
                "minimum": 2,
                "maximum": 5
              },
              "waitBetweenTries": {
                "type": "integer",
                "minimum": 0,
                "maximum": 5000
              },
              "alwaysOutputData": {
                "type": "boolean"
              },
              "executeOnce": {
                "type": "boolean"
              },
              "errorWorkflow": {
                "type": "string",
                "description": "ID of a SEPARATE workflow to run whenever THIS workflow fails — the common best-practice way to send failure alerts (email, Slack, etc.) or log errors via a shared, reusable handler. The referenced workflow must contain an Error Trigger node; find its ID with search_workflows. Pass \"DEFAULT\" to clear it. There are two ways to handle failures: (a) a dedicated/shared error workflow set here, or (b) an Error Trigger node placed directly inside THIS workflow (n8n fires it automatically on failure, no setting needed). When the user asks for error handling, ask which pattern they prefer before choosing. When errorWorkflow is set, it takes precedence over a same-workflow Error Trigger for the failing run. Failure handling fires for production executions only, not manual/test runs. Distinct from per-node onError/retry (setNodeSettings)."
              },
              "timezone": {
                "type": "string",
                "description": "IANA timezone used by Schedule Triggers and date/time operations, e.g. \"America/New_York\". Pass \"DEFAULT\" to inherit the instance timezone."
              },
              "executionOrder": {
                "type": "string",
                "enum": [
                  "v0",
                  "v1"
                ],
                "description": "Node execution order. \"v1\" is the default for new workflows; \"v0\" is legacy."
              },
              "saveExecutionProgress": {
                "anyOf": [
                  {
                    "type": "boolean"
                  },
                  {
                    "type": "string",
                    "const": "DEFAULT"
                  }
                ],
                "description": "Save execution data after each node finishes. Allows resuming/inspecting partial runs at the cost of speed."
              },
              "saveManualExecutions": {
                "anyOf": [
                  {
                    "type": "boolean"
                  },
                  {
                    "type": "string",
                    "const": "DEFAULT"
                  }
                ],
                "description": "Whether manual (test) executions are saved to the execution list."
              },
              "saveDataErrorExecution": {
                "type": "string",
                "enum": [
                  "DEFAULT",
                  "all",
                  "none"
                ],
                "description": "Whether to store execution data for failed runs."
              },
              "saveDataSuccessExecution": {
                "type": "string",
                "enum": [
                  "DEFAULT",
                  "all",
                  "none"
                ],
                "description": "Whether to store execution data for successful runs."
              },
              "executionTimeout": {
                "type": "integer",
                "description": "Maximum execution time in seconds before a run is stopped. Use a positive number of seconds (not exceeding the instance maximum, enforced server-side), or -1 for unlimited (no timeout)."
              },
              "timeSavedPerExecution": {
                "type": "integer",
                "minimum": 0,
                "description": "Estimated time saved per execution, in minutes (used for insights/reporting)."
              },
              "callerPolicy": {
                "type": "string",
                "enum": [
                  "any",
                  "none",
                  "workflowsFromAList",
                  "workflowsFromSameOwner"
                ],
                "description": "Which workflows may call this one via the Execute Sub-workflow node. Defaults to \"workflowsFromSameOwner\"."
              },
              "callerIds": {
                "type": "string",
                "description": "Comma-separated workflow IDs allowed to call this workflow (only used with callerPolicy \"workflowsFromAList\")."
              }
            },
            "additionalProperties": false,
            "description": "For setNodeSettings or setWorkflowSettings."
          },
          "name": {
            "type": "string",
            "maxLength": 128,
            "description": "Only used for setWorkflowMetadata."
          },
          "description": {
            "type": "string",
            "maxLength": 255,
            "description": "Only used for setWorkflowMetadata."
          },
          "names": {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "For addTags / removeTags."
          },
          "nodeGroups": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "id": {
                  "type": "string"
                },
                "name": {
                  "type": "string"
                },
                "nodeNames": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "description": {
                  "type": "string"
                }
              },
              "required": [
                "name",
                "nodeNames"
              ],
              "additionalProperties": false
            },
            "description": "For setNodeGroups. Replaces all node groups; pass [] to clear. Group members are node names, not ids."
          }
        },
        "required": [
          "type"
        ],
        "additionalProperties": false,
        "description": "Workflow update operation. Provide fields matching type."
      },
      "minItems": 1,
      "maxItems": 100,
      "description": "Ordered operations to apply atomically (max 100). If any op fails, nothing is saved."
    },
    "versionName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 80,
      "description": "Short summary of what this update changes, shown in the workflow's version history (e.g. \"Added Slack notification after HTTP request\"). Always provide it."
    },
    "versionDescription": {
      "type": "string",
      "maxLength": 1000,
      "description": "Longer description of what changed and why, shown in the version history alongside the version name."
    }
  },
  "required": [
    "workflowId",
    "operations"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-update-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-archive-workflow`

Archive a workflow in n8n by its ID.

id: `ae07a9eb39d84fb087c72c7da74c7feb`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to archive"
    }
  },
  "required": [
    "workflowId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-archive-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-move-workflows-to-folder`

Move one or more existing workflows into a folder (or to the project root). The destination folder must be in the same project as the workflows. Resolve the folder by name with search_folders first; when multiple folders match the name, ask the user which one they meant before moving. After moving, confirm the destination to the user by folder name, not ID. Moves may partially succeed — report any entries in `failed` to the user.

id: `58651f34baa54ceaa8a43e218dd3ac10`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowIds": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "minItems": 1,
      "maxItems": 20,
      "description": "The IDs of the workflows to move (up to 20 at a time)"
    },
    "folderId": {
      "type": "string",
      "maxLength": 36,
      "description": "The ID of the destination folder. It must belong to the project that owns the workflows — use search_folders to find it by name. Pass \"0\" to move the workflows to the project root."
    }
  },
  "required": [
    "workflowIds",
    "folderId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-move-workflows-to-folder`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-update-folder`

Rename a folder and/or move it under another folder in the same project. Resolve folders by name with search_folders first; when multiple folders match a name, ask the user which one they meant before updating. After the update, confirm the result to the user using folder names, not IDs.

id: `fa071929ccaa4036b8cb14c5bc7089fa`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "description": "The ID of the project the folder belongs to. Use search_projects to resolve a project name to an ID."
    },
    "folderId": {
      "type": "string",
      "maxLength": 36,
      "description": "The ID of the folder to update. Use search_folders to find it by name."
    },
    "name": {
      "allOf": [
        {
          "type": "string"
        },
        {
          "type": "string",
          "maxLength": 128
        }
      ],
      "description": "New name for the folder (rename)"
    },
    "parentFolderId": {
      "type": "string",
      "maxLength": 36,
      "description": "New parent folder ID to move the folder under. Must belong to the same project and must not be a descendant of the folder being moved. Pass \"0\" to move the folder to the project root. Omit to leave the folder where it is."
    }
  },
  "required": [
    "projectId",
    "folderId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-update-folder`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-create-folder`

Create a folder in a project, optionally nested under an existing folder. Requires a projectId — use search_projects first if needed. If the user named a parent folder, resolve it with search_folders; when multiple folders match the name, ask the user which one they meant before creating. After creation, confirm the folder to the user by name, not ID.

id: `29edc084ac3e4732819eee49255a6249`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "description": "The ID of the project to create the folder in. Use search_projects to resolve a project name to an ID."
    },
    "name": {
      "allOf": [
        {
          "type": "string"
        },
        {
          "type": "string",
          "maxLength": 128
        }
      ],
      "description": "The name of the folder to create"
    },
    "parentFolderId": {
      "type": "string",
      "maxLength": 36,
      "description": "Optional parent folder ID to nest the new folder under. Must belong to the same project — use search_folders to find it. Omit it or pass \"0\" to create the folder at the project root."
    }
  },
  "required": [
    "projectId",
    "name"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-create-folder`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-folders`

Search for folders within a project. Use this to resolve a folder name to an ID before creating a workflow in a folder, creating or updating a folder, or moving workflows into a folder. Each result includes the folder's full name path — when multiple folders match a name, use the paths to ask the user which one they meant. Requires a projectId — use search_projects first if needed.

id: `85c1800df2244a68935c2fa3585ebbca`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "description": "The ID of the project to search folders in"
    },
    "query": {
      "type": "string",
      "description": "Filter folders by name (case-insensitive partial match)"
    },
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 100,
      "description": "Limit the number of results (max 100)"
    }
  },
  "required": [
    "projectId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-folders`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-projects`

Search for projects accessible to the current user. Call this whenever the user names a project — pass the name as the query, then use the resolved ID with tools that take a projectId. Results are ranked with exact case-insensitive name matches first. If no exact match is found but multiple partials are returned, the response includes a `hint` field telling you to clarify with the user before acting; follow it instead of guessing. The response also includes `teamProjectsEnabled` — when false, team projects are not licensed on this instance, so default to creating in the caller's personal project unless the user explicitly picks one of the returned accessible projects.

id: `97a62c4ac2bc4c2799b56609e51730cf`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Filter projects by name (case-insensitive partial match). Pass the exact project name the user mentioned — results are ranked with exact case-insensitive matches first, then partial matches."
    },
    "type": {
      "type": "string",
      "enum": [
        "personal",
        "team"
      ],
      "description": "Filter by project type. 'team' for shared team projects, 'personal' for personal projects."
    },
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 100,
      "description": "Limit the number of results (max 100)"
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-projects`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-create-workflow-from-code`

Create a workflow in n8n from validated SDK code. This tool expects code that already follows the n8n Workflow SDK patterns and has passed validate_workflow. If code fails to parse, call get_workflow_sdk_reference, rewrite the code using the reference, validate again, then retry creation. If the user named a target project, resolve it via search_projects before calling this tool; when projectId is omitted, the workflow is created in the user's personal project. If the user named a target folder, resolve it via search_folders. If you used n8n skills while preparing this workflow, pass their identifiers in skillsUsed. After creation, always tell the user which project — and folder, if any — the workflow landed in (see the targetProject and targetFolder fields in the response).

id: `a471956e5bb64a43b3a2ebaa21b29fe8`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "description": "Full TypeScript/JavaScript workflow code using the n8n Workflow SDK. Must be validated first with validate_workflow."
    },
    "skillsUsed": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "IDs of n8n skills used to prepare this call, e.g. \"workflow-builder\". An optional plugin prefix is allowed, e.g. \"n8n-skills:workflow-builder\". Entries are normalized server-side (trimmed, lowercased, deduped); invalid identifiers are dropped."
    },
    "name": {
      "type": "string",
      "maxLength": 128,
      "description": "Optional workflow name. If not provided, uses the name from the code."
    },
    "description": {
      "type": "string",
      "description": "Workflow description. Longer text is shortened to 255 chars before saving."
    },
    "versionName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 80,
      "description": "Short summary of this initial version, shown in the workflow's version history (e.g. \"Initial Slack notification workflow\"). Always provide it."
    },
    "versionDescription": {
      "type": "string",
      "maxLength": 1000,
      "description": "Longer description of what this version does, shown in the version history alongside the version name."
    },
    "projectId": {
      "type": "string",
      "description": "Project ID to create the workflow in. If the user named a project (e.g. 'in my Marketing project'), you MUST call search_projects first to resolve the name to an ID and pass it here — do not guess. If search_projects returns multiple partial matches with no exact match, ask the user to clarify before creating the workflow. Only omit this field when the user did not mention a project at all; in that case it defaults to the user's personal project."
    },
    "folderId": {
      "type": "string",
      "description": "Optional folder ID to create the workflow in. Requires projectId to be set. Use search_folders to find a folder by name within a project; when multiple folders match the name, ask the user which one they meant before creating."
    }
  },
  "required": [
    "code"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-create-workflow-from-code`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-validate-node-config`

Validate a node's config the moment you write it — before assembling create_workflow_from_code or calling update_workflow. Read-only and needs no existing workflow, so use it freely while composing. Unlike the write tools (which validate only as they mutate), this returns isolated per-node, per-parameter errors with no graph noise, and can check several candidate configs in one call so you wire only the one that passes. For langchain tool subnodes (nodes wired via ai_tool), set isToolNode: true so the schema evaluates the correct displayOptions branch. Schema-level only — for connections, required inputs, triggers, and credentials use validate_workflow.

id: `580613d74dba4addad0b4b6c7b0c6323`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "nodes": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "Optional node name. Echoed back in the result so callers can correlate."
          },
          "type": {
            "type": "string",
            "description": "Full node type, e.g. \"n8n-nodes-base.set\" or \"@n8n/n8n-nodes-langchain.agent\"."
          },
          "typeVersion": {
            "type": "number",
            "exclusiveMinimum": 0,
            "default": 1,
            "description": "Node type version. Defaults to 1."
          },
          "parameters": {
            "type": "object",
            "additionalProperties": {},
            "default": {},
            "description": "Node parameters object — same shape as workflow JSON."
          },
          "subnodes": {
            "description": "Optional subnode config for AI parent nodes (e.g. langchain agent): `{ model, memory, tools: [...] }` of `{ type, version }` refs."
          },
          "isToolNode": {
            "type": "boolean",
            "description": "Set to true when validating a node that is wired as an AI tool subnode (ai_tool connection). Adjusts which displayOptions branch is evaluated."
          }
        },
        "required": [
          "type"
        ],
        "additionalProperties": false
      },
      "minItems": 1,
      "maxItems": 50,
      "description": "One or more node configurations to validate independently."
    }
  },
  "required": [
    "nodes"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-validate-node-config`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-validate-workflow`

Validate n8n Workflow SDK code. Required before creating or updating workflows from code. If you have not already read get_workflow_sdk_reference, call that first; guessing SDK syntax commonly creates invalid workflows.

id: `f0d279d5a4154821b1516cccce9ecaa4`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "description": "Full TypeScript/JavaScript workflow code using the n8n Workflow SDK. Must include the workflow export."
    }
  },
  "required": [
    "code"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-validate-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-explore-node-resources`

Resolve the real values behind a node's resource locator or load-options dropdown (e.g. Slack channels, Google Sheets tabs, OpenAI models). Use this after get_node_types so you ground RLC and load-options parameters in real IDs instead of inventing them. Requires a credential ID from list_credentials — the call runs as the current user with that credential.

id: `688024c0036d434dbea06fa28ad9fc55`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "nodeType": {
      "type": "string",
      "description": "Fully-qualified node type ID from search_nodes / get_node_types, e.g. \"n8n-nodes-base.slack\"."
    },
    "version": {
      "type": "number",
      "description": "Node version, e.g. 4.7. Must match a version returned by search_nodes."
    },
    "methodName": {
      "type": "string",
      "description": "The exact method name from the node's `@searchListMethod` or `@loadOptionsMethod` annotation in the type definition. Call get_node_types first to read the real method name. Do not invent or guess."
    },
    "methodType": {
      "type": "string",
      "enum": [
        "listSearch",
        "loadOptions"
      ],
      "description": "\"listSearch\" for `@searchListMethod` annotations (supports filter/pagination); \"loadOptions\" for `@loadOptionsMethod` annotations."
    },
    "credentialType": {
      "type": "string",
      "description": "Credential type key for the node, e.g. \"slackApi\" or \"googleSheetsOAuth2Api\"."
    },
    "credentialId": {
      "type": "string",
      "description": "ID of a credential the user can access, obtained from list_credentials."
    },
    "filter": {
      "type": "string",
      "description": "Optional search/filter text to narrow results."
    },
    "paginationToken": {
      "type": "string",
      "description": "Pagination token from a previous call to fetch the next page (listSearch only)."
    },
    "currentNodeParameters": {
      "type": "object",
      "additionalProperties": {},
      "description": "Current node parameters for dependent lookups. Some methods require prior selections — e.g. listing sheets within a spreadsheet needs `{ documentId: { __rl: true, mode: \"id\", value: \"<spreadsheetId>\" } }`. Check the type definition's displayOptions to know which parameters a method depends on."
    }
  },
  "required": [
    "nodeType",
    "version",
    "methodName",
    "methodType",
    "credentialType",
    "credentialId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-explore-node-resources`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-best-practices`

Required planning step when building a workflow, and only then. Get best-practices guidance (recommended nodes, patterns, and common pitfalls) for a specific workflow technique before searching for nodes or writing code. Call once per relevant technique. Use technique="list" first if unsure which techniques apply.

id: `8e9929675a744106ab1c9ffae86b9033`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "technique": {
      "anyOf": [
        {
          "type": "string",
          "enum": [
            "scheduling",
            "chatbot",
            "form_input",
            "scraping_and_research",
            "monitoring",
            "enrichment",
            "triage",
            "content_generation",
            "document_processing",
            "data_extraction",
            "data_analysis",
            "data_transformation",
            "data_persistence",
            "notification",
            "knowledge_base",
            "human_in_the_loop",
            "web_app"
          ]
        },
        {
          "type": "string",
          "const": "list"
        }
      ],
      "description": "Workflow technique key (e.g. \"chatbot\", \"scheduling\", \"triage\") to fetch best-practices guidance for. Pass \"list\" to discover all available techniques."
    }
  },
  "required": [
    "technique"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-best-practices`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-node-types`

Get TypeScript type definitions for n8n nodes. Returns exact parameter names and structures. MUST be called before writing workflow code or configuring node-backed tools — guessing parameter names creates invalid configurations. Pass nodeIds as an array of objects like { nodeId: "n8n-nodes-base.gmail" }. Include discriminators (resource/operation/mode) from search_nodes results.

id: `42755476267840f491bb494a9adc8005`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "nodeIds": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "nodeId": {
            "type": "string",
            "description": "The node type ID (e.g. \"n8n-nodes-base.gmail\")"
          },
          "version": {
            "type": "string",
            "description": "Specific version (e.g. \"2.1\")"
          },
          "resource": {
            "type": "string",
            "description": "Resource discriminator (e.g. \"message\")"
          },
          "operation": {
            "type": "string",
            "description": "Operation discriminator (e.g. \"send\")"
          },
          "mode": {
            "type": "string",
            "description": "Mode discriminator"
          }
        },
        "required": [
          "nodeId"
        ],
        "additionalProperties": false
      },
      "minItems": 1,
      "description": "Node type requests to get definitions for. Always pass an array of objects, even for a single node. Include discriminators from search_nodes results when available."
    }
  },
  "required": [
    "nodeIds"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-node-types`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-nodes`

Search for n8n nodes by service name, trigger type, or utility function. Set usage="agentTool" to return only Agent-compatible tool nodes. Returns node IDs, discriminators (resource/operation/mode), and related nodes needed for get_node_types.

id: `d4d1571494114356b4d6b2aea6689242`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "queries": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "minItems": 1,
      "description": "Search queries for n8n nodes — service names (e.g. \"gmail\", \"slack\"), trigger types (e.g. \"schedule trigger\", \"webhook\"), or utility nodes (e.g. \"set\", \"if\", \"merge\", \"code\")"
    },
    "usage": {
      "type": "string",
      "enum": [
        "workflow",
        "agentTool"
      ],
      "description": "Use agentTool to return only nodes that can be configured as Agent tools; defaults to workflow"
    }
  },
  "required": [
    "queries"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-nodes`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-data-table-rows`

Read rows from a data table, with optional filtering, sorting and pagination. Use search_data_tables to find the data table ID and its columns first.

id: `4ffca8ab923c4e658b56e0aec4e0b460`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "dataTableId": {
      "type": "string",
      "description": "The ID of the data table to read rows from"
    },
    "projectId": {
      "type": "string",
      "description": "The project ID the data table belongs to"
    },
    "filter": {
      "type": "object",
      "properties": {
        "type": {
          "type": "string",
          "enum": [
            "and",
            "or"
          ],
          "default": "and",
          "description": "How to combine the filters: all must match (and) or any may match (or)"
        },
        "filters": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "columnName": {
                "type": "string",
                "minLength": 1,
                "maxLength": 63,
                "pattern": "^[a-zA-Z][a-zA-Z0-9_]*$",
                "description": "Column to filter on. System columns 'id', 'createdAt' and 'updatedAt' are also allowed"
              },
              "condition": {
                "type": "string",
                "enum": [
                  "eq",
                  "neq",
                  "like",
                  "ilike",
                  "gt",
                  "gte",
                  "lt",
                  "lte"
                ],
                "default": "eq",
                "description": "Comparison operator. 'like' (case-sensitive) and 'ilike' (case-insensitive) match substrings; include % wildcards for custom patterns. 'neq' also matches rows where the column is null"
              },
              "value": {
                "type": [
                  "string",
                  "number",
                  "boolean",
                  "null"
                ],
                "description": "Value to compare against. For date columns, pass an ISO 8601 string. Pass null with eq/neq to match rows where the column is null / not null"
              }
            },
            "required": [
              "columnName",
              "value"
            ],
            "additionalProperties": false
          },
          "minItems": 1
        }
      },
      "required": [
        "filters"
      ],
      "additionalProperties": false,
      "description": "Filter conditions to select rows. Omit to return all rows"
    },
    "sortBy": {
      "type": "string",
      "pattern": "^[^:]+:(asc|desc)$",
      "description": "Sort order as '<columnName>:<asc|desc>', e.g. 'createdAt:desc'"
    },
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 100,
      "description": "Limit the number of results (max 100)"
    },
    "skip": {
      "type": "integer",
      "minimum": 0,
      "description": "Number of rows to skip, for paginating through large result sets"
    }
  },
  "required": [
    "dataTableId",
    "projectId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-data-table-rows`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-add-data-table-rows`

Insert rows into an existing data table. Each row is an object mapping column names to values. Use search_data_tables to find the data table ID first.

id: `c0637d930e0b4463a012ab686b68c2b3`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "dataTableId": {
      "type": "string",
      "description": "The ID of the data table to insert rows into"
    },
    "projectId": {
      "type": "string",
      "description": "The project ID the data table belongs to"
    },
    "rows": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": {
          "type": [
            "string",
            "number",
            "boolean",
            "null"
          ]
        }
      },
      "minItems": 1,
      "maxItems": 1000,
      "description": "Array of row objects to insert. Each object maps column names to values. Maximum 1000 rows per call."
    }
  },
  "required": [
    "dataTableId",
    "projectId",
    "rows"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-add-data-table-rows`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-rename-data-table-column`

Rename a column in a data table.

id: `3641590211024875a43f91276d1a5bc2`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "dataTableId": {
      "type": "string",
      "description": "The ID of the data table containing the column"
    },
    "projectId": {
      "type": "string",
      "description": "The project ID the data table belongs to"
    },
    "columnId": {
      "type": "string",
      "description": "The ID of the column to rename"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 63,
      "pattern": "^[a-zA-Z][a-zA-Z0-9_]*$",
      "description": "The new column name"
    }
  },
  "required": [
    "dataTableId",
    "projectId",
    "columnId",
    "name"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-rename-data-table-column`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-delete-data-table-column`

Delete a column from a data table. This permanently removes the column and all its data.

id: `ef94ca8b840842ae9158ddaa6876490b`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "dataTableId": {
      "type": "string",
      "description": "The ID of the data table containing the column"
    },
    "projectId": {
      "type": "string",
      "description": "The project ID the data table belongs to"
    },
    "columnId": {
      "type": "string",
      "description": "The ID of the column to delete"
    }
  },
  "required": [
    "dataTableId",
    "projectId",
    "columnId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-delete-data-table-column`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-add-data-table-column`

Add a new column to an existing data table.

id: `ebedd5111d14473f9725beab96843c26`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "dataTableId": {
      "type": "string",
      "description": "The ID of the data table to add a column to"
    },
    "projectId": {
      "type": "string",
      "description": "The project ID the data table belongs to"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 63,
      "pattern": "^[a-zA-Z][a-zA-Z0-9_]*$",
      "description": "Column name. Must start with a letter, contain only letters, numbers, and underscores (max 63 chars)"
    },
    "type": {
      "type": "string",
      "enum": [
        "string",
        "number",
        "boolean",
        "date"
      ],
      "description": "The data type of the new column"
    }
  },
  "required": [
    "dataTableId",
    "projectId",
    "name",
    "type"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-add-data-table-column`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-rename-data-table`

Rename an existing data table.

id: `2ac9706735cc4bc189d45aff0710d8b8`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "dataTableId": {
      "type": "string",
      "description": "The ID of the data table to rename"
    },
    "projectId": {
      "type": "string",
      "description": "The project ID the data table belongs to"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 128,
      "description": "The new name for the data table"
    }
  },
  "required": [
    "dataTableId",
    "projectId",
    "name"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-rename-data-table`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-create-data-table`

Create a new data table with the specified columns. Use search_projects to find a project ID first.

id: `8e5270d8c0ad4d619ddd82b26d2c1cff`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "projectId": {
      "type": "string",
      "description": "The project ID where the data table will be created"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 128,
      "description": "The name of the data table (must be unique within the project)"
    },
    "columns": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "minLength": 1,
            "maxLength": 63,
            "pattern": "^[a-zA-Z][a-zA-Z0-9_]*$",
            "description": "Column name. Must start with a letter, contain only letters, numbers, and underscores (max 63 chars)"
          },
          "type": {
            "type": "string",
            "enum": [
              "string",
              "number",
              "boolean",
              "date"
            ],
            "description": "The data type of the column"
          }
        },
        "required": [
          "name",
          "type"
        ],
        "additionalProperties": false
      },
      "minItems": 1,
      "description": "The columns to create in the data table. At least one column is required."
    }
  },
  "required": [
    "projectId",
    "name",
    "columns"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-create-data-table`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-data-tables`

Search for data tables accessible to the current user. Use this to find a data table ID before modifying or adding data to it.

id: `170e18181ec04a209c4aff99bb0fb86e`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Filter data tables by name (case-insensitive partial match)"
    },
    "projectId": {
      "type": "string",
      "description": "Filter by project ID"
    },
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 100,
      "description": "Limit the number of results (max 100)"
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-data-tables`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-list-workflow-tags`

List all workflow tags in the instance.

id: `d362dd8a76964800b07834ac69484c61`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 500,
      "description": "Limit the number of results (max 500)"
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-list-workflow-tags`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-list-n8n-connect-services`

List n8n credits coverage: node and credential types the platform can provide managed credentials for, plus supported resource+operation combinations, minimum type versions, and hidden node properties. Use this to decide which nodes let the user skip credential setup.

id: `a71aaae5e3e2408f94e21b8ef12ed3b5`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-list-n8n-connect-services`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-list-credentials`

List credentials the current user can access. Use this to find a credential ID before referencing it anywhere one is required. Prefer reusing a credential already used by another node in the workflow (get_workflow_details with detailLevel 'full' shows the credentials on each node); when the workflow has none of that type and multiple candidates exist, ask the user which one to use rather than picking one. Never returns credential secret data.

id: `a3b9a3266f8645cc90ba7bbd9c95b3e1`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 200,
      "description": "Limit the number of results (max 200)"
    },
    "query": {
      "type": "string",
      "description": "Filter credentials by name (partial match)"
    },
    "type": {
      "type": "string",
      "description": "Filter by credential type (e.g. \"slackApi\", \"httpHeaderAuth\"). Partial match."
    },
    "projectId": {
      "type": "string",
      "description": "Restrict results to credentials belonging to this project"
    },
    "onlySharedWithMe": {
      "type": "boolean",
      "description": "Only return credentials shared directly with the current user"
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-list-credentials`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-test-workflow`

Test a workflow using pin data to bypass external services. Trigger nodes, nodes with credentials, and HTTP Request nodes are pinned (use simulated data). Other nodes (Set, If, Code, etc.) execute normally — including credential-free I/O nodes like Execute Command or file read/write nodes. Use prepare_workflow_pin_data to generate the pin data first.

id: `cf3a5c4c1f9e442eaadd87afc3d62807`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to test"
    },
    "pinData": {
      "type": "object",
      "additionalProperties": {
        "type": "array",
        "items": {
          "type": "object",
          "additionalProperties": {}
        }
      },
      "description": "Pin data for all workflow nodes. Use the prepare_workflow_pin_data tool to generate this. Keys are node names, values are arrays of items. Each item MUST be wrapped in a \"json\" property, e.g. [{\"json\": {\"id\": \"123\", \"name\": \"test\"}}]. Do NOT pass flat objects like [{\"id\": \"123\"}]."
    },
    "triggerNodeName": {
      "type": "string",
      "description": "Optional name of the trigger node to start execution from. Useful for workflows with multiple triggers. Defaults to the first trigger node found."
    },
    "timeout": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 3600,
      "description": "Optional timeout in seconds before the test execution is interrupted. Defaults to 300 seconds. Increase this to test workflows that take longer to run."
    }
  },
  "required": [
    "workflowId",
    "pinData"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-test-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-prepare-workflow-pin-data`

Prepare test pin data for a workflow. Trigger nodes, nodes with credentials, and HTTP Request nodes need pin data. Logic nodes (Set, If, Code, etc.) and credential-free I/O nodes (Execute Command, file read/write) execute normally without pin data. Returns JSON Schemas describing the expected output shape for each node that needs pin data — schemas are derived from past execution output shapes or node type definitions. No actual user data is returned. You should generate realistic sample data for the schemas, use empty defaults for nodes without schema, merge everything into a single pinData object, and pass it to test_workflow.

id: `52b3ceee8e254b51a84ab615e9749c90`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to generate test pin data for"
    }
  },
  "required": [
    "workflowId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-prepare-workflow-pin-data`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-unpublish-workflow`

Unpublish (deactivate) a workflow to stop it from being available for production execution.

id: `8c12e2ccd9f24677a35ac1a7e9be6c29`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to unpublish"
    }
  },
  "required": [
    "workflowId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-unpublish-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-publish-workflow`

Publish (activate) a workflow to make it available for production execution. This creates an active version from the current draft.

id: `bda113af339641a0b6ec38269ee1d3d2`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to publish"
    },
    "versionId": {
      "type": "string",
      "description": "Optional version ID to publish. If not provided, publishes the current draft version."
    }
  },
  "required": [
    "workflowId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-publish-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-versions-diff`

Compare two saved versions of a workflow and return what changed between them: nodes added (with their full content), removed, or modified (with a field-level delta per modified node) and connections added or removed. Pass the older version as fromVersionId and the newer one as toVersionId, using version IDs from get_workflow_history.

id: `8a3c59b245ae418a8854a344be5e47c7`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow the versions belong to"
    },
    "fromVersionId": {
      "type": "string",
      "description": "The base (older) version ID, as returned by get_workflow_history"
    },
    "toVersionId": {
      "type": "string",
      "description": "The target (newer) version ID, as returned by get_workflow_history"
    }
  },
  "required": [
    "workflowId",
    "fromVersionId",
    "toVersionId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-versions-diff`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-version`

Retrieve the full content (nodes, connections, node groups) of a specific workflow version from its history. Use the versionId from get_workflow_history.

id: `b8470802857d4154b5cb55e24ddfe33c`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow the version belongs to"
    },
    "versionId": {
      "type": "string",
      "description": "The version ID to retrieve, as returned by get_workflow_history"
    }
  },
  "required": [
    "workflowId",
    "versionId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-version`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-history`

List the saved version history of a workflow (newest first), so you can inspect how it changed over time and pick a version to retrieve or restore.

id: `e76b6b9c262642d89c6dbc638e3329e7`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to read version history for"
    },
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 50,
      "description": "Limit the number of results (max 50)"
    },
    "offset": {
      "type": "integer",
      "minimum": 0,
      "description": "Number of versions to skip for pagination (default 0)"
    }
  },
  "required": [
    "workflowId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-history`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-details`

Get detailed information about a specific workflow including trigger details

id: `b7fe069369a64e4d8155d7a79cf06288`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to retrieve"
    },
    "detailLevel": {
      "type": "string",
      "enum": [
        "full",
        "execution"
      ],
      "default": "full",
      "description": "Level of detail to return. 'full' (default) includes the complete workflow payload. 'execution' returns only the workflow metadata and trigger information needed to run it — prefer it when the goal is just to execute the workflow via execute_workflow."
    }
  },
  "required": [
    "workflowId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-details`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-workflow-executions`

Search for workflow executions with optional filters. Returns execution metadata including status, timing, and workflow ID.

id: `caef9905d40a43c5880caaad8f999630`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "Filter executions by workflow ID"
    },
    "status": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": [
          "canceled",
          "crashed",
          "error",
          "new",
          "running",
          "success",
          "unknown",
          "waiting"
        ]
      },
      "description": "Filter by execution status(es)"
    },
    "startedAfter": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 timestamp — only return executions that started after this time"
    },
    "startedBefore": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 timestamp — only return executions that started before this time"
    },
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 200,
      "description": "Limit the number of results (max 200)"
    },
    "lastId": {
      "type": "string",
      "description": "Cursor for pagination — pass the last execution ID from the previous page"
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-workflow-executions`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-get-workflow-execution`

Get workflow execution details by execution ID and workflow ID. By default returns metadata only. Set includeData to true to include node execution data, optionally filtered by nodeNames and truncated by truncateData.

id: `31fe2a10d7724d4fb05871fbd467d39b`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow the execution belongs to"
    },
    "executionId": {
      "type": "string",
      "description": "The ID of the execution to retrieve"
    },
    "includeData": {
      "type": "boolean",
      "description": "Whether to include the full execution result data. Defaults to false (metadata only). Set to true to include node inputs/outputs. Use `false` to quickly check execution status"
    },
    "nodeNames": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "When includeData is true, return data only for these node names. If omitted, data for all nodes is included."
    },
    "truncateData": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "description": "When includeData is true, limit the number of data items returned per node output to this value. If omitted, all items are returned."
    }
  },
  "required": [
    "workflowId",
    "executionId"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-get-workflow-execution`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-execute-workflow`

Execute a workflow by ID. Returns the execution ID immediately without waiting for completion. Before executing always ensure you know the input schema by first using the get_workflow_details tool and consulting workflow description; pass detailLevel 'execution' to that tool when running the workflow is all you need, since the full graph is not required here.

id: `c88a87f97bed44e1a4d11455127436c9`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "workflowId": {
      "type": "string",
      "description": "The ID of the workflow to execute"
    },
    "executionMode": {
      "type": "string",
      "enum": [
        "manual",
        "production"
      ],
      "description": "Required execution intent. Use \"manual\" for testing or validating the current workflow, including tests against live external services. Use \"production\" only when intentionally running the published workflow as a live execution."
    },
    "triggerNodeName": {
      "type": "string",
      "description": "Name of the trigger node to execute. Required when providing inputs. If omitted, the workflow must have exactly one trigger that does not require inputs (Schedule Trigger, or Manual Trigger in manual mode). Use get_workflow_details to see available trigger names."
    },
    "inputs": {
      "anyOf": [
        {
          "type": "object",
          "properties": {
            "chatInput": {
              "type": "string",
              "description": "Input for chat-based workflows"
            }
          },
          "required": [
            "chatInput"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "formData": {
              "type": "object",
              "additionalProperties": {},
              "description": "Input data for form-based workflows"
            }
          },
          "required": [
            "formData"
          ],
          "additionalProperties": false
        },
        {
          "type": "object",
          "properties": {
            "webhookData": {
              "type": "object",
              "properties": {
                "method": {
                  "type": "string",
                  "enum": [
                    "GET",
                    "POST",
                    "PUT",
                    "DELETE",
                    "PATCH",
                    "HEAD",
                    "OPTIONS"
                  ],
                  "default": "GET",
                  "description": "HTTP method (defaults to GET)"
                },
                "query": {
                  "type": "object",
                  "additionalProperties": {
                    "type": "string"
                  },
                  "description": "Query string parameters"
                },
                "body": {
                  "type": "object",
                  "additionalProperties": {},
                  "description": "Request body data (main webhook payload)"
                },
                "headers": {
                  "type": "object",
                  "additionalProperties": {
                    "type": "string"
                  },
                  "description": "HTTP headers (e.g., authorization, content-type)"
                }
              },
              "additionalProperties": false,
              "description": "Input data for webhook-based workflows"
            }
          },
          "required": [
            "webhookData"
          ],
          "additionalProperties": false
        }
      ],
      "description": "Trigger payload. Required for webhook, chat, and form triggers. Must be omitted for schedule and manual triggers. Use get_workflow_details to see the expected payload for each trigger."
    }
  },
  "required": [
    "workflowId",
    "executionMode"
  ],
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-execute-workflow`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-search-workflows`

Search for workflows with optional filters. Returns a preview of each workflow.

id: `e6f27e613ac04ecc9bdac460c3297bd8`

enabled: `True`

Exposed by: `n8n`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "exclusiveMinimum": 0,
      "maximum": 200,
      "description": "Limit the number of results (max 200)"
    },
    "query": {
      "type": "string",
      "description": "Filter by name or description"
    },
    "projectId": {
      "type": "string"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Filter by tag names (AND semantics — workflow must have all)."
    },
    "sortBy": {
      "type": "string",
      "enum": [
        "updatedAt:desc",
        "updatedAt:asc",
        "createdAt:desc",
        "createdAt:asc",
        "name:asc",
        "name:desc"
      ],
      "description": "Sort order for results (default: updatedAt:desc). Use updatedAt:desc to find the most recently edited workflows first."
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-search-workflows`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-github-app-webhook`

The GitHub App's webhook, as GitHub sees it: target URL, subscribed events, recent deliveries.

    Added 2026-10-02 (Claude Code · Opus 5.5) to find why a push does not start a deployment.
    Push-to-deploy needs, in order: GitHub sends a `push` delivery to this URL, Coolify answers it,
    and the app has auto-deploy on and a watch path matching a changed file. This tool shows the
    first two; get_application shows the rest (`settings.is_auto_deploy_enabled`, `watch_paths`).

    Read-only. The App's private key and webhook secret are never returned.

    Parameters
    ----------
    github_app_id : int        # Coolify's id for the GitHub App source (the apps' `source_id`; 2)
    repository : str, optional # "owner/name" or "name": only deliveries for that repository
    limit : int                # deliveries to fetch from GitHub before filtering (max 100)
    delivery_id : int, optional
        One delivery in full: its push ref and changed files, and what Coolify answered.

    Returns
    -------
    dict  {app, hook, deliveries: [{id, delivered_at, event, status, status_code, repository}],
           delivery?: {event, ref, changed_files, response_status, response_body}}

id: `852c778297df4136ab5d5ed8ed31cd42`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "github_app_id": {
      "default": 2,
      "title": "Github App Id",
      "type": "integer"
    },
    "repository": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Repository"
    },
    "limit": {
      "default": 30,
      "title": "Limit",
      "type": "integer"
    },
    "delivery_id": {
      "anyOf": [
        {
          "type": "integer"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Delivery Id"
    }
  },
  "title": "github_app_webhookArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-github-app-webhook`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `memsearch-status`

memsearch server health: collection, row count, embedder, versions.

probe_embedding: also embed one short text to prove the embedding endpoint answers.

id: `344bba31ea074191801acd08da9f65bb`

enabled: `True`

Exposed by: `memsearch`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "probe_embedding": {
      "default": false,
      "title": "Probe Embedding",
      "type": "boolean"
    }
  },
  "title": "statusArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `memsearch-status`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `memsearch-recall`

Search, expand the best notes and pack them for an agent: the memory answer in one call.

query: what to recall.
top_k: notes to include (1-10).
max_chars: total text budget (default 8000).
Each note is headed "[n] date · heading · agent · score".

id: `bcc8ac143d40488b8f81bcc666efcdc2`

enabled: `True`

Exposed by: `memsearch`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "query": {
      "title": "Query",
      "type": "string"
    },
    "top_k": {
      "default": 5,
      "title": "Top K",
      "type": "integer"
    },
    "max_chars": {
      "default": 8000,
      "title": "Max Chars",
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "title": "recallArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `memsearch-recall`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `memsearch-expand`

The whole journal section (one note) around a chunk, rebuilt from its chunks in Milvus.

chunk_hash: from a search hit.
max_chars: text limit (0 = no limit).
Returns file, date, heading, agent, line range, the note's anchor (session, turn, transcript path)
and the cleaned text. Lines the ingest filter dropped before indexing are not stored, so they are
not shown.

id: `816be88ddadb4d45913404ab6a23b333`

enabled: `True`

Exposed by: `memsearch`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "chunk_hash": {
      "title": "Chunk Hash",
      "type": "string"
    },
    "max_chars": {
      "default": 12000,
      "title": "Max Chars",
      "type": "integer"
    }
  },
  "required": [
    "chunk_hash"
  ],
  "title": "expandArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `memsearch-expand`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `memsearch-search`

Search the shared memsearch memory (hybrid dense + BM25, the same search as `memsearch search`).

query: what to look for, in natural language or exact words.
top_k: results to return (1-20).
source_contains: optional substring the source path must contain, e.g. "2026-10-02" for one day's
    journal or "memory\codex" for the imported Codex memories.
max_chars: per-result text limit (0 = no limit).
Each hit has chunk_hash (for expand), file, date, heading, agent, score and cleaned text.
A query none of whose words occurs in the collection returns no hits.

id: `6e6390a909364a0b8ba6f9d76f15ddef`

enabled: `True`

Exposed by: `memsearch`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "query": {
      "title": "Query",
      "type": "string"
    },
    "top_k": {
      "default": 8,
      "title": "Top K",
      "type": "integer"
    },
    "source_contains": {
      "default": "",
      "title": "Source Contains",
      "type": "string"
    },
    "max_chars": {
      "default": 700,
      "title": "Max Chars",
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "title": "searchArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `memsearch-search`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-plan-claim-followup`

Save a planned document search, investigation, legal research or discovery action for a claim.

id: `0d1d8331f0904cf7bde197ae30e9d55c`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "claim_id": {
      "type": "string"
    },
    "expected_revision": {
      "type": "integer"
    },
    "kind": {
      "type": "string"
    },
    "description": {
      "type": "string"
    },
    "gap_id": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    }
  },
  "required": [
    "claim_id",
    "expected_revision",
    "kind",
    "description"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-plan-claim-followup`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-open-probata-legal-response`

Open one legal response over an existing Probata entity/event ID and pinned version.

id: `d0b430b0a8b64cdeb2d1dbfa339a6dac`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "origin": {
      "additionalProperties": true,
      "type": "object"
    },
    "response": {
      "default": "",
      "type": "string"
    }
  },
  "required": [
    "origin"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-open-probata-legal-response`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-create-case-claim`

Save a private claim or response. This does not accept evidence or send a request.

id: `aa39659bc20a4ac4aa9ac6349a7aea9c`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "text": {
      "type": "string"
    },
    "kind": {
      "default": "assertion",
      "type": "string"
    },
    "claimant": {
      "default": "",
      "type": "string"
    },
    "response": {
      "default": "",
      "type": "string"
    }
  },
  "required": [
    "text"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-create-case-claim`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-case-claim-gaps`

Read claim-specific evidence gaps and planned follow-up actions.

id: `b1a1fed73fda40868772dc2edf587e25`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {},
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-case-claim-gaps`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-case-claims`

Read the same saved claims, legal responses and live Probata source pointers as the workdesk.

id: `edf14f8d9627416ca8dbd5f247e30fdd`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {},
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-case-claims`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-coolify-api`

Call any Coolify v4 REST endpoint directly. Escape hatch for the long tail.

    Covers everything without a dedicated tool: teams, private keys, GitHub
    apps, database backups, and server administration. See
    references/CLI-PARITY.md for the exact method+path of every such operation.

    Parameters
    ----------
    method : str   # GET, POST, PATCH, PUT, DELETE
    path : str     # path under the API base, e.g. "/teams" or
                   # "/databases/abc-123/backups". Leading slash optional;
                   # do NOT include the /api/v1 prefix (it is in the base URL).
    body : dict, optional    # JSON request body for POST/PATCH/PUT
    params : dict, optional  # query string parameters
    confirm_destructive : bool
        Required True for any non-GET/HEAD method. This is a deliberate
        speed bump: the passthrough can delete anything the token can reach
        and has no per-resource name guard like delete_application does.

    Examples
    --------
    List teams:            method="GET",  path="/teams"
    Current team:          method="GET",  path="/teams/current"
    List private keys:     method="GET",  path="/security/keys"
    List GitHub apps:      method="GET",  path="/github-apps"
    List db backups:       method="GET",  path="/databases/{uuid}/backups"
    Validate a server:     method="GET",  path="/servers/{uuid}/validate"
    Server resources:      method="GET",  path="/servers/{uuid}/resources"
    Create a private key:  method="POST", path="/security/keys",
                           body={"name": "...", "private_key": "..."},
                           confirm_destructive=True

    Do / Don't
    ----------
    Do: prefer a dedicated tool when one exists — they carry guards, retries,
        and response post-processing this raw path does not.
    Don't: paste a secret value into `body` and then echo the result — the
        transcript keeps both.

    Returns
    -------
    The decoded JSON response, or None for 204/empty bodies.

id: `22c35ff9da0d40fd84a1c6d31edc0b58`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "method": {
      "title": "Method",
      "type": "string"
    },
    "path": {
      "title": "Path",
      "type": "string"
    },
    "body": {
      "anyOf": [
        {
          "additionalProperties": true,
          "type": "object"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Body"
    },
    "params": {
      "anyOf": [
        {
          "additionalProperties": true,
          "type": "object"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Params"
    },
    "confirm_destructive": {
      "default": false,
      "title": "Confirm Destructive",
      "type": "boolean"
    }
  },
  "required": [
    "method",
    "path"
  ],
  "title": "coolify_apiArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-coolify-api`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-cancel-deployment`

Cancel a queued or in-progress deployment by deployment UUID.

    POST /deployments/{uuid}/cancel. Endpoint taken from the v4 OpenAPI spec
    and NOT verified live against this instance — if it 404s, the deployment
    can still be stopped by stop_application on the target app.

    Parameters
    ----------
    deployment_uuid : str  # from list_deployments or a deploy_* response

id: `05aa10cd77e540cf9f689781b1396a6e`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "deployment_uuid": {
      "title": "Deployment Uuid",
      "type": "string"
    }
  },
  "required": [
    "deployment_uuid"
  ],
  "title": "cancel_deploymentArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-cancel-deployment`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-deployments`

List deployments currently queued or running across the whole team.

    GET /deployments — team-wide and in-flight only. For one app's deployment
    HISTORY (including finished ones) use list_deployments_for_app instead.

id: `848fb0c0bb444cd8b76b1e65ad03323a`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {},
  "title": "list_deploymentsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-deployments`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-delete-project`

Delete a project by UUID. DESTRUCTIVE — requires confirm_name.

    Do / Don't
    ----------
    Do: list what the project still holds first (list_applications /
        list_databases / list_services) — deleting a project takes its
        resources with it.

id: `4d0532f4ee1a4cba81109f3b29769a5b`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "confirm_name": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Confirm Name"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "delete_projectArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-delete-project`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-update-project`

Rename a project or change its description. PATCH /projects/{uuid}.

    Only the fields you pass are sent; omitted fields are left unchanged.

id: `ac5e48870034473ab53e624462cb3bac`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "name": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Name"
    },
    "description": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Description"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "update_projectArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-update-project`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-create-project`

Create a new project (the container for applications/databases/services).

    POST /projects. Returns the record including the uuid you pass as
    project_uuid to create_application / create_database / create_service.

id: `1bb11ee5993e438989fee57c78adcb6f`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "name": {
      "title": "Name",
      "type": "string"
    },
    "description": {
      "default": "",
      "title": "Description",
      "type": "string"
    }
  },
  "required": [
    "name"
  ],
  "title": "create_projectArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-create-project`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-delete-application-env`

Delete one environment variable from an application by its env UUID.

    DELETE /applications/{uuid}/envs/{env_uuid}.

    Do / Don't
    ----------
    Do: get env_uuid from list_application_envs immediately before deleting.
    Do: redeploy afterwards — a removed env stays baked into the running
        container until the next deploy renders a fresh compose.

id: `fbabc4fe7990493ab815264cbb91115b`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "env_uuid": {
      "title": "Env Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid",
    "env_uuid"
  ],
  "title": "delete_application_envArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-delete-application-env`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-application-envs`

List environment variables on an application (key, value, uuid, flags).

    GET /applications/{uuid}/envs. Use this to find an env's `uuid` before
    delete_application_env, and to audit what is actually set versus what the
    compose expects.

    Requires
    --------
    Token needs `read:sensitive` to see VALUES; without it keys come back with
    values stripped.

id: `4f180dec7b8e4b69aeeaf1e491a73c39`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "list_application_envsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-application-envs`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-delete-service`

Delete a service (compose stack) by UUID. DESTRUCTIVE — requires confirm_name.

id: `5826604032fd4618957729628ff3b970`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "confirm_name": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Confirm Name"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "delete_serviceArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-delete-service`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-restart-service`

Restart a service (compose stack) by UUID. POST /services/{uuid}/restart.

id: `aa116e39f7c44e8cbe3027ebb66af327`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "restart_serviceArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-restart-service`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-stop-service`

Stop a running service (compose stack) by UUID. POST /services/{uuid}/stop.

id: `e222ffc6fdea451d8637df391c4481d5`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "stop_serviceArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-stop-service`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-start-service`

Start a stopped service (compose stack) by UUID. POST /services/{uuid}/start.

id: `bc5a33b8f81b4ecdb4c6a274771c457e`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "start_serviceArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-start-service`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-create-service`

Create a multi-container service from a raw docker-compose document.

    POST /services. Distinct from create_application: a *service* is a compose
    stack Coolify manages as one unit; an *application* is git-backed and
    rebuilt on push.

    Parameters
    ----------
    docker_compose_raw : str
        The compose file content as a YAML string (not a path, not a dict).

    Do / Don't
    ----------
    Do: parse-check the compose locally before sending — Coolify accepts the
        create and only fails at deploy time on a malformed document.
    Do: check_port_collision for every host port the compose binds.
    Don't: bind host port 8080 — coolify-proxy owns it on every node.

id: `00a1be886ab34232a50e545a21a33477`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "project_uuid": {
      "title": "Project Uuid",
      "type": "string"
    },
    "server_uuid": {
      "title": "Server Uuid",
      "type": "string"
    },
    "name": {
      "title": "Name",
      "type": "string"
    },
    "docker_compose_raw": {
      "title": "Docker Compose Raw",
      "type": "string"
    },
    "environment_name": {
      "default": "production",
      "title": "Environment Name",
      "type": "string"
    },
    "description": {
      "default": "",
      "title": "Description",
      "type": "string"
    },
    "instant_deploy": {
      "default": false,
      "title": "Instant Deploy",
      "type": "boolean"
    }
  },
  "required": [
    "project_uuid",
    "server_uuid",
    "name",
    "docker_compose_raw"
  ],
  "title": "create_serviceArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-create-service`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-delete-database`

Delete a standalone database by UUID. DESTRUCTIVE — requires confirm_name.

    Same guard as delete_application: the name is re-read and must match.

    Do / Don't
    ----------
    Do: take a backup first (coolify_api POST /databases/{uuid}/backups/{backup_uuid}/execute)
        — deleting a database destroys its volume.
    Don't: pass a uuid you haven't re-read this session.

id: `3fa7b8bf754046cc8ee45fcf2953d0d5`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "confirm_name": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Confirm Name"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "delete_databaseArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-delete-database`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-restart-database`

Restart a standalone database by UUID. POST /databases/{uuid}/restart.

id: `d1af3b9078cf401cae3eb04f1d9d3ae6`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "restart_databaseArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-restart-database`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-stop-database`

Stop a running standalone database by UUID. POST /databases/{uuid}/stop.

    Do / Don't
    ----------
    Don't: `docker stop` the container directly — same orphan problem as
    applications; Coolify owns the lifecycle.

id: `8716cb4ca9064adb8c6d106937a6442f`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "stop_databaseArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-stop-database`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-start-database`

Start a stopped standalone database by UUID. POST /databases/{uuid}/start.

id: `9145789a2e6744e0815ae31d03750ce4`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "start_databaseArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-start-database`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-create-database`

Create a standalone database. One tool covers all 8 Coolify engine types.

    POST /databases/{db_type} — replaces the old CLI's eight separate
    `databases create-*` commands.

    Parameters
    ----------
    db_type : str
        One of: postgresql, mysql, mariadb, mongodb, redis, keydb,
        clickhouse, dragonfly.
    options : dict, optional
        Engine-specific fields passed through verbatim, e.g.
        {"postgres_user": "admin", "postgres_password": "...",
         "postgres_db": "myapp"} for postgresql, or {"redis_password": "..."}
        for redis. Omitted keys get Coolify's generated defaults.

    Do / Don't
    ----------
    Do: confirm project_uuid + server_uuid via get_infrastructure_overview first.
    Do: run check_port_collision if you plan to expose the DB on a host port.
    Don't: hand-write a password you then log — the transcript keeps it.

    Returns
    -------
    dict — created database record (uuid, name). Start it with start_database.

id: `450df9f54ae143d6a7ebf44612994faf`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "project_uuid": {
      "title": "Project Uuid",
      "type": "string"
    },
    "server_uuid": {
      "title": "Server Uuid",
      "type": "string"
    },
    "db_type": {
      "title": "Db Type",
      "type": "string"
    },
    "name": {
      "title": "Name",
      "type": "string"
    },
    "environment_name": {
      "default": "production",
      "title": "Environment Name",
      "type": "string"
    },
    "description": {
      "default": "",
      "title": "Description",
      "type": "string"
    },
    "options": {
      "anyOf": [
        {
          "additionalProperties": true,
          "type": "object"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Options"
    }
  },
  "required": [
    "project_uuid",
    "server_uuid",
    "db_type",
    "name"
  ],
  "title": "create_databaseArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-create-database`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-update-application`

Repoint an existing application's build inputs (PATCH /applications/{uuid}).

    Added 2026-09-28 (Claude Code · Opus 5): repointing an app at a different compose file or
    base directory had no tool, so it would have meant a raw coolify_api call. Changing an app in
    place is how a service gets replaced without standing up a parallel stack.
    is_auto_deploy_enabled added 2026-10-02 (Claude Code · Opus 5.5).

    Only the fields you pass are touched. Writes nothing unless confirm=True; then it re-reads the
    application and checks every field it set actually took. If any did not, the previous values
    are written back and the result says so, so a half-applied repoint never survives silently.

    Parameters
    ----------
    uuid : str                       # application UUID (list_applications)
    base_directory : str, optional   # build context root, e.g. "/" for the monorepo root
    docker_compose_location : str, optional  # compose path RELATIVE TO base_directory
    watch_paths : str, optional      # newline-separated globs, matched against repository root
    is_auto_deploy_enabled : bool, optional
        Whether a push deploys the app. Off on every app by owner rule (2026-09-20, reaffirmed
        2026-10-02): deploys are explicit. Read back from the record's `settings`.
    confirm : bool                   # must be True to write; False reports the diff only

    Returns
    -------
    dict  {uuid, changes: {field: {from, to}}, written, verified, restored?}

id: `ef51dbd382ef4ebba54da489f3f1e2cd`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "name": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Name"
    },
    "description": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Description"
    },
    "base_directory": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Base Directory"
    },
    "docker_compose_location": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Docker Compose Location"
    },
    "watch_paths": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Watch Paths"
    },
    "git_repository": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Git Repository"
    },
    "git_branch": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Git Branch"
    },
    "is_auto_deploy_enabled": {
      "anyOf": [
        {
          "type": "boolean"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Is Auto Deploy Enabled"
    },
    "confirm": {
      "default": false,
      "title": "Confirm",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "update_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-update-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-set-service-image`

Point one container of a Coolify service at a new image tag, without exposing its compose.

    Reads the service's compose, changes ONLY services..image, writes it back, then
    re-reads it and checks that nothing else changed. If the read-back differs anywhere other
    than that one field, the original compose is restored. Secret values never appear in the
    result. Follow with a deploy of the service to start the new image.

    Parameters
    ----------
    uuid : str          # service UUID (get_service / list_services)
    service_name : str  # the compose service key, e.g. "docstore"
    image : str         # new image reference, e.g. "propria-docstore:0.8.1-r7"
    confirm : bool      # must be True to write; False reports what would change

    Returns
    -------
    dict  {uuid, service, old_image, new_image, written, verified, restored?}

id: `60f86a108bcc400b8967e50d4bcc041c`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "service_name": {
      "title": "Service Name",
      "type": "string"
    },
    "image": {
      "title": "Image",
      "type": "string"
    },
    "confirm": {
      "default": false,
      "title": "Confirm",
      "type": "boolean"
    }
  },
  "required": [
    "uuid",
    "service_name",
    "image"
  ],
  "title": "set_service_imageArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-set-service-image`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-service-env`

Environment variables a service declares in its compose, grouped by compose service.

    Added 2026-09-30 (Claude Code · Opus 5) because redacting get_service closed the only route
    to these values, and one job legitimately needs them: migrating them into a secrets manager.
    Applications have list_application_envs; services keep their environment inside
    docker_compose_raw, which get_service masks.

    This returns the env pairs and NOTHING else -- no compose document, no webhook secrets, no
    image, volume or network detail -- so reading a value here does not also expose the rest of
    the stack the way an unredacted get_service would.

    Parameters
    ----------
    uuid : str     # service UUID (list_services)
    reveal : bool  # False (default) gives names with value LENGTHS, enough to inventory;
                   # True gives the values, and is the only way to read them through this plugin.

    Returns
    -------
    dict  {uuid, name, services: {: {: value | ""}}, revealed}

id: `cded4574918d45778b06e922affa39f1`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "reveal": {
      "default": false,
      "title": "Reveal",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "get_service_envArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-service-env`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `atomic-tools-atomic-tools`

Probata's atomic evidence tools: 43 tools in 7 families (parsers, extractors, repair and engine probes). Browse first, then run one.
- documents (2; extract.text): extract-docling, extract-text
- engine (2; engine.certify, engine.inspect): poppler-certify-text, poppler-inspect
- geo_map (1; viz.geo_map): leaflet
- ingest (1; ingest.context-drain): context-drain
- messages (11; parse.facebook, parse.imessage, parse.messages-csv, parse.messages-transcript, parse.sms-xml, parse.snapchat, parse.whatsapp): facebook-html, facebook-json, imessage-html, imessage-pdf, imessage-txt, messaging-csv, +5 more
- repair (10; repair.audit, repair.derive, repair.detect, repair.flag, repair.inspect, repair.preview, repair.quarantine, repair.quarantine-plan): audit-verify, capabilities, detect, flag-damaged, pdf-derived, pdf-inspect, +4 more
- transcripts (16; parse.transcript): chatgpt-custom-gpt-md, chatgpt-official, chatgpt-share, claude-ai-export, claude-code, claude-code-jsonl, +10 more
Use: path='' lists families; path='' lists its tools; path='.' shows the full contract; add run={source_ref, args} to execute it. source_ref is a locator (upload://, r2://, b2://), never a host path; omit it for tools that take no input.

id: `6b027d31f78242f2b5115c4ca4fb2374`

enabled: `True`

Exposed by: `atomic-tools`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "'' lists families; '<family>' lists its tools; '<family>.<tool>' shows the full contract"
    },
    "run": {
      "type": [
        "null",
        "object"
      ],
      "properties": {
        "source_ref": {
          "type": "string",
          "description": "locator of the input (upload://, r2://, b2://); omit for tools that take no input"
        },
        "args": {
          "type": "object",
          "description": "tool options such as format or sample_limit; never a host path",
          "additionalProperties": true
        }
      },
      "description": "set to execute the tool named by path",
      "additionalProperties": false
    }
  },
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `atomic-tools-atomic-tools`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-order-images-by-original-time`

Put a series of screenshots or photos in chronological order by original capture time.
`images` is a list of {"filename", "content_base64", optional "takeout_sidecar_json"}; use the
original filenames. Returns `ordered` (earliest first: filename, sha256, resolved time, source,
confidence, conflict flag, device) and `unresolved` for files with no recoverable time. Times
without a timezone are device-local wall-clock; mixing them with UTC values is flagged.

id: `1de83ecef20a4dc6b1e8299734dc181b`

enabled: `True`

Exposed by: `advocatio`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "images": {
      "items": {
        "additionalProperties": true,
        "type": "object"
      },
      "type": "array"
    }
  },
  "required": [
    "images"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-order-images-by-original-time`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-ocr-image`

Read the text in a screenshot or photographed document with Tesseract OCR (PNG, JPG, WEBP,
TIFF, BMP, GIF). Returns the full text, each line with its confidence and pixel box, word count
and mean confidence. `layout`: "auto", "block" (one column, e.g. a message thread), or "sparse"
(scattered UI text). OCR text is a machine-read derivative, never equal to a native export.

id: `eb294aac484e4af8b818f0d837324692`

enabled: `True`

Exposed by: `advocatio`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "filename": {
      "type": "string"
    },
    "content_base64": {
      "type": "string"
    },
    "language": {
      "default": "eng",
      "type": "string"
    },
    "layout": {
      "default": "auto",
      "type": "string"
    }
  },
  "required": [
    "filename",
    "content_base64"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-ocr-image`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-read-file-metadata`

Read every metadata field exiftool finds in a file: photos and screenshots (EXIF capture time,
GPS, phone/camera make, model and serial, editing software, XMP edit history), video, audio,
office files, PDFs. `original_time` resolves when the image was originally captured and names its
source (EXIF, Takeout sidecar, embedded creation time, or the device-generated filename), with
every candidate listed and a conflict flag when they disagree. Pass the ORIGINAL filename: names
like Screenshot_20240312-141502.png carry the capture time. `takeout_sidecar_json` is the text of
a Google Takeout `.json` sidecar if one exists. The file is never changed or kept.
Files over 25 MiB go to POST /v1/documents:metadata instead.

id: `9f9d21a6d6ec4f9badb47f8d79fa0c2e`

enabled: `True`

Exposed by: `advocatio`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "filename": {
      "type": "string"
    },
    "content_base64": {
      "type": "string"
    },
    "takeout_sidecar_json": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    }
  },
  "required": [
    "filename",
    "content_base64"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-read-file-metadata`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-scrub-pdf-metadata`

Write a copy of an owner-produced PDF with its document-info dictionary and XMP packet removed,
then re-read it with exiftool. Returns the stored `.scrubbed.pdf` name, sha256, the fields
removed and any authored fields still present. Never use on an evidence original.

id: `20d832f819604958b835b27baaeea236`

enabled: `True`

Exposed by: `advocatio`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "filename": {
      "type": "string"
    },
    "content_base64": {
      "type": "string"
    }
  },
  "required": [
    "filename",
    "content_base64"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-scrub-pdf-metadata`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `advocatio-convert-office-document-to-pdf`

Convert an owner-produced office file (DOCX, DOC, ODT, RTF, TXT, XLSX, ODS, PPTX, ODP) to PDF
with LibreOffice. `filename` keeps its office extension; `content_base64` is the file's bytes.
Returns the stored PDF name, sha256 and size; fetch it from /v1/documents/renders/{output_name}.

id: `f163482a9fb84726b3973d2bedb3c366`

enabled: `True`

Exposed by: `advocatio`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "filename": {
      "type": "string"
    },
    "content_base64": {
      "type": "string"
    }
  },
  "required": [
    "filename",
    "content_base64"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `advocatio-convert-office-document-to-pdf`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docstore-query`

Invoke a discovered operation. Read mode rejects writes; explicit write mode retains operation validation.

id: `a4b67cdd7ecc43dd88cc7fa50878225e`

enabled: `True`

Exposed by: `propria-docstore`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "additionalProperties": false,
  "properties": {
    "operation": {
      "type": "string"
    },
    "arguments": {
      "anyOf": [
        {
          "additionalProperties": true,
          "type": "object"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "mode": {
      "default": "read",
      "enum": [
        "read",
        "write"
      ],
      "type": "string"
    }
  },
  "required": [
    "operation"
  ],
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docstore-query`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docstore-get`

Retrieve a document by returned record ID; preserves status and body.

id: `bc52c18614e9491e883ad09f08ce2c79`

enabled: `True`

Exposed by: `propria-docstore`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "additionalProperties": false,
  "properties": {
    "record_id": {
      "type": "string"
    }
  },
  "required": [
    "record_id"
  ],
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docstore-get`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docstore-search`

Primary documentation search: CocoIndex/NIM vectors searched in SurrealDB; compact uses bounded DuckDB presentation.

id: `893c2daea6864003a315da316baf7c56`

enabled: `True`

Exposed by: `propria-docstore`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "maxLength": 2048,
      "minLength": 2,
      "type": "string"
    },
    "domain": {
      "enum": [
        "probata",
        "proffer",
        "consignatio",
        "advocatio",
        "vestigia",
        "indagatio",
        "intake",
        "workbench",
        "knowledge",
        "memory",
        "infra",
        "docs"
      ],
      "type": "string"
    },
    "limit": {
      "default": 8,
      "maximum": 20,
      "minimum": 1,
      "type": "integer"
    },
    "kind": {
      "default": "doc",
      "enum": [
        "doc",
        "adr",
        "decision",
        "handoff",
        "todo",
        "review",
        "blueprint",
        "reference",
        "infrastructure"
      ],
      "type": "string"
    },
    "status": {
      "default": "all",
      "enum": [
        "active",
        "all"
      ],
      "type": "string"
    },
    "rerank": {
      "default": true,
      "type": "boolean"
    },
    "presentation": {
      "default": "compact",
      "enum": [
        "compact",
        "full"
      ],
      "type": "string"
    }
  },
  "required": [
    "query",
    "domain"
  ],
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docstore-search`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docstore-health`

Read actual health from the configured documentation API.

id: `e8e8ace753bf48539c2bfa20f55732b3`

enabled: `True`

Exposed by: `propria-docstore`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docstore-health`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docstore-capabilities`

Discover capability groups; request one group's operations or one operation's input schema.

id: `c83a58cc9a3f4e8e87ce56bdeb39acd9`

enabled: `True`

Exposed by: `propria-docstore`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "additionalProperties": false,
  "properties": {
    "group": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "operation": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    }
  },
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docstore-capabilities`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-status`

One-call diagnostic: is Octopoda actually working?

Returns mode (local/cloud), backend type, agent count, embedding
availability, dashboard URL if running locally. Designed to answer
'is it wired up?' in a single call so users don't have to probe
individual tools to discover broken plumbing.

id: `a2436387da2843079ff0499653201639`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {},
  "title": "octopoda_statusArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-status`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-search-filtered`

Search memories with combined filters. All filters are AND-combined.

Args:
    agent_id: The agent to search
    query: Semantic search query (optional)
    tags: Comma-separated tags to filter by (optional)
    importance: Filter by importance: "critical", "normal", or "low" (optional)
    max_age_days: Only return memories from the last N days (optional)

id: `426fb39a26784940a613924ba6ba6827`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "query": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Query"
    },
    "tags": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Tags"
    },
    "importance": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Importance"
    },
    "max_age_days": {
      "anyOf": [
        {
          "type": "integer"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Max Age Days"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_search_filteredArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-search-filtered`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-update-progress`

Update progress on an agent's current goal.

Args:
    agent_id: The agent to update
    progress: Overall progress 0.0 to 1.0 (optional)
    milestone_index: Mark a specific milestone as complete (optional)
    note: Progress note to log (optional)

id: `97b6da9771d7475c8dacdb991b03a71b`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "progress": {
      "anyOf": [
        {
          "type": "number"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Progress"
    },
    "milestone_index": {
      "anyOf": [
        {
          "type": "integer"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Milestone Index"
    },
    "note": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Note"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_update_progressArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-update-progress`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-get-goal`

Get the current goal and progress for an agent.

Args:
    agent_id: The agent to check

id: `e9cdf5c8d58143e380b04fdd9fd20662`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_get_goalArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-get-goal`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-set-goal`

Set a goal for an agent with optional milestones. Goals are tracked
persistently and integrate with drift detection.

Args:
    agent_id: The agent to set a goal for
    goal: Description of what the agent should accomplish
    milestones: Optional comma-separated list of milestone descriptions

id: `914aedac07e44d0bb10987885f5ac69d`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "goal": {
      "title": "Goal",
      "type": "string"
    },
    "milestones": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Milestones"
    }
  },
  "required": [
    "agent_id",
    "goal"
  ],
  "title": "octopoda_set_goalArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-set-goal`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-broadcast`

Broadcast a message to all agents. Any agent can read broadcasts.

Args:
    agent_id: The broadcasting agent
    message: Message to broadcast
    message_type: "info", "request", "response", or "alert"

id: `2305b79d878d457780dfbe4fafdf16d4`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "message": {
      "title": "Message",
      "type": "string"
    },
    "message_type": {
      "default": "info",
      "title": "Message Type",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "message"
  ],
  "title": "octopoda_broadcastArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-broadcast`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-read-messages`

Read messages from an agent's inbox.

Args:
    agent_id: The agent whose inbox to read
    unread_only: If true, only return unread messages

id: `d902590ad16e4df6820bcafd769257ce`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "unread_only": {
      "default": false,
      "title": "Unread Only",
      "type": "boolean"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_read_messagesArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-read-messages`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-send-message`

Send a message from one agent to another. Creates an inbox/outbox
system for asynchronous agent-to-agent communication.

Args:
    agent_id: The sending agent
    to_agent: The receiving agent ID
    message: Message content (plain text or JSON string)
    message_type: "info", "request", "response", or "alert"

id: `b3d6357d2a2f4ae8b37fcf6ed0ed4679`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "to_agent": {
      "title": "To Agent",
      "type": "string"
    },
    "message": {
      "title": "Message",
      "type": "string"
    },
    "message_type": {
      "default": "info",
      "title": "Message Type",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "to_agent",
    "message"
  ],
  "title": "octopoda_send_messageArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-send-message`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-loop-history`

Get loop detection alert history for pattern analysis. Shows how
loop behavior has evolved over time, broken down by hour and type.
Automatically detects recurring patterns.

Args:
    agent_id: The agent to analyze
    hours: How many hours of history to analyze (default 24, max 168)

id: `ca870203f3e64bb09038918f04547656`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "hours": {
      "default": 24,
      "title": "Hours",
      "type": "integer"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_loop_historyArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-loop-history`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-loop-status`

Get comprehensive loop detection status for an agent. Combines 5
signals: write similarity, key overwrites, velocity spikes, alert
frequency, and goal drift. Returns severity (green/yellow/orange/red)
with actionable recovery suggestions.

Use this when you suspect an agent is stuck, looping, or behaving
abnormally. The severity score tells you how urgently to intervene.

Args:
    agent_id: The agent to check for loops

id: `5f8270609b3b4990b312790caec5ce48`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_loop_statusArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-loop-status`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-consolidate`

Find and optionally merge duplicate memories. Duplicates degrade
retrieval quality because similar but stale memories surface alongside
current ones.

Args:
    agent_id: The agent whose memories to consolidate
    dry_run: If true (default), reports duplicates without deleting them

id: `219680171d124c29bf6f7491f938e70b`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "dry_run": {
      "default": false,
      "title": "Dry Run",
      "type": "boolean"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_consolidateArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-consolidate`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-memory-health`

Check the health of an agent's memory. Returns a score from 0-100
with actionable recommendations for improving retrieval quality.

Args:
    agent_id: The agent to check

id: `f50d9ee1d1c546b7a17f1eb99292bf77`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_memory_healthArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-memory-health`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-forget-stale`

Forget old memories to keep the agent's knowledge fresh.
Critical memories are always preserved regardless of age.

Args:
    agent_id: The agent to clean up
    max_age_days: Delete memories older than this many days (default 7)

id: `2cc463754a8c42dba89ced1b1dd4381b`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "max_age_days": {
      "default": 7,
      "title": "Max Age Days",
      "type": "integer"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_forget_staleArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-forget-stale`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-forget`

Explicitly forget (delete) a specific memory. Use when a memory is
no longer relevant or correct.

Args:
    agent_id: The agent whose memory to forget
    key: The memory key to delete

id: `2273782f1c854bf9a20ba5725e608051`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "key": {
      "title": "Key",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "key"
  ],
  "title": "octopoda_forgetArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-forget`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-log-decision`

Log an agent decision with full audit trail.

Args:
    agent_id: The agent making the decision
    decision: What was decided ("allow", "deny", or "escalate")
    reasoning: Why this decision was made
    context: Optional context. Accepts either a dict OR a JSON string
             (the MCP framework auto-parses JSON-shaped strings into
             dicts before validation, so the type accepts both).

Note: previously typed `str | None`, which failed validation when the
framework auto-parsed JSON strings into dicts. Reported May 2026 audit.

id: `4947ef76e6854ba48084b76eb37ae7c9`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "decision": {
      "title": "Decision",
      "type": "string"
    },
    "reasoning": {
      "title": "Reasoning",
      "type": "string"
    },
    "context": {
      "anyOf": [
        {
          "additionalProperties": true,
          "type": "object"
        },
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Context"
    }
  },
  "required": [
    "agent_id",
    "decision",
    "reasoning"
  ],
  "title": "octopoda_log_decisionArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-log-decision`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-get-context`

Get relevant context from memory before generating a response.
Returns memories related to the query, ready to inject into your prompt.

Args:
    agent_id: The agent whose memories to search
    query: The current user message or topic
    limit: Max memories to retrieve (default 10)

id: `29a6a9068f7a4ff89a2ebc0422a04ea7`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "query": {
      "title": "Query",
      "type": "string"
    },
    "limit": {
      "default": 10,
      "title": "Limit",
      "type": "integer"
    }
  },
  "required": [
    "agent_id",
    "query"
  ],
  "title": "octopoda_get_contextArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-get-context`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-process-conversation`

Process a conversation turn — automatically extracts and stores memories.
Call this after your agent responds to learn from the conversation.

Args:
    agent_id: The agent to store memories for
    user_message: What the user said
    assistant_response: What the assistant replied

id: `1ddbade23aba42a796ab859a351c3421`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "user_message": {
      "title": "User Message",
      "type": "string"
    },
    "assistant_response": {
      "title": "Assistant Response",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "user_message",
    "assistant_response"
  ],
  "title": "octopoda_process_conversationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-process-conversation`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-agent-stats`

Get performance statistics and analytics for an agent.

Args:
    agent_id: The agent to get stats for

id: `264a943df8f7482ba6c4bff92fc5683c`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_agent_statsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-agent-stats`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-list-agents`

List all registered agents in your Octopoda account.

id: `c452f534ffac453aa8ab779c9974beba`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {},
  "title": "octopoda_list_agentsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-list-agents`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-read-shared`

Read from shared memory written by any agent.

Args:
    agent_id: The agent reading the data
    key: Shared memory key to read
    space: Memory space name (default "global")

id: `12883eb08f95473abc10c0f3b98269d4`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "key": {
      "title": "Key",
      "type": "string"
    },
    "space": {
      "default": "global",
      "title": "Space",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "key"
  ],
  "title": "octopoda_read_sharedArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-read-shared`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-share`

Write to shared memory that other agents can read.

Args:
    agent_id: The agent writing the data
    key: Shared memory key
    value: Data to share (JSON string or plain text)
    space: Memory space name (default "global")

id: `0b497cf9d6aa4737bd383557dc512932`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "key": {
      "title": "Key",
      "type": "string"
    },
    "value": {
      "title": "Value",
      "type": "string"
    },
    "space": {
      "default": "global",
      "title": "Space",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "key",
    "value"
  ],
  "title": "octopoda_shareArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-share`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-restore`

Restore agent memory from a snapshot. Reverts to the saved state.

Args:
    agent_id: The agent whose memory to restore
    label: Snapshot label to restore from (latest if omitted)

id: `6be83907e91644ebab876819ed2221d6`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "label": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Label"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_restoreArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-restore`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-snapshot`

Take a snapshot (checkpoint) of all agent memory. Use before risky operations.

Args:
    agent_id: The agent whose memory to snapshot
    label: Optional label for the snapshot (auto-generated if omitted)

id: `c8d6dedc2e33497ab457d29ae1232628`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "label": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Label"
    }
  },
  "required": [
    "agent_id"
  ],
  "title": "octopoda_snapshotArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-snapshot`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-related`

Query the knowledge graph for an entity and its connections.
Shows what other entities are related and how.

Args:
    agent_id: The agent whose knowledge graph to query
    entity: Entity name to look up (e.g. "London", "Alice")

id: `e08200994dc543d58edd1f57e15ca22a`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "entity": {
      "title": "Entity",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "entity"
  ],
  "title": "octopoda_relatedArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-related`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-recall-history`

Get the full timeline of how a memory changed over time.
Shows all versions with timestamps for when each was valid.

Args:
    agent_id: The agent whose memory to inspect
    key: The memory key to get history for

id: `f2e660fcbbec4db29cff6f314ff4db69`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "key": {
      "title": "Key",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "key"
  ],
  "title": "octopoda_recall_historyArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-recall-history`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-recall-similar`

Search an agent's memories by meaning (semantic similarity).
Finds memories related to the query even if the exact words don't match.

Args:
    agent_id: The agent whose memories to search
    query: Natural language query (e.g. "what food does the user like?")
    limit: Maximum number of results (default 10)

id: `53645e00124b47fd9b3fd919209692e7`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "query": {
      "title": "Query",
      "type": "string"
    },
    "limit": {
      "default": 10,
      "title": "Limit",
      "type": "integer"
    }
  },
  "required": [
    "agent_id",
    "query"
  ],
  "title": "octopoda_recall_similarArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-recall-similar`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-search`

Search an agent's memories by key prefix.

Args:
    agent_id: The agent whose memories to search
    prefix: Key prefix to search for (e.g. "task:" finds "task:current", "task:history")
    limit: Maximum number of results (default 20)

id: `8b29e0b61e824c0e87f34deac5b31095`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "prefix": {
      "title": "Prefix",
      "type": "string"
    },
    "limit": {
      "default": 20,
      "title": "Limit",
      "type": "integer"
    }
  },
  "required": [
    "agent_id",
    "prefix"
  ],
  "title": "octopoda_searchArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-search`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-recall`

Retrieve a stored memory by key.

Args:
    agent_id: The agent whose memory to read
    key: The memory key to retrieve

id: `c15e72a9e297408091a49ad162a8a52d`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "key": {
      "title": "Key",
      "type": "string"
    }
  },
  "required": [
    "agent_id",
    "key"
  ],
  "title": "octopoda_recallArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-recall`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `octopoda-octopoda-remember`

Store a persistent memory for an AI agent. Memory is stored in the cloud and persists across sessions.

Args:
    agent_id: Unique identifier for the agent (e.g. "research_bot", "code_assistant")
    key: Memory key (e.g. "user_preference", "task:current")
    value: Data to store (JSON string or plain text)
    tags: Optional list of tags for categorization

id: `6065c7e5f63a40639a06342e75b11eba`

enabled: `True`

Exposed by: `agent-memory`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "agent_id": {
      "title": "Agent Id",
      "type": "string"
    },
    "key": {
      "title": "Key",
      "type": "string"
    },
    "value": {
      "title": "Value",
      "type": "string"
    },
    "tags": {
      "anyOf": [
        {
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Tags"
    }
  },
  "required": [
    "agent_id",
    "key",
    "value"
  ],
  "title": "octopoda_rememberArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `octopoda-octopoda-remember`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `cloudflare-docs-migrate-pages-to-workers-guide`

ALWAYS read this guide before migrating Pages projects to Workers.

id: `fefa910b7d87463f89296d50eded7de3`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {}
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `cloudflare-docs-migrate-pages-to-workers-guide`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `cloudflare-docs-search-cloudflare-documentation`

Search the Cloudflare documentation.

		This tool should be used to answer any question about Cloudflare products or features, including:
		- Workers, Pages, R2, Images, Stream, D1, Durable Objects, KV, Workflows, Hyperdrive, Queues
		- AI Search, Workers AI, Vectorize, AI Gateway, Browser Run
		- Zero Trust, Access, Tunnel, Gateway, Browser Isolation, WARP, DDOS, Magic Transit, Magic WAN
		- CDN, Cache, DNS, Zaraz, Argo, Rulesets, Terraform, Account and Billing

		Results are returned as semantically similar chunks to the query.

id: `087022052bdd4573bda0917e6ac4b3b8`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "query": {
      "type": "string"
    }
  },
  "required": [
    "query"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `cloudflare-docs-search-cloudflare-documentation`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-docs-sendfeedback`

Report an issue in the documentation of n8n Docs so the team can fix it. Use it whenever, while helping a user, you come across content that is outdated, contradictory, missing information, or otherwise unhelpful. Also use it when the user themselves reports a problem with the docs, even if you could not verify it yourself. If it's your own observation, do a quick sanity check that the issue is real before reporting — no need to exhaustively re-read the page. Send one call per distinct issue and do not report the same issue twice in a conversation. Do not use this tool to confirm that a page is accurate; it is for reporting problems only.

id: `52b9e96b242046e5a78041d2bd1307c0`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "content": {
      "type": "string",
      "minLength": 1,
      "maxLength": 2048,
      "description": "Explain the issue in full, as if writing to a documentation maintainer who never saw this conversation. Describe what is wrong, where on the page it appears (quote the exact sentence or section title when possible), what the user was trying to do, and, when relevant, what the correct or expected information should be. Write a few clear, specific sentences in English. Never include personal or confidential information from the conversation. Up to 2048 characters."
    },
    "pageUrl": {
      "type": "string",
      "description": "The full URL of the page the issue is about (e.g. https://docs.n8n.io//getting-started), so the finding is linked to the exact page."
    },
    "goal": {
      "type": "string",
      "maxLength": 1024,
      "description": "The broader end goal you were ultimately trying to accomplish (as/on behalf of the user) when you hit this issue. Gives the team the context you were working towards. Optional."
    }
  },
  "required": [
    "content",
    "pageUrl"
  ],
  "additionalProperties": false,
  "$schema": "http://json-schema.org/draft-07/schema#"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-docs-sendfeedback`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-docs-getpage`

Fetch the full markdown content of a specific documentation page from n8n Docs. Use this when you have a page URL and want to read its content. Accepts full URLs (e.g. https://docs.n8n.io//getting-started). Since `searchDocumentation` returns partial content, use `getPage` to retrieve the complete page when you need more details. The content includes links you can follow to navigate to related pages.

id: `d3c9dd6f471242b7a4129dfda3de1761`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "description": "The URL of the page to fetch"
    }
  },
  "required": [
    "url"
  ],
  "additionalProperties": false,
  "$schema": "http://json-schema.org/draft-07/schema#"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-docs-getpage`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `n8n-docs-searchdocumentation`

Search across the documentation to find relevant information, code examples, API references, and guides. Use this tool when you need to answer questions about n8n Docs, find specific documentation, understand how features work, or locate implementation details. The search returns contextual content with titles and direct links to the documentation pages.

id: `713bef62368642389fb385e6ba63e33c`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "additionalProperties": false,
  "$schema": "http://json-schema.org/draft-07/schema#"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `n8n-docs-searchdocumentation`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `agno-docs-submit-docs-feedback`

Report a problem with a documentation page so the docs team can fix it.

Use when a page is incorrect, outdated, confusing, incomplete, or has a broken
example — not for product support requests or questions.

Args:
    path: The site path of the page the feedback is about, e.g. `/first-agent`.
    feedback: What is incorrect, outdated, missing, or confusing (5–4000 characters).

id: `3b140138a867454596ef24fdae4ae8e4`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "path": {
      "type": "string",
      "description": "Documentation page path, e.g. `/first-agent`."
    },
    "feedback": {
      "maxLength": 4000,
      "minLength": 5,
      "type": "string",
      "description": "What is incorrect, outdated, missing, or confusing."
    }
  },
  "required": [
    "path",
    "feedback"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `agno-docs-submit-docs-feedback`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `agno-docs-query-docs-filesystem`

Run a read-only shell-like command against a virtual filesystem containing
ONLY the documentation pages as `.md` files. This is NOT a shell on any real
machine — commands are emulated in-process against indexed content.

This is how you read a documentation page: `cat /agents/overview` or
`head -80 /agents/overview.md` (the .md extension is optional). To find exact
keywords, class names, or parameters use `rg`; to see the docs layout use
`tree` or `ls`.

Examples:
- `tree / -L 2` — top-level docs layout
- `rg -il "rate limit" /` — pages mentioning "rate limit"
- `rg -C 3 "output_schema" /agents` — matches with 3 lines of context
- `rg -c -F "Agent(" /examples/agents` — how many matching lines per page (-F: literal)
- `head -80 /first-agent /agents/overview` — read several pages at once

Supported: ls, tree, find, rg/grep (-i -l -c -w -F -C/-A/-B n), cat, head,
tail, wc. No pipes, no writes, no network; every call is stateless and
output is bounded to 30,000 characters plus a continuation notice. Follow
that notice to continue a page read or narrow a search. Incomplete scans
cannot establish that an identifier is absent. Missing paths and invalid
commands are explained in the output. Cite pages by URL (the `url` from
search_docs, or the docs site base plus the path without `.md`).

Args:
    command: One shell-like command, e.g. `rg -il "keyword" /` or `head -80 /path/page`.

id: `b1f41767395c45658c7a79ee3ce3f22e`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "command": {
      "type": "string",
      "description": "One shell-like command, e.g. `rg -il \"keyword\" /` or `head -80 /path/page`."
    }
  },
  "required": [
    "command"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `agno-docs-query-docs-filesystem`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `agno-docs-search-docs`

Search the Agno documentation for relevant content, code examples, and guides.

Returns the best-matching sections (up to 10, at most 3 from any one page) with
the site path and URL of the page each came from. Use natural language for
`query`. When the question is vague or uses words the docs may not, add up to
three `alternatives`: the same question in the docs' vocabulary, e.g. the
likely identifier (`enable_agentic_memory`) and a page-title style wording
("Agent with Memory"). Every phrasing is searched and the results are merged,
so one call is enough. For exact keyword or regex matches, for browsing the
docs structure, or to read a full page, use `query_docs_filesystem` instead.

Result JSON: {"results": [{"path", "url", "title", "content", "confidence", "revision"}, ...]}.
`content` is one section of the page (heading and body, markdown), `title` is
its "Page › Section" breadcrumb. `confidence` is a retrieval relevance score,
not the probability that an answer is correct or a feature exists. Results
are already ranked; verify that their content supports the requested API.
`revision` identifies the retrieved page version.

An empty results list means no sections were returned. If `partial` or
`truncated` is true, coverage is incomplete: inspect `warnings` and
`omitted_count`, narrow the query or read the relevant page. Incomplete or
low-relevance results do not establish that a feature is absent. An `error`
explains a failed search; correct invalid input, or use query_docs_filesystem
with `rg` when search is unavailable.

Args:
    query: Natural-language search query, e.g. "give an agent memory of past chats".
    alternatives: Up to three other phrasings of the same question, in the docs' vocabulary.

id: `a004b82696eb41099e8ee12b7d3bb2fb`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "query": {
      "maxLength": 500,
      "minLength": 1,
      "type": "string",
      "description": "Natural-language question or a documented identifier."
    },
    "alternatives": {
      "anyOf": [
        {
          "items": {
            "maxLength": 500,
            "minLength": 1,
            "type": "string"
          },
          "maxItems": 3,
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "description": "Up to three alternative phrasings of the same question."
    }
  },
  "required": [
    "query"
  ],
  "type": "object",
  "additionalProperties": false
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `agno-docs-search-docs`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `context7-query-docs`

Retrieves and queries up-to-date documentation and code examples from Context7 for any programming library or framework.

You must call 'Resolve Context7 Library ID' tool first to obtain the exact Context7-compatible library ID required to use this tool, UNLESS the user explicitly provides a library ID in the format '/org/project' or '/org/project/version' in their query.

Do not call this tool more than 3 times per question.

id: `91de665aca174cf589492288440ae3e1`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "libraryId": {
      "type": "string",
      "description": "Exact Context7-compatible library ID (e.g., '/mongodb/docs', '/vercel/next.js', '/supabase/supabase', '/vercel/next.js/v14.3.0-canary.87') retrieved from 'resolve-library-id' or directly from user query in the format '/org/project' or '/org/project/version'."
    },
    "query": {
      "type": "string",
      "description": "What to look up in the library's documentation, scoped to a single concept. Be specific and include relevant details, but keep each query to one topic — if the user's question spans multiple distinct concepts, make a separate call per concept instead of combining them, unless the question is about how the concepts interact. Good: 'How to set up authentication with JWT in Express.js' or 'React useEffect cleanup function examples'. Bad (too vague): 'auth' or 'hooks'. Bad (too broad): 'routing and auth and caching in Next.js'. The query is sent to the Context7 API for processing. Do not include any sensitive or confidential information such as API keys, passwords, credentials, personal data, or proprietary code in your query."
    }
  },
  "required": [
    "libraryId",
    "query"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `context7-query-docs`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `context7-resolve-library-id`

Resolves a package/product name to a Context7-compatible library ID and returns matching libraries.

You MUST call this function before 'Query Documentation' tool to obtain a valid Context7-compatible library ID UNLESS the user explicitly provides a library ID in the format '/org/project' or '/org/project/version' in their query.

Each result includes:
- Library ID: Context7-compatible identifier (format: /org/project)
- Name: Library or package name
- Description: Short summary
- Code Snippets: Number of available code examples
- Source Reputation: Authority indicator (High, Medium, Low, or Unknown)
- Benchmark Score: Quality indicator (100 is the highest score)
- Versions: List of versions if available. Use one of those versions if the user provides a version in their query. The format of the version is /org/project/version.

For best results, select libraries based on name match, source reputation, snippet coverage, benchmark score, and relevance to your use case.

Selection Process:
1. Analyze the query to understand what library/package the user is looking for
2. Return the most relevant match based on:
- Name similarity to the query (exact matches prioritized)
- Description relevance to the query's intent
- Documentation coverage (prioritize libraries with higher Code Snippet counts)
- Source reputation (consider libraries with High or Medium reputation more authoritative)
- Benchmark Score: Quality indicator (100 is the highest score)

Response Format:
- Return the selected library ID in a clearly marked section
- Provide a brief explanation for why this library was chosen
- If multiple good matches exist, acknowledge this but proceed with the most relevant one
- If no good matches exist, clearly state this and suggest query refinements

For ambiguous queries, request clarification before proceeding with a best-guess match.

IMPORTANT: Do not call this tool more than 3 times per question. If you cannot find what you need after 3 calls, use the best result you have.

id: `17b331d1084b4d1cadaf6979cc770b66`

enabled: `True`

Exposed by: `dev-docs`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "type": "object",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "query": {
      "type": "string",
      "description": "What to look up in the library's documentation. This is used to rank library results by relevance to what the user is trying to accomplish. The query is sent to the Context7 API for processing. Do not include any sensitive or confidential information such as API keys, passwords, credentials, personal data, or proprietary code in your query."
    },
    "libraryName": {
      "type": "string",
      "description": "Library name to search for and retrieve a Context7-compatible library ID. Use the official library name with proper punctuation — e.g., 'Next.js' instead of 'nextjs', 'Customer.io' instead of 'customerio', 'Three.js' instead of 'threejs'."
    }
  },
  "required": [
    "query",
    "libraryName"
  ]
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `context7-resolve-library-id`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-use`

Switch the active namespace and/or database. At least one of `namespace` or `database` must be provided; both can be set in a single call. Returns the resolved context.

id: `637d0279bf274e8b8b19cdef981f7149`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "database": {
      "description": "Database to switch to. If set without `namespace`, switches DB under\nthe current namespace.",
      "type": [
        "string",
        "null"
      ]
    },
    "namespace": {
      "description": "Namespace to switch to. Either this or `database` must be set.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "title": "UseParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-use`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-upsert`

UPSERT records with CONTENT, MERGE, or PATCH mode. Data is bound as a typed variable. `where_clause` is a SurrealQL expression fragment -- use the `query` tool with $param bindings for dynamic values.

id: `d78410b7c8724c7d83b83957ba9d3c18`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "content_data": {
      "description": "JSON data for CONTENT mode (replaces entire record). Bound as\n$data. Embed typed SurrealDB values via `{\"$ql\": \"<expr>\"}` (e.g.\n`{\"price\": {\"$ql\": \"9.99dec\"}}`)."
    },
    "merge_data": {
      "description": "JSON data for MERGE mode (merges with existing). Bound as $data.\nSame `$ql` escape applies for typed values."
    },
    "patch_data": {
      "description": "JSON patch operations for PATCH mode. Bound as $data."
    },
    "target": {
      "description": "Target table or record (e.g. \"person\" or \"person:john\").",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause (SurrealQL expression fragment). For dynamic\nvalues use `$param` bindings via the raw `query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "UpsertParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-upsert`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-update`

UPDATE existing records with CONTENT, MERGE, or PATCH mode. Data is bound as a typed variable. `where_clause` is a SurrealQL expression fragment -- use the `query` tool with $param bindings for dynamic values.

id: `6089716e312b44e19696752b0815ce03`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "content_data": {
      "description": "JSON data for CONTENT mode. Bound as $data. Embed typed\nSurrealDB values via `{\"$ql\": \"<expr>\"}`."
    },
    "merge_data": {
      "description": "JSON data for MERGE mode. Bound as $data. Same `$ql` escape\napplies."
    },
    "patch_data": {
      "description": "JSON patch operations for PATCH mode. Bound as $data."
    },
    "target": {
      "description": "Target table or record (e.g. \"person\" or \"person:john\").",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause (SurrealQL expression fragment). For dynamic\nvalues use `$param` bindings via the raw `query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "UpdateParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-update`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-select`

SELECT records with optional filtering, sorting, and pagination. `fields`, `where_clause`, `order_clause`, `group_clause`, and `split_clause` are raw SurrealQL expression fragments -- use the `query` tool with $param bindings for dynamic values.

id: `010eadc3e7cf481b9f8629a61e52bd49`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "fetch_clause": {
      "description": "Optional `FETCH` clause: comma-separated SurrealQL expression\nfragment naming record-link fields to hydrate (e.g.\n`customer, items.*.product`). Use this to follow `record<...>`\nreferences in one round-trip rather than issuing follow-up\nqueries.",
      "type": [
        "string",
        "null"
      ]
    },
    "fields": {
      "description": "Optional projection list (SurrealQL expression fragment; defaults\nto `*`). For dynamic values use `$param` bindings via the raw\n`query` tool.",
      "type": [
        "string",
        "null"
      ]
    },
    "group_clause": {
      "description": "Optional `GROUP BY` clause (SurrealQL expression fragment).",
      "type": [
        "string",
        "null"
      ]
    },
    "limit_clause": {
      "description": "Optional `LIMIT` value.",
      "format": "uint64",
      "minimum": 0,
      "type": [
        "integer",
        "null"
      ]
    },
    "order_clause": {
      "description": "Optional `ORDER BY` clause (SurrealQL expression fragment, e.g.\n`name ASC`).",
      "type": [
        "string",
        "null"
      ]
    },
    "split_clause": {
      "description": "Optional `SPLIT` clause (SurrealQL expression fragment).",
      "type": [
        "string",
        "null"
      ]
    },
    "start_clause": {
      "description": "Optional `START` value for pagination.",
      "format": "uint64",
      "minimum": 0,
      "type": [
        "integer",
        "null"
      ]
    },
    "target": {
      "description": "Table or record target (e.g. \"person\", \"person:john\"). A single\nidentifier or record id; comma-separated multi-target selects are\nnot supported here -- use the raw `query` tool for those.",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause as a SurrealQL expression fragment (e.g.\n`age > 18`). For dynamic values use `$param` bindings via the raw\n`query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "SelectParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-select`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-run`

Invoke a SurrealQL function (e.g. `math::sum`, `string::concat`, `fn::my_function`) with typed argument bindings. Arguments are bound natively; the function name is restricted to `identifier(::identifier)*`. Permissions and capabilities are enforced by SurrealDB.

id: `cee19efd5ac24736842b0117a92ee73f`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "args": {
      "description": "Optional argument list. Values are bound with their native types --\nnumbers stay numbers, objects stay objects. Embed typed SurrealDB\nvalues via `{\"$ql\": \"<expr>\"}` -- e.g. pass a record id as\n`{\"$ql\": \"person:alice\"}` or a decimal as `{\"$ql\": \"9.99dec\"}`.",
      "items": true,
      "type": [
        "array",
        "null"
      ]
    },
    "function": {
      "description": "Function name. Examples: `math::sum`, `string::concat`, `fn::my_function`.",
      "type": "string"
    }
  },
  "required": [
    "function"
  ],
  "title": "RunParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-run`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-relate`

RELATE records to create graph edges (from->table->to). Optional content is bound as a typed variable.

id: `6200c939a1954ec6b094278bee019c9d`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "content_data": {
      "description": "Optional JSON content data for the edge record. Bound as $data.\nEmbed typed SurrealDB values via `{\"$ql\": \"<expr>\"}` (e.g.\n`{\"order\": {\"$ql\": \"order:o1\"}, \"quantity\": 2}`)."
    },
    "from": {
      "description": "Source record(s) (e.g. \"person:john\").",
      "type": "string"
    },
    "table": {
      "description": "Edge table name (e.g. \"knows\", \"wrote\").",
      "type": "string"
    },
    "with": {
      "description": "Target record(s) (e.g. \"person:bob\").",
      "type": "string"
    }
  },
  "required": [
    "from",
    "table",
    "with"
  ],
  "title": "RelateParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-relate`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-query`

Execute a SurrealQL query with optional parameterized inputs. Use $param syntax for placeholders and provide bindings in the parameters object.

id: `fd8ed669b55c44568ecd4a93b9a7a554`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "parameters": {
      "description": "Optional JSON object of parameter bindings (e.g. {\"name\": \"John\", \"age\": 30}).\nValues are bound with their native types -- numbers stay numbers, objects stay objects.\nUse `{\"$ql\": \"<surrealql expr>\"}` to embed typed SurrealDB values\nsuch as decimals, datetimes, durations, record ids, or uuids\n(e.g. `{\"price\": {\"$ql\": \"9.99dec\"}, \"user\": {\"$ql\": \"person:alice\"}}`)."
    },
    "query": {
      "description": "The SurrealQL query to execute. Use $param syntax for parameter placeholders.",
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "title": "QueryParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-query`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-list`

Enumerate schema entities of a single kind. `kind` is one of: namespaces, nodes, databases, tables, functions, analyzers, params, apis, buckets, models, modules, sequences, configs, users, accesses, fields, indexes, events. Set `table` for fields/indexes/events. Set `scope` (root|ns|db) for users/accesses.

id: `279a613be63145cf9b3069673c176f0f`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$defs": {
    "ListKind": {
      "description": "Kinds of schema entities that `list` can enumerate.",
      "enum": [
        "namespaces",
        "nodes",
        "databases",
        "tables",
        "functions",
        "analyzers",
        "params",
        "apis",
        "buckets",
        "models",
        "modules",
        "sequences",
        "configs",
        "users",
        "accesses",
        "fields",
        "indexes",
        "events"
      ],
      "type": "string"
    },
    "ListScope": {
      "description": "Scope selector for kinds that exist at multiple levels.",
      "enum": [
        "root",
        "ns",
        "db"
      ],
      "type": "string"
    }
  },
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "kind": {
      "$ref": "#/$defs/ListKind",
      "description": "Kind of entity to enumerate."
    },
    "scope": {
      "anyOf": [
        {
          "$ref": "#/$defs/ListScope"
        },
        {
          "type": "null"
        }
      ],
      "description": "Required when `kind` is one of: users, accesses. One of: root, ns, db."
    },
    "table": {
      "description": "Required when `kind` is one of: fields, indexes, events.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "kind"
  ],
  "title": "ListParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-list`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-insert`

INSERT records into a table. Data is bound as a typed variable. Supports IGNORE and RELATION flags.

id: `aef2513ce7744be9adb64d7cf36869f6`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "data": {
      "description": "JSON array of objects or single object to insert. Bound as $data\nvariable. Use `{\"$ql\": \"<surrealql expr>\"}` anywhere in the tree\nto embed a typed SurrealDB value (decimal, datetime, duration,\nrecord id, uuid, ...)."
    },
    "ignore": {
      "default": false,
      "description": "Whether to ignore duplicate key errors.",
      "type": "boolean"
    },
    "relation": {
      "default": false,
      "description": "Whether this is a relation insert.",
      "type": "boolean"
    },
    "target": {
      "description": "Target table to insert into.",
      "type": "string"
    }
  },
  "required": [
    "target",
    "data"
  ],
  "title": "InsertParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-insert`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-info`

Dump full schema information for a scope. Target: 'root', 'ns', 'db', or a table name. Defaults to the most specific current context. Use `list` when you only need entities of one kind.

id: `2178fa74bd7948a8a5d40e13d14542a2`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "target": {
      "description": "Scope: \"root\", \"ns\", \"db\", or a table name. Defaults to most specific context.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "title": "InfoParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-info`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-graphql`

Execute a GraphQL query or mutation against the active namespace and database. The database must have a `DEFINE CONFIG GRAPHQL` statement. Provide GraphQL variables as a JSON object; returns the GraphQL `{ data, errors }` response envelope (GraphQL execution errors appear in `errors`, not as a tool error).

id: `2d04140a93404932aaa812b5bba8408f`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "operation": {
      "description": "Optional operation name, used to select an operation when the document\ndefines more than one named operation.",
      "type": [
        "string",
        "null"
      ]
    },
    "query": {
      "description": "The GraphQL document to execute (a query or mutation operation).\nSubscriptions, which require a streaming transport, are not supported\nthrough this tool.",
      "type": "string"
    },
    "variables": {
      "description": "Optional JSON object of GraphQL variables (e.g. {\"id\": \"person:tobie\"})."
    }
  },
  "required": [
    "query"
  ],
  "title": "GraphqlParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-graphql`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-gql`

Execute a GQL (ISO/IEC 39075) query with optional parameter bindings, e.g. `MATCH (p:person) RETURN p.name AS name`. GQL is an experimental capability that must be enabled on the server (`--allow-experimental gql`); otherwise the call returns a capability error.

id: `7711885f7bd24046bcc6d5bce08a5edd`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "parameters": {
      "description": "Optional JSON object of parameter bindings (e.g. {\"name\": \"John\"}).\nValues are bound with their native types -- numbers stay numbers,\nobjects stay objects. Use `{\"$ql\": \"<surrealql expr>\"}` to embed typed\nSurrealDB values such as decimals, datetimes, durations, record ids, or\nuuids."
    },
    "query": {
      "description": "The GQL (ISO/IEC 39075) query to execute, e.g.\n`MATCH (p:person) RETURN p.name AS name ORDER BY name`.",
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "title": "GqlParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-gql`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-delete`

DELETE records with an optional WHERE clause. `where_clause` is a SurrealQL expression fragment -- use the `query` tool with $param bindings for dynamic values.

id: `6eedc0f51aa24bc4a9460b04f4517239`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "target": {
      "description": "Target table or record to delete.",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause (SurrealQL expression fragment). For dynamic\nvalues use `$param` bindings via the raw `query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "DeleteParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-delete`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `mem-create`

CREATE a new record with optional content data. Data is bound as a typed variable.

id: `a6ffc89d6c7e4faebc4f11de3631f248`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "data": {
      "description": "JSON data for the record content. Bound as $data variable.\nUse `{\"$ql\": \"<surrealql expr>\"}` anywhere in the tree to embed a\ntyped SurrealDB value (decimal, datetime, duration, record id,\nuuid, ...) -- e.g. `{\"price\": {\"$ql\": \"9.99dec\"}, \"customer\":\n{\"$ql\": \"customer:alice\"}}`."
    },
    "target": {
      "description": "Target table or record ID (e.g. \"person\" or \"person:john\").",
      "type": "string"
    }
  },
  "required": [
    "target"
  ],
  "title": "CreateParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `mem-create`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-use`

Switch the active namespace and/or database. At least one of `namespace` or `database` must be provided; both can be set in a single call. Returns the resolved context.

id: `cc142a6e32074be6a83b29b705bd1e7e`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "database": {
      "description": "Database to switch to. If set without `namespace`, switches DB under\nthe current namespace.",
      "type": [
        "string",
        "null"
      ]
    },
    "namespace": {
      "description": "Namespace to switch to. Either this or `database` must be set.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "title": "UseParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-use`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-upsert`

UPSERT records with CONTENT, MERGE, or PATCH mode. Data is bound as a typed variable. `where_clause` is a SurrealQL expression fragment -- use the `query` tool with $param bindings for dynamic values.

id: `79399b07809e494ba69cfc20c92a7eb8`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "content_data": {
      "description": "JSON data for CONTENT mode (replaces entire record). Bound as\n$data. Embed typed SurrealDB values via `{\"$ql\": \"<expr>\"}` (e.g.\n`{\"price\": {\"$ql\": \"9.99dec\"}}`)."
    },
    "merge_data": {
      "description": "JSON data for MERGE mode (merges with existing). Bound as $data.\nSame `$ql` escape applies for typed values."
    },
    "patch_data": {
      "description": "JSON patch operations for PATCH mode. Bound as $data."
    },
    "target": {
      "description": "Target table or record (e.g. \"person\" or \"person:john\").",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause (SurrealQL expression fragment). For dynamic\nvalues use `$param` bindings via the raw `query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "UpsertParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-upsert`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-update`

UPDATE existing records with CONTENT, MERGE, or PATCH mode. Data is bound as a typed variable. `where_clause` is a SurrealQL expression fragment -- use the `query` tool with $param bindings for dynamic values.

id: `be63e353e3064412a49f4cb0c1b88433`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "content_data": {
      "description": "JSON data for CONTENT mode. Bound as $data. Embed typed\nSurrealDB values via `{\"$ql\": \"<expr>\"}`."
    },
    "merge_data": {
      "description": "JSON data for MERGE mode. Bound as $data. Same `$ql` escape\napplies."
    },
    "patch_data": {
      "description": "JSON patch operations for PATCH mode. Bound as $data."
    },
    "target": {
      "description": "Target table or record (e.g. \"person\" or \"person:john\").",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause (SurrealQL expression fragment). For dynamic\nvalues use `$param` bindings via the raw `query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "UpdateParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-update`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-select`

SELECT records with optional filtering, sorting, and pagination. `fields`, `where_clause`, `order_clause`, `group_clause`, and `split_clause` are raw SurrealQL expression fragments -- use the `query` tool with $param bindings for dynamic values.

id: `d7b98dade699400099090f53e7ee0a3a`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "fetch_clause": {
      "description": "Optional `FETCH` clause: comma-separated SurrealQL expression\nfragment naming record-link fields to hydrate (e.g.\n`customer, items.*.product`). Use this to follow `record<...>`\nreferences in one round-trip rather than issuing follow-up\nqueries.",
      "type": [
        "string",
        "null"
      ]
    },
    "fields": {
      "description": "Optional projection list (SurrealQL expression fragment; defaults\nto `*`). For dynamic values use `$param` bindings via the raw\n`query` tool.",
      "type": [
        "string",
        "null"
      ]
    },
    "group_clause": {
      "description": "Optional `GROUP BY` clause (SurrealQL expression fragment).",
      "type": [
        "string",
        "null"
      ]
    },
    "limit_clause": {
      "description": "Optional `LIMIT` value.",
      "format": "uint64",
      "minimum": 0,
      "type": [
        "integer",
        "null"
      ]
    },
    "order_clause": {
      "description": "Optional `ORDER BY` clause (SurrealQL expression fragment, e.g.\n`name ASC`).",
      "type": [
        "string",
        "null"
      ]
    },
    "split_clause": {
      "description": "Optional `SPLIT` clause (SurrealQL expression fragment).",
      "type": [
        "string",
        "null"
      ]
    },
    "start_clause": {
      "description": "Optional `START` value for pagination.",
      "format": "uint64",
      "minimum": 0,
      "type": [
        "integer",
        "null"
      ]
    },
    "target": {
      "description": "Table or record target (e.g. \"person\", \"person:john\"). A single\nidentifier or record id; comma-separated multi-target selects are\nnot supported here -- use the raw `query` tool for those.",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause as a SurrealQL expression fragment (e.g.\n`age > 18`). For dynamic values use `$param` bindings via the raw\n`query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "SelectParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-select`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-run`

Invoke a SurrealQL function (e.g. `math::sum`, `string::concat`, `fn::my_function`) with typed argument bindings. Arguments are bound natively; the function name is restricted to `identifier(::identifier)*`. Permissions and capabilities are enforced by SurrealDB.

id: `c7fc6dde621149aabdc9277d039cb375`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "args": {
      "description": "Optional argument list. Values are bound with their native types --\nnumbers stay numbers, objects stay objects. Embed typed SurrealDB\nvalues via `{\"$ql\": \"<expr>\"}` -- e.g. pass a record id as\n`{\"$ql\": \"person:alice\"}` or a decimal as `{\"$ql\": \"9.99dec\"}`.",
      "items": true,
      "type": [
        "array",
        "null"
      ]
    },
    "function": {
      "description": "Function name. Examples: `math::sum`, `string::concat`, `fn::my_function`.",
      "type": "string"
    }
  },
  "required": [
    "function"
  ],
  "title": "RunParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-run`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-relate`

RELATE records to create graph edges (from->table->to). Optional content is bound as a typed variable.

id: `1ffce596960b4377b2e94c436895e407`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "content_data": {
      "description": "Optional JSON content data for the edge record. Bound as $data.\nEmbed typed SurrealDB values via `{\"$ql\": \"<expr>\"}` (e.g.\n`{\"order\": {\"$ql\": \"order:o1\"}, \"quantity\": 2}`)."
    },
    "from": {
      "description": "Source record(s) (e.g. \"person:john\").",
      "type": "string"
    },
    "table": {
      "description": "Edge table name (e.g. \"knows\", \"wrote\").",
      "type": "string"
    },
    "with": {
      "description": "Target record(s) (e.g. \"person:bob\").",
      "type": "string"
    }
  },
  "required": [
    "from",
    "table",
    "with"
  ],
  "title": "RelateParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-relate`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-query`

Execute a SurrealQL query with optional parameterized inputs. Use $param syntax for placeholders and provide bindings in the parameters object.

id: `5df6375672674c5a89c213f7752eaea9`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "parameters": {
      "description": "Optional JSON object of parameter bindings (e.g. {\"name\": \"John\", \"age\": 30}).\nValues are bound with their native types -- numbers stay numbers, objects stay objects.\nUse `{\"$ql\": \"<surrealql expr>\"}` to embed typed SurrealDB values\nsuch as decimals, datetimes, durations, record ids, or uuids\n(e.g. `{\"price\": {\"$ql\": \"9.99dec\"}, \"user\": {\"$ql\": \"person:alice\"}}`)."
    },
    "query": {
      "description": "The SurrealQL query to execute. Use $param syntax for parameter placeholders.",
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "title": "QueryParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-query`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-list`

Enumerate schema entities of a single kind. `kind` is one of: namespaces, nodes, databases, tables, functions, analyzers, params, apis, buckets, models, modules, sequences, configs, users, accesses, fields, indexes, events. Set `table` for fields/indexes/events. Set `scope` (root|ns|db) for users/accesses.

id: `9c619f2179664290962a7a785e2bf149`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$defs": {
    "ListKind": {
      "description": "Kinds of schema entities that `list` can enumerate.",
      "enum": [
        "namespaces",
        "nodes",
        "databases",
        "tables",
        "functions",
        "analyzers",
        "params",
        "apis",
        "buckets",
        "models",
        "modules",
        "sequences",
        "configs",
        "users",
        "accesses",
        "fields",
        "indexes",
        "events"
      ],
      "type": "string"
    },
    "ListScope": {
      "description": "Scope selector for kinds that exist at multiple levels.",
      "enum": [
        "root",
        "ns",
        "db"
      ],
      "type": "string"
    }
  },
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "kind": {
      "$ref": "#/$defs/ListKind",
      "description": "Kind of entity to enumerate."
    },
    "scope": {
      "anyOf": [
        {
          "$ref": "#/$defs/ListScope"
        },
        {
          "type": "null"
        }
      ],
      "description": "Required when `kind` is one of: users, accesses. One of: root, ns, db."
    },
    "table": {
      "description": "Required when `kind` is one of: fields, indexes, events.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "kind"
  ],
  "title": "ListParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-list`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-insert`

INSERT records into a table. Data is bound as a typed variable. Supports IGNORE and RELATION flags.

id: `652c29d2870444b38b044641a1241749`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "data": {
      "description": "JSON array of objects or single object to insert. Bound as $data\nvariable. Use `{\"$ql\": \"<surrealql expr>\"}` anywhere in the tree\nto embed a typed SurrealDB value (decimal, datetime, duration,\nrecord id, uuid, ...)."
    },
    "ignore": {
      "default": false,
      "description": "Whether to ignore duplicate key errors.",
      "type": "boolean"
    },
    "relation": {
      "default": false,
      "description": "Whether this is a relation insert.",
      "type": "boolean"
    },
    "target": {
      "description": "Target table to insert into.",
      "type": "string"
    }
  },
  "required": [
    "target",
    "data"
  ],
  "title": "InsertParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-insert`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-info`

Dump full schema information for a scope. Target: 'root', 'ns', 'db', or a table name. Defaults to the most specific current context. Use `list` when you only need entities of one kind.

id: `450b3dd255944a1296960ad6cc98feb4`

enabled: `True`

Exposed by: `surrealdb`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "target": {
      "description": "Scope: \"root\", \"ns\", \"db\", or a table name. Defaults to most specific context.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "title": "InfoParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-info`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-graphql`

Execute a GraphQL query or mutation against the active namespace and database. The database must have a `DEFINE CONFIG GRAPHQL` statement. Provide GraphQL variables as a JSON object; returns the GraphQL `{ data, errors }` response envelope (GraphQL execution errors appear in `errors`, not as a tool error).

id: `2642fd6d6c2549b397ff6b5b2a9af6e5`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "operation": {
      "description": "Optional operation name, used to select an operation when the document\ndefines more than one named operation.",
      "type": [
        "string",
        "null"
      ]
    },
    "query": {
      "description": "The GraphQL document to execute (a query or mutation operation).\nSubscriptions, which require a streaming transport, are not supported\nthrough this tool.",
      "type": "string"
    },
    "variables": {
      "description": "Optional JSON object of GraphQL variables (e.g. {\"id\": \"person:tobie\"})."
    }
  },
  "required": [
    "query"
  ],
  "title": "GraphqlParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-graphql`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-gql`

Execute a GQL (ISO/IEC 39075) query with optional parameter bindings, e.g. `MATCH (p:person) RETURN p.name AS name`. GQL is an experimental capability that must be enabled on the server (`--allow-experimental gql`); otherwise the call returns a capability error.

id: `c110f97ec4ff40e8a5789d447d7cc0da`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "parameters": {
      "description": "Optional JSON object of parameter bindings (e.g. {\"name\": \"John\"}).\nValues are bound with their native types -- numbers stay numbers,\nobjects stay objects. Use `{\"$ql\": \"<surrealql expr>\"}` to embed typed\nSurrealDB values such as decimals, datetimes, durations, record ids, or\nuuids."
    },
    "query": {
      "description": "The GQL (ISO/IEC 39075) query to execute, e.g.\n`MATCH (p:person) RETURN p.name AS name ORDER BY name`.",
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "title": "GqlParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-gql`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-delete`

DELETE records with an optional WHERE clause. `where_clause` is a SurrealQL expression fragment -- use the `query` tool with $param bindings for dynamic values.

id: `8ee57a247e634f3ca74185c206c2f4de`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "target": {
      "description": "Target table or record to delete.",
      "type": "string"
    },
    "where_clause": {
      "description": "Optional `WHERE` clause (SurrealQL expression fragment). For dynamic\nvalues use `$param` bindings via the raw `query` tool.",
      "type": [
        "string",
        "null"
      ]
    }
  },
  "required": [
    "target"
  ],
  "title": "DeleteParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-delete`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `docs-create`

CREATE a new record with optional content data. Data is bound as a typed variable.

id: `750462ba2c7a43b99fea293007bb8e5a`

enabled: `True`

Exposed by: no virtual-server association returned; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "data": {
      "description": "JSON data for the record content. Bound as $data variable.\nUse `{\"$ql\": \"<surrealql expr>\"}` anywhere in the tree to embed a\ntyped SurrealDB value (decimal, datetime, duration, record id,\nuuid, ...) -- e.g. `{\"price\": {\"$ql\": \"9.99dec\"}, \"customer\":\n{\"$ql\": \"customer:alice\"}}`."
    },
    "target": {
      "description": "Target table or record ID (e.g. \"person\" or \"person:john\").",
      "type": "string"
    }
  },
  "required": [
    "target"
  ],
  "title": "CreateParams",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `docs-create`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-application-logs`

Get recent runtime container logs for an application by UUID.

    This is CONTAINER STDOUT/STDERR (GET /applications/{uuid}/logs), distinct
    from deployment build logs (get_deployment). Use this for "why is my app
    crashing at runtime"; use get_deployment for "why did the build fail."

    Parameters
    ----------
    uuid : str
    lines : int  # number of lines from the end of the log (API default 100)

    Requires
    --------
    Token needs `read:sensitive` permission — logs may contain secrets.

id: `e289b6e3365548629aebb2849dfc523b`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "lines": {
      "default": 100,
      "title": "Lines",
      "type": "integer"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "get_application_logsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-application-logs`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-deployments-for-app`

List past/current deployments for one application by app UUID.

    GET /deployments/applications/{uuid} — separate from GET /deployments
    (which only lists deployments currently queued/in-progress across the
    whole team, not scoped to one app).

    Parameters
    ----------
    uuid : str   # application UUID (not deployment_uuid)
    skip : int   # pagination offset (default 0)
    take : int   # page size (default 10)
    full : bool  # return whole records, including the build logs

    Summary by default: each record carries the entire build log, so ten deployments came back
    at 2.5 MB. Use get_deployment(deployment_uuid) for one deployment's logs.

id: `103e3363f9ca43b8a12ad5b90f1318ae`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "skip": {
      "default": 0,
      "title": "Skip",
      "type": "integer"
    },
    "take": {
      "default": 10,
      "title": "Take",
      "type": "integer"
    },
    "full": {
      "default": false,
      "title": "Full",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "list_deployments_for_appArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-deployments-for-app`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-deployment`

Get one deployment's status and logs by deployment UUID, with errors pre-extracted.

    The raw `logs` field from the API is a JSON-encoded STRING of an array of
    log-line objects (not a plain string, not pre-parsed JSON) — this tool
    parses it and surfaces the tail + any error-looking lines so you don't
    have to re-parse it yourself every call.

    Parameters
    ----------
    deployment_uuid : str  # from deploy_application / restart_application /
                            # start_application response, or list_deployments_for_app

    Requires
    --------
    The API token needs `read:sensitive` permission to see the `logs` field —
    without it, Coolify strips logs from the response (removeSensitiveData()).

    Returns
    -------
    dict
        {
          deployment_uuid, status, application_id, server_id,
          log_line_count: int,
          last_lines: list[str],       # tail of the log (up to 40 lines)
          error_lines: list[str],      # lines containing error/fail/exception (case-insens.)
          raw_logs_available: bool,    # False if token lacks read:sensitive
        }

id: `76353fb35f2d4da4bc8b30a5f246f095`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "deployment_uuid": {
      "title": "Deployment Uuid",
      "type": "string"
    }
  },
  "required": [
    "deployment_uuid"
  ],
  "title": "get_deploymentArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-deployment`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-start-application`

Start a stopped application by UUID (deploys and starts containers).

    Pre-deploy guard (2026-10-02)
    -----------------------------
    If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
    report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
    output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
    A guarded call returns {"guard": {...}, "coolify": }; an unguarded call returns
    Coolify's answer unchanged.

    Parameters
    ----------
    uuid : str
    force : bool           # force rebuild
    instant_deploy : bool  # skip the deploy queue

    Do / Don't
    ----------
    Don't: use `docker start ` directly on a Coolify-managed
    container — Coolify tracks desired state itself; starting outside the API
    creates an orphan container Coolify doesn't know is running (verified
    local lesson — leads to port conflicts and duplicate containers on the
    next Coolify-triggered deploy). Always start/stop through this tool or
    the Coolify UI.

    Returns
    -------
    dict — {"message": "...", "deployment_uuid": "..."} (may be None if the
    app was already running).

id: `e756795e32e445ebb50eef1538b89907`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "force": {
      "default": false,
      "title": "Force",
      "type": "boolean"
    },
    "instant_deploy": {
      "default": false,
      "title": "Instant Deploy",
      "type": "boolean"
    },
    "check_only": {
      "default": false,
      "title": "Check Only",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "start_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-start-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-stop-application`

Stop a running application by UUID.

    Pre-deploy guard (2026-10-02)
    -----------------------------
    If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
    report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
    output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
    A guarded call returns {"guard": {...}, "coolify": }; an unguarded call returns
    Coolify's answer unchanged.

    Parameters
    ----------
    uuid : str
    docker_cleanup : bool  # prune networks/volumes after stop (API default True)

    Do / Don't
    ----------
    Do: check_port_collision first if you're stopping one app to free a port
        for another (this IS the "stop the old app" step in that workflow).
    Don't: docker-stop the container manually outside Coolify — Coolify owns
        the container lifecycle and will fight a manually-stopped container
        on its next reconciliation pass, leaving it in a confused state.

    Returns
    -------
    dict — {"message": "Application stopping request queued."}

id: `b91afd41f9b547bea3657be4c5ab7250`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "docker_cleanup": {
      "default": true,
      "title": "Docker Cleanup",
      "type": "boolean"
    },
    "check_only": {
      "default": false,
      "title": "Check Only",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "stop_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-stop-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-restart-application`

Restart a running application by UUID (stop + start; rebuilds from current image).

    Pre-deploy guard (2026-10-02)
    -----------------------------
    If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
    report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
    output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
    A guarded call returns {"guard": {...}, "coolify": }; an unguarded call returns
    Coolify's answer unchanged.

    Do / Don't
    ----------
    Do: use this for a quick container bounce (e.g. picking up a restarted
        dependency) when you do NOT need a fresh build.
    Don't: use this expecting new env values to apply from a compose baked at
        an earlier deploy — use deploy_application for that.

    Returns
    -------
    dict — {"message": "Restart request queued.", "deployment_uuid": "..."}

id: `8dd000c49488443192a64027a1d7cd0d`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "check_only": {
      "default": false,
      "title": "Check Only",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "restart_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-restart-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-deploy-application`

Trigger a deployment for an application (or database/service) by UUID.

    Pre-deploy guard (2026-10-02)
    -----------------------------
    If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
    report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
    output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
    A guarded call returns {"guard": {...}, "coolify": }; an unguarded call returns
    Coolify's answer unchanged.

    Uses POST /deploy?uuid=&force= (Coolify 4.3 answers 405 to GET; a JSON body {"uuid","force"} is also
    accepted by the API; GET+query is used here since it needs no body).

    Verification (owner ruling 2026-09-08)
    --------------------------------------
    A finished deployment is NOT a working app. After the deployment reaches
    `finished`, load the user-facing URL/port (expect 200, or 401 when auth is
    on), confirm the container is `healthy`, and for tailscale sidecars hit the
    served hostname. Report "deployed, not yet verified" until that is done.
    Also: relative binds of repo files in the compose become EMPTY DIRECTORIES
    (Coolify keeps no checkout beside the rendered compose) — use absolute host
    paths for mounted config files.

    When to use
    -----------
    - After upsert_application_envs, to bake new env values into the running
      container (Coolify renders env into compose at deploy time — env
      changes do NOT reach the container until a redeploy).
    - After changing git_branch/docker_compose_location via the Coolify UI.
    - Redeploying the current commit (force=False reuses cache where possible;
      force=True does a clean rebuild).

    Parameters
    ----------
    uuid : str    # application (or db/service) UUID
    force : bool  # force rebuild without cache (default False)

    Returns
    -------
    dict — {"message": "...", "deployment_uuid": "..."}. Pass deployment_uuid
    to get_deployment to poll status/logs.

id: `699df86735f342edb1b945be0ccdb473`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "force": {
      "default": false,
      "title": "Force",
      "type": "boolean"
    },
    "check_only": {
      "default": false,
      "title": "Check Only",
      "type": "boolean"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "deploy_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-deploy-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-check-port-collision`

Report which applications bind a given host port (read-only, safe).

    Primary verification tool before a cutover deploy. Parses each
    application's `docker_compose_raw` (the reliable source for compose apps,
    since `ports_mappings` is None for compose apps) and `ports_mappings`
    (for dockerfile apps) to find host-port bindings.

    Parameters
    ----------
    port : int          # the host port to check, e.g. 5432
    server_uuid : str, optional  # if given, restrict to apps on that server.
        NOTE: Coolify's /applications list does not expose a clean server_uuid
        field, so server filtering is best-effort via the app's network/destination.
        When in doubt, leave None and inspect the returned app names.

    When to use
    -----------
    - Before deploying a new app that binds a known port — confirm nothing
      already holds it.
    - Before a split cutover — confirm the bundled app still holds the port
      (so you know to stop it before starting the new app).

    Returns
    -------
    dict
        {
          port: int,
          collides: bool,
          apps: [{uuid, name, status, ports: [int]}],  # apps that bind this port
          next_actions: list[str],
        }

id: `714b8ee122aa462ebb0a2cbc1d97baeb`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "port": {
      "title": "Port",
      "type": "integer"
    },
    "server_uuid": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Server Uuid"
    }
  },
  "required": [
    "port"
  ],
  "title": "check_port_collisionArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-check-port-collision`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-delete-application`

Delete an application by UUID. DESTRUCTIVE — requires confirm_name to match.

    The confirm_name guard prevents accidental deletion when an agent passes a
    stale or wrong uuid. The name must match the application's current name.

    Do / Don't
    ----------
    Do: pass confirm_name = the app's exact current name (from get_application).
    Do: snapshot any needed config/env FIRST — delete is irreversible.
    Don't: pass a uuid you haven't just re-read in this session.
    Don't: ever pass a wildcard or empty string for either argument.

id: `f193d19d6753465da44f909de9a57b81`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    },
    "confirm_name": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Confirm Name"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "delete_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-delete-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-upsert-application-envs`

Set/update environment variables on an application (bulk upsert).

    Tries PATCH /applications/{uuid}/envs/bulk first; falls back to per-key
    POST /applications/{uuid}/envs if the bulk endpoint is unavailable.

    Parameters
    ----------
    application_uuid : str
    envs : dict[str, str]   # {KEY: "value", ...} — keys not present are left unchanged
    is_runtime : bool       # available at runtime (default True)
    is_buildtime : bool     # available at build time (default True)

    Do / Don't
    ----------
    Do: group all env for one app into a single call (bulk path is one round-trip).
    Do: remember Coolify bakes env VALUES into the rendered compose at deploy —
        changing env requires a redeploy of the app for it to take effect.
    Don't: put the token or other secrets here as literal values you then log —
        this tool never logs values, but the agent's transcript might.

    Returns
    -------
    dict — {method: "bulk"|"per-key", updated: int, failed: list}

id: `d83b61b5feb34b9f847053feaf6a08ba`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "application_uuid": {
      "title": "Application Uuid",
      "type": "string"
    },
    "envs": {
      "additionalProperties": true,
      "title": "Envs",
      "type": "object"
    },
    "is_runtime": {
      "default": true,
      "title": "Is Runtime",
      "type": "boolean"
    },
    "is_buildtime": {
      "default": true,
      "title": "Is Buildtime",
      "type": "boolean"
    }
  },
  "required": [
    "application_uuid",
    "envs"
  ],
  "title": "upsert_application_envsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-upsert-application-envs`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-create-application`

Create a new Coolify application (default: docker-compose from a git repo).

    Mirrors the proven data-tier pattern: repo Cursedpotential/mcp-platform-agno-mcp,
    branch main, github_app_id 2, compose location "/".

    When to use
    -----------
    - Splitting a bundled compose app into independent Coolify apps.
    - Adding a new service from a git-hosted compose file.
    NOT for: standalone databases (use Coolify UI) or importing images.

    Parameters
    ----------
    project_uuid : str  # from list_projects
    server_uuid : str   # from list_servers
    name : str          # human label, also becomes the default fqdn slug
    git_repository : str
        Meaning depends on `type`:
          dockercompose / public -> "owner/repo" (or full URL for public)
          dockerfile             -> the Dockerfile path or inline content
          dockerimage            -> the image reference, e.g. "nginx:latest"
    git_branch : str
    docker_compose_location : str  # path within repo to compose.yaml; "/" = root
    github_app_id : int  # Coolify GitHub App source id (2 on this instance)
    type : str
        Build type. One of:
          "dockercompose" (default) — private GitHub App + compose file
          "public"                  — public git repo, nixpacks build
          "dockerfile"              — build from a Dockerfile
          "dockerimage"             — deploy a prebuilt image, no build
    description : str

    Do / Don't
    ----------
    Do: confirm project_uuid + server_uuid via get_infrastructure_overview first.
    Do: run check_port_collision after create, before starting, to avoid binding wars.
    Don't: pass a placeholder repo — Coolify will accept the create but the deploy will fail.

    Returns
    -------
    dict — the created application record (uuid, name, status). Use the uuid
    with upsert_application_envs to set runtime env, then trigger a deploy.
    Auto-deploy is switched off right after creation (owner rule: deploys are explicit);
    `is_auto_deploy_enabled` in the result is the read-back.

id: `5a55b168a3bf4242bac4d6843d2e27c8`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "project_uuid": {
      "title": "Project Uuid",
      "type": "string"
    },
    "server_uuid": {
      "title": "Server Uuid",
      "type": "string"
    },
    "name": {
      "title": "Name",
      "type": "string"
    },
    "git_repository": {
      "title": "Git Repository",
      "type": "string"
    },
    "git_branch": {
      "title": "Git Branch",
      "type": "string"
    },
    "docker_compose_location": {
      "default": "/",
      "title": "Docker Compose Location",
      "type": "string"
    },
    "github_app_id": {
      "default": 2,
      "title": "Github App Id",
      "type": "integer"
    },
    "type": {
      "default": "dockercompose",
      "title": "Type",
      "type": "string"
    },
    "description": {
      "default": "",
      "title": "Description",
      "type": "string"
    }
  },
  "required": [
    "project_uuid",
    "server_uuid",
    "name",
    "git_repository",
    "git_branch"
  ],
  "title": "create_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-create-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-service`

Full details for one service (multi-container stack) by UUID. Secret values are redacted.

id: `5526381b88cc4717b256ec579fa5c2fa`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "get_serviceArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-service`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-services`

List services (multi-container stacks; summary: uuid, name, status).

    The raw record carries docker_compose_raw with every environment value in plain text, so the
    summary is the default. full=True returns the whole record, redacted.

id: `511bd8e2ca2c436e92a885546c9f867d`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "page": {
      "default": 1,
      "title": "Page",
      "type": "integer"
    },
    "per_page": {
      "default": 50,
      "title": "Per Page",
      "type": "integer"
    },
    "full": {
      "default": false,
      "title": "Full",
      "type": "boolean"
    }
  },
  "title": "list_servicesArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-services`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-database`

Full details for one standalone database by UUID.

id: `4e6884596d4043b0b900b0e5d662cbc2`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "get_databaseArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-database`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-databases`

List standalone databases (summary: uuid, name, status, type). full=True for whole records.

id: `2854c41649664b1c9cad688a5e411cf1`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "page": {
      "default": 1,
      "title": "Page",
      "type": "integer"
    },
    "per_page": {
      "default": 50,
      "title": "Per Page",
      "type": "integer"
    },
    "full": {
      "default": false,
      "title": "Full",
      "type": "boolean"
    }
  },
  "title": "list_databasesArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-databases`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-application`

Full details for one application by UUID (93-field record: status, env, compose, ports, health...).

    Do / Don't
    ----------
    Do: pass the exact uuid from list_applications.
    Don't: guess or wildcard — there is no fuzzy match; a wrong uuid 404s.

id: `ec2bcabba3924a6e9882ab8752254c1a`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "get_applicationArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-application`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-applications`

List applications (summary: uuid, name, status, fqdn, git_repository).

    Parameters
    ----------
    tag : str, optional
        Filter by tag name.
    full : bool
        Return whole records (redacted) instead of the summary.

id: `6b607e27791c4a3e8230663504a73428`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "page": {
      "default": 1,
      "title": "Page",
      "type": "integer"
    },
    "per_page": {
      "default": 50,
      "title": "Per Page",
      "type": "integer"
    },
    "tag": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null,
      "title": "Tag"
    },
    "full": {
      "default": false,
      "title": "Full",
      "type": "boolean"
    }
  },
  "title": "list_applicationsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-applications`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-projects`

List projects (summary: uuid, name, description). full=True returns whole records.

id: `4c3c690569304c92b71d7750f5859feb`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "page": {
      "default": 1,
      "title": "Page",
      "type": "integer"
    },
    "per_page": {
      "default": 50,
      "title": "Per Page",
      "type": "integer"
    },
    "full": {
      "default": false,
      "title": "Full",
      "type": "boolean"
    }
  },
  "title": "list_projectsArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-projects`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-server`

Full details for one server by UUID.

    Parameters
    ----------
    uuid : str
        Server UUID. Get it from list_servers or get_infrastructure_overview.

id: `775801014fa94e2c8fe76ed84ef1a9f4`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "uuid": {
      "title": "Uuid",
      "type": "string"
    }
  },
  "required": [
    "uuid"
  ],
  "title": "get_serverArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-server`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-list-servers`

List servers owned by the authenticated team. Returns summary (uuid, name, ip, is_reachable).

    Discovery
    ---------
    Use get_infrastructure_overview for a one-shot fleet map, or this for
    paginated server listing. Pass the returned uuid to get_server for details.

id: `15015536d6d942ab8f214b9b62896d61`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {
    "page": {
      "default": 1,
      "title": "Page",
      "type": "integer"
    },
    "per_page": {
      "default": 50,
      "title": "Per Page",
      "type": "integer"
    }
  },
  "title": "list_serversArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-list-servers`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

## `coolify-write-get-infrastructure-overview`

Coolify version, all servers, projects with resource counts, and aggregates.

    Discovery
    ---------
    Start here. Single call returns the whole topology — servers, projects,
    and counts of applications/databases/services — so you can pick the right
    UUID before calling the per-resource tools.

    When to use
    -----------
    - First call in a session to map the fleet.
    - Before any write op, to confirm the target server/project/uuid exists.
    - NOT for: drilling into one resource (use get_application/get_server/etc).

    Returns
    -------
    dict
        {
          coolify_version: str,
          servers: list[dict],          # name, uuid, ip, is_reachable
          projects: list[dict],         # name, uuid (per-project app count
                                        # needs an /environments call — drill via
                                        # list_applications for per-app detail)
          counts: {applications, databases, services, servers},
        }

id: `686b323426d347aba09239943cc8a8b8`

enabled: `True`

Exposed by: `coolify-write`; see [[Code/wiki/mcp-servers]].

Input contract:

```json
{
  "properties": {},
  "title": "get_infrastructure_overviewArguments",
  "type": "object"
}
```

Source: `assets/tool-inventory.json` → `live_registry.contextforge_tools` → `coolify-write-get-infrastructure-overview`; registry adapter [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`.

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
