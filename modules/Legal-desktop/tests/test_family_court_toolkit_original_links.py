"""Bounded original-link projection tests for the shared Workdesk record reader.

Byline: OpenAI Codex · GPT-6-Luna · 2026-10-05.
"""

from legal_workspace.services import family_court_toolkit as toolkit


def test_original_links_keep_exact_opaque_version_and_fixed_host(monkeypatch):
    """Preserve an opaque provider version and build the fixed hosted PDF URL.

    Inputs are synthetic binding metadata; output is one proxy envelope.
    The read-only query is mocked, so no shared store or source bytes are touched.
    """
    binding_id = "library_file:" + "a" * 64
    version_id = "opaque:version/one + two"
    monkeypatch.setenv("TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE", "case-scope")
    calls = []

    def query(surql, params):
        calls.append((surql, params))
        return [{
            "binding_id": binding_id,
            "original_pointer": {
                "version_id": version_id,
                "sha256": "b" * 64,
                "size": 1234,
                "content_type": "application/pdf",
            },
        }]

    monkeypatch.setattr(toolkit, "_query", query)
    links = toolkit._original_links("source:authority")

    assert len(links) == 1
    assert links[0].version_id == version_id
    assert links[0].bytes == 1234
    assert links[0].href == (
        "https://family-court.tilapia-skilift.ts.net/api/library/original?"
        "binding_id=library_file%3A" + "a" * 64 + "&version_id=opaque%3Aversion%2Fone+%2B+two"
    )
    assert "provider = 'b2'" in calls[0][0]
    assert "bucket = 'salem-data'" in calls[0][0]
    assert "KnowledgeBase/legal/" in calls[0][0]
    assert "FROM library_file_alias WHERE record_id = $record_id LIMIT 8" in calls[0][0]
    assert "record_id = $record_id OR <string> id IN $aliases" in calls[0][0]
    assert calls[0][1] == {"record_id": "source:authority", "scope": "case-scope"}


def test_alias_only_source_mapping_uses_scoped_library_file_metadata(monkeypatch):
    """Resolve a source alias through the same scoped B2 pointer validation.

    Inputs are an exact source ID and synthetic alias-resolved metadata; output
    is a hosted link. The query is mocked, so no shared store or source bytes
    are accessed.
    """
    monkeypatch.setenv("TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE", "case-scope")
    binding_id = "library_file:" + "f" * 64
    calls = []

    def query(surql, params):
        calls.append((surql, params))
        return [{
            "binding_id": binding_id,
            "original_pointer": {
                "version_id": "source-primary-pdf-v1",
                "sha256": "9" * 64,
                "size": 2048,
                "content_type": "application/pdf",
            },
        }]

    monkeypatch.setattr(toolkit, "_query", query)
    links = toolkit._original_links("source:primary-authority")

    assert len(links) == 1
    assert links[0].binding_id == binding_id
    assert links[0].version_id == "source-primary-pdf-v1"
    assert calls[0][1] == {"record_id": "source:primary-authority", "scope": "case-scope"}
    assert "library_file_alias" in calls[0][0]
    assert "account_scope = $scope" in calls[0][0]
    assert "legal_root = 'consignatio/casevault/KnowledgeBase/legal/'" in calls[0][0]


def test_malformed_locator_does_not_change_the_exact_record_version(monkeypatch):
    """Skip a malformed pointer while preserving the canonical record and version.

    Input is a synthetic exact record and an invalid mapped locator. Output
    proves the record body/version remain unchanged and links stay separate.
    """
    monkeypatch.setenv("TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE", "case-scope")
    version_id = "sha256:" + "c" * 64
    body = {"title": "Synthetic source", "body": "Exact source text", "id": "source:authority"}

    def query(surql, params):
        if "original_pointer" in surql:
            return [{
                "binding_id": "library_file:" + "d" * 64,
                "original_pointer": {
                    "version_id": "bad\nlocator",
                    "sha256": "e" * 64,
                    "size": 1234,
                    "content_type": "application/pdf",
                },
            }]
        return {"tb": "source", "id": "authority", "version": version_id, "record": body.copy()}

    monkeypatch.setattr(toolkit, "_query", query)
    result = toolkit.get_record("source:authority")

    assert result is not None
    assert result.version == version_id
    assert result.record == {"title": "Synthetic source", "body": "Exact source text"}
    assert result.original_links == []


def test_missing_mapping_returns_empty_links_and_preserves_unknown_body_fields(monkeypatch):
    """Treat a missing B2 binding as an empty envelope field without changing the record.

    Input is a synthetic record plus no mapping rows; output preserves its
    exact version and all existing fields, including unknown personal fields.
    """
    monkeypatch.setenv("TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE", "case-scope")
    version_id = "store-owned-version"
    body = {"display_name": "Exact name", "custom_field": {"unknown": True}}

    def query(surql, params):
        if "original_pointer" in surql:
            return []
        return {"tb": "source", "id": "personal", "version": version_id, "record": body.copy()}

    monkeypatch.setattr(toolkit, "_query", query)
    result = toolkit.get_record("source:personal")

    assert result is not None
    assert result.version == version_id
    assert result.record == body
    assert result.original_links == []
