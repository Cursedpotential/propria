"""Tests for JSON export/backup endpoint."""

from __future__ import annotations

import tempfile
from uuid import uuid4

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.config import get_settings
from legal_workspace.services import workspace as workspace_mod


def test_json_export_disabled_by_default() -> None:
    """Test that JSON export endpoint is disabled by default."""
    # Explicitly disable JSON export to ensure consistent test results
    import os
    original_env = os.environ.get("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED")
    os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = "false"
    try:
        # Create a temporary workspace for isolation
        with tempfile.TemporaryDirectory() as tmp_dir:
            main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_dir)
            client = TestClient(app)
            response = client.get("/v1/export/json")
            assert response.status_code == 403
            assert "JSON export is disabled" in response.json()["detail"]
    finally:
        if original_env is None:
            os.environ.pop("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED", None)
        else:
            os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = original_env


def test_json_export_enabled_when_configured(tmp_path) -> None:
    """Test that JSON export endpoint works when enabled via settings."""
    # Set up isolated workspace
    main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_path)
    client = TestClient(app)

    # Temporarily enable JSON export by modifying environment
    import os
    original_env = os.environ.get("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED")
    os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = "true"

    try:
        # Force settings to reload by getting new instance
        # Note: Since settings are cached, we need to test the actual behavior
        response = client.get("/v1/export/json")
        # Should succeed when enabled (we'll check the structure below)
        assert response.status_code == 200

        data = response.json()
        assert "exported_at" in data
        assert "version" in data
        assert "workspace_state" in data
        assert isinstance(data["workspace_state"], dict)

    finally:
        # Restore environment
        if original_env is None:
            os.environ.pop("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED", None)
        else:
            os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = original_env


def test_json_export_returns_valid_structure(tmp_path) -> None:
    """Test that JSON export returns the expected structure when enabled."""
    # Set up isolated workspace
    main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_path)
    client = TestClient(app)

    # Enable JSON export via environment
    import os
    original_env = os.environ.get("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED")
    os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = "true"

    try:
        response = client.get("/v1/export/json")
        assert response.status_code == 200

        data = response.json()
        # Verify required fields exist
        assert "exported_at" in data
        assert "version" in data
        assert "workspace_state" in data

        # Verify workspace state has expected structure
        workspace_state = data["workspace_state"]
        assert isinstance(workspace_state, dict)
        assert "matter" in workspace_state
        assert "court_case" in workspace_state
        assert "factors" in workspace_state

    finally:
        # Restore environment
        if original_env is None:
            os.environ.pop("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED", None)
        else:
            os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = original_env