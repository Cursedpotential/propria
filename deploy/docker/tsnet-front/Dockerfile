# tsnet-front — the tailnet identity for a service that cannot embed one
# (today: the Python Workbench). See modules/engine/cmd/tsnet-front/main.go.
#
# Hermetic vendored build, same contract as every other engine service:
# modules/engine/vendor/ snapshots the full module graph including
# tailscale.com, so the build needs no network access and no sibling
# modules/forks/sbv checkout. Build context is the repo root (`context: .`).
#
# This image publishes NO port and declares no EXPOSE: its only listener is the
# tsnet one, which is the entire point (owner directive 2026-09-07).
#
# Toolchain pinned to match modules/engine/go.mod (`go 1.26.6`). A 1.25 builder
# fails the build outright with GOTOOLCHAIN=local; do not bump go.mod without
# bumping every Go Dockerfile in this directory in the same change.
#
# Byline: Claude Code subagent · Opus 5 · 2026-09-07.
FROM golang:1.26-bookworm AS builder

WORKDIR /src/engine
COPY modules/engine/ ./
# CGO stays on and fts5 is set for parity with the other engine images: the
# vendored module graph is shared, and a divergent build tag set has bitten
# this tree before.
RUN CGO_ENABLED=1 go build \
    -mod=vendor \
    -tags fts5 \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/tsnet-front \
    ./cmd/tsnet-front

FROM debian:bookworm-slim

# /data/tsnet holds the tsnet node state (its machine identity) and MUST be a
# persistent bind mount, or the service re-registers as a new tailnet node on
# every deploy and the Service FQDN churns.
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 platform \
    && useradd --system --uid 10001 --gid platform --home-dir /nonexistent --shell /usr/sbin/nologin platform \
    && install -d -o platform -g platform -m 0750 /data/tsnet

COPY --from=builder /out/tsnet-front /usr/local/bin/tsnet-front

USER 10001:10001
ENV TSNET_STATE_DIR=/data/tsnet \
    TSNET_AUTHKEY_FILE=/run/secrets/tsnet-authkey \
    TSNET_TAGS=tag:docker

ENTRYPOINT ["/usr/local/bin/tsnet-front"]
