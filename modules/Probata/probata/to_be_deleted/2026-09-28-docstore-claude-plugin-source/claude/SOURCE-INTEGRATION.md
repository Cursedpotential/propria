# Source integration boundary

This 0.7.0 directory is the canonical **Claude** client, not a replacement for the Codex root bundle or for the remote FastMCP application.

To merge it into the uploaded repository source, apply `patches/CLAUDE-CLIENT-SOURCE.patch` from the repository root. The uploaded source files use CRLF line endings in places, so use `git apply --ignore-space-change patches/CLAUDE-CLIENT-SOURCE.patch` if plain `git apply` rejects only line-ending context. Keep the marketplace/source entry for Claude pointed at `source/plugins/docstore/claude` (or the corresponding repository path). Do not point Claude at the parent Codex bundle and the child Claude bundle simultaneously.

The separate `patches/REMOTE-CONTROL-TOOL-DESCRIPTION.patch` changes only the stale `docstore_handoff_write` Python docstring. Deploying that tiny server-source patch is recommended so MCP tool discovery presents the same safe supersession semantics as the client skill. It is not required for the already-correct wrapper behavior.
