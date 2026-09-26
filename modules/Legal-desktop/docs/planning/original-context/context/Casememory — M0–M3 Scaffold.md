<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# Casememory — M0–M3 Scaffold

Graphiti's `add_episode` triggers an async LLM extraction pipeline that produces entities/edges with `valid_at`/`invalid_at` fields captured during edge processing, and Zep's Fact Invalidation concept stores the invalidation timestamp on the edge itself rather than deleting it. This bi-temporal design (real-world validity time vs. transaction time when the system learned/unlearned a fact) is exactly the substrate T2 requires. Below is the full M0–M3 scaffold.[^1][^2][^3][^4][^5]

## 1. Directory Tree

```
casememory/
├── cli.py
├── config/
│   ├── config.py
│   └── .env.example
├── ingest/
│   ├── episode.py
│   ├── hasher.py
│   ├── identity.py
│   └── loaders/
│       ├── text_loader.py
│       ├── json_loader.py
│       └── message_loader.py
├── graph/
│   ├── client.py
│   ├── schema.py
│   ├── invalidation.py
│   └── versions.py
├── extraction/
│   ├── provider.py
│   ├── nim_provider.py
│   ├── portkey_gateway.py
│   └── lineage.py
├── audit/
│   ├── log.py
│   ├── verify.py
│   └── schema.sql
├── crypto/
│   └── __init__.py
├── retention/
│   └── __init__.py
├── observations/
│   └── __init__.py
├── mcp/
│   └── __init__.py
├── sdk/
│   └── python/
│       └── casememory/
│           ├── __init__.py
│           ├── client.py
│           └── types.py
├── infra/
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── coolify/
│   │   ├── README.md
│   │   └── service.yaml.example
│   └── .env.example
├── tests/
│   ├── test_t1_source_lineage.py
│   ├── test_t2_bitemporal.py
│   ├── test_t3_extraction_lineage.py
│   ├── test_t4_audit_chain.py
│   ├── test_t4_audit_tamper_detection.py
│   └── conftest.py
├── .github/
│   └── workflows/
│       └── ci.yml
├── docs/
│   ├── HANDOFF.md
│   ├── DEPLOYMENT.md
│   └── PROVIDERS.md
├── pyproject.toml
├── ruff.toml
└── README.md
```


## 2. Stub Files

```python
# casememory/config/config.py
"""
Coolify-style env-var config loader for Casememory.

All secrets are injected by Coolify's secret store as environment variables
into the container. This module is the ONLY place that reads os.environ for
runtime config; nothing else should call os.environ directly.

Satisfies: deployment assumption "Secrets managed by Coolify... via a small
config.py loader; never hard-code, never commit an .env value".
"""
from __future__ import annotations

import os
import subprocess
from dataclasses import dataclass, field
from functools import lru_cache


@dataclass(frozen=True)
class GraphConfig:
    bolt_uri: str
    user: str
    password: str
    database: str = "neo4j"  # DozerDB default; verify against dozerdb.org docs


@dataclass(frozen=True)
class LLMConfig:
    provider: str  # "nim" | "openai" | "anthropic" | "gemini"
    api_base: str
    api_key: str
    model: str
    embedding_model: str
    reranker_model: str | None = None
    portkey_api_key: str | None = None
    portkey_virtual_key: str | None = None

    @property
    def use_portkey(self) -> bool:
        return bool(self.portkey_api_key)


@dataclass(frozen=True)
class IdentityConfig:
    human_user_id: str
    service_name: str = "casememory"
    git_commit_hash: str = field(default_factory=lambda: _resolve_git_commit())


@dataclass(frozen=True)
class AuditConfig:
    sqlite_path: str


@dataclass(frozen=True)
class CasememoryConfig:
    graph: GraphConfig
    llm: LLMConfig
    identity: IdentityConfig
    audit: AuditConfig


def _resolve_git_commit() -> str:
    """
    Resolve the git commit hash of the running code (T1 ingestor identity).

    Order of resolution:
      1. GIT_COMMIT_HASH env var (set via Docker build-arg at image build time).
      2. `git rev-parse HEAD` shelled out at container start (fallback, dev only).
    TODO(T1): confirm Coolify passes build-args through to ENV; if not, bake
    the value into a file at build time (see infra/Dockerfile).
    """
    env_val = os.environ.get("GIT_COMMIT_HASH")
    if env_val:
        return env_val
    try:
        out = subprocess.run(
            ["git", "rev-parse", "HEAD"], capture_output=True, text=True, check=True
        )
        return out.stdout.strip()
    except Exception:
        return "unknown"


@lru_cache(maxsize=1)
def load_config() -> CasememoryConfig:
    """Load and validate all config from environment variables. Cached (singleton)."""
    graph = GraphConfig(
        bolt_uri=_require("DOZERDB_BOLT_URI"),
        user=_require("DOZERDB_USER"),
        password=_require("DOZERDB_PASSWORD"),
        database=os.environ.get("DOZERDB_DATABASE", "neo4j"),
    )
    llm = LLMConfig(
        provider=os.environ.get("LLM_PROVIDER", "nim"),
        api_base=_require("LLM_API_BASE"),
        api_key=_require("LLM_API_KEY"),
        model=_require("LLM_MODEL"),
        embedding_model=_require("LLM_EMBEDDING_MODEL"),
        reranker_model=os.environ.get("LLM_RERANKER_MODEL"),
        portkey_api_key=os.environ.get("PORTKEY_API_KEY"),
        portkey_virtual_key=os.environ.get("PORTKEY_VIRTUAL_KEY"),
    )
    identity = IdentityConfig(human_user_id=_require("CASEMEMORY_HUMAN_USER_ID"))
    audit = AuditConfig(sqlite_path=os.environ.get("AUDIT_SQLITE_PATH", "/data/audit.db"))
    return CasememoryConfig(graph=graph, llm=llm, identity=identity, audit=audit)


def _require(name: str) -> str:
    val = os.environ.get(name)
    if not val:
        raise RuntimeError(f"Missing required env var: {name}")
    return val
```

```bash
# casememory/config/.env.example
# Placeholders only. Real values live in Coolify's secret store as env vars.

DOZERDB_BOLT_URI=bolt://dozerdb:7687
DOZERDB_USER=neo4j
DOZERDB_PASSWORD=
DOZERDB_DATABASE=neo4j

LLM_PROVIDER=nim
LLM_API_BASE=https://integrate.api.nvidia.com/v1
LLM_API_KEY=
LLM_MODEL=
LLM_EMBEDDING_MODEL=
LLM_RERANKER_MODEL=

# Optional — set to route all model traffic through Portkey
PORTKEY_API_KEY=
PORTKEY_VIRTUAL_KEY=

CASEMEMORY_HUMAN_USER_ID=matthew-salem
AUDIT_SQLITE_PATH=/data/audit.db

# Set at build time via Docker build-arg; do not set manually in prod
GIT_COMMIT_HASH=
```

```python
# casememory/ingest/episode.py
"""
Episode dataclass carrying T1 source-lineage fields.

Mirrors Zep/Graphiti's "Episode" concept (help.getzep.com/concepts) while
extending it with chain-of-custody metadata required by T1.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from enum import Enum


class EpisodeType(str, Enum):
    """// verify against graphiti-core EpisodeType (text | json | message)."""

    TEXT = "text"
    JSON = "json"
    MESSAGE = "message"


@dataclass(frozen=True)
class IngestorIdentity:
    """T1: who/what performed ingestion."""

    human_user_id: str
    service_name: str
    git_commit_hash: str


@dataclass(frozen=True)
class SourceLineage:
    """
    T1 fields captured per ingested episode.

    - sha256_hash: hex digest of raw content bytes (see ingest/hasher.py).
    - original_path: file path or message identifier from the source system.
    - ingestion_timestamp: UTC ISO-8601, time the episode was received here.
    - ingestor: IngestorIdentity (human, service, git commit).
    - mime_type / byte_length: describe the raw content.
    - signature: optional; null in M1, wired for a future BYOK signing step.
    """

    sha256_hash: str
    original_path: str
    ingestion_timestamp: datetime
    ingestor: IngestorIdentity
    mime_type: str
    byte_length: int
    signature: str | None = None


@dataclass(frozen=True)
class Episode:
    """
    A single unit of content to be added to the graph via graph.add
    (Graphiti's add_episode). content is stored verbatim, never mutated.
    """

    episode_id: str
    episode_type: EpisodeType
    content: str
    name: str
    source_description: str
    lineage: SourceLineage

    def to_graphiti_kwargs(self) -> dict:
        """
        TODO(T1): map to graphiti_core.Graphiti.add_episode(...) kwargs
        (name, episode_body, source, source_description, reference_time).
        // verify against graphiti-core add_episode signature —
        https://help.getzep.com/graphiti/core-concepts/adding-episodes
        """
        raise NotImplementedError
```

```python
# casememory/ingest/hasher.py
"""SHA-256 content hashing for T1 source lineage."""
from __future__ import annotations

import hashlib


def sha256_hex(raw: bytes) -> str:
    """Return the hex-encoded SHA-256 digest of raw content bytes."""
    return hashlib.sha256(raw).hexdigest()


def sha256_hex_of_text(text: str, encoding: str = "utf-8") -> str:
    """Convenience wrapper: hash of a text payload using a fixed encoding."""
    return sha256_hex(text.encode(encoding))
```

```python
# casememory/ingest/identity.py
"""
Ingestor identity resolution for T1.

Resolves the three identity components:
  (a) human user id — from config.identity.human_user_id
  (b) service name — from config.identity.service_name
  (c) git commit hash — from config.identity.git_commit_hash (build-arg or
      `git rev-parse HEAD`, resolved once in config.py at process start)
"""
from __future__ import annotations

from casememory.config.config import CasememoryConfig
from casememory.ingest.episode import IngestorIdentity


def resolve_ingestor_identity(config: CasememoryConfig) -> IngestorIdentity:
    """Build an IngestorIdentity snapshot from the loaded config."""
    return IngestorIdentity(
        human_user_id=config.identity.human_user_id,
        service_name=config.identity.service_name,
        git_commit_hash=config.identity.git_commit_hash,
    )
```

```python
# casememory/ingest/loaders/text_loader.py
"""Loader that turns a raw text file/string into an Episode with T1 lineage."""
from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path
from uuid import uuid4

from casememory.config.config import CasememoryConfig
from casememory.ingest.episode import Episode, EpisodeType, SourceLineage
from casememory.ingest.hasher import sha256_hex
from casememory.ingest.identity import resolve_ingestor_identity


def load_text_file(path: Path, config: CasememoryConfig) -> Episode:
    """
    Read a text file from disk, compute T1 lineage, and build an Episode.

    TODO(T1): byte_length must be len(raw_bytes), not len(decoded string).
    """
    raw_bytes = path.read_bytes()
    lineage = SourceLineage(
        sha256_hash=sha256_hex(raw_bytes),
        original_path=str(path),
        ingestion_timestamp=datetime.now(timezone.utc),
        ingestor=resolve_ingestor_identity(config),
        mime_type="text/plain",
        byte_length=len(raw_bytes),
    )
    return Episode(
        episode_id=str(uuid4()),
        episode_type=EpisodeType.TEXT,
        content=raw_bytes.decode("utf-8"),
        name=path.name,
        source_description=f"text file: {path}",
        lineage=lineage,
    )
```

```python
# casememory/ingest/loaders/json_loader.py
"""Loader that turns a JSON file/payload into an Episode with T1 lineage."""
from __future__ import annotations

import json
from datetime import datetime, timezone
from pathlib import Path
from uuid import uuid4

from casememory.config.config import CasememoryConfig
from casememory.ingest.episode import Episode, EpisodeType, SourceLineage
from casememory.ingest.hasher import sha256_hex
from casememory.ingest.identity import resolve_ingestor_identity


def load_json_file(path: Path, config: CasememoryConfig) -> Episode:
    """
    Read a JSON file, validate it parses, compute T1 lineage.

    TODO(T3 boundary): the raw JSON string (not a re-serialized version) is
    what must be stored as episode content so hashes remain verifiable.
    """
    raw_bytes = path.read_bytes()
    json.loads(raw_bytes)  # validate only; do not mutate/reformat
    lineage = SourceLineage(
        sha256_hash=sha256_hex(raw_bytes),
        original_path=str(path),
        ingestion_timestamp=datetime.now(timezone.utc),
        ingestor=resolve_ingestor_identity(config),
        mime_type="application/json",
        byte_length=len(raw_bytes),
    )
    return Episode(
        episode_id=str(uuid4()),
        episode_type=EpisodeType.JSON,
        content=raw_bytes.decode("utf-8"),
        name=path.name,
        source_description=f"json file: {path}",
        lineage=lineage,
    )
```

```python
# casememory/ingest/loaders/message_loader.py
"""Loader for chat/message-style episodes (Zep 'Thread' analogue)."""
from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

from casememory.config.config import CasememoryConfig
from casememory.ingest.episode import Episode, EpisodeType, SourceLineage
from casememory.ingest.hasher import sha256_hex_of_text
from casememory.ingest.identity import resolve_ingestor_identity


def load_message(
    message_id: str, role: str, content: str, config: CasememoryConfig
) -> Episode:
    """
    Build an Episode from a single message. original_path is the
    message_id (e.g. "thread:<id>:msg:<n>") per T1's "message identifier".

    TODO(T3 boundary): this maps toward thread.add_messages in the future
    SDK layer (sdk/python/casememory/client.py); M1 only builds the Episode.
    // verify thread.add_messages semantics: https://help.getzep.com/concepts
    """
    lineage = SourceLineage(
        sha256_hash=sha256_hex_of_text(content),
        original_path=message_id,
        ingestion_timestamp=datetime.now(timezone.utc),
        ingestor=resolve_ingestor_identity(config),
        mime_type="text/plain",
        byte_length=len(content.encode("utf-8")),
    )
    return Episode(
        episode_id=str(uuid4()),
        episode_type=EpisodeType.MESSAGE,
        content=content,
        name=f"{role}:{message_id}",
        source_description=f"message from role={role}",
        lineage=lineage,
    )
```

```python
# casememory/graph/client.py
"""
Graphiti-core wrapper connecting to DozerDB via Neo4j Bolt.

DozerDB is a Neo4j-Enterprise-compatible fork (https://dozerdb.org); Graphiti
requires Neo4j 5.26+ semantics, so the pinned DozerDB image tag must track
that parity (documented in docs/HANDOFF.md and infra/docker-compose.yml).
"""
from __future__ import annotations

from typing import Any, Protocol

from casememory.config.config import GraphConfig
from casememory.ingest.episode import Episode


class GraphClient(Protocol):
    """Public seam so extraction/audit layers depend on a protocol, not a concrete driver."""

    async def add_episode(self, episode: Episode) -> dict[str, Any]:
        ...

    async def search(self, query: str, **kwargs: Any) -> list[dict[str, Any]]:
        ...

    async def close(self) -> None:
        ...


class GraphitiDozerDBClient:
    """
    Concrete GraphClient backed by graphiti_core.Graphiti + Neo4jDriver.

    TODO(graph.add): construct graphiti_core.Graphiti(uri, user, password)
    and call .add_episode(...). // verify exact constructor signature against
    graphiti-core — https://github.com/getzep/graphiti
    TODO: wire extraction.provider.LLMProvider/Embedder/Reranker into the
    Graphiti instance's llm_client / embedder / cross_encoder args.
    """

    def __init__(self, config: GraphConfig) -> None:
        self._config = config
        self._driver: Any = None  # TODO: graphiti_core.driver.Neo4jDriver

    async def connect(self) -> None:
        """Open the Bolt connection and run Graphiti's build_indices_and_constraints()."""
        raise NotImplementedError

    async def add_episode(self, episode: Episode) -> dict[str, Any]:
        """
        Call graphiti_core add_episode with episode.content verbatim.
        Extraction lineage (T3) must be captured by the LLMProvider wrapper,
        not here; this method only orchestrates the call and returns the
        raw Graphiti result for audit logging.
        """
        raise NotImplementedError

    async def search(self, query: str, **kwargs: Any) -> list[dict[str, Any]]:
        """// verify against graphiti-core Graphiti.search / graph.search (Zep concepts)."""
        raise NotImplementedError

    async def close(self) -> None:
        raise NotImplementedError
```

```python
# casememory/graph/schema.py
"""
T2/T3 field extensions layered onto Graphiti's EntityNode / EntityEdge.

Graphiti's core EntityEdge already carries valid_at / invalid_at /
created_at / expired_at per Zep's bi-temporal model (blog.getzep.com/
beyond-static-knowledge-graphs). This module defines Casememory's
*additional* attributes (T3 extraction lineage, T1 episode back-reference)
stored as edge/node properties, using Graphiti's custom entity/edge type
mechanism.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class BitemporalFields:
    """
    T2 — mirrors graphiti_core.edges.EntityEdge temporal fields.
    // verify field names against graphiti-core EntityEdge —
    https://help.getzep.com/graphiti/working-with-data/
    """

    valid_at: datetime | None
    invalid_at: datetime | None
    created_at: datetime
    expired_at: datetime | None


@dataclass(frozen=True)
class ExtractionLineageRef:
    """T3 — foreign-key style reference from an edge/node to its extraction record."""

    extraction_record_id: str
    source_episode_id: str
    span_start: int
    span_end: int


# TODO(T3): register as a Graphiti "custom edge type" so extraction lineage
# rides alongside each Fact/EntityEdge instance.
# // verify custom edge type registration API — help.getzep.com/graphiti/working-with-data/
CUSTOM_EDGE_TYPE_NAME = "CasememoryFact"
```

```python
# casememory/graph/invalidation.py
"""
Append-only invalidation logic (T2).

Nothing is ever hard-deleted. "Invalidating" a fact means writing
invalid_at/expired_at on the existing edge and, where Graphiti's own
contradiction-resolution creates a *new* edge, letting that new edge exist
alongside the old one — never overwriting or removing the old edge.
"""
from __future__ import annotations

from datetime import datetime
from typing import Protocol


class InvalidatesFact(Protocol):
    async def __call__(self, fact_uuid: str, invalid_at: datetime, reason: str) -> str:
        ...


async def invalidate_fact(fact_uuid: str, invalid_at: datetime, reason: str) -> str:
    """
    Mark a fact edge invalid as of invalid_at. Returns the audit-log operation
    id for T4 (audit/log.py must be called by the orchestrating layer, not
    here, to keep graph and audit writes decoupled per the spec).

    TODO(T2): call graphiti_core's own invalidation path if it exposes one
    directly, else set invalid_at/expired_at via a Cypher MERGE that only
    SETs those two properties — never DELETE, never SET valid_at retroactively.
    // verify against https://help.getzep.com/facts (Fact Invalidation)
    """
    raise NotImplementedError


def assert_append_only(operation: str) -> None:
    """Guard used by graph/client.py callers: raise if operation implies mutation/deletion."""
    forbidden = {"DELETE", "DETACH DELETE", "REMOVE"}
    if operation.strip().upper().startswith(tuple(forbidden)):
        raise RuntimeError(f"Forbidden mutating operation attempted: {operation!r}")
```

```python
# casememory/graph/versions.py
"""get_fact_versions(fact_uuid) — T2 requirement to query all historical versions of a fact."""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class FactVersion:
    fact_uuid: str
    content: str
    valid_at: datetime | None
    invalid_at: datetime | None
    created_at: datetime
    expired_at: datetime | None


async def get_fact_versions(fact_uuid: str) -> list[FactVersion]:
    """
    Return every version of a fact (original + superseding edges/invalidation
    records) ordered by created_at ascending. Because invalidation is
    append-only (graph/invalidation.py), this must never lose a row.

    TODO(T2): implement as a Cypher query across all edges sharing a lineage
    key (e.g. `fact_group_id`) rather than a single edge uuid, since
    invalidation may produce sibling edges rather than mutating one edge.
    // verify against graphiti-core's edge/episode versioning model —
    https://help.getzep.com/graphiti/working-with-data/
    """
    raise NotImplementedError
```

```python
# casememory/extraction/provider.py
"""
Provider-agnostic protocols for LLM / Embedder / Reranker.

Design goal: swapping NIM -> OpenAI -> Anthropic -> Gemini is a config
change only. If a swap ever requires touching this file's call sites, the
protocol is wrong per the prompt's own constraint.
"""
from __future__ import annotations

from typing import Any, Protocol


class LLMCallResult(Protocol):
    """Normalized shape every provider implementation must return for T3 lineage capture."""

    text: str
    raw_response: Any
    request_id: str | None
    confidence: float | None


class LLMProvider(Protocol):
    async def complete(self, prompt: str, **kwargs: Any) -> LLMCallResult:
        ...


class Embedder(Protocol):
    async def embed(self, texts: list[str]) -> list[list[float]]:
        ...


class Reranker(Protocol):
    async def rerank(self, query: str, documents: list[str]) -> list[float]:
        ...
```

```python
# casememory/extraction/nim_provider.py
"""NVIDIA NIM implementation of LLMProvider / Embedder / Reranker."""
from __future__ import annotations

from typing import Any

from casememory.config.config import LLMConfig
from casememory.extraction.provider import Embedder, LLMProvider, Reranker


class NIMLLMProvider(LLMProvider):
    """
    Talks to NVIDIA NIM hosted endpoints (OpenAI-compatible chat completions).
    // verify request/response shape against https://docs.nvidia.com/nim
    """

    def __init__(self, config: LLMConfig) -> None:
        self._config = config

    async def complete(self, prompt: str, **kwargs: Any) -> Any:
        """
        TODO: POST to config.api_base chat/completions with config.model.
        If config.use_portkey, this call must instead go through
        extraction.portkey_gateway.PortkeyGateway — decided by the factory
        in this module, not by branching in graph/client.py.
        """
        raise NotImplementedError


class NIMEmbedder(Embedder):
    async def embed(self, texts: list[str]) -> list[list[float]]:
        """TODO: call config.embedding_model endpoint. // verify docs.nvidia.com/nim embeddings API"""
        raise NotImplementedError


class NIMReranker(Reranker):
    async def rerank(self, query: str, documents: list[str]) -> list[float]:
        """TODO: call config.reranker_model endpoint, or return NotImplementedError if unset."""
        raise NotImplementedError


def build_provider_from_config(config: LLMConfig) -> LLMProvider:
    """
    Factory: the ONLY place that decides direct-NIM vs Portkey-routed.
    Downstream code always depends on LLMProvider, never on this class.
    """
    if config.use_portkey:
        from casememory.extraction.portkey_gateway import PortkeyGateway

        return PortkeyGateway(inner=NIMLLMProvider(config), config=config)
    return NIMLLMProvider(config)
```

```python
# casememory/extraction/portkey_gateway.py
"""
Optional Portkey gateway wrapper.

When PORTKEY_API_KEY is set, all outbound model traffic is routed through
Portkey so a court can later cross-reference Portkey's own request logs by
request_id (T3). Wraps any LLMProvider without changing its call signature.
"""
from __future__ import annotations

from typing import Any

from casememory.config.config import LLMConfig
from casememory.extraction.provider import LLMProvider


class PortkeyGateway(LLMProvider):
    """
    Decorator around an inner LLMProvider that rewrites the HTTP call to go
    through Portkey's gateway (base URL + virtual key headers), and extracts
    the Portkey request id from response headers for T3.
    // verify header names / base URL pattern against https://docs.portkey.ai
    """

    def __init__(self, inner: LLMProvider, config: LLMConfig) -> None:
        self._inner = inner
        self._config = config

    async def complete(self, prompt: str, **kwargs: Any) -> Any:
        """
        TODO: set headers x-portkey-api-key, x-portkey-virtual-key
        (config.portkey_virtual_key) and read x-portkey-request-id (or
        equivalent) from the response for lineage.py to store.
        """
        raise NotImplementedError
```

```python
# casememory/extraction/lineage.py
"""
T3 extraction lineage capture.

Every LLM call that produces an entity or edge must be wrapped so this
module records the full verbatim record before the caller ever sees the
parsed result.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class ExtractionRecord:
    """
    T3 fields:
      - provider_name: "nim" | "openai" | "portkey-routed"
      - model_name: model identifier + version string as returned/sent
      - prompt_verbatim / response_verbatim: exact strings, no normalization
      - start_time / end_time: UTC
      - span_start / span_end: char offsets into the source episode content
      - confidence: float or None (explicitly null-marked, not omitted)
      - portkey_request_id: None unless routed through Portkey
    """

    extraction_id: str
    source_episode_id: str
    provider_name: str
    model_name: str
    prompt_verbatim: str
    response_verbatim: str
    start_time: datetime
    end_time: datetime
    span_start: int
    span_end: int
    confidence: float | None
    confidence_is_null: bool
    portkey_request_id: str | None


async def capture_extraction(
    *,
    source_episode_id: str,
    provider_name: str,
    model_name: str,
    prompt: str,
    call_fn,
    span_start: int,
    span_end: int,
) -> ExtractionRecord:
    """
    Wrap a single LLM call, timing it and recording the verbatim prompt and
    response. Callers in graph/client.py must persist the returned
    ExtractionRecord via audit/log.py (operation="extraction") AND attach
    extraction_id to the resulting EntityEdge/EntityNode (graph/schema.py
    ExtractionLineageRef).

    TODO(T3): call_fn is expected to be an LLMProvider.complete bound call;
    exact wiring against graphiti-core's extraction hooks needs verification.
    // verify against graphiti-core prompt library extraction call sites —
    https://github.com/getzep/graphiti
    """
    raise NotImplementedError
```

```python
# casememory/audit/schema.sql
-- T4 append-only audit log schema. SQLite, WAL mode.
-- One row per ingestion, extraction, search, invalidation, or admin action.
-- This file is applied once at first boot (see audit/log.py:init_db).

PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS audit_log (
    row_id            INTEGER PRIMARY KEY AUTOINCREMENT,
    ts_utc            TEXT    NOT NULL,          -- ISO-8601 UTC
    actor             TEXT    NOT NULL,          -- human_user_id or service_name
    operation         TEXT    NOT NULL,          -- ingestion | extraction | search | invalidation | admin
    target_uuids      TEXT    NOT NULL,          -- JSON array of UUID strings
    payload_json       TEXT    NOT NULL,          -- operation-specific JSON payload, verbatim
    prev_hash         TEXT,                      -- hash of previous row, NULL for row 1
    row_hash          TEXT    NOT NULL           -- sha256(canonical_row || prev_hash)
);

CREATE INDEX IF NOT EXISTS idx_audit_log_ts ON audit_log(ts_utc);
CREATE INDEX IF NOT EXISTS idx_audit_log_operation ON audit_log(operation);
```

```python
# casememory/audit/log.py
"""
T4 — append-only Merkle-chained audit log in SQLite (WAL mode).

This DB is a SEPARATE file from the graph DB; the graph never writes to it
directly. Every ingestion, extraction, search, invalidation, or admin action
elsewhere in the codebase must call append_entry() here.
"""
from __future__ import annotations

import hashlib
import json
import sqlite3
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path


@dataclass(frozen=True)
class AuditEntry:
    ts_utc: str
    actor: str
    operation: str
    target_uuids: list[str]
    payload: dict


def init_db(sqlite_path: str) -> sqlite3.Connection:
    """Create the DB file + schema (audit/schema.sql) if not present, enable WAL."""
    conn = sqlite3.connect(sqlite_path, isolation_level=None)
    conn.execute("PRAGMA journal_mode=WAL;")
    schema_sql = (Path(__file__).parent / "schema.sql").read_text()
    conn.executescript(schema_sql)
    return conn


def _canonical_row_repr(entry: AuditEntry) -> str:
    """Deterministic JSON serialization used as Merkle-chain input."""
    return json.dumps(
        {
            "ts_utc": entry.ts_utc,
            "actor": entry.actor,
            "operation": entry.operation,
            "target_uuids": sorted(entry.target_uuids),
            "payload": entry.payload,
        },
        sort_keys=True,
        separators=(",", ":"),
    )


def append_entry(conn: sqlite3.Connection, entry: AuditEntry) -> str:
    """
    Append one row: compute row_hash = sha256(canonical_row + prev_hash),
    chaining to the previous row's hash. Never UPDATE or DELETE existing rows.
    Returns the new row_hash.
    """
    cur = conn.execute("SELECT row_hash FROM audit_log ORDER BY row_id DESC LIMIT 1;")
    row = cur.fetchone()
    prev_hash = row[^0] if row else ""
    canonical = _canonical_row_repr(entry)
    row_hash = hashlib.sha256((canonical + prev_hash).encode("utf-8")).hexdigest()
    conn.execute(
        "INSERT INTO audit_log (ts_utc, actor, operation, target_uuids, payload_json, "
        "prev_hash, row_hash) VALUES (?, ?, ?, ?, ?, ?, ?);",
        (
            entry.ts_utc,
            entry.actor,
            entry.operation,
            json.dumps(entry.target_uuids),
            json.dumps(entry.payload),
            prev_hash or None,
            row_hash,
        ),
    )
    return row_hash


def now_iso_utc() -> str:
    return datetime.now(timezone.utc).isoformat()
```

```python
# casememory/audit/verify.py
"""`audit verify` CLI subcommand — tamper detection by recomputing the Merkle chain."""
from __future__ import annotations

import json
import sqlite3
from dataclasses import dataclass

from casememory.audit.log import AuditEntry, _canonical_row_repr
import hashlib


@dataclass(frozen=True)
class VerificationResult:
    ok: bool
    total_rows: int
    first_bad_row_id: int | None
    detail: str


def verify_chain(conn: sqlite3.Connection) -> VerificationResult:
    """
    Recompute row_hash for every row in row_id order and compare to the
    stored value; also check that stored prev_hash matches the actual
    previous row's row_hash. First mismatch is reported and iteration stops.
    """
    cur = conn.execute(
        "SELECT row_id, ts_utc, actor, operation, target_uuids, payload_json, "
        "prev_hash, row_hash FROM audit_log ORDER BY row_id ASC;"
    )
    prev_hash = ""
    count = 0
    for row_id, ts_utc, actor, operation, target_uuids_json, payload_json, stored_prev, stored_hash in cur:
        count += 1
        entry = AuditEntry(
            ts_utc=ts_utc,
            actor=actor,
            operation=operation,
            target_uuids=json.loads(target_uuids_json),
            payload=json.loads(payload_json),
        )
        expected_prev = stored_prev or ""
        if expected_prev != prev_hash:
            return VerificationResult(False, count, row_id, "prev_hash mismatch")
        canonical = _canonical_row_repr(entry)
        recomputed = hashlib.sha256((canonical + prev_hash).encode("utf-8")).hexdigest()
        if recomputed != stored_hash:
            return VerificationResult(False, count, row_id, "row_hash mismatch — tamper detected")
        prev_hash = stored_hash
    return VerificationResult(True, count, None, "chain intact")
```

```python
# casememory/crypto/__init__.py
"""
Interface boundary for the deferred BYOK envelope-encryption feature.

M0-M3 only defines the seam; implementation is out of scope here.
Future work must satisfy KeySource without changing callers in
ingest/episode.py (SourceLineage.signature) or extraction/lineage.py.
"""
from __future__ import annotations

from typing import Protocol


class KeySource(Protocol):
    """A future BYOK key provider. No implementation in M0-M3."""

    async def get_signing_key(self) -> bytes:
        ...

    async def get_encryption_key(self) -> bytes:
        ...
```

```python
# casememory/retention/__init__.py
"""Interface boundary for the deferred retention/legal-hold engine. No implementation here."""
from __future__ import annotations

from typing import Protocol


class RetentionPolicy(Protocol):
    """Future policy object; must be consultable before any purge-eligible operation (there are none in M0-M3, since nothing is ever deleted)."""

    def is_legal_hold(self, target_uuid: str) -> bool:
        ...

    def retention_expiry(self, target_uuid: str) -> "datetime | None":  # noqa: F821
        ...
```

```python
# casememory/observations/__init__.py
"""Interface boundary for deferred Observation detectors (Zep 'Observation' concept). No implementation here."""
from __future__ import annotations

from typing import Protocol


class ObservationDetector(Protocol):
    """Future scheduled-job detector producing Observations from the graph."""

    async def detect(self, since: "datetime") -> list[dict]:  # noqa: F821
        ...
```

```python
# casememory/mcp/__init__.py
"""
Placeholder for the deferred Graphiti MCP-server-style overlay
(github.com/getzep/graphiti/blob/main/mcp_server/README.md).

This module will host an MCP server exposing graph.add / graph.search /
thread.get_user_context as MCP tools, backed by sdk/python/casememory/client.py.
Not built in M0-M3.
"""
```

```python
# casememory/sdk/python/casememory/types.py
"""Typed request/response models for the public Casememory client API."""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class AddEpisodeResult:
    episode_id: str
    audit_row_hash: str


@dataclass(frozen=True)
class FactVersionDTO:
    fact_uuid: str
    content: str
    valid_at: datetime | None
    invalid_at: datetime | None
    created_at: datetime
    expired_at: datetime | None


@dataclass(frozen=True)
class ContextBlock:
    """Mirrors Zep's Context Block concept (help.getzep.com/concepts)."""

    text: str
    facts: list[str]
```

```python
# casememory/sdk/python/casememory/client.py
"""
Public typed client API — the contract the future Agno toolkit wraps.

Mirrors the shape of agno.tools.zep.ZepTools (docs.agno.com/examples/
integrations/memory/zep-integration): add_episode, add_message,
search_memory, get_context_block, get_fact_versions. Renaming/wrapping
this later should require no structural changes.
"""
from __future__ import annotations

from casememory.sdk.python.casememory.types import (
    AddEpisodeResult,
    ContextBlock,
    FactVersionDTO,
)


class CasememoryClient:
    """
    Thin typed facade over graph/client.py + audit/log.py + extraction/*.
    All methods are async because Graphiti's core operations are async.
    """

    def __init__(self, config) -> None:  # config: CasememoryConfig
        self._config = config

    async def add_episode(
        self, *, name: str, content: str, source_description: str, original_path: str
    ) -> AddEpisodeResult:
        """TODO: build ingest.episode.Episode, call graph/client.add_episode, log to audit."""
        raise NotImplementedError

    async def add_message(self, *, thread_id: str, role: str, content: str) -> AddEpisodeResult:
        """TODO: delegate to ingest/loaders/message_loader.py then add_episode path."""
        raise NotImplementedError

    async def search_memory(self, *, query: str, limit: int = 10) -> list[dict]:
        """TODO: delegate to graph/client.search; log a 'search' audit row (T4)."""
        raise NotImplementedError

    async def get_context_block(self, *, thread_id: str) -> ContextBlock:
        """// verify against Zep thread.get_user_context — help.getzep.com/concepts"""
        raise NotImplementedError

    async def get_fact_versions(self, *, fact_uuid: str) -> list[FactVersionDTO]:
        """TODO: delegate to graph/versions.get_fact_versions and map to DTO."""
        raise NotImplementedError
```

```python
# casememory/sdk/python/casememory/__init__.py
"""Public Casememory Python SDK package."""
from casememory.sdk.python.casememory.client import CasememoryClient

__all__ = ["CasememoryClient"]
```

```python
# casememory/cli.py
"""Typer CLI entrypoint for Casememory: ingest, search, audit verify, healthz check."""
from __future__ import annotations

import typer

app = typer.Typer(help="Casememory CLI")
audit_app = typer.Typer(help="Audit log operations")
app.add_typer(audit_app, name="audit")


@app.command()
def ingest(path: str, kind: str = typer.Option("text", help="text|json|message")) -> None:
    """TODO: dispatch to ingest/loaders/* based on kind, then CasememoryClient.add_episode."""
    raise NotImplementedError


@app.command()
def search(query: str) -> None:
    """TODO: call CasememoryClient.search_memory and print results."""
    raise NotImplementedError


@audit_app.command("verify")
def audit_verify() -> None:
    """TODO: open audit sqlite via config, call audit/verify.verify_chain, print result."""
    raise NotImplementedError


@app.command()
def healthz() -> None:
    """Print OK if config loads and graph/audit connections succeed. Used by Docker HEALTHCHECK."""
    raise NotImplementedError


if __name__ == "__main__":
    app()
```

```python
# casememory/tests/conftest.py
"""Shared pytest fixtures: DozerDB test fixture (or mock), stub LLM provider."""
from __future__ import annotations

import pytest


@pytest.fixture
def stub_llm_provider():
    """
    Deterministic fake implementing extraction.provider.LLMProvider for
    T3 tests — returns a fixed prompt/response pair with a known request_id.
    """
    raise NotImplementedError


@pytest.fixture
def dozerdb_fixture():
    """
    TODO: spin up (or connect to) a disposable DozerDB instance for
    integration tests; skip/mark xfail if DOZERDB_TEST_URI is unset in CI.
    """
    raise NotImplementedError


@pytest.fixture
def tmp_audit_db(tmp_path):
    """Return a path to a fresh SQLite audit DB for T4 tests."""
    return tmp_path / "audit_test.db"
```

```python
# casememory/tests/test_t1_source_lineage.py
"""T1: verify every ingested episode captures hash, path, timestamp, identity, mime, length."""
from __future__ import annotations


def test_text_loader_captures_sha256_and_identity(tmp_path):
    """TODO: write a temp text file, load via ingest.loaders.text_loader, assert lineage fields."""
    raise NotImplementedError


def test_signature_field_is_null_in_m1(tmp_path):
    """TODO: assert SourceLineage.signature is None until BYOK phase implements it."""
    raise NotImplementedError
```

```python
# casememory/tests/test_t2_bitemporal.py
"""T2: verify invalidation is additive and get_fact_versions never loses a prior version."""
from __future__ import annotations


def test_invalidate_fact_does_not_delete_prior_edge():
    """TODO: assert graph/invalidation.assert_append_only raises on DELETE-style ops."""
    raise NotImplementedError


def test_get_fact_versions_returns_all_prior_versions():
    """TODO: seed multiple versions, assert graph/versions.get_fact_versions returns all, ordered."""
    raise NotImplementedError
```

```python
# casememory/tests/test_t3_extraction_lineage.py
"""T3: verify every LLM call is recorded verbatim with span, timestamps, provider, request id."""
from __future__ import annotations


def test_capture_extraction_records_verbatim_prompt_and_response(stub_llm_provider):
    """TODO: call extraction.lineage.capture_extraction with stub_llm_provider, assert equality."""
    raise NotImplementedError


def test_confidence_null_is_explicitly_marked():
    """TODO: assert confidence_is_null=True when provider returns no confidence score."""
    raise NotImplementedError
```

```python
# casememory/tests/test_t4_audit_chain.py
"""T4: verify Merkle-style hash chaining across sequential audit_log rows."""
from __future__ import annotations

from casememory.audit.log import AuditEntry, append_entry, init_db, now_iso_utc


def test_chain_links_rows(tmp_audit_db):
    conn = init_db(str(tmp_audit_db))
    h1 = append_entry(
        conn,
        AuditEntry(now_iso_utc(), "matthew-salem", "ingestion", ["uuid-1"], {"foo": "bar"}),
    )
    h2 = append_entry(
        conn,
        AuditEntry(now_iso_utc(), "casememory", "extraction", ["uuid-2"], {"baz": 1}),
    )
    assert h1 != h2
    row = conn.execute("SELECT prev_hash FROM audit_log WHERE row_hash = ?;", (h2,)).fetchone()
    assert row[^0] == h1
```

```python
# casememory/tests/test_t4_audit_tamper_detection.py
"""T4: verify audit/verify.verify_chain detects tampering by recomputation."""
from __future__ import annotations

from casememory.audit.log import AuditEntry, append_entry, init_db, now_iso_utc
from casememory.audit.verify import verify_chain


def test_tamper_detected_on_payload_edit(tmp_audit_db):
    conn = init_db(str(tmp_audit_db))
    append_entry(conn, AuditEntry(now_iso_utc(), "matthew-salem", "ingestion", ["uuid-1"], {"a": 1}))
    conn.execute("UPDATE audit_log SET payload_json = ? WHERE row_id = 1;", ('{"a": 999}',))
    result = verify_chain(conn)
    assert result.ok is False
    assert result.first_bad_row_id == 1
```

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  lint-type-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.13"
      - run: pip install -e ".[dev]"
      - run: ruff check .
      - run: mypy casememory
      - run: pytest -q
```

```yaml
# casememory/infra/docker-compose.yml
version: "3.9"
services:
  dozerdb:
    image: graphstack/dozerdb:5.24.0-1.0  # TODO: pin to a verified DozerDB tag — https://dozerdb.org
    environment:
      NEO4J_AUTH: ${DOZERDB_USER}/${DOZERDB_PASSWORD}
    volumes:
      - dozerdb_data:/data
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:7474"]
      interval: 15s
      timeout: 5s
      retries: 5
    # No public port publishing — bind only to the tailnet interface via
    # Coolify network config; see docs/DEPLOYMENT.md.

  casememory:
    build:
      context: .
      dockerfile: infra/Dockerfile
      args:
        GIT_COMMIT_HASH: ${GIT_COMMIT_HASH}
    depends_on:
      dozerdb:
        condition: service_healthy
    environment:
      DOZERDB_BOLT_URI: bolt://dozerdb:7687
      DOZERDB_USER: ${DOZERDB_USER}
      DOZERDB_PASSWORD: ${DOZERDB_PASSWORD}
      LLM_PROVIDER: ${LLM_PROVIDER}
      LLM_API_BASE: ${LLM_API_BASE}
      LLM_API_KEY: ${LLM_API_KEY}
      LLM_MODEL: ${LLM_MODEL}
      LLM_EMBEDDING_MODEL: ${LLM_EMBEDDING_MODEL}
      PORTKEY_API_KEY: ${PORTKEY_API_KEY}
      PORTKEY_VIRTUAL_KEY: ${PORTKEY_VIRTUAL_KEY}
      CASEMEMORY_HUMAN_USER_ID: ${CASEMEMORY_HUMAN_USER_ID}
      AUDIT_SQLITE_PATH: /data/audit.db
    volumes:
      - audit_data:/data
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/healthz')"]
      interval: 15s
      timeout: 5s
      retries: 5

volumes:
  dozerdb_data:
  audit_data:
```

```dockerfile
# casememory/infra/Dockerfile
# Multi-stage Python 3.13 build.
FROM python:3.13-slim AS builder
WORKDIR /app
COPY pyproject.toml .
COPY casememory ./casememory
RUN pip install --no-cache-dir --prefix=/install .

FROM python:3.13-slim
ARG GIT_COMMIT_HASH
ENV GIT_COMMIT_HASH=${GIT_COMMIT_HASH}
WORKDIR /app
COPY --from=builder /install /usr/local
COPY casememory ./casememory
EXPOSE 8000
HEALTHCHECK --interval=15s --timeout=5s --retries=5 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/healthz')"
CMD ["python", "-m", "casememory.cli", "serve"]
```

```markdown
# casememory/infra/coolify/README.md
# Deploying Casememory on Coolify

1. Create a new Coolify "Docker Compose" resource pointing at `infra/docker-compose.yml`.
2. Add all variables from `infra/.env.example` as Coolify secrets (never commit real values).
3. Attach the Coolify server's Tailscale interface; do NOT expose public ports for `dozerdb` or `casememory`.
4. Set restart policy `unless-stopped` for both services.
5. Point Coolify's health check at the `casememory` service `/healthz` endpoint (see docs/DEPLOYMENT.md).

See `service.yaml.example` for a reference service definition.
```

```yaml
# casememory/infra/coolify/service.yaml.example
# Illustrative only — Coolify's actual schema may differ; verify against https://coolify.io/docs
name: casememory
type: docker-compose
compose_file: infra/docker-compose.yml
network: tailnet-only
healthcheck_path: /healthz
restart_policy: unless-stopped
secrets_from: coolify-secret-store
```

```bash
# casememory/infra/.env.example
# Coolify secret variable names — placeholders only.
DOZERDB_USER=neo4j
DOZERDB_PASSWORD=
LLM_PROVIDER=nim
LLM_API_BASE=
LLM_API_KEY=
LLM_MODEL=
LLM_EMBEDDING_MODEL=
PORTKEY_API_KEY=
PORTKEY_VIRTUAL_KEY=
CASEMEMORY_HUMAN_USER_ID=matthew-salem
GIT_COMMIT_HASH=
```

```toml
# pyproject.toml
[project]
name = "casememory"
version = "0.1.0"
requires-python = ">=3.13"
dependencies = [
  "graphiti-core",
  "typer",
  "httpx",
]

[project.optional-dependencies]
dev = ["pytest", "ruff", "mypy"]

[build-system]
requires = ["setuptools>=68"]
build-backend = "setuptools.build_meta"
```

```toml
# ruff.toml
line-length = 100
target-version = "py313"
[lint]
select = ["E", "F", "I", "UP"]
```

```markdown
# README.md
# Casememory

Single-user agent memory platform on Graphiti + DozerDB, deployed via Coolify
behind Tailscale. This checkout covers milestones M0-M3 only. See
`docs/HANDOFF.md` for scope, architecture, and the deferred-feature seam
registry; `docs/DEPLOYMENT.md` for Coolify/Tailscale/DozerDB operations;
`docs/PROVIDERS.md` for NIM/Portkey configuration.
```


## 3. docs/HANDOFF.md

```markdown
# HANDOFF.md

## a. Purpose and scope
Casememory organizes and preserves evidence for a contested civil matter using
a temporal knowledge graph (Graphiti + DozerDB) with forensic-grade lineage.
This pass (M0-M3) delivers: repo/infra bootstrap, episode ingestion with
source lineage (T1), an append-only Merkle-chained audit log (T4), verbatim
LLM extraction lineage capture (T3), and append-only bi-temporal invalidation
(T2). Dashboard, BYOK, retention/legal hold, observations, MCP overlay, Agno
toolkit, and evidence export are explicitly deferred — their interface seams
are defined but not implemented.

## b. Architecture (text diagram)

```

[source file / message]
│
▼
ingest/loaders/*  ──►  ingest/hasher.py (SHA-256)
│                        │
▼                        ▼
ingest/episode.py (Episode + SourceLineage, T1)  ──► ingest/identity.py
│
▼
graph/client.py.add_episode()  ──calls──►  graphiti_core Graphiti.add_episode
│                                          │
│                                          ▼
│                              extraction/provider.py (LLMProvider)
│                                    │ (direct NIM or via portkey_gateway.py)
│                                    ▼
│                          extraction/lineage.py.capture_extraction (T3)
│                                    │
▼                                    ▼
graph/schema.py (bi-temporal fields + ExtractionLineageRef, T2/T3)
│
▼
DozerDB (Neo4j Bolt) — append-only edges; invalidation.py never deletes
│
▼
audit/log.py.append_entry() — Merkle-chained SQLite row for EVERY step above
│
▼
Queryable state: graph/versions.get_fact_versions(), sdk client.search_memory()

```

The graph DB and the audit SQLite DB are separate files; the graph never
writes to the audit DB directly — the orchestrating Python code (future
`sdk/python/casememory/client.py` implementations) calls both explicitly.

## c. Interface boundary registry

| Deferred feature | Seam module | Shape future implementation must satisfy |
|---|---|---|
| Dashboard/UI | none in this pass — will consume `sdk/python/casememory/client.py` | Read-only calls to `CasememoryClient` methods |
| BYOK envelope encryption | `crypto/__init__.py` | `KeySource` protocol (`get_signing_key`, `get_encryption_key`); consumed by `ingest.episode.SourceLineage.signature` and `extraction/lineage.py` |
| Retention / legal hold | `retention/__init__.py` | `RetentionPolicy` protocol (`is_legal_hold`, `retention_expiry`); must be consulted before any future purge-eligible op (none exist yet, since nothing is deleted) |
| Observations | `observations/__init__.py` | `ObservationDetector` protocol (`detect(since)`); scheduled job runner not built |
| MCP overlay | `mcp/__init__.py` | Future MCP server wrapping `CasememoryClient`; mirrors `getzep/graphiti/mcp_server` |
| Agno toolkit | `sdk/python/casememory/client.py` | Method names/shapes mirror `agno.tools.zep.ZepTools` (`add_episode`, `add_message`, `search_memory`, `get_context_block`, `get_fact_versions`) so wrapping is a rename [cite:web reference: docs.agno.com/examples/integrations/memory/zep-integration] |
| Evidence export / PDF exhibits | none yet | Will read from `audit/log.py` + `graph/versions.py`; no seam module created this pass per explicit constraint |

## d. Milestone plan reference
- M0: repo bootstrap, docker-compose infra, CI, health checks (this pass).
- M1: episode ingestion + T1/T4 (this pass).
- M2: extraction lineage, T3 (this pass).
- M3: bi-temporal invalidation semantics, T2 (this pass).
- M4+: dashboard, BYOK, retention/legal hold, observations, MCP overlay, Agno toolkit, evidence export, FRE/MRE mapping doc — all deferred to follow-up prompts.

## e. Open questions requiring a decision before M4
1. Exact DozerDB image tag/version to pin, confirmed compatible with the Neo4j 5.26+ API surface Graphiti requires — needs a live compatibility check against https://dozerdb.org release notes.
2. Whether `graph.add`/`add_episode` in graphiti-core exposes a hook to inject custom edge properties (T3 `ExtractionLineageRef`) natively, or whether Casememory must post-process edges after Graphiti's extraction completes.
3. Whether Portkey's response headers expose a stable request-id field name to store as `portkey_request_id`.
4. Confirmation of `fact_group_id`-style linkage needed for `get_fact_versions` across superseding edges (vs. single-edge mutation) in the installed graphiti-core version.
5. Legal-hold/retention triggers and their exact required fields, to correctly shape the `RetentionPolicy` protocol before implementation begins.
```


## 4. docs/DEPLOYMENT.md

```markdown
# DEPLOYMENT.md

## a. Coolify project layout
- One Coolify project "casememory" containing a single Docker Compose
  resource (`infra/docker-compose.yml`) with two services: `dozerdb` and
  `casememory`.
- Secrets (DOZERDB_PASSWORD, LLM_API_KEY, PORTKEY_API_KEY, etc.) are stored
  in Coolify's secret store and injected as environment variables; no
  `.env` file with real values is ever committed.
- Health checks: Coolify polls the `casememory` service's `/healthz`
  endpoint; `dozerdb` uses its own container HEALTHCHECK.
- Restart policy: `unless-stopped` for both services.

## b. Tailscale ACL example (device + VPS tailnet interface only)
```json
{
  "acls": [
    {
      "action": "accept",
      "src": ["matthew-device"],
      "dst": ["vps-tailnet-node:*"]
    }
  ],
  "tagOwners": {
    "tag:casememory-vps": ["autogroup:admin"]
  }
}
```

Bind `dozerdb` and `casememory` only to the VPS's tailnet IP (`100.x.x.x`)
or `127.0.0.1` for inter-container traffic — never `0.0.0.0` on a public
interface, per Tailscale's kb guidance on tailnet-only exposure.
// verify exact ACL JSON schema against https://tailscale.com/kb

## c. DozerDB image tag, initial DB creation, credential rotation

- Pin the DozerDB tag explicitly in `docker-compose.yml` (placeholder used
in this pass; confirm against https://dozerdb.org before first deploy —
see HANDOFF.md open question 1).
- On first boot, DozerDB auto-creates the default database using
`NEO4J_AUTH=user/password`; Graphiti's `build_indices_and_constraints()`
(called from `graph/client.py.connect()`) creates required indices.
- Credential rotation: update `DOZERDB_PASSWORD` in Coolify's secret store,
redeploy `dozerdb` with `NEO4J_AUTH` updated, then redeploy `casememory`
so it picks up the new value — no code changes required.


## d. First-boot procedure and health verification

1. Deploy the compose stack via Coolify.
2. Wait for `dozerdb` healthcheck to pass (Bolt/HTTP port up).
3. `casememory` waits on `depends_on: service_healthy` before starting.
4. Verify via Tailscale: `curl http://<tailnet-ip>:8000/healthz` returns 200.
5. Run `casememory audit verify` once to confirm the audit chain initializes cleanly.

## e. Backup and snapshot procedure

- DozerDB: snapshot the `dozerdb_data` Docker volume (e.g. via Coolify's
volume backup feature or a cron `docker run --rm -v dozerdb_data:/data ... tar` job) on a schedule; store off-VPS.
- Audit SQLite: because it is WAL-mode, checkpoint before copying
(`PRAGMA wal_checkpoint(FULL);`) then copy the single `.db` file from the
`audit_data` volume; verify the copy with `casememory audit verify`
immediately after each backup to catch corruption early.
- Never restore a backup by mutating the live audit file in place — restore
to a fresh path and re-verify the chain before switching over.

```

## 5. docs/PROVIDERS.md

```markdown
# PROVIDERS.md

## a. NIM endpoint configuration
Required env vars: `LLM_PROVIDER=nim`, `LLM_API_BASE` (NVIDIA NIM base URL,
e.g. `https://integrate.api.nvidia.com/v1`), `LLM_API_KEY`, `LLM_MODEL`,
`LLM_EMBEDDING_MODEL`, optional `LLM_RERANKER_MODEL`.
// verify exact endpoint paths and auth header format against
https://docs.nvidia.com/nim

## b. Portkey virtual-key configuration
Set `PORTKEY_API_KEY` and `PORTKEY_VIRTUAL_KEY` to route all outbound model
calls through Portkey's gateway instead of hitting NIM directly. The
`extraction/nim_provider.py` factory (`build_provider_from_config`) checks
`LLMConfig.use_portkey` and wraps the base provider in
`extraction/portkey_gateway.py.PortkeyGateway` — no other code changes.
// verify virtual-key header names against https://docs.portkey.ai

## c. Swapping providers (NIM → OpenAI → Anthropic → Gemini)
Only these env vars change; no code is touched, per the `LLMProvider`
protocol contract in `extraction/provider.py`:

| Target | LLM_PROVIDER | LLM_API_BASE | LLM_API_KEY | LLM_MODEL |
|---|---|---|---|---|
| NIM | nim | NIM base URL | NIM key | NIM model id |
| OpenAI | openai | OpenAI base URL | OpenAI key | e.g. gpt-4.1 |
| Anthropic | anthropic | Anthropic base URL | Anthropic key | e.g. claude model id |
| Gemini | gemini | Gemini base URL | Gemini key | Gemini model id |

If a swap ever requires editing `graph/client.py` or `extraction/lineage.py`,
the `LLMProvider`/`Embedder`/`Reranker` protocols are wrong and must be
fixed — this is a hard constraint from the scaffold spec, not a suggestion.

## d. What T3 lineage captures per provider (trial reconstruction)
For every extraction call: `provider_name`, `model_name` (with version
string as sent), `prompt_verbatim`, `response_verbatim`, `start_time`/
`end_time` (UTC), `span_start`/`span_end` into the source episode, a
`confidence` value or explicit null marker, and — when routed through
Portkey — the `portkey_request_id` so opposing counsel or the court can
cross-reference Portkey's own independent request logs for the same call.
This record set is designed so the exact model call can be reconstructed
without relying on Casememory's own logs alone.
```


## Next Actions for the Human (≤10 items)

1. Confirm the DozerDB Docker image tag and verify Neo4j 5.26+ parity against dozerdb.org release notes before first deploy.
2. Verify the exact `graphiti_core.Graphiti.add_episode` constructor and call signature in the installed version and replace `// verify` TODOs in `graph/client.py`.
3. Confirm how Graphiti exposes `valid_at`/`invalid_at` on `EntityEdge` in your installed version to finalize `graph/schema.py`.
4. Decide whether custom edge types are the right mechanism to carry T3 `ExtractionLineageRef`, per help.getzep.com/graphiti/working-with-data.
5. Provision Tailscale ACLs restricting the tailnet to your device and the VPS node only, then apply the example policy.
6. Create the Coolify project, wire all secrets from `infra/.env.example` into Coolify's secret store, and deploy the compose stack.
7. Run `casememory audit verify` after first boot to confirm a clean Merkle chain baseline.
8. Decide on NVIDIA NIM model/embedding/reranker choices and populate the corresponding env vars in Coolify.
9. Set up the DozerDB and audit-SQLite backup cadence per DEPLOYMENT.md section e.
10. When ready, invite the M4 follow-up prompt with: `Scaffold M4-M9 of Casememory: dashboard, BYOK, retention/legal hold, observations, MCP overlay, Agno toolkit, evidence export, and FRE/MRE mapping doc.`
<span style="display:none">[^10][^11][^12][^13][^14][^15][^6][^7][^8][^9]</span>

<div align="center">⁂</div>

[^1]: https://blog.getzep.com/beyond-static-knowledge-graphs/

[^2]: https://help.getzep.com/facts

[^3]: https://codepointer.substack.com/p/agent-memory-systems-and-knowledge

[^4]: https://help.getzep.com/v2/concepts

[^5]: https://arxiv.org/html/2501.13956v1

[^6]: https://help.getzep.com/graphiti/core-concepts/adding-episodes

[^7]: https://www.falkordb.com/blog/graphiti-get-started/

[^8]: https://www.youtube.com/watch?v=F4hwuLlISP4

[^9]: https://datapace.ai/blog/ai-agent-memory-tools-2026

[^10]: https://pypi.org/project/graphiti-core/0.8.8/

[^11]: https://www.youtube.com/watch?v=H2Cb5wbcRzo

[^12]: https://cortexdb.ai/blog/cortexdb-vs-zep

[^13]: https://github.com/getzep/graphiti

[^14]: https://github.com/getzep/graphiti/issues/912

[^15]: https://www.digitalapplied.com/blog/open-source-agent-memory-mem0-letta-zep-compared

