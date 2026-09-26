# Byline: Grok · grok-4.6 · 2026-08-18
# Host-local API. No Docker.
Set-Location $PSScriptRoot\..
uv run uvicorn legal_workspace.api.main:app --app-dir api --host 127.0.0.1 --port 8010
