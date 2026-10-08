"""Read exact-version query artifacts through the existing B2 client.

Inputs: a workflow-returned artifact reference. Output: bounded verified JSON.
Effects: one object read; use for query findings, never arbitrary browser URLs.
"""
import hashlib
import json
from urllib.parse import urlsplit

from app.repo.object_store_client import get_store_client, validate_source_key
from app.service.proffer_errors import ProfferError
from app.types.analysis import AnalysisArtifact

QUERY_PREFIX = "consignatio/casevault/DerivedKnowledge/analysis/queries/"


def read_query_artifact(ref: AnalysisArtifact) -> dict:
    """Fetch one bounded B2 version and verify its exact length and SHA-256.

    Input: trusted workflow reference. Output: JSON dict. Effects: read/close one
    stream; choose over a presigned/latest URL which would discard version pins.
    """
    uri = urlsplit(ref.uri)
    key = uri.path.lstrip("/")
    if uri.scheme != "b2" or uri.netloc != "salem-data" or uri.query or uri.fragment or not key.startswith(QUERY_PREFIX):
        raise ProfferError("Analysis returned a result outside its query-artifact location", 502)
    try:
        validate_source_key(key)
        response = get_store_client("b2").get_object(Bucket=uri.netloc, Key=key, VersionId=ref.version_id)
        stream = response["Body"]
        try:
            if response.get("VersionId") != ref.version_id or response.get("ContentLength") != ref.bytes:
                raise ProfferError("Analysis result version or length changed", 502)
            raw = stream.read(ref.bytes + 1)
        finally:
            stream.close()
    except ProfferError:
        raise
    except Exception:
        raise ProfferError("Analysis result could not be read from its exact B2 version", 502) from None
    if len(raw) != ref.bytes or hashlib.sha256(raw).hexdigest() != ref.sha256:
        raise ProfferError("Analysis result does not match its recorded hash", 502)
    try:
        body = json.loads(raw)
    except (ValueError, UnicodeDecodeError):
        raise ProfferError("Analysis result is not valid JSON", 502) from None
    if not isinstance(body, dict):
        raise ProfferError("Analysis result is not a findings document", 502)
    return body
