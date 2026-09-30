# Tracked LibreChat runtime configuration.
#
# The upstream application image remains pinned by digest. Baking the
# repository-owned, secret-free config into the derived image makes a normal
# Coolify rebuild deploy the exact reviewed MCP routing contract; operators no
# longer have to copy a second host file that can drift from Git.
#
# Byline: Codex · GPT-5.6-Sol · 2026-08-29
# Byline: Claude Code · Fable 5.1 · 2026-09-26 (COPY path from the Coolify build context = base_directory)

FROM ghcr.io/danny-avila/librechat@sha256:c5db3331b845e1f289f8d04c0c77936c4bbe372f76730a804abc1c37e44d23a9

# The build context is the Probata module root (Coolify base_directory).
COPY deploy/docker/librechat/librechat.yaml /app/librechat.yaml
