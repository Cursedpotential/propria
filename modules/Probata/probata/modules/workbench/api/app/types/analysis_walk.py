"""Validate the actual Go walk request and immutable result references.

Inputs: existing projection pins and explicit dates. Outputs: bounded contracts.
Effects: none; pick for analytical walks, never ingestion or evidence promotion.
"""
from typing import Literal
from urllib.parse import parse_qs, unquote, urlsplit

from pydantic import AwareDatetime, BaseModel, ConfigDict, Field, model_validator

from app.types.analysis import AnalysisArtifact, AnalysisQuery

WALK_PREFIX = "consignatio/casevault/DerivedKnowledge/analysis/walks/"


class AnalysisWalk(BaseModel):
    """Keep a selected graph scope and one to twenty increasing historical cutoffs.

    Input/output: exact Go body without server-owned mode/case. Effects: none.
    Choose instead of a single query when comparing successive knowledge dates.
    """
    model_config = ConfigDict(extra="forbid")
    scope: AnalysisQuery
    cutoffs: list[AwareDatetime] = Field(min_length=1, max_length=20)

    @model_validator(mode="after")
    def ordered_dates(self):
        """Match Go's as-lived, unpaged, final-horizon contract without adding a gate."""
        if self.scope.perspective != "as_lived" or self.scope.cursor:
            raise ValueError("Walks require an unpaged as-lived scope")
        if any(right <= left for left, right in zip(self.cutoffs, self.cutoffs[1:])) or self.cutoffs[-1] != self.scope.horizon:
            raise ValueError("Cutoffs must increase and end at the selected horizon")
        return self


class WalkArtifact(AnalysisArtifact):
    """Preserve Go's version-query locator and prefixed SHA-256 exactly.

    Input/output: actual B2 derivative reference. Effects: none; normalize only
    for the shared verified reader, preserving this original reference in replies.
    """
    sha256: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")

    def reader_reference(self) -> AnalysisArtifact:
        """Resolve the exact B2 key/version without allowing arbitrary storage locations.

        Input: immutable reference. Output: shared-reader format. Effects: none;
        use only with the fixed walk derivative prefix.
        """
        uri = urlsplit(self.uri)
        key = unquote(uri.path.lstrip("/"))
        if (uri.scheme != "b2" or uri.netloc != "salem-data" or uri.fragment
                or not key.startswith(WALK_PREFIX) or ".." in key
                or parse_qs(uri.query, strict_parsing=True) != {"versionId": [self.version_id]}
                or self.version_id == "null"):
            raise ValueError("Walk artifact is outside its exact-version location")
        return AnalysisArtifact(uri=f"b2://salem-data/{key}", version_id=self.version_id,
            sha256=self.sha256.removeprefix("sha256:"), bytes=self.bytes)


class WalkReference(BaseModel):
    """Wrap one real immutable walk artifact; input/output metadata, no effects."""
    model_config = ConfigDict(extra="forbid")
    artifact: WalkArtifact


class WalkResult(BaseModel):
    """Retain the completed workflow's references and actual coverage status.

    Input/output: Go result metadata. Effects: none; choose before protected reads.
    """
    model_config = ConfigDict(extra="forbid")
    artifact: WalkArtifact
    checkpoints: list[WalkReference] = Field(min_length=2, max_length=21)
    partial: bool
    review_status: Literal["pending"]
