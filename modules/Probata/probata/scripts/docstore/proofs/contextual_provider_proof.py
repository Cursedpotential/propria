"""Read one to three real document chunks and compare remote embedding inputs without writes.

Run only on the VPS after coordinating with the active index owner. Requires
the existing Docstore database and DOCSTORE_LLM_* provider configuration. This
never imports the indexing App, runs CDC, applies schema, or writes vectors.
Outputs establish provider/input compatibility, not retrieval quality gains.
"""
from __future__ import annotations

import argparse
import asyncio
import json
import os
import sys
import time
from dataclasses import replace
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import sq
from contextual_retrieval import ContextPolicy, build_request, describe_context, digest, search_metadata
from nim_input import embed_input


def rows(value):
    """Unwrap the existing SDK's bounded query response shape.

    Input: query result. Output: row list. Side effects: none. Use for proof reads.
    """
    while isinstance(value, list) and len(value) == 1 and isinstance(value[0], list):
        value = value[0]
    return value if isinstance(value, list) else [value] if value else []


async def run(paths: list[str], *, expected_hashes: dict[str, str] | None = None) -> list[dict]:
    """Compare contextual versus raw inputs for a bounded set of real stored chunks.

    Inputs: one to three exact canonical source paths. Output: hashes, lengths,
    description and provider dimensions. Side effects: SELECTs, one chat call and
    two existing-adapter embedding calls per source; no database or indexing writes.
    Use before source-scoped activation, never as a corpus benchmark.
    """
    from cocoindex.ops.litellm import LiteLLMEmbedder

    if not 1 <= len(paths) <= 3 or len(set(paths)) != len(paths):
        raise ValueError("Choose one to three unique canonical source paths")
    policy = ContextPolicy(model=os.environ.get("DOCSTORE_LLM_MODEL", ""),
                           api_base=os.environ.get("DOCSTORE_LLM_BASE_URL", ""),
                           embedding_model=os.environ.get("EMBED_MODEL", "nvidia/nemotron-3-embed-1b"),
                           disable_thinking=os.environ.get("DOCSTORE_LLM_DISABLE_THINKING") == "1")
    key = os.environ.get("NVIDIA_API_KEY") or os.environ.get("NVIDIA_NIM_API_KEY")
    if not key:
        raise ValueError("Existing NVIDIA embedding credentials are required")
    # The flow's exact configured adapter; no guessed wire format or input_type.
    embedder = LiteLLMEmbedder(f"openai/{policy.embedding_model}",
                              api_base=os.environ.get("NVIDIA_API_BASE", "https://integrate.api.nvidia.com/v1"),
                              api_key=key)
    db = await sq.connect("docs", "probata", "docs")
    output = []
    try:
        for path in paths:
            docs = rows(await db.query("SELECT source_path, content_hash, title, body FROM document WHERE source_path=$path LIMIT 1;", {"path": path}))
            if not docs:
                raise ValueError("Requested real document is absent")
            doc = docs[0]
            if expected_hashes is not None and doc["content_hash"] != expected_hashes.get(path):
                raise ValueError("Document version differs from the authorized proof source")
            if digest(doc["body"]) != doc["content_hash"]:
                raise ValueError("Stored body/hash validation failed")
            chunks = rows(await db.query("SELECT text, ordinal, heading FROM chunk WHERE source_path=$path ORDER BY ordinal LIMIT 3;", {"path": path}))
            chunk = next((c for c in chunks if 0 < len(c["text"]) <= 2000 and c["text"] in doc["body"]), None)
            if chunk is None:
                raise ValueError("Choose a document with an exact stored-body chunk match for this proof")
            request = build_request(source_path=path, source_hash=doc["content_hash"], title=doc["title"],
                                    heading=chunk.get("heading") or "", ordinal=chunk["ordinal"],
                                    offset=doc["body"].index(chunk["text"]), chunk_text=chunk["text"],
                                    normalized_document=doc["body"], policy=policy)
            request = replace(request, rendering="stored-body exact substring; no additional normalization; proof locator")
            generation = {}
            description = await describe_context(request, policy, api_key=os.environ.get("DOCSTORE_LLM_API_KEY", ""), telemetry=generation)
            description, search, provenance = search_metadata(request, policy, description)
            started = time.monotonic()
            raw_vector = await embedder.embed(embed_input(chunk["text"], allow_truncation=False))
            raw_ms = round((time.monotonic() - started) * 1000)
            started = time.monotonic()
            contextual_vector = await embedder.embed(embed_input(search, allow_truncation=False))
            contextual_ms = round((time.monotonic() - started) * 1000)
            output.append({"source_path": path, "ordinal": chunk["ordinal"], "source_hash": doc["content_hash"],
                           "original_chunk_hash": digest(chunk["text"]), "raw_chars": len(chunk["text"]),
                           "contextual_chars": len(search), "description": description,
                           "embedding_dimensions": [len(raw_vector), len(contextual_vector)],
                           "generation": generation, "embedding_latency_ms": [raw_ms, contextual_ms],
                           "embedding_input_type": None, "embedding_adapter": "cocoindex.ops.litellm.LiteLLMEmbedder.embed",
                           "provenance": json.loads(provenance), "writes": 0})
    finally:
        await db.close()
    return output


def main() -> None:
    """Run the explicitly bounded provider proof from canonical path arguments.

    Inputs: --path repeated at most three times. Output: JSON proof. Side effects:
    those documented by run. Pick this after review and coordination with index owner.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--path", action="append", required=True)
    args = parser.parse_args()
    print(json.dumps(asyncio.run(run(args.path)), ensure_ascii=True, indent=2))


if __name__ == "__main__":
    main()
