"""Bounded TwelveLabs v1.3 transport for the existing atomic tool runtime.

No SDK dependency, polling, database writes, or implicit HTTP retries. Credentials
come from the service environment or its configured secrets file, never a payload.
"""

from __future__ import annotations

import http.client
import json
import math
import os
import re
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path
from typing import Any

API_ROOT = "https://api.twelvelabs.io/v1.3"
MAX_RESPONSE_BYTES = 4 * 1024 * 1024
_KEY_NAMES = ("TWELVELABS_API_KEY", "TWELVE_LABS_API_KEY")


class TwelveLabsError(RuntimeError):
    """Expose a sanitized provider failure without credentials or source URLs."""

    def __init__(self, code: str, *, status: int | None = None, ambiguous: bool = False):
        super().__init__(f"TwelveLabs {code}" + (f" (HTTP {status})" if status else ""))
        self.code = code
        self.status = status
        self.ambiguous = ambiguous


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    """Reject redirects so the API key never reaches another host."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def _api_key() -> str:
    """Load only the API key from environment or the operator-configured env file.

    The file is parsed as literal assignments, never executed or interpolated.
    Inputs: process configuration. Output: secret string. Side effect: file read.
    Use TWELVELABS_ENV_FILE for Propria's .secrets/twelvelabs.env on deployment.
    """
    for name in _KEY_NAMES:
        value = os.environ.get(name, "").strip()
        if value:
            return value
    configured = os.environ.get("TWELVELABS_ENV_FILE")
    if configured:
        try:
            for line in Path(configured).read_text(encoding="utf-8-sig").splitlines():
                match = re.fullmatch(r"\s*(?:export\s+)?([A-Z_][A-Z0-9_]*)\s*=\s*(.*?)\s*", line)
                if match and match[1] in _KEY_NAMES:
                    value = match[2]
                    if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
                        value = value[1:-1]
                    else:
                        value = re.split(r"\s+#", value, maxsplit=1)[0].strip()
                    if value:
                        return value
        except (OSError, UnicodeError):
            raise TwelveLabsError("credentials_file_unavailable") from None
    raise TwelveLabsError("credentials_missing")


def identifier(value: Any, name: str) -> str:
    """Validate a provider identifier before placing it in an API path."""
    if not isinstance(value, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,200}", value):
        raise ValueError(f"{name} must be a provider identifier")
    return value


class TwelveLabsClient:
    """Perform one bounded request per operation against the fixed provider host."""

    def __init__(self) -> None:
        self._key = _api_key()
        try:
            self.timeout = float(os.environ.get("TWELVELABS_TIMEOUT_SECONDS", "60"))
        except ValueError:
            raise ValueError("TWELVELABS_TIMEOUT_SECONDS must be numeric") from None
        if not math.isfinite(self.timeout) or not 1 <= self.timeout <= 300:
            raise ValueError("TWELVELABS_TIMEOUT_SECONDS must be between 1 and 300")
        self._opener = urllib.request.build_opener(_NoRedirect())

    def request(
        self,
        method: str,
        path: str,
        *,
        body: dict[str, Any] | None = None,
        form: dict[str, str] | None = None,
        query: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Send one request and preserve provider JSON, including annotations.

        Inputs: internal path and bounded JSON/form fields. Output: provider object.
        Side effects: external request; POST can incur charges or create resources.
        No retries: a lost POST response is ambiguous and must be reconciled by the
        orchestrator before resubmission. GET calls can be retried by the caller.
        """
        if not path.startswith("/") or ".." in path or "?" in path or "#" in path:
            raise ValueError("invalid internal API path")
        headers = {"x-api-key": self._key, "Accept": "application/json"}
        data = None
        if form is not None:
            boundary = "twelvelabs-" + uuid.uuid4().hex
            parts = []
            for key, value in form.items():
                parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n')
            data = ("".join(parts) + f"--{boundary}--\r\n").encode()
            headers["Content-Type"] = f"multipart/form-data; boundary={boundary}"
        elif body is not None:
            data = json.dumps(body, allow_nan=False).encode()
            headers["Content-Type"] = "application/json"
        if data is not None and len(data) > 128 * 1024:
            raise ValueError("TwelveLabs request exceeds 128 KiB")
        url = API_ROOT + path
        if query:
            url += "?" + urllib.parse.urlencode(query)
        request = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with self._opener.open(request, timeout=self.timeout) as response:
                raw = response.read(MAX_RESPONSE_BYTES + 1)
        except urllib.error.HTTPError as exc:
            status = exc.code
            exc.close()
            raise TwelveLabsError(
                "provider_rejected_request", status=status, ambiguous=method != "GET" and status >= 500
            ) from None
        except (TimeoutError, urllib.error.URLError, OSError, http.client.HTTPException):
            raise TwelveLabsError("transport_failed", ambiguous=method != "GET") from None
        if len(raw) > MAX_RESPONSE_BYTES:
            raise TwelveLabsError("response_too_large", ambiguous=method != "GET")
        try:
            result = json.loads(raw)
        except (ValueError, UnicodeError):
            raise TwelveLabsError("invalid_json", ambiguous=method != "GET") from None
        if not isinstance(result, dict):
            raise TwelveLabsError("invalid_response", ambiguous=method != "GET")
        return result
