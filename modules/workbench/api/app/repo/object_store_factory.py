# Byline: Claude Code · Fable 5.1 · 2026-09-20 (split out of object_store_client.py)
"""Build one S3-compatible client from a runtime-mounted credential document.

Every configured object store (R2, B2, any S3-compatible provider) uses this
one builder; nothing here names a provider, endpoint or bucket.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path

import boto3
from botocore.config import Config


@dataclass(frozen=True)
class R2Config:
    endpoint_url: str
    region: str
    access_key_id: str
    secret_access_key: str
    session_token: str | None = None


def build_store_client(config_path: str, scheme: str):
    if not config_path:
        raise RuntimeError(f"Platform object store {scheme} is not configured")
    try:
        payload = json.loads(Path(config_path).read_text(encoding="utf-8"))
    except (OSError, ValueError) as error:
        raise RuntimeError(f"Platform object store {scheme} configuration could not be loaded") from error
    allowed = {"endpoint_url", "region", "access_key_id", "secret_access_key", "session_token"}
    if not isinstance(payload, dict) or set(payload) - allowed:
        raise RuntimeError(f"Platform object store {scheme} configuration is invalid")
    required = ("endpoint_url", "region", "access_key_id", "secret_access_key")
    if any(not isinstance(payload.get(key), str) or not payload[key].strip() for key in required):
        raise RuntimeError(f"Platform object store {scheme} configuration is invalid")
    if payload.get("session_token") is not None and not isinstance(payload["session_token"], str):
        raise RuntimeError(f"Platform object store {scheme} configuration is invalid")
    config = R2Config(**payload)
    return boto3.client(
        "s3",
        endpoint_url=config.endpoint_url,
        region_name=config.region,
        aws_access_key_id=config.access_key_id,
        aws_secret_access_key=config.secret_access_key,
        aws_session_token=config.session_token,
        config=Config(signature_version="s3v4", s3={"addressing_style": "path"}),
    )
