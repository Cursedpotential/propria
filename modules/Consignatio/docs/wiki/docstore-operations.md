---
title: "Docstore operations"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# Docstore operations

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Docstore exposes five MCP entry points and discovers the following operations through `docstore_capabilities`.

Read each operation schema before calling it. Invoke with `docstore_query`, supplying `operation`, `arguments`, and `mode`. Read mode rejects mutations; write mode retains governance checks.

```json
{"operation":"docstore_revision_state","arguments":{"document_key":"document:casebible_catalog_guide_20261004"},"mode":"read"}
```

Discovery evidence: [docstore-operation-schemas.json:1](<_worktrees/custom-tools-wiki-20261004/modules/Consignatio/docs/wiki/assets/docstore-operation-schemas.json:1>) · SHA-256 `614e71a9f3e8c26cabb2d8c14463b8f80457049850f710bea05f88deccd8272d`

## `docstore_adr`

Manage authoritative ADRs, version-checked edits, legacy imports and generated projections.

Read only: `False`. Capability group: `adr`.

```json
{
  "additionalProperties": false,
  "properties": {
    "action": {
      "enum": [
        "list",
        "create",
        "update",
        "migration-plan",
        "migration-apply",
        "projections",
        "verify"
      ],
      "type": "string"
    },
    "payload": {
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
    }
  },
  "required": [
    "action"
  ],
  "type": "object"
}
```

## `docstore_diagnostics`

Report release, processing mode and endpoint identity. Does not start indexing.

Read only: `True`. Capability group: `diagnostics`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_upgrade_apply`

Apply the checksum-bound plan transactionally; no index run or legacy deletion.

Read only: `False`. Capability group: `diagnostics`.

```json
{
  "additionalProperties": false,
  "properties": {
    "plan_id": {
      "type": "string"
    }
  },
  "required": [
    "plan_id"
  ],
  "type": "object"
}
```

## `docstore_upgrade_plan`

Dry-run additive migrations and return an exact plan ID.

Read only: `True`. Capability group: `diagnostics`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_upgrade_status`

Read current version, migration checksums and missing schema.

Read only: `True`. Capability group: `diagnostics`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_upgrade_verify`

Verify release ledger and live required schema after an upgrade.

Read only: `True`. Capability group: `diagnostics`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_approve_revision`

Record explicitly authorized approval of exact current content revision/hash; never infer approval from active status.

Read only: `False`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "approval": {
      "additionalProperties": false,
      "properties": {
        "document_key": {
          "pattern": "^(document|adr|note):[A-Za-z0-9_-]{1,128}$",
          "type": "string"
        },
        "revision_number": {
          "minimum": 1,
          "type": "integer"
        },
        "raw_sha256": {
          "pattern": "^[a-f0-9]{64}$",
          "type": "string"
        },
        "expected_generation": {
          "minimum": 1,
          "type": "integer"
        },
        "actor": {
          "maxLength": 100,
          "minLength": 1,
          "type": "string"
        },
        "rationale": {
          "maxLength": 2000,
          "minLength": 1,
          "type": "string"
        },
        "source_ref": {
          "maxLength": 1000,
          "minLength": 1,
          "type": "string"
        },
        "request_key": {
          "pattern": "^[A-Za-z0-9_-]{1,128}$",
          "type": "string"
        }
      },
      "required": [
        "document_key",
        "revision_number",
        "raw_sha256",
        "expected_generation",
        "actor",
        "rationale",
        "source_ref",
        "request_key"
      ],
      "type": "object"
    }
  },
  "required": [
    "approval"
  ],
  "type": "object"
}
```

## `docstore_capture_revision`

Capture bounded document content/history under a stable logical key; metadata-only rename retains revision. Does not run indexing.

Read only: `False`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "revision": {
      "additionalProperties": false,
      "properties": {
        "document_key": {
          "pattern": "^(document|adr|note):[A-Za-z0-9_-]{1,128}$",
          "type": "string"
        },
        "source_path": {
          "maxLength": 1000,
          "minLength": 1,
          "type": "string"
        },
        "title": {
          "maxLength": 300,
          "minLength": 1,
          "type": "string"
        },
        "body": {
          "maxLength": 1048576,
          "type": "string"
        },
        "expected_generation": {
          "minimum": 0,
          "type": "integer"
        },
        "actor": {
          "maxLength": 100,
          "minLength": 1,
          "type": "string"
        },
        "source_ref": {
          "maxLength": 1000,
          "minLength": 1,
          "type": "string"
        }
      },
      "required": [
        "document_key",
        "source_path",
        "title",
        "body",
        "expected_generation",
        "actor",
        "source_ref"
      ],
      "type": "object"
    }
  },
  "required": [
    "revision"
  ],
  "type": "object"
}
```

## `docstore_flags`

Read scoped flags independently of semantic similarity, with authority and lifecycle status.

Read only: `True`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "domain": {
      "maxLength": 64,
      "minLength": 1,
      "type": "string"
    },
    "priority": {
      "default": "critical",
      "enum": [
        "critical",
        "high",
        "normal"
      ],
      "type": "string"
    },
    "status": {
      "default": "active",
      "enum": [
        "active",
        "superseded",
        "retracted"
      ],
      "type": "string"
    },
    "limit": {
      "default": 20,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "domain"
  ],
  "type": "object"
}
```

## `docstore_reconcile_packet`

Persist a bounded JSON/Markdown conflict packet through the external governed adapter.

Read only: `False`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "maxLength": 2048,
      "minLength": 2,
      "type": "string"
    },
    "mode": {
      "default": "auto",
      "enum": [
        "auto",
        "all",
        "selected"
      ],
      "type": "string"
    },
    "stores": {
      "anyOf": [
        {
          "items": {
            "enum": [
              "smart_explore",
              "ccc",
              "docstore",
              "codex_memory",
              "claude_memory",
              "cnf",
              "remember",
              "memsearch"
            ],
            "type": "string"
          },
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "project_root": {
      "default": "",
      "maxLength": 512,
      "type": "string"
    },
    "limit": {
      "default": 20,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## `docstore_reconcile_query`

Query selected structural, semantic, Docstore, Codex, Claude, and memory stores with provenance.

Read only: `True`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "maxLength": 2048,
      "minLength": 2,
      "type": "string"
    },
    "mode": {
      "default": "auto",
      "enum": [
        "auto",
        "all",
        "selected"
      ],
      "type": "string"
    },
    "stores": {
      "anyOf": [
        {
          "items": {
            "enum": [
              "smart_explore",
              "ccc",
              "docstore",
              "codex_memory",
              "claude_memory",
              "cnf",
              "remember",
              "memsearch"
            ],
            "type": "string"
          },
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "project_root": {
      "default": "",
      "maxLength": 512,
      "type": "string"
    },
    "limit": {
      "default": 20,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## `docstore_reconcile_repair`

Persist the canonical bounded repair packet; source repair still requires an owning agent.

Read only: `False`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "maxLength": 2048,
      "minLength": 2,
      "type": "string"
    },
    "mode": {
      "default": "auto",
      "enum": [
        "auto",
        "all",
        "selected"
      ],
      "type": "string"
    },
    "stores": {
      "anyOf": [
        {
          "items": {
            "enum": [
              "smart_explore",
              "ccc",
              "docstore",
              "codex_memory",
              "claude_memory",
              "cnf",
              "remember",
              "memsearch"
            ],
            "type": "string"
          },
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "project_root": {
      "default": "",
      "maxLength": 512,
      "type": "string"
    },
    "limit": {
      "default": 20,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## `docstore_reconcile_validate`

Require explicit per-store state and clean Docstore attribution when Docstore is selected.

Read only: `True`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "maxLength": 2048,
      "minLength": 2,
      "type": "string"
    },
    "mode": {
      "default": "auto",
      "enum": [
        "auto",
        "all",
        "selected"
      ],
      "type": "string"
    },
    "stores": {
      "anyOf": [
        {
          "items": {
            "enum": [
              "smart_explore",
              "ccc",
              "docstore",
              "codex_memory",
              "claude_memory",
              "cnf",
              "remember",
              "memsearch"
            ],
            "type": "string"
          },
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "project_root": {
      "default": "",
      "maxLength": 512,
      "type": "string"
    },
    "limit": {
      "default": 20,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## `docstore_related_updates`

Query notes, ADRs, decision log and indexed documents before updates. Compact, literal keyword search; includes inactive records and explicit partial coverage.

Read only: `True`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "term": {
      "maxLength": 200,
      "minLength": 2,
      "type": "string"
    },
    "limit": {
      "default": 10,
      "maximum": 20,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "term"
  ],
  "type": "object"
}
```

## `docstore_revision_state`

Read current logical identity, revision metadata, approvals and path aliases; distinguishes changed-since-approval from indexing status.

Read only: `True`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "document_key": {
      "pattern": "^(document|adr|note):[A-Za-z0-9_-]{1,128}$",
      "type": "string"
    }
  },
  "required": [
    "document_key"
  ],
  "type": "object"
}
```

## `docstore_selected_update_plan`

Validate selected source bytes against current revisions; never starts indexing or proves bootstrap.

Read only: `True`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "request": {
      "additionalProperties": false,
      "properties": {
        "files": {
          "items": {
            "additionalProperties": false,
            "properties": {
              "document_key": {
                "pattern": "^(document|adr|note):[A-Za-z0-9_-]{1,128}$",
                "type": "string"
              },
              "source_path": {
                "maxLength": 1000,
                "minLength": 6,
                "type": "string"
              },
              "expected_generation": {
                "minimum": 1,
                "type": "integer"
              },
              "expected_revision_number": {
                "minimum": 1,
                "type": "integer"
              },
              "expected_raw_sha256": {
                "pattern": "^[a-f0-9]{64}$",
                "type": "string"
              }
            },
            "required": [
              "document_key",
              "source_path",
              "expected_generation",
              "expected_revision_number",
              "expected_raw_sha256"
            ],
            "type": "object"
          },
          "maxItems": 20,
          "minItems": 1,
          "type": "array"
        }
      },
      "required": [
        "files"
      ],
      "type": "object"
    }
  },
  "required": [
    "request"
  ],
  "type": "object"
}
```

## `docstore_set_flags`

Set priority/authority/status with expected revision and audit history; does not modify source or trigger CDC.

Read only: `False`. Capability group: `governance`.

```json
{
  "additionalProperties": false,
  "properties": {
    "flag": {
      "additionalProperties": false,
      "properties": {
        "subject": {
          "pattern": "^(document|adr|todo|note):[A-Za-z0-9_-]{1,128}$",
          "type": "string"
        },
        "title": {
          "maxLength": 300,
          "minLength": 1,
          "type": "string"
        },
        "summary": {
          "maxLength": 4000,
          "minLength": 1,
          "type": "string"
        },
        "priority": {
          "enum": [
            "critical",
            "high",
            "normal"
          ],
          "type": "string"
        },
        "authority": {
          "enum": [
            "owner_decision",
            "verified_finding",
            "proposal"
          ],
          "type": "string"
        },
        "status": {
          "default": "active",
          "enum": [
            "active",
            "superseded",
            "retracted"
          ],
          "type": "string"
        },
        "domains": {
          "items": {
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
          "maxItems": 12,
          "minItems": 1,
          "type": "array"
        },
        "source_ref": {
          "maxLength": 1000,
          "minLength": 1,
          "type": "string"
        },
        "rationale": {
          "maxLength": 2000,
          "minLength": 1,
          "type": "string"
        },
        "actor": {
          "maxLength": 100,
          "minLength": 1,
          "type": "string"
        },
        "expected_revision": {
          "minimum": 0,
          "type": "integer"
        }
      },
      "required": [
        "subject",
        "title",
        "summary",
        "priority",
        "authority",
        "domains",
        "source_ref",
        "rationale",
        "actor",
        "expected_revision"
      ],
      "type": "object"
    }
  },
  "required": [
    "flag"
  ],
  "type": "object"
}
```

## `docstore_graph`

Read a bounded relationship neighborhood for an exact document ID.

Read only: `True`. Capability group: `graphs`.

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

## `docstore_graph_entity_upsert`

Resolve/create one canonical entity using normalized names and explicit aliases.

Read only: `False`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "name": {
      "type": "string"
    },
    "kind": {
      "type": "string"
    },
    "aliases": {
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
      "default": null
    }
  },
  "required": [
    "name",
    "kind"
  ],
  "type": "object"
}
```

## `docstore_graph_path`

Find a bounded directed graph path; not-found applies only to the requested bounds.

Read only: `True`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "start": {
      "type": "string"
    },
    "end": {
      "type": "string"
    },
    "relation": {
      "default": "related_to",
      "type": "string"
    },
    "depth": {
      "default": 4,
      "type": "integer"
    },
    "limit": {
      "default": 100,
      "type": "integer"
    }
  },
  "required": [
    "start",
    "end"
  ],
  "type": "object"
}
```

## `docstore_graph_query`

Run a depth-one, allowlisted, parameter-bound graph query and return JSON or an inline export.

Read only: `True`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "subject": {
      "maxLength": 256,
      "minLength": 1,
      "type": "string"
    },
    "relation_types": {
      "default": [
        "links_to",
        "cites",
        "supersedes"
      ],
      "items": {
        "enum": [
          "links_to",
          "cites",
          "supersedes"
        ],
        "type": "string"
      },
      "type": "array"
    },
    "direction": {
      "default": "both",
      "enum": [
        "in",
        "out",
        "both"
      ],
      "type": "string"
    },
    "limit": {
      "default": 25,
      "maximum": 200,
      "minimum": 1,
      "type": "integer"
    },
    "source_prefix": {
      "default": "",
      "maxLength": 256,
      "type": "string"
    },
    "observed_from": {
      "default": "",
      "maxLength": 48,
      "type": "string"
    },
    "observed_to": {
      "default": "",
      "maxLength": 48,
      "type": "string"
    },
    "export_format": {
      "default": "json",
      "enum": [
        "json",
        "csv",
        "graphml",
        "mermaid"
      ],
      "type": "string"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "required": [
    "subject"
  ],
  "type": "object"
}
```

## `docstore_graph_query_preview`

Validate and preview a depth-one graph query, including its maximum result count.

Read only: `True`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "subject": {
      "maxLength": 256,
      "minLength": 1,
      "type": "string"
    },
    "relation_types": {
      "default": [
        "links_to",
        "cites",
        "supersedes"
      ],
      "items": {
        "enum": [
          "links_to",
          "cites",
          "supersedes"
        ],
        "type": "string"
      },
      "type": "array"
    },
    "direction": {
      "default": "both",
      "enum": [
        "in",
        "out",
        "both"
      ],
      "type": "string"
    },
    "limit": {
      "default": 25,
      "maximum": 200,
      "minimum": 1,
      "type": "integer"
    },
    "source_prefix": {
      "default": "",
      "maxLength": 256,
      "type": "string"
    },
    "observed_from": {
      "default": "",
      "maxLength": 48,
      "type": "string"
    },
    "observed_to": {
      "default": "",
      "maxLength": 48,
      "type": "string"
    },
    "export_format": {
      "default": "json",
      "enum": [
        "json",
        "csv",
        "graphml",
        "mermaid"
      ],
      "type": "string"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "required": [
    "subject"
  ],
  "type": "object"
}
```

## `docstore_graph_relate`

Create one native SurrealDB relationship; allowed types and record IDs are validated.

Read only: `False`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "start": {
      "type": "string"
    },
    "end": {
      "type": "string"
    },
    "relation": {
      "default": "related_to",
      "type": "string"
    }
  },
  "required": [
    "start",
    "end"
  ],
  "type": "object"
}
```

## `docstore_graph_schema`

Read fixed graph node/relation types, bounds, export formats, and live counts.

Read only: `True`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_knowledge_graph`

Read bounded native SurrealDB relationships across documents, entities and statements.

Read only: `True`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "start": {
      "type": "string"
    },
    "relation": {
      "default": "about",
      "type": "string"
    },
    "depth": {
      "default": 1,
      "type": "integer"
    },
    "limit": {
      "default": 50,
      "type": "integer"
    }
  },
  "required": [
    "start"
  ],
  "type": "object"
}
```

## `docstore_surrealql_read`

Bounded native SELECT fields/traversals FROM table_or_record LIMIT n. Expressions and writes are rejected.

Read only: `True`. Capability group: `graphs`.

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## `docstore_handoff_write`

Write and verify one governed Docstore handoff. Pass `supersedes` (document:<id> list)
for the specific prior handoff(s) this replaces. Omitted supersedes becomes [];
it never supersedes every same-domain handoff.

Read only: `False`. Capability group: `handoff`.

```json
{
  "additionalProperties": false,
  "properties": {
    "handoff": {
      "additionalProperties": false,
      "properties": {
        "title": {
          "maxLength": 300,
          "minLength": 1,
          "type": "string"
        },
        "body": {
          "maxLength": 1048576,
          "minLength": 1,
          "type": "string"
        },
        "domains": {
          "items": {
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
          "maxItems": 12,
          "minItems": 1,
          "type": "array"
        },
        "supersedes": {
          "items": {
            "type": "string"
          },
          "maxItems": 50,
          "type": "array"
        }
      },
      "required": [
        "title",
        "body",
        "domains"
      ],
      "type": "object"
    }
  },
  "required": [
    "handoff"
  ],
  "type": "object"
}
```

## `docstore_cdc_runs`

Read bounded append-only worker receipts; execution completion is not per-document CDC proof.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "run_id": {
      "anyOf": [
        {
          "pattern": "^[a-f0-9]{32}$",
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "limit": {
      "default": 20,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "type": "object"
}
```

## `docstore_index_execute`

Start full-source CocoIndex reconciliation; selected docs are exact verification targets.

Read only: `False`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "paths": {
      "anyOf": [
        {
          "items": {
            "type": "string"
          },
          "maxItems": 20,
          "minItems": 1,
          "type": "array"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "full_reprocess": {
      "default": false,
      "type": "boolean"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_index_full`

Start one governed full-source CocoIndex reconciliation. 0.8.1-r3: stored documents missing from the source are held, not retracted, unless named in retract_paths.

Read only: `False`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "full_reprocess": {
      "default": false,
      "type": "boolean"
    },
    "tracking_rebuild": {
      "default": false,
      "type": "boolean"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    },
    "retract_paths": {
      "anyOf": [
        {
          "items": {
            "type": "string"
          },
          "maxItems": 1000,
          "type": "array"
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

## `docstore_index_plan`

Hash explicitly selected local Markdown docs without embeddings, indexing or store writes.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "paths": {
      "items": {
        "type": "string"
      },
      "maxItems": 20,
      "minItems": 1,
      "type": "array"
    }
  },
  "required": [
    "paths"
  ],
  "type": "object"
}
```

## `docstore_index_selected`

Admit selected Markdown paths as verification targets while reconciling the complete source.

Read only: `False`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "paths": {
      "items": {
        "type": "string"
      },
      "maxItems": 20,
      "minItems": 1,
      "type": "array"
    },
    "full_reprocess": {
      "default": false,
      "type": "boolean"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "required": [
    "paths"
  ],
  "type": "object"
}
```

## `docstore_pipeline_identity`

Verify the deployed worker app/environment identity from its durable latest run.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_project_source`

Get one registered documentation source by stable project ID; read docstore_project_sources for IDs.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "project_id": {
      "type": "string"
    }
  },
  "required": [
    "project_id"
  ],
  "type": "object"
}
```

## `docstore_project_sources`

List governed Propria documentation roots and declared ingestion state; does not scan or index files.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_retraction_plan`

Preview documents outside the current five-root source snapshot; no writes.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_run_cancel`

Request cancellation of the exact active run launched by this worker API.

Read only: `False`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "run_id": {
      "pattern": "^[a-f0-9]{32}$",
      "type": "string"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "required": [
    "run_id"
  ],
  "type": "object"
}
```

## `docstore_run_current`

Read the durable current worker run status.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_run_get`

Read one durable run by exact ID.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "run_id": {
      "pattern": "^[a-f0-9]{32}$",
      "type": "string"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "required": [
    "run_id"
  ],
  "type": "object"
}
```

## `docstore_run_list`

List the newest durable Docstore runs, with one terminal/current record per run.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "limit": {
      "default": 20,
      "maximum": 100,
      "minimum": 1,
      "type": "integer"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_run_status`

Read the current run or one exact run ID from the deployed worker API.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "run_id": {
      "anyOf": [
        {
          "pattern": "^[a-f0-9]{32}$",
          "type": "string"
        },
        {
          "type": "null"
        }
      ],
      "default": null
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_source_apply`

Apply an exact source plan, quarantining replaced files; no index run. 0.8.1-r3: every document the plan would retract must be named in retract, or the apply is refused.

Read only: `False`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "files": {
      "items": {
        "additionalProperties": true,
        "type": "object"
      },
      "type": "array"
    },
    "plan_id": {
      "type": "string"
    },
    "retract": {
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
      "default": null
    }
  },
  "required": [
    "files",
    "plan_id"
  ],
  "type": "object"
}
```

## `docstore_source_plan`

Dry-run a complete hash-validated five-root source sync; no embedding or writes.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "files": {
      "items": {
        "additionalProperties": true,
        "type": "object"
      },
      "type": "array"
    }
  },
  "required": [
    "files"
  ],
  "type": "object"
}
```

## `docstore_source_read`

0.8.1-r3: exact mirror copies (content + sha256) of named project/path keys, for hash-verified restores; bounded, unsent keys return in remaining.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "paths": {
      "items": {
        "type": "string"
      },
      "type": "array"
    }
  },
  "required": [
    "paths"
  ],
  "type": "object"
}
```

## `docstore_verify_index`

Compare selected source fingerprints with stored documents/chunks; reports lag without running CDC or claiming execution proof.

Read only: `True`. Capability group: `index`.

```json
{
  "additionalProperties": false,
  "properties": {
    "paths": {
      "items": {
        "type": "string"
      },
      "maxItems": 20,
      "minItems": 1,
      "type": "array"
    }
  },
  "required": [
    "paths"
  ],
  "type": "object"
}
```

## `docstore_memory_recall`

Recall independent remote shared memory (active rows in scope and its descendants), with
server-side query embedding, BM25 + vector fusion and DuckDB packing. Scope root is "propria".

Read only: `True`. Capability group: `memory`.

```json
{
  "additionalProperties": false,
  "properties": {
    "query": {
      "maxLength": 2000,
      "minLength": 1,
      "type": "string"
    },
    "scope": {
      "default": "propria",
      "pattern": "^propria(/[a-z0-9_-]+)*$",
      "type": "string"
    },
    "limit": {
      "default": 10,
      "maximum": 50,
      "minimum": 1,
      "type": "integer"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## `docstore_memory_remember`

Write one claim through the memory service's governed fn::remember. Call with
docstore_query(operation="docstore_memory_remember", mode="write", arguments={"payload": {...}}).
Required: kind, claim, evidence, agent. Optional: scope (default "propria"), detail, confidence,
observed_at, force, supersede, reason.
Duplicate guard (0.8.1-r6, 2026-09-28): an active row in the same scope conflicts if BM25 finds every
claim word in it, or its cosine distance is <= 0.10, or its distance is <= 0.20 AND word overlap
(Jaccard) is >= 0.35. Then nothing is written and the call fails HTTP 409 listing the conflicting ids
with dist and overlap; retry with supersede:"<id>" and a reworded claim to replace one, or force:true.
Success returns {outcome: "written"|"superseded", id, superseded, scope}.
Errors: 422 invalid payload (every problem listed), 409 near-duplicate or exact claim already stored,
503/502 memory service unreachable or failed.

Read only: `False`. Capability group: `memory`.

```json
{
  "additionalProperties": false,
  "properties": {
    "payload": {
      "additionalProperties": false,
      "description": "One durable claim for the shared agent memory (SurrealDB probata_memory/memory).",
      "properties": {
        "kind": {
          "description": "What sort of claim: an owner rule is usually constraint, preference or correction.",
          "enum": [
            "correction",
            "preference",
            "observation",
            "handoff",
            "fact",
            "constraint",
            "decision"
          ],
          "type": "string"
        },
        "claim": {
          "description": "One self-contained sentence an agent can act on without the conversation. Unique per scope: the exact text can never be written twice, even after it is superseded or retracted.",
          "maxLength": 599,
          "minLength": 11,
          "type": "string"
        },
        "evidence": {
          "description": "Where the claim comes from: owner quote with date/time, doc id, file path or session anchor.",
          "maxLength": 2000,
          "minLength": 1,
          "type": "string"
        },
        "agent": {
          "description": "Who writes it, e.g. \"Claude Code · Opus 5.5\".",
          "maxLength": 200,
          "minLength": 1,
          "type": "string"
        },
        "scope": {
          "default": "propria",
          "description": "Hierarchical scope. \"propria\" is the whole project; \"propria/<module>[/<agent>]\" narrows it. Recall of a scope includes its descendants.",
          "pattern": "^propria(/[a-z0-9_-]+)*$",
          "type": "string"
        },
        "detail": {
          "anyOf": [
            {
              "maxLength": 8000,
              "type": "string"
            },
            {
              "type": "null"
            }
          ],
          "default": null,
          "description": "Optional longer explanation: why, how to apply."
        },
        "confidence": {
          "anyOf": [
            {
              "maximum": 1,
              "minimum": 0,
              "type": "number"
            },
            {
              "type": "null"
            }
          ],
          "default": null,
          "description": "0-1; the store defaults to 0.6. Owner rules: 0.95-1."
        },
        "observed_at": {
          "anyOf": [
            {
              "type": "string"
            },
            {
              "type": "null"
            }
          ],
          "default": null,
          "description": "ISO-8601 time the claim was observed; defaults to now."
        },
        "force": {
          "default": false,
          "description": "Write even though near-duplicates exist, keeping both. Use only when the claims really differ.",
          "type": "boolean"
        },
        "supersede": {
          "anyOf": [
            {
              "pattern": "^memory:[A-Za-z0-9_]+$",
              "type": "string"
            },
            {
              "type": "null"
            }
          ],
          "default": null,
          "description": "Replace this ACTIVE memory id: the new row is written, linked ->supersedes-> the old one, and the old row becomes status superseded (never deleted). The claim text must differ from the old claim."
        },
        "reason": {
          "anyOf": [
            {
              "maxLength": 1000,
              "type": "string"
            },
            {
              "type": "null"
            }
          ],
          "default": null,
          "description": "Why a supersession happened; stored in decision_log."
        }
      },
      "required": [
        "kind",
        "claim",
        "evidence",
        "agent"
      ],
      "type": "object"
    }
  },
  "required": [
    "payload"
  ],
  "type": "object"
}
```

## `coco_docstore_search`

Primary documentation search: CocoIndex/NIM vectors searched in SurrealDB; compact uses bounded DuckDB presentation.

Read only: `True`. Capability group: `retrieval`.

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

## `docstore_attribution_verify`

Run a fresh read-only exact path and normalized-content-hash comparison.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "type": "object"
}
```

## `docstore_cancel_run`

Request cancellation of the exact run launched by this API process.

Read only: `False`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {
    "run_id": {
      "pattern": "^[a-f0-9]{32}$",
      "type": "string"
    },
    "index_kind": {
      "const": "docs",
      "default": "docs",
      "type": "string"
    }
  },
  "required": [
    "run_id"
  ],
  "type": "object"
}
```

## `docstore_capabilities`

Describe supported operations and isolation configuration; not a live job-status report.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_compact`

Preferred low-noise retrieval: DuckDB column/row tables, body excerpts, no vectors; preserves flags and diagnostics. No arbitrary SQL.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {
    "operation": {
      "enum": [
        "search",
        "flags",
        "graph",
        "document"
      ],
      "type": "string"
    },
    "query": {
      "default": "",
      "maxLength": 2048,
      "type": "string"
    },
    "domain": {
      "default": "docs",
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
    "record_id": {
      "default": "",
      "maxLength": 140,
      "type": "string"
    }
  },
  "required": [
    "operation"
  ],
  "type": "object"
}
```

## `docstore_context_pack`

Normalize federated candidates through DuckDB with retained provenance and a byte budget.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {
    "rows": {
      "items": {
        "additionalProperties": true,
        "type": "object"
      },
      "type": "array"
    },
    "limit": {
      "default": 20,
      "type": "integer"
    },
    "budget": {
      "default": 8000,
      "type": "integer"
    }
  },
  "required": [
    "rows"
  ],
  "type": "object"
}
```

## `docstore_get`

Retrieve a document by returned record ID; preserves status and body.

Read only: `True`. Capability group: `retrieval`.

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

## `docstore_health`

Read actual health from the configured documentation API.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_search`

Compatibility name for full hybrid search; prefer coco_docstore_search for agent retrieval.

Read only: `True`. Capability group: `retrieval`.

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
    }
  },
  "required": [
    "query",
    "domain"
  ],
  "type": "object"
}
```

## `docstore_stats`

Read document/chunk/edge counts and vector-index status; not source freshness or job completion.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

## `docstore_surrealist`

Return credential-free Surrealist/Studio viewer links and the dedicated Docstore connection recipe.

Read only: `True`. Capability group: `retrieval`.

```json
{
  "additionalProperties": false,
  "properties": {},
  "type": "object"
}
```

