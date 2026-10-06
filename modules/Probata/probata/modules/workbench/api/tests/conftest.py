# Byline: Claude Code · Sonnet (agent) · 2026-07-19
import pytest
from app.config import settings
from app.runtime import operating_mode
from httpx import ASGITransport, AsyncClient
from main import app


@pytest.fixture(autouse=True)
def fixed_case_boundary_fixture(monkeypatch, request):
    """Keep route-unit stubs focused on their operation, not engine-header I/O.

    Inputs: pytest test identity. Output: neutral case configuration. Effects:
    patches test-only scope admission. The canonical policy acceptance suite
    retains the real verification helper and tests wrong configuration/receipts.
    Use for isolated BFF unit tests, never live integration proof.
    """
    from app.runtime import health
    from app.service import imported

    monkeypatch.setattr(health, "check_lancedb_connectivity", lambda: False)
    monkeypatch.setattr(health, "check_connectivity", lambda: False)
    monkeypatch.setattr(imported, "warm", lambda: None)
    monkeypatch.setattr(settings, "proffer_matter_id", "11111111-1111-4111-8111-111111111111")
    monkeypatch.setattr(settings, "proffer_court_case_id", "22222222-2222-4222-8222-222222222222")
    if request.module.__name__.endswith("test_proffer_mode_isolation"):
        return

    async def verified_scope(_mode):
        return None

    monkeypatch.setattr(operating_mode, "verify_case_scope", verified_scope)


@pytest.fixture
async def client():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac
