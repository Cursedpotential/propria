"""Simple test for JSON export/backup endpoint."""

from uuid import uuid4
from datetime import UTC, datetime

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services import workspace as workspace_mod


def test_json_export_endpoint_exists(tmp_path) -> None:
    """Test that JSON export endpoint exists and requires enablement."""
    # Use temporary directory for isolation like the working test
    main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_path)
    client = TestClient(app)

    # Temporarily disable JSON export by overriding environment
    import os
    original_env = os.environ.get("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED")
    os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = "false"

    try:
        # Should be disabled when explicitly set to false
        response = client.get("/v1/export/json")
        assert response.status_code == 403
        assert "JSON export is disabled" in response.json()["detail"]
    finally:
        # Restore environment
        if original_env is None:
            os.environ.pop("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED", None)
        else:
            os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = original_env


def test_json_export_works_when_enabled(tmp_path) -> None:
    """Test that JSON export works when enabled via environment."""
    # Set up isolated workspace
    main_mod.WORKSPACE = workspace_mod.get_workspace(tmp_path)
    client = TestClient(app)

    # Enable JSON export via environment
    import os
    original_env = os.environ.get("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED")
    os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = "true"

    try:
        response = client.get("/v1/export/json")
        # Should succeed when enabled
        assert response.status_code == 200

        data = response.json()
        # Verify basic structure
        assert "exported_at" in data
        assert "version" in data
        assert "workspace_state" in data
        assert isinstance(data["workspace_state"], dict)
        assert "matter" in data["workspace_state"]
        assert "court_case" in data["workspace_state"]

    finally:
        # Restore environment
        if original_env is None:
            os.environ.pop("LEGAL_WORKSPACE_JSON_EXPORT_ENABLED", None)
        else:
            os.environ["LEGAL_WORKSPACE_JSON_EXPORT_ENABLED"] = original_env