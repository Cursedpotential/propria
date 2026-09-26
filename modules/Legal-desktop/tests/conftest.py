"""Pytest configuration for Legal-Workspace tests.

This file sets up test fixtures and overrides for the test environment.

Byline: Codex · GPT-5 · 2026-09-12
"""

import os

import pytest
from fastapi.testclient import TestClient
from legal_workspace.api.main import app
from legal_workspace.contracts.source_package import LegalSourcePackage, ReviewState
from legal_workspace.services.source_package import ImportResult
from legal_workspace.services.workspace import Workspace


def seed_synthetic_approved_package(
    workspace: Workspace, package: LegalSourcePackage
) -> ImportResult:
    """Arrange downstream domain tests without crossing the held D08 import gate.

    This writes synthetic fixture state directly; it is not a producer verifier
    or a public import path and must never be used to assert import acceptance.
    """
    accepted = [item for item in package.items if item.review_state is ReviewState.APPROVED]
    omitted = tuple(str(item.item_id) for item in package.items if item.review_state is not ReviewState.APPROVED)
    selected = package.model_copy(update={"items": accepted})
    state = workspace.load()
    state.package = selected
    state.omitted_item_ids = list(omitted)
    workspace._write(state, action="synthetic-test-fixture")
    return ImportResult(accepted=selected, omitted_item_ids=omitted, blocked=False)


@pytest.fixture(autouse=True)
def explicit_unit_test_auth_bypass():
    """Keep legacy domain tests isolated; test_auth exercises the real boundary."""

    previous = os.environ.get("LEGAL_WORKSPACE_BYPASS_AUTH")
    os.environ["LEGAL_WORKSPACE_BYPASS_AUTH"] = "true"
    yield
    if previous is None:
        os.environ.pop("LEGAL_WORKSPACE_BYPASS_AUTH", None)
    else:
        os.environ["LEGAL_WORKSPACE_BYPASS_AUTH"] = previous


@pytest.fixture
def client_with_auth():
    """Test client operating under the explicit unit-test bypass."""

    return TestClient(app)
