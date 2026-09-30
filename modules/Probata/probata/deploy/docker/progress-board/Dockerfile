# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Propria progress board (the /progress pages and the JSON the portal's widgets read). Same Node
# image the hand-built Coolify service runs; the source is baked in instead of bind-mounted from
# /data/dashboards/progress-board. Runtime data that is not source stays on the host and is mounted
# by deploy/progress-board.yaml: intake-build/ (Intake preview releases), data/ (synced
# URGENT-TODO.md) and the secrets.
FROM node:22.22.3-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund
COPY . .
USER 1000:1000
EXPOSE 3020
CMD ["node", "server.mjs"]
