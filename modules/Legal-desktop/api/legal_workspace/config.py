"""Service-name and authentication configuration. Tailnet IPs are forbidden.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
> _Auth boundary: Codex · GPT-5 · 2026-09-12_
"""

from __future__ import annotations

import re

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

from legal_workspace.db.engine import default_sqlite_url

_IP_RE = re.compile(r"(?:^|[\s/=])(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?")


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    app_name: str = "legal-workspace"
    environment: str = Field(default="dev", alias="LEGAL_WORKSPACE_ENV")
    legal_api_service: str = Field(default="legal-api")
    legal_web_service: str = Field(default="legal-web")
    legal_postgres_service: str = Field(default="legal-postgres")
    evidence_platform_service: str = Field(default="evidence-platform")
    model_gateway_service: str = Field(default="model-gateway")
    legal_renderer_service: str = Field(default="legal-renderer")
    legal_api_port: int = Field(default=8010)
    display_timezone: str = Field(default="America/New_York")
    evidence_platform_base_url: str = Field(default="http://evidence-platform:8000")
    model_gateway_base_url: str = Field(default="http://model-gateway:4000")
    legal_renderer_base_url: str = Field(default="http://legal-renderer:3000")
    database_url: str = Field(default_factory=default_sqlite_url)
    invoke_models: bool = Field(default=False, alias="LEGAL_WORKSPACE_INVOKE_MODELS")
    debug_json: bool = Field(default=False, alias="LEGAL_WORKSPACE_DEBUG_JSON")
    bypass_auth: bool = Field(default=False, alias="LEGAL_WORKSPACE_BYPASS_AUTH")
    json_export_enabled: bool = Field(default=False, alias="LEGAL_WORKSPACE_JSON_EXPORT_ENABLED")
    contextforge_gateway_token: str | None = Field(default=None, alias="CF_GATEWAY_TOKEN")
    authentik_issuer: str = Field(default="", alias="AUTHENTIK_ISSUER")
    authentik_audience: str = Field(default="", alias="AUTHENTIK_AUDIENCE")
    authentik_jwks_url: str = Field(default="", alias="AUTHENTIK_JWKS_URL")
    authentik_allowed_groups: str = Field(default="", alias="AUTHENTIK_ALLOWED_GROUPS")
    authentik_review_groups: str = Field(
        default="advocatio-users", alias="AUTHENTIK_REVIEW_GROUPS"
    )
    legal_bff_signing_secret: str = Field(default="", alias="LEGAL_BFF_SIGNING_SECRET")
    mcp_gateway_token: str = Field(default="", alias="LEGAL_MCP_GATEWAY_TOKEN")
    tailnet_owner_access: bool = Field(default=True, alias="LEGAL_TAILNET_OWNER_ACCESS")
    auth_clock_skew_seconds: int = Field(default=30, alias="LEGAL_AUTH_CLOCK_SKEW_SECONDS")
    # Consignatio catalog (read-only). Empty URL = evidence desk reports "not configured".
    consignatio_catalog_url: str = Field(default="", alias="CONSIGNATIO_CATALOG_URL")
    consignatio_catalog_bindings_file: str = Field(default="", alias="CONSIGNATIO_CATALOG_BINDINGS")
    # B2 (S3-compatible) read access for short-lived artifact links. No mount, no mirror.
    b2_key_id: str = Field(default="", alias="B2_KEY_ID")
    b2_application_key: str = Field(default="", alias="B2_APPLICATION_KEY")
    b2_s3_endpoint: str = Field(default="", alias="B2_S3_ENDPOINT")
    b2_region: str = Field(default="", alias="B2_REGION")
    b2_link_ttl_seconds: int = Field(default=600, alias="B2_LINK_TTL_SECONDS")

    @field_validator(
        "consignatio_catalog_url",
        "legal_api_service",
        "legal_web_service",
        "legal_postgres_service",
        "evidence_platform_service",
        "model_gateway_service",
        "legal_renderer_service",
        "evidence_platform_base_url",
        "model_gateway_base_url",
        "legal_renderer_base_url",
        "database_url",
        "authentik_issuer",
        "authentik_jwks_url",
    )
    @classmethod
    def no_embedded_ips(cls, value: str) -> str:
        if _IP_RE.search(value):
            raise ValueError("service configuration must use names, not tailnet IPs")
        return value


def get_settings() -> Settings:
    return Settings()
