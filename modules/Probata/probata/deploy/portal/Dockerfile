# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Propria Homepage portal image. One Dockerfile, two instances: INSTANCE=tailnet or INSTANCE=public.
# The configuration is baked in (Coolify's render directory holds no repository checkout, so a
# relative bind of a repo file would mount an empty directory). shared/ is identical for both
# instances; <instance>/ carries only that instance's services.yaml.
# Base pinned to the exact image the hand-made containers ran on 2026-09-26: Homepage v1.13.2,
# multi-arch index digest below (verified against ghcr.io the same day).
FROM ghcr.io/gethomepage/homepage:v1.13.2@sha256:a0b71c8e757298d02560186bab9fbe3fc2d375c523a62cc1019177b37e48aa28
ARG INSTANCE
RUN test "$INSTANCE" = tailnet || test "$INSTANCE" = public
COPY shared/ /app/config/
COPY ${INSTANCE}/ /app/config/
