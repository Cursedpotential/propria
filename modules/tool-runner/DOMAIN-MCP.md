# Propria domain MCP and human runner

Owner scope: October 17, 2026. One package; independent domain backends; progressive agents; unrestricted human catalog.

## Entry points

- Single agent connection: https://mcp.mitechconsult.com/mcp
- Human tool runner: https://mcp.mitechconsult.com/runner/
- Connection marketplace: https://mcp.mitechconsult.com/runner/?view=connections

The hub advertises six actions. Each domain advertises five. Native tools and full Think methods are discovered only when selected. The human runner intentionally loads the entire authorized catalog and permits direct selection. Package choices never filter that human catalog.

| Domain | MCP endpoint | Backing catalog entries |
|---|---|---|
| n8n | https://mcp.mitechconsult.com/n8n/mcp | 54 |
| docstore | https://mcp.mitechconsult.com/docstore/mcp | 5 |
| storage | https://mcp.mitechconsult.com/storage/mcp | 28 |
| platform | https://mcp.mitechconsult.com/platform/mcp | 44 |
| think | https://mcp.mitechconsult.com/think/mcp | 101 |
| search | https://mcp.mitechconsult.com/search/mcp | 16 |
| admin | https://mcp.mitechconsult.com/admin/mcp | 43 |

## Agent controls

### domains
List domain connections and open the full human runner or marketplace.

### discover
Find native tools or Think processes in one domain; summaries only.

### inspect
Load the exact selected tool schema, effects and source guidance.

### read
Execute one native tool declared read-only, or load a complete Think activation packet.

### execute
Execute a selected native operation; it may modify/delete data or deploy infrastructure. Inspect first and obtain user authorization.

### resource
Read domain guidance or expand a result/Think resource by exact URI.

## Package and authorization

The portable ZIP contains one root plugin manifest, all selected domain MCP connections, domain skills and the original Think resource files. Hosts supporting portable multi-server plugins can install the selected connections together. A ChatGPT custom remote connection can instead use the single hub URL; native tools remain separated behind it. This does not silently install other ChatGPT apps, bypass consent, or grant new permissions.

The service retains existing Authentik sign-in, browser-bound OAuth state, PKCE, server-side tokens, CSRF checks, native gateway authorization and duplicate-submission protection. Write/admin dispatch is labeled as potentially destructive. No service-account token is embedded in the plugin or served to JavaScript.

## Data and exclusions

Octopoda was removed from all gateway tool associations, its virtual server and upstream registration. Its hosted application was stopped without pruning data. Existing memories and case data were not altered.

DuckDB processes JSON projections in memory only. Original native results are not rewritten; truncation is explicit and credential-bound result resources allow expansion. No new persistent DuckDB database or search index is created.

Think resources are copied from the existing owner-installed plugin, preserving source hashes and licenses. Micro/Focused/Full/Adaptive, task/conversation scope, and silent/normal/report output are independent. An activation packet instructs the calling host; it does not assert that a remote model or another conversation executed it.

## Verification

Run `python -m unittest test_app.py test_domains.py -q` from this directory. Run `node test_ui.cjs` with jsdom available. Production acceptance must verify short-route OAuth challenges, authenticated discovery and real read calls, the runner page and container health. Unit tests alone are not deployment proof.
