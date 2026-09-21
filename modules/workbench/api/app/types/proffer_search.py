"""Bounded server-side search filter for the preview message projection.

The owner's threads run to years of messages, so narrowing happens in the
engine's PostgreSQL projection, never over the rows a browser happens to have
loaded. This module owns only the request-side contract; the response totals
live on ProfferPreviewMessagesResponse.
"""

from __future__ import annotations

from datetime import datetime
from typing import Annotated

from pydantic import BaseModel, ConfigDict, StringConstraints, model_validator

# Matches previewmodel.MaxMessageQueryBytes in the Go engine. Kept bounded so a
# search is always one COUNT plus one page scan.
MAX_QUERY_LENGTH = 200

SearchNeedle = Annotated[str, StringConstraints(strip_whitespace=True, max_length=MAX_QUERY_LENGTH)]
ParticipantID = Annotated[str, StringConstraints(strip_whitespace=True, max_length=128)]


class PreviewMessageFilter(BaseModel):
    """One bounded narrowing of a preview's messages.

    The default instance narrows nothing and reproduces the pre-existing
    unfiltered paging behaviour exactly.
    """

    model_config = ConfigDict(extra="forbid")

    query: SearchNeedle | None = None
    has_attachments: bool = False
    sender: ParticipantID | None = None
    sent_from: datetime | None = None
    sent_to: datetime | None = None

    @model_validator(mode="after")
    def _check_bounds(self) -> PreviewMessageFilter:
        if self.sent_from is not None and self.sent_to is not None and self.sent_to < self.sent_from:
            raise ValueError("to must not precede from")
        return self

    def is_empty(self) -> bool:
        """True when this filter narrows nothing."""
        return not any(
            (self.query, self.has_attachments, self.sender, self.sent_from, self.sent_to)
        )

    def as_query_params(self) -> dict[str, str]:
        """Render the filter as the engine's documented query parameters.

        Only active predicates are sent. An absent parameter means "no
        predicate", which is exactly how the engine reads it, so a dropped value
        can never silently widen the result set into a different filter's
        cursor scope.
        """
        params: dict[str, str] = {}
        if self.query:
            params["q"] = self.query
        if self.has_attachments:
            params["has_attachments"] = "true"
        if self.sender:
            params["sender"] = self.sender
        if self.sent_from is not None:
            params["from"] = self.sent_from.isoformat().replace("+00:00", "Z")
        if self.sent_to is not None:
            params["to"] = self.sent_to.isoformat().replace("+00:00", "Z")
        return params


__all__ = ["MAX_QUERY_LENGTH", "PreviewMessageFilter"]
