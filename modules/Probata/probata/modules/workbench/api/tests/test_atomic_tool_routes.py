"""Check that the production Workbench app exposes the source-pinned action API."""


def test_atomic_tool_routes_are_in_the_production_app() -> None:
    """Require both action start and status paths in the real app's OpenAPI map.

    Inputs: the production FastAPI app. Outputs: route/method assertions.
    Effects: none; choose this to catch a BFF router omitted from main.py.
    """
    import main

    paths = main.app.openapi()["paths"]
    assert "post" in paths["/api/atomic-tool-actions"]
    assert "get" in paths["/api/atomic-tool-actions/{workflow_id}"]
