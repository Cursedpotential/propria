"""Attach per-file contextual policies while preserving complete live source membership.

The adapter forwards every original key and every watch lifecycle call. Only
explicitly selected file values acquire a new memo input; other values remain
the original objects and retain their existing CocoIndex memo identities.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Mapping

from contextual_retrieval import ContextPolicy, selected_policy


def contextual_items(items, canonical_prefix: str, env: Mapping[str, str]):
    """Preserve the original source object unless contextual retrieval is enabled.

    Inputs: full live items/prefix/configuration. Output: original map or complete
    adapter. Side effects: none. Pick this to keep the disabled source path exact.
    """
    return ContextualSourceItems(items, canonical_prefix, env) if env.get("DOCSTORE_CONTEXT_ENABLED") == "1" else items


@dataclass(frozen=True)
class PolicyFile:
    """Associate one existing FileLike resource with a derived search policy.

    Inputs: original resource and policy. Output: a transparent read proxy.
    Side effects: none until read. Use only for selected files so siblings retain
    their original object types, state hooks and memo fingerprints.
    """

    file: object
    context_policy: ContextPolicy

    @property
    def file_path(self):
        """Return the unchanged source locator from the wrapped file.

        Input: wrapped resource. Output: original FilePath. Side effects: none.
        Use for existing canonical source-path construction.
        """
        return self.file.file_path

    async def read(self, size: int = -1) -> bytes:
        """Read original bytes through the resource's existing cache.

        Input: optional byte limit. Output: unchanged bytes. Side effects: source
        read when uncached. Use instead of opening a second source handle.
        """
        return await self.file.read(size)

    def __coco_memo_key__(self):
        """Include policy plus the original resource and its content-state hook.

        Input: wrapped resource/policy. Output: structured memo key. Side effects:
        none. Use to invalidate one selected file without changing global versions.
        """
        return self.file, self.context_policy


class ContextualSourceItems:
    """Map selected file values while retaining the full original live map.

    Inputs: existing live items, canonical prefix and immutable environment copy.
    Output: a LiveMapView-compatible adapter. Side effects: delegates original
    source iteration/watch only. Use instead of filtering a CocoIndex source.
    """

    def __init__(self, items, canonical_prefix: str, env: Mapping[str, str]):
        """Capture the complete source and a stable policy configuration.

        Inputs: live items, prefix, environment. Output: adapter. Side effects:
        none. Pick this when enriching explicitly selected canonical source paths.
        """
        self.items = items
        self.canonical_prefix = canonical_prefix
        # Source/memo metadata carries configuration only, never provider keys.
        keys = ("DOCSTORE_CONTEXT_ENABLED", "DOCSTORE_CONTEXT_SOURCES", "DOCSTORE_LLM_MODEL",
                "DOCSTORE_LLM_BASE_URL", "DOCSTORE_LLM_DISABLE_THINKING", "EMBED_MODEL")
        self.env = {key: env[key] for key in keys if key in env}

    def transform(self, file):
        """Return the original file or a policy proxy for an explicitly selected source.

        Input: source file. Output: unchanged object or PolicyFile. Side effects:
        validates selected configuration. Use for both scans and live updates.
        """
        path = self.canonical_prefix + file.file_path.path.as_posix()
        policy = selected_policy(path, self.env)
        return file if policy is None else PolicyFile(file, policy)

    async def __aiter__(self):
        """Forward every original membership key with its optional policy value.

        Input: complete underlying iterator. Output: all keyed entries. Side
        effects: original source scan. Use for complete snapshot reconciliation.
        """
        async for key, file in self.items:
            yield key, self.transform(file)

    async def watch(self, subscriber):
        """Forward live lifecycle calls and enrich only update values.

        Input: CocoIndex subscriber. Output: delegated watch lifetime. Side
        effects: original watcher. All deletes/readiness/full scans retain ownership.
        """
        await self.items.watch(_ContextSubscriber(subscriber, self))


class _ContextSubscriber:
    """Delegate a live subscriber while mapping file update values.

    Inputs: subscriber and source adapter. Output: forwarding subscriber.
    Side effects: delegated calls. Use only inside ContextualSourceItems.watch.
    """

    def __init__(self, subscriber, source):
        """Capture one subscriber and its deterministic source transform.

        Inputs: subscriber/source. Output: proxy. Side effects: none. Use for one watch.
        """
        self.subscriber = subscriber
        self.source = source

    def __getattr__(self, name):
        """Forward lifecycle and committed-state operations without modification.

        Input: attribute name. Output: underlying attribute. Side effects: none.
        Use for all subscriber operations except update.
        """
        return getattr(self.subscriber, name)

    async def update(self, key, file):
        """Forward the original key with its source-specific policy value.

        Inputs: original key/file. Output: original mount handle. Side effects:
        delegates the existing component update. Use for source watch notifications.
        """
        return await self.subscriber.update(key, self.source.transform(file))
