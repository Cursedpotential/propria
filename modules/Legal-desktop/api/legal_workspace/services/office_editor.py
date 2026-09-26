"""Collabora WOPI adapter; durable scoped sessions and document-level locks.

Protocol references: https://sdk.collaboraonline.com/docs/How_to_integrate.html
and https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/putfile
Tokens are capabilities for one private working document, never review/release authority.
"""

from __future__ import annotations

import hashlib
import os
import secrets
import sqlite3
import time
import xml.etree.ElementTree as ET
from contextlib import contextmanager
from functools import lru_cache
from pathlib import Path
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

import httpx

from legal_workspace.services.work_documents import WorkDocumentService


class OfficeError(ValueError):
    def __init__(self, message: str, status: int = 400, lock: str | None = None):
        super().__init__(message)
        self.status = status
        self.lock = lock


def request_token(query_token: str | None, authorization: str | None) -> str:
    bearer = None
    if authorization:
        scheme, _, value = authorization.partition(" ")
        if scheme.lower() != "bearer" or not value:
            raise OfficeError("Invalid editor token", 401)
        bearer = value
    if query_token and bearer and query_token != bearer:
        raise OfficeError("Conflicting editor tokens", 401)
    token = query_token or bearer
    if not token or len(token) > 256:
        raise OfficeError("Editor token required", 401)
    return token


class OfficeEditor:
    SESSION_SECONDS = 8 * 60 * 60
    LOCK_SECONDS = 30 * 60

    def __init__(self, documents: WorkDocumentService, store_dir: Path):
        self.documents = documents
        self.path = Path(store_dir) / "legal.sqlite"
        self.path.parent.mkdir(parents=True, exist_ok=True)
        with self.connection() as conn:
            conn.executescript("""
                CREATE TABLE IF NOT EXISTS office_editor_sessions (
                    token_hash TEXT PRIMARY KEY, document_id TEXT NOT NULL,
                    actor TEXT NOT NULL, expires REAL NOT NULL, revision INTEGER NOT NULL
                );
                CREATE TABLE IF NOT EXISTS office_editor_locks (
                    document_id TEXT PRIMARY KEY, lock_value TEXT NOT NULL,
                    expires REAL NOT NULL, revision INTEGER NOT NULL
                );
            """)

    @contextmanager
    def connection(self):
        conn = sqlite3.connect(self.path, timeout=30)
        conn.row_factory = sqlite3.Row
        try:
            with conn:
                yield conn
        finally:
            conn.close()

    @staticmethod
    def token_hash(token: str) -> str:
        return hashlib.sha256(token.encode()).hexdigest()

    def authorize(self, document_id: str, token: str) -> dict:
        with self.connection() as conn:
            row = conn.execute(
                "SELECT * FROM office_editor_sessions WHERE token_hash=? AND document_id=?",
                (self.token_hash(token), document_id),
            ).fetchone()
        if row is None or row["expires"] <= time.time():
            raise OfficeError("Editor session expired or invalid; reopen the document", 401)
        return dict(row)

    def create_session(self, document_id: str, actor: str, editor_url: str) -> dict:
        doc = self.documents.get(document_id)
        if doc.format not in {"odt", "docx"}:
            raise OfficeError("Interactive office editing supports ODT and DOCX")
        token = secrets.token_urlsafe(48)
        expires = time.time() + self.SESSION_SECONDS
        with self.connection() as conn:
            conn.execute(
                "INSERT INTO office_editor_sessions VALUES (?, ?, ?, ?, ?)",
                (self.token_hash(token), document_id, actor, expires, doc.current_revision),
            )
        return {
            "document_id": document_id,
            "revision": doc.current_revision,
            "editor_url": editor_url,
            "access_token": token,
            "access_token_ttl": int(expires * 1000),
            "expires_at": expires,
        }

    def info(self, document_id: str, token: str) -> dict:
        session = self.authorize(document_id, token)
        doc = self.documents.get(document_id)
        data = self.documents.read_bytes(document_id, doc.current_revision)
        return {
            "BaseFileName": doc.filename,
            "OwnerId": "advocatio",
            "Size": len(data),
            "Version": str(doc.current_revision),
            "UserId": session["actor"],
            "UserFriendlyName": "Workspace editor",
            "UserCanWrite": True,
            "ReadOnly": False,
            "SupportsUpdate": True,
            "SupportsLocks": True,
            "SupportsGetLock": True,
            "UserCanNotWriteRelative": True,
            "DisableExport": False,
            "PostMessageOrigin": os.environ.get("OFFICE_HOST_ORIGIN", ""),
        }

    def read(self, document_id: str, token: str) -> tuple[bytes, int]:
        self.authorize(document_id, token)
        doc = self.documents.get(document_id)
        return self.documents.read_bytes(document_id, doc.current_revision), doc.current_revision

    @staticmethod
    def live_lock(conn, document_id: str):
        return conn.execute(
            "SELECT * FROM office_editor_locks WHERE document_id=? AND expires>?",
            (document_id, time.time()),
        ).fetchone()

    def lock(
        self,
        document_id: str,
        token: str,
        operation: str,
        value: str = "",
        old_value: str | None = None,
    ) -> str:
        session = self.authorize(document_id, token)
        if operation != "GET_LOCK" and (not value or len(value.encode()) > 1024):
            raise OfficeError("A lock value of at most 1024 bytes is required")
        # Obtain the document before taking the writer lock; save CAS rechecks revision.
        doc = self.documents.get(document_id)
        with self.connection() as conn:
            conn.execute("BEGIN IMMEDIATE")
            current = self.live_lock(conn, document_id)
            current_value = current["lock_value"] if current else ""
            if operation == "GET_LOCK":
                return current_value
            expected = old_value if old_value is not None else value
            if current and current_value != expected:
                raise OfficeError("Document is locked by another editor", 409, current_value)
            if (operation in {"UNLOCK", "REFRESH_LOCK"} or old_value is not None) and not current:
                raise OfficeError("Document lock expired", 409, "")
            if operation == "UNLOCK":
                conn.execute(
                    "UPDATE office_editor_locks SET expires=0 WHERE document_id=?", (document_id,)
                )
            elif operation in {"LOCK", "REFRESH_LOCK"}:
                if not current and doc.current_revision != session["revision"]:
                    raise OfficeError("Document changed; reopen the editor", 409, "")
                revision = current["revision"] if current else doc.current_revision
                conn.execute(
                    "INSERT INTO office_editor_locks VALUES (?, ?, ?, ?) "
                    "ON CONFLICT(document_id) DO UPDATE SET lock_value=excluded.lock_value, "
                    "expires=excluded.expires, revision=excluded.revision",
                    (document_id, value, time.time() + self.LOCK_SECONDS, revision),
                )
            else:
                raise OfficeError("Unsupported WOPI operation", 501)
        return ""

    def save(self, document_id: str, token: str, value: str, data: bytes):
        session = self.authorize(document_id, token)
        with self.connection() as conn:
            current = self.live_lock(conn, document_id)
        if current is None or current["lock_value"] != value:
            raise OfficeError(
                "A matching live editor lock is required",
                409,
                current["lock_value"] if current else "",
            )
        try:
            result = self.documents.save(
                document_id,
                data,
                current["revision"],
                actor=session["actor"],
                origin="office",
                office_lock=value,
            )
        except ValueError as exc:
            # Never silently overwrite an intervening save or expired/replaced lock.
            from legal_workspace.services.work_documents import DocumentLocked, RevisionConflict

            if isinstance(exc, (DocumentLocked, RevisionConflict)):
                with self.connection() as conn:
                    latest = self.live_lock(conn, document_id)
                raise OfficeError(
                    "Document changed or lock expired; reopen the editor",
                    409,
                    latest["lock_value"] if latest else "",
                ) from exc
            raise
        with self.connection() as conn:
            conn.execute(
                "UPDATE office_editor_sessions SET revision=? WHERE token_hash=?",
                (result.current_revision, self.token_hash(token)),
            )
        return result


@lru_cache(maxsize=8)
def _discovery(internal: str, minute: int) -> bytes:
    response = httpx.get(f"{internal}/hosting/discovery", timeout=10)
    response.raise_for_status()
    if len(response.content) > 2 * 1024 * 1024 or b"<!DOCTYPE" in response.content:
        raise ValueError("invalid discovery document")
    return response.content


def discover_editor_url(extension: str, document_id: str) -> str:
    """Use Collabora discovery, retaining versioned browser path and action parameters."""
    public = os.environ.get("OFFICE_EDITOR_PUBLIC_URL", "").rstrip("/")
    internal = os.environ.get("OFFICE_EDITOR_INTERNAL_URL", "").rstrip("/")
    callback = os.environ.get("OFFICE_WOPI_BASE_URL", "").rstrip("/")
    if not public or not internal or not callback:
        raise OfficeError("Interactive office service is not configured", 503)
    if urlsplit(public).scheme != "https":
        raise OfficeError("Office browser URL must use HTTPS", 503)
    try:
        root = ET.fromstring(_discovery(internal, int(time.time() // 60)))
    except (httpx.HTTPError, ET.ParseError, ValueError) as exc:
        raise OfficeError("Interactive office service is unavailable", 503) from exc
    action = next(
        (
            item
            for item in root.iter("action")
            if item.get("ext") == extension and item.get("name") == "edit"
        ),
        None,
    )
    if action is None or not action.get("urlsrc"):
        raise OfficeError("Office service does not support this document format", 503)
    src = urlsplit(action.get("urlsrc", ""))
    base = urlsplit(public)
    # Discovery may include optional <...> placeholders; omit unresolved values.
    params = [(k, v) for k, v in parse_qsl(src.query) if "<" not in k + v]
    params.append(("WOPISrc", f"{callback}/wopi/files/{document_id}"))
    prefix = base.path.rstrip("/")
    path = src.path if src.path.startswith(prefix + "/") else prefix + src.path
    return urlunsplit((base.scheme, base.netloc, path, urlencode(params), ""))
