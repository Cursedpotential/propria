---
title: "MCP servers and exposed tools"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# MCP servers and exposed tools

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

These are existing virtual servers returned by ContextForge. Attach the required server in your MCP client; its tool names are listed below. Enabled registration does not prove every upstream call works.

## `family-court-console`

Michigan Family Court Console for Claude Code and Codex: the family-court-toolkit plugin's tools, resources and prompts, served by the hosted family-court-console app. Tool names carry the `family-court-` prefix.

Server ID: `0b85f64905be468ba477d75cb2a80323`. Enabled: `True`.

- `family-court-court-language-review`
- `family-court-case-put`
- `family-court-case-query`
- `family-court-case-search`
- `family-court-case-graph`
- `family-court-case-factor-map`
- `family-court-case-timeline`
- `family-court-case-export`
- `family-court-case-import`
- `family-court-case-summary`
- `family-court-case-status`
- `family-court-case-docket`
- `family-court-case-memo`
- `family-court-case-evidence-log`
- `family-court-case-eval`
- `family-court-case-reference`
- `family-court-case-record`
- `family-court-case-source`
- `family-court-open-dashboard`
- `family-court-route-issue`
- `family-court-calculate-planning-date`
- `family-court-get-packet-plan`
- `family-court-get-checklist`
- `family-court-audit-sources`
- `family-court-search-guide`
- `family-court-build-chronology`
- `family-court-case-facts`
- `family-court-survival-guide`

## `n8n`

n8n workflow automation for Claude Code and Codex: search, read, validate, build, publish and run workflows, inspect executions, manage data tables, folders and n8n Agents. The instance's own MCP server (casebible-n8n on ovh-files) behind ContextForge.

Server ID: `c961807d29e24cd790987ff05d940e7c`. Enabled: `True`.

- `n8n-search-workflows`
- `n8n-execute-workflow`
- `n8n-get-workflow-execution`
- `n8n-search-workflow-executions`
- `n8n-get-workflow-details`
- `n8n-get-workflow-history`
- `n8n-get-workflow-version`
- `n8n-get-workflow-versions-diff`
- `n8n-publish-workflow`
- `n8n-unpublish-workflow`
- `n8n-prepare-workflow-pin-data`
- `n8n-test-workflow`
- `n8n-list-credentials`
- `n8n-list-n8n-connect-services`
- `n8n-list-workflow-tags`
- `n8n-search-data-tables`
- `n8n-create-data-table`
- `n8n-rename-data-table`
- `n8n-add-data-table-column`
- `n8n-delete-data-table-column`
- `n8n-rename-data-table-column`
- `n8n-add-data-table-rows`
- `n8n-get-data-table-rows`
- `n8n-search-nodes`
- `n8n-get-node-types`
- `n8n-get-workflow-best-practices`
- `n8n-explore-node-resources`
- `n8n-validate-workflow`
- `n8n-validate-node-config`
- `n8n-create-workflow-from-code`
- `n8n-search-projects`
- `n8n-search-folders`
- `n8n-create-folder`
- `n8n-update-folder`
- `n8n-move-workflows-to-folder`
- `n8n-archive-workflow`
- `n8n-update-workflow`
- `n8n-restore-workflow-version`
- `n8n-get-workflow-sdk-reference`
- `n8n-search-agents`
- `n8n-get-agent`
- `n8n-create-agent`
- `n8n-mutate-agent`
- `n8n-validate-agent`
- `n8n-call-agent`
- `n8n-publish-agent`
- `n8n-unpublish-agent`
- `n8n-revert-agent`
- `n8n-list-agent-versions`
- `n8n-delete-agent`
- `n8n-discover-agent-assets`
- `n8n-verify-agent-mcp-server`
- `n8n-update-agent-integration`
- `n8n-get-agent-builder-reference`

## `memsearch`

memsearch shared agent memory for Claude Code and Codex: search (hybrid dense + BM25), expand (a whole note rebuilt from its chunks), recall (search + expand, packed) and status. The memsearch plugin's own server, hosted as memsearch-mcp on ovh-files; data comes only from Milvus.

Server ID: `3b578fe024194f4ca1ef93c8ad362d95`. Enabled: `True`.

- `memsearch-search`
- `memsearch-expand`
- `memsearch-recall`
- `memsearch-status`

## `surrealdb`

SurrealDB's own MCP tools for the Propria instances: docs-* = surreal-docs (ns probata, db docs); mem-* = surreal-case instance (ns probata_memory db memory by default; ns fct db case via an inline USE). Rules and access notes: the surrealdb plugin's surrealdb-deployments skill.

Server ID: `b67cbe99052a46209acad26308a871e9`. Enabled: `True`.

- `docs-query`
- `mem-info`
- `mem-list`
- `docs-list`
- `docs-run`
- `docs-select`
- `docs-info`
- `mem-select`
- `mem-run`
- `mem-query`

## `atomic-tools`

Probata atomic tools through the tool gateway.

Server ID: `85016c5244f6440eb665e2e9c9338eb1`. Enabled: `True`.

- `atomic-tools-atomic-tools`

## `advocatio`

Advocatio legal workdesk file tools: metadata read and scrub, OCR, Office to PDF, image ordering.

Server ID: `48d341d4e7b441618ec79c96e6671dc1`. Enabled: `True`.

- `advocatio-ocr-image`
- `advocatio-convert-office-document-to-pdf`
- `advocatio-order-images-by-original-time`
- `advocatio-read-file-metadata`
- `advocatio-scrub-pdf-metadata`

## `coolify-write`

Coolify operations (read, create, envs, deploy and lifecycle for applications, databases and services, plus the raw API passthrough): the coolify-write plugin's own server, hosted as coolify-mcp on ovh-app.

Server ID: `e0bc95b5e93148e785154c350fc72830`. Enabled: `True`.

- `coolify-write-list-deployments-for-app`
- `coolify-write-list-servers`
- `coolify-write-get-database`
- `coolify-write-get-infrastructure-overview`
- `coolify-write-get-server`
- `coolify-write-restart-service`
- `coolify-write-upsert-application-envs`
- `coolify-write-get-application-logs`
- `coolify-write-get-application`
- `coolify-write-list-databases`
- `coolify-write-list-projects`
- `coolify-write-get-service`
- `coolify-write-create-database`
- `coolify-write-start-database`
- `coolify-write-create-service`
- `coolify-write-delete-database`
- `coolify-write-stop-database`
- `coolify-write-deploy-application`
- `coolify-write-stop-application`
- `coolify-write-check-port-collision`
- `coolify-write-get-deployment`
- `coolify-write-start-service`
- `coolify-write-restart-database`
- `coolify-write-delete-service`
- `coolify-write-cancel-deployment`
- `coolify-write-create-project`
- `coolify-write-coolify-api`
- `coolify-write-delete-project`
- `coolify-write-list-application-envs`
- `coolify-write-list-services`
- `coolify-write-create-application`
- `coolify-write-set-service-image`
- `coolify-write-list-applications`
- `coolify-write-list-deployments`
- `coolify-write-github-app-webhook`
- `coolify-write-restart-application`
- `coolify-write-update-project`
- `coolify-write-get-service-env`
- `coolify-write-stop-service`
- `coolify-write-start-application`
- `coolify-write-update-application`
- `coolify-write-delete-application`
- `coolify-write-delete-application-env`

## `propria-docstore`

Consolidated Propria Docstore 0.8.1 hosted ctl

Server ID: `aca1b85df0ef49acaf152617f043bc96`. Enabled: `True`.

- `docstore-capabilities`
- `docstore-search`
- `docstore-get`
- `docstore-health`
- `docstore-query`

## `agent-memory`

Octopoda shared agent memory MCP

Server ID: `a14b17330a3d432e8eb1a87369b8af8c`. Enabled: `True`.

- `octopoda-octopoda-agent-stats`
- `octopoda-octopoda-share`
- `octopoda-octopoda-read-shared`
- `octopoda-octopoda-process-conversation`
- `octopoda-octopoda-consolidate`
- `octopoda-octopoda-forget`
- `octopoda-octopoda-broadcast`
- `octopoda-octopoda-related`
- `octopoda-octopoda-get-context`
- `octopoda-octopoda-forget-stale`
- `octopoda-octopoda-search-filtered`
- `octopoda-octopoda-log-decision`
- `octopoda-octopoda-recall-similar`
- `octopoda-octopoda-loop-status`
- `octopoda-octopoda-remember`
- `octopoda-octopoda-get-goal`
- `octopoda-octopoda-restore`
- `octopoda-octopoda-search`
- `octopoda-octopoda-set-goal`
- `octopoda-octopoda-update-progress`
- `octopoda-octopoda-status`
- `octopoda-octopoda-send-message`
- `octopoda-octopoda-recall`
- `octopoda-octopoda-list-agents`
- `octopoda-octopoda-snapshot`
- `octopoda-octopoda-loop-history`
- `octopoda-octopoda-read-messages`
- `octopoda-octopoda-recall-history`
- `octopoda-octopoda-memory-health`

## `dev-docs`

Developer docs MCP: context7, agno-docs, n8n-docs, cloudflare-docs

Server ID: `e6bf594590134686a2f10990c244f97b`. Enabled: `True`.

- `context7-resolve-library-id`
- `n8n-docs-searchdocumentation`
- `n8n-docs-getpage`
- `cloudflare-docs-search-cloudflare-documentation`
- `context7-query-docs`
- `agno-docs-query-docs-filesystem`
- `n8n-docs-sendfeedback`
- `agno-docs-submit-docs-feedback`
- `agno-docs-search-docs`
- `cloudflare-docs-migrate-pages-to-workers-guide`

