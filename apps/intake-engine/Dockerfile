# Byline: Claude Code · Opus 5 · 2026-09-17
# Hosted Intake engine image. Build context = repository root (Coolify clones the repo).
FROM rust:1.91.1-bookworm AS chef
RUN cargo install cargo-chef --locked --version 0.1.73
WORKDIR /src

FROM chef AS planner
COPY rust-toolchain.toml ./
COPY apps/src-tauri/src apps/src-tauri/src
COPY apps/src-tauri/Cargo.toml apps/src-tauri/Cargo.toml
COPY apps/intake-engine apps/intake-engine
WORKDIR /src/apps/intake-engine
RUN cargo chef prepare --recipe-path /recipe.json

FROM chef AS builder
COPY rust-toolchain.toml ./
COPY --from=planner /recipe.json /recipe.json
# Path dependencies (the headless tauri shim) must exist for the dependency layer.
COPY apps/intake-engine/crates apps/intake-engine/crates
WORKDIR /src/apps/intake-engine
RUN cargo chef cook --release --recipe-path /recipe.json
WORKDIR /src
COPY apps/src-tauri/src apps/src-tauri/src
COPY apps/src-tauri/Cargo.toml apps/src-tauri/Cargo.toml
COPY apps/intake-engine apps/intake-engine
WORKDIR /src/apps/intake-engine
RUN cargo build --release --bin intake-engine && cp target/release/intake-engine /intake-engine

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates libssl3 ffmpeg git bash procps tini \
 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /intake-engine /usr/local/bin/intake-engine
COPY scripts/intake-portkey-chat.json /app/config/intake-portkey-chat.json
ENV INTAKE_ENGINE_BIND=0.0.0.0:8790 \
    INTAKE_ENGINE_STATE=/state \
    INTAKE_MOUNT_ROOT=/srv/openlist \
    HOME=/srv/openlist \
    XDG_CONFIG_HOME=/state/xdg/config \
    XDG_DATA_HOME=/state/xdg/data \
    XDG_CACHE_HOME=/state/xdg/cache \
    XDG_STATE_HOME=/state/xdg/state \
    HISTFILE=/state/xdg/bash_history \
    INTAKE_PORTKEY_CONFIG_FILE=/app/config/intake-portkey-chat.json
EXPOSE 8790
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD bash -c 'exec 3<>/dev/tcp/127.0.0.1/8790 && printf "GET /healthz HTTP/1.0\r\n\r\n" >&3 && grep -q "200 OK" <&3'
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/intake-engine"]
