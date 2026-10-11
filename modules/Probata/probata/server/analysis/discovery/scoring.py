"""Retrieve independent detector labels with bounded local CPU embeddings.

Inputs: episode contexts, human-readable hypotheses and immutable human feedback.
Outputs: reviewable scores and cited candidates, including empty/abstained results.
Effects: injected encoder calls only; choose reversible exemplar ranking over
training on model agreement or treating similarity as proof.
"""

from __future__ import annotations

import json
import math
import os
from functools import lru_cache
from pathlib import Path
from typing import Any
from uuid import NAMESPACE_URL, uuid5

from .context import assemble_context, attribution, order_feedback, timestamp
from .storage import DiscoveryInvalid, fingerprint

MODEL_VERSION = "discovery-cpu-v1"
SCORE_REVISION = "prototype-exemplar-v3"
DEFAULT_HYPOTHESES = [
    {"hypothesis_id": "threat", "category": "threat", "text": "A person threatens physical harm or intimidation."},
    {
        "hypothesis_id": "coercion",
        "category": "control",
        "text": "A person uses demands or pressure to control another.",
    },
    {"hypothesis_id": "demeaning", "category": "verbal", "text": "A person humiliates, insults or demeans another."},
]
ABSTENTION_PROTOTYPES = [
    "A neutral factual update or ordinary cooperative conversation without threatening or controlling behavior.",
    "A person explicitly denies intent to harm, threaten or control another, or negates an alleged behavior.",
]


def model_config(batch: dict[str, Any], *, injected: bool = False) -> dict[str, Any]:
    """Pin explicit local model settings before scoring and retry/cache lookup.

    Inputs: batch scoring config and explicit local runtime settings. Outputs:
    immutable configuration. Effects: environment reads only; choose a required
    model revision over silently changing weights under an existing score key.
    """
    config = batch.get("scoring") or batch.get("config") or batch.get("model_config") or {}
    if not config:
        try:
            config = json.loads(os.environ.get("ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON", "{}"))
        except ValueError:
            raise DiscoveryInvalid("discovery runtime model configuration is invalid JSON") from None
    if not isinstance(config, dict):
        raise DiscoveryInvalid("discovery model configuration must be an object")
    config = dict(config)
    if not injected:
        config.setdefault("model_path", os.environ.get("ADAPTIVE_DISCOVERY_MODEL_PATH", ""))
        config.setdefault("model_revision", os.environ.get("ADAPTIVE_DISCOVERY_MODEL_REVISION", ""))
        config.setdefault("backend", os.environ.get("ADAPTIVE_DISCOVERY_MODEL_BACKEND", "onnx"))
        if not config["model_revision"]:
            raise DiscoveryInvalid("discovery local model requires explicit model_revision")
    else:
        config.setdefault("model_revision", "injected-fixture-v1")
    return config


def local_encoder(config: dict[str, Any]) -> Any:
    """Load the existing CPU encoder only from an explicitly retained local model.

    Inputs: model_path and optional backend. Outputs: SentenceEncoder. Effects:
    lazy local model loading; choose over a provider or automatic model download.
    """
    path = Path(config.get("model_path", ""))
    if not config.get("model_path") or not path.is_absolute() or not path.is_dir():
        raise DiscoveryInvalid("discovery requires an injected encoder or existing absolute local model_path")
    return _cached_encoder(str(path), config["model_revision"], config.get("backend", "onnx"))


@lru_cache(maxsize=2)
def _cached_encoder(path: str, revision: str, backend: str) -> Any:
    """Reuse one pinned local CPU model without changing the Activity data contract.

    Inputs: existing local path and explicit immutable model revision/backend.
    Outputs: encoder. Effects: lazy local model allocation; choose bounded cache
    reuse over loading a fresh model for every scoring batch.
    """
    from server.analysis.cpu_lane_classifier import load_cpu_encoder

    return load_cpu_encoder(path, backend=backend)


def _bound(config: dict[str, Any], name: str, default: int, ceiling: int) -> int:
    """Validate a requested positive CPU bound without silently dropping coverage.

    Inputs: config/limit. Outputs: integer. Effects: none; choose rejection over
    truncating records and implying complete analysis.
    """
    value = config.get(name, default)
    if isinstance(value, bool) or not isinstance(value, int) or not 1 <= value <= ceiling:
        raise DiscoveryInvalid("discovery scoring bound is invalid: " + name)
    return value


def _vectors(encoder: Any, texts: list[str], batch_size: int) -> list[list[float]]:
    """Normalize and validate bounded encoder vectors for cosine retrieval.

    Inputs: injected encoder/text/batch bound. Outputs: normalized vectors.
    Effects: CPU calls only; choose explicit finite dimension validation over
    trusting an encoder's normalization or malformed output.
    """
    vectors = []
    dimension = None
    for offset in range(0, len(texts), batch_size):
        raw = encoder.encode(
            texts[offset : offset + batch_size],
            normalize_embeddings=True,
            batch_size=batch_size,
            show_progress_bar=False,
        )
        if hasattr(raw, "tolist"):
            raw = raw.tolist()
        if len(raw) != len(texts[offset : offset + batch_size]):
            raise DiscoveryInvalid("encoder returned an invalid vector count")
        for value in raw:
            vector = [float(component) for component in value]
            dimension = dimension or len(vector)
            if (
                not vector
                or len(vector) > 4096
                or len(vector) != dimension
                or not all(math.isfinite(c) for c in vector)
            ):
                raise DiscoveryInvalid("encoder returned invalid vector dimensions or values")
            norm = math.sqrt(sum(c * c for c in vector))
            if not math.isfinite(norm):
                raise DiscoveryInvalid("encoder returned an invalid vector magnitude")
            vectors.append([c / norm for c in vector] if norm else [0.0] * len(vector))
    return vectors


def _similarity(left: list[float], right: list[float]) -> float:
    """Compare two validated vectors without interpreting similarity as certainty.

    Inputs: vectors. Outputs: cosine score. Effects: none; choose for candidate
    ranking only, never acceptance or factual verification.
    """
    return sum(a * b for a, b in zip(left, right, strict=True))


def _feedback(batch: dict[str, Any]) -> list[dict[str, Any]]:
    """Admit bounded explicit human feedback in this snapshot's immutable scope.

    Inputs: batch scope/revision/feedback/filter. Outputs: compatible examples.
    Effects: none; choose human-only examples and reject future as-lived feedback.
    """
    feedback = batch.get("feedback") or []
    if not isinstance(feedback, list) or len(feedback) > 256:
        raise DiscoveryInvalid("discovery feedback count exceeds bound")
    out = []
    filters = batch.get("filters") or {}
    cutoff = timestamp(filters.get("cutoff"))
    for item in feedback:
        if not isinstance(item, dict) or not item.get("feedback_id"):
            raise DiscoveryInvalid("feedback identity is incomplete")
        if item.get("scope", batch.get("scope")) != batch.get("scope"):
            raise DiscoveryInvalid("feedback scope differs from snapshot")
        if item.get("dataset_id", batch.get("dataset_id")) != batch.get("dataset_id"):
            raise DiscoveryInvalid("feedback dataset differs from snapshot")
        if item.get("source", "human") != "human" or not item.get("actor", {}).get("subject_uid"):
            raise DiscoveryInvalid("feedback requires an explicit human reviewer")
        if cutoff is not None and filters.get("perspective") == "as_lived":
            created = timestamp(item.get("created_at"))
            if created is None or created > cutoff:
                raise DiscoveryInvalid("feedback falls outside as_lived cutoff")
        if len(item.get("example_text", "")) > 4000:
            raise DiscoveryInvalid("feedback example text exceeds bound")
        out.append(item)
    return order_feedback(out)


def _feedback_direction(item: dict[str, Any], label: str) -> int:
    """Interpret one human correction as a label-specific positive, negative or neutral.

    Inputs: immutable desired/prior label IDs and feedback kind. Outputs: signed
    exemplar direction. Effects: none; choose explicit correction differences
    over treating desired labels as rejection or partial review as a full negative.
    """
    desired = set(item.get("labels") or [])
    previous = set(item.get("previous_labels") or [])
    kind = item.get("kind")
    if kind in {"correct_labels", "wrong_behavior", "partial_match"}:
        if kind == "partial_match" and not desired:
            return 0
        if label in desired:
            return 1
        return -1 if label in previous - desired else 0
    declared = desired or previous
    if item.get("hypothesis_id"):
        declared = declared | {item["hypothesis_id"]}
    if label not in declared:
        return 0
    if kind in {"confirm", "relevant"}:
        return 1
    if kind in {"reject", "irrelevant"}:
        return -1
    return 0


def _citations(records: list[dict[str, Any]], segments: list[tuple[int, int]]) -> list[dict[str, Any]]:
    """Carry native locator identity and exact text offsets into derived candidates.

    Inputs: ordered records/retrieved offsets. Outputs: bounded citations. Effects: none; choose
    explicit uncertainty for missing locators over fabricated source coordinates.
    """
    citations = []
    cursor = 0
    for record in records:
        record_start, record_stop = cursor, cursor + len(record["text"])
        cursor = record_stop + 1
        spans = sorted(
            {
                (max(start, record_start) - record_start, min(stop, record_stop) - record_start)
                for start, stop in segments
                if max(start, record_start) < min(stop, record_stop)
            }
        )
        for locator in record.get("locators") or []:
            if not locator.get("locator_id") or locator.get("source_version_id") != record["source_version_id"]:
                raise DiscoveryInvalid("record locator identity differs from source version")
            for start, stop in spans:
                citations.append(
                    {
                        "record_id": record["record_id"],
                        "locator_id": locator["locator_id"],
                        "quote": record["text"][start:stop],
                        "start_char": start,
                        "end_char": stop,
                        "source_version_id": record["source_version_id"],
                        "generation_id": record["generation_id"],
                        "ordinal": record.get("ordinal", 0),
                        "native_locator": locator,
                    }
                )
    if len(citations) > 100:
        raise DiscoveryInvalid("retrieved native citation count exceeds bound")
    return citations


def score_batch(batch: dict[str, Any], *, encoder: Any = None) -> dict[str, Any]:
    """Score one immutable bounded snapshot using independent descriptions/examples.

    Inputs: versioned batch and optional CPU encoder. Outputs: deterministic
    derived result with per-label scores and explicit zero-label/abstention counts.
    Effects: encoder calls only; choose for discovery candidates before human
    review, never canonical facts, paid providers or model-agreement training.
    """
    if batch.get("schema") != "adaptive-discovery.batch.v1" or batch.get("model_version") != MODEL_VERSION:
        raise DiscoveryInvalid("discovery schema or model version differs")
    for field in ("run_id", "dataset_id", "selection_revision", "feedback_revision"):
        if batch.get(field) is None or batch.get(field) == "":
            raise DiscoveryInvalid("discovery snapshot revision identity is incomplete")
    if not isinstance(batch.get("scope"), dict) or batch["scope"].get("mode") not in {"LIVE", "DEV"}:
        raise DiscoveryInvalid("discovery snapshot requires explicit operating scope")
    contexts = assemble_context(batch)
    feedback = _feedback(batch)
    config = model_config(batch, injected=encoder is not None)
    window_chars = _bound(config, "max_window_chars", 1400, 4000)
    max_windows = _bound(config, "max_windows", 1024, 2048)
    batch_size = _bound(config, "batch_size", 16, 64)
    max_candidates = _bound(batch, "max_candidates", 100, 100)
    threshold = config.get("threshold", 0.55)
    weight = config.get("feedback_weight", 0.15)
    abstention_margin = config.get("abstention_margin", 0.02)
    if isinstance(threshold, bool) or not isinstance(threshold, (int, float)) or not 0 <= threshold <= 1:
        raise DiscoveryInvalid("discovery threshold must be between zero and one")
    if isinstance(weight, bool) or not isinstance(weight, (int, float)) or not 0 <= weight <= 0.5:
        raise DiscoveryInvalid("discovery feedback weight must be between zero and one half")
    if (
        isinstance(abstention_margin, bool)
        or not isinstance(abstention_margin, (int, float))
        or not 0 <= abstention_margin <= 0.5
    ):
        raise DiscoveryInvalid("discovery abstention margin is invalid")
    hypotheses = batch.get("hypotheses") or DEFAULT_HYPOTHESES
    if not isinstance(hypotheses, list) or not 1 <= len(hypotheses) <= 32:
        raise DiscoveryInvalid("discovery hypothesis count exceeds bound")
    prototype_spans = []
    texts = []
    for hypothesis in hypotheses:
        descriptions = [hypothesis.get("text", ""), *(hypothesis.get("prototypes") or [])]
        if (
            not hypothesis.get("hypothesis_id")
            or len(descriptions) > 8
            or any(not isinstance(text, str) or not text.strip() or len(text) > 4000 for text in descriptions)
        ):
            raise DiscoveryInvalid("discovery hypothesis descriptions are invalid")
        if hypothesis["hypothesis_id"] in [h["hypothesis_id"] for h in hypotheses[: len(prototype_spans)]]:
            raise DiscoveryInvalid("discovery hypothesis identity is duplicated")
        start = len(texts)
        texts.extend(descriptions)
        prototype_spans.append((start, len(texts)))
    neutral_descriptions = config.get("abstention_prototypes", ABSTENTION_PROTOTYPES)
    if (
        not isinstance(neutral_descriptions, list)
        or len(neutral_descriptions) > 8
        or any(not isinstance(text, str) or not text.strip() or len(text) > 4000 for text in neutral_descriptions)
    ):
        raise DiscoveryInvalid("discovery abstention descriptions are invalid")
    neutral_start = len(texts)
    texts.extend(neutral_descriptions)
    neutral_stop = len(texts)
    episode_spans = []
    windows = 0
    for context in contexts:
        start = len(texts)
        text = "\n".join(record["text"] for record in context["records"])
        pieces = [text[index : index + window_chars] for index in range(0, len(text), window_chars)] or [""]
        texts.extend(pieces)
        windows += len(pieces)
        episode_spans.append((start, len(texts)))
    if windows > max_windows:
        raise DiscoveryInvalid("discovery episode windows exceed bound")
    feedback_vectors = {}
    records_by_id = {record["record_id"]: record for record in batch.get("records") or []}
    for item in feedback:
        example = item.get("example_text") or "\n".join(
            records_by_id[identity]["text"] for identity in item.get("record_ids") or [] if identity in records_by_id
        )
        if example:
            feedback_vectors[item["feedback_id"]] = len(texts)
            texts.append(example[:window_chars])
    vectors = _vectors(encoder or local_encoder(config), texts, batch_size) if contexts else []
    candidates = []
    zero_labels = 0
    unlocated = 0
    for context, (start, stop) in zip(contexts, episode_spans, strict=True):
        records = context["records"]
        candidate_id = str(
            uuid5(
                NAMESPACE_URL,
                "propria:adaptive-discovery:"
                + fingerprint(
                    [
                        batch["dataset_id"],
                        batch["scope"],
                        context["episode_id"],
                        [(record["record_id"], record.get("ordinal", 0)) for record in records],
                    ]
                ),
            )
        )
        scoped = [
            item
            for item in feedback
            if item.get("candidate_id") == candidate_id
            or set(item.get("record_ids") or []) & {record["record_id"] for record in records}
        ]
        scores = {}
        detectors = {}
        best_windows = {}
        label_neutral_scores = {}
        for hypothesis, (pstart, pstop) in zip(hypotheses, prototype_spans, strict=True):
            label = hypothesis["hypothesis_id"]
            score, best_window = max(
                (_similarity(window, prototype), index)
                for index, window in enumerate(vectors[start:stop])
                for prototype in vectors[pstart:pstop]
            )
            best_windows[label] = best_window
            label_neutral_scores[label] = max(
                (
                    _similarity(vectors[start + best_window], prototype)
                    for prototype in vectors[neutral_start:neutral_stop]
                ),
                default=-1.0,
            )
            adjustments = []
            for item in feedback:
                sign = _feedback_direction(item, label)
                index = feedback_vectors.get(item["feedback_id"])
                if sign and index is not None:
                    similarity = max(_similarity(window, vectors[index]) for window in vectors[start:stop])
                    adjustments.append(sign * max(0.0, similarity) * weight)
            score = round(max(0.0, min(1.0, score + sum(adjustments) / max(1, len(adjustments)))), 6)
            scores[label] = score
            detector = hypothesis.get("detector") or hypothesis.get("category") or label
            detectors[detector] = max(detectors.get(detector, 0), score)
        labels = sorted(
            label
            for label, score in scores.items()
            if score >= threshold and score > label_neutral_scores[label] + abstention_margin
        )
        if not labels:
            zero_labels += 1
            continue
        details = attribution(records, scoped)
        citations = _citations(
            records,
            [(best_windows[label] * window_chars, (best_windows[label] + 1) * window_chars) for label in labels],
        )
        if not citations:
            unlocated += 1
            continue
        if any(not record.get("locators") for record in records):
            details["uncertainty"].append("Some occurrences have no native cited locator.")
        if batch.get("truncated"):
            details["uncertainty"].append("The selected input batch was truncated upstream; context may be incomplete.")
        if any(item.get("kind") == "missing_context" for item in scoped):
            details["uncertainty"].append("A reviewer marked this episode as missing context.")
        candidates.append(
            {
                "candidate_id": candidate_id,
                "episode_id": context["episode_id"],
                "title": "Possible " + ", ".join(labels),
                "summary": "Review the cited episode context.",
                "labels": labels,
                "label_scores": scores,
                "detector_scores": detectors,
                "abstention_score": round(max(0.0, *(label_neutral_scores[label] for label in labels)), 6),
                "score": max(scores[label] for label in labels),
                "citations": citations,
                "rationale": "CPU description retrieval with scoped human exemplar adjustments.",
                "context_records": [
                    {key: value for key, value in record.items() if key != "text"} for record in records
                ],
                "review_state": "unreviewed",
                **details,
            }
        )
    candidates.sort(key=lambda candidate: (-candidate["score"], candidate["candidate_id"]))
    if len(candidates) > max_candidates:
        raise DiscoveryInvalid("candidate bound exceeded; select a smaller batch")
    pins = {
        field: batch.get(field, "")
        for field in (
            "selection_revision",
            "input_revision",
            "group_revision",
            "identity_revision",
            "feedback_revision",
        )
    }
    pins["input_revision"] = batch.get("input_revision") or fingerprint(
        [
            {
                key: record.get(key)
                for key in (
                    "record_id",
                    "source_version_id",
                    "generation_id",
                    "record_revision",
                    "input_revision",
                    "ordinal",
                )
            }
            for record in batch.get("records") or []
        ]
    )
    pins["group_revision"] = batch.get("group_revision") or fingerprint(
        [[context["episode_id"], [record["record_id"] for record in context["records"]]] for context in contexts]
    )
    pins["identity_revision"] = batch.get("identity_revision") or fingerprint(
        [
            [record["record_id"], record.get("participants"), record.get("attributed_actor_ids")]
            for record in batch.get("records") or []
        ]
    )
    pins["model_revision"] = config["model_revision"]
    pins["config_revision"] = fingerprint(config)
    return {
        "schema": "adaptive-discovery.result.v1",
        "run_id": batch["run_id"],
        "feedback_revision": batch["feedback_revision"],
        "model_version": MODEL_VERSION,
        "score_revision": SCORE_REVISION,
        "abstained": zero_labels + unlocated,
        "pins": pins,
        "candidates": candidates,
        "counts": {
            "episodes": len(contexts),
            "candidates": len(candidates),
            "zero_labels": zero_labels,
            "abstained": zero_labels + unlocated,
            "unlocated": unlocated,
        },
        "model_metadata": {
            "model_path": config.get("model_path", "injected"),
            "model_revision": config["model_revision"],
            "backend": config.get("backend", ""),
        },
    }
