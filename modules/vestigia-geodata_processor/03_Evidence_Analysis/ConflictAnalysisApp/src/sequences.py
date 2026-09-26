from __future__ import annotations
from typing import Dict, Any, List
from datetime import datetime, timedelta
import math

def minutes_between(a: datetime, b: datetime) -> float:
    return abs((b - a).total_seconds())/60.0

def detect_sequences(df, rules_sequences: Dict[str, Any], sentiment_series=None) -> List[Dict[str, Any]]:
    """Detect sequences as defined in sequences.yaml against a pandas DataFrame `df`
    with columns: datetime, sender, message, behavior_tags (comma string optional).
    """
    seq_events = []
    if df is None or df.empty or not rules_sequences:
        return seq_events

    # Ensure sorted
    df = df.sort_values("datetime").reset_index(drop=True)
    tags_col = "behavior_tags" if "behavior_tags" in df.columns else None
    sentiment_col = "sentiment" if "sentiment" in df.columns else None

    def msg_tags(i) -> List[str]:
        if not tags_col: return []
        val = df.at[i, tags_col]
        if isinstance(val, float) and math.isnan(val): return []
        return [t.strip() for t in str(val).split(",") if t.strip()]

    for seq_name, spec in rules_sequences.items():
        steps = spec.get("steps", [])
        require_order = bool(spec.get("require_order", True))
        score = spec.get("score", 1)
        if not steps: continue

        # Sliding scan
        i = 0
        N = len(df)
        while i < N:
            start_i = i
            who_anchor = None  # track sender of previous step
            ok = True
            matched_idxs = [start_i]
            ts0 = df.at[start_i, "datetime"]
            sender0 = df.at[start_i, "sender"]
            # Check first step on i as candidate
            def step_matches(step_idx, msg_idx, prev_idx):
                step = steps[step_idx]
                sender = df.at[msg_idx, "sender"]
                tstamp = df.at[msg_idx, "datetime"]
                tags = msg_tags(msg_idx)
                # who
                who = step.get("who", "any")
                if who == "same" and prev_idx is not None:
                    if sender != df.at[prev_idx, "sender"]: return False
                if who == "other" and prev_idx is not None:
                    if sender == df.at[prev_idx, "sender"]: return False
                # tag_in
                want_tags = step.get("tag_in")
                if want_tags:
                    if not any(w in tags for w in want_tags):
                        return False
                # sentiment req
                want_sent = step.get("sentiment")
                if want_sent:
                    s = df.at[msg_idx, sentiment_col] if sentiment_col else 0.0
                    if want_sent == "negative" and s > - (step.get("intensity_min", 0.5)):
                        return False
                    if want_sent == "positive" and s < (step.get("intensity_min", 0.5)):
                        return False
                # silence window cannot be tested here (needs gap)
                return True

            # First step must match current i
            if not step_matches(0, i, None):
                i += 1
                continue

            prev_idx = i
            # Walk next steps
            cursor = i + 1
            step_i = 1
            while step_i < len(steps) and cursor < N:
                step = steps[step_i]
                within = step.get("within_minutes")
                # Special: silence_minutes_min
                if "silence_minutes_min" in step:
                    # Look ahead for next message and check gap
                    if cursor < N:
                        gap = minutes_between(df.at[prev_idx, "datetime"], df.at[cursor, "datetime"]) if cursor < N else 0
                        if gap >= float(step["silence_minutes_min"]):
                            matched_idxs.append(prev_idx)  # end on prev
                            step_i += 1
                            continue
                        else:
                            cursor += 1
                            continue
                # Co-occur terms (look within window)
                if "cooccur_terms" in step:
                    ok2 = False
                    j = cursor
                    while j < N:
                        if within and minutes_between(df.at[prev_idx, "datetime"], df.at[j, "datetime"]) > float(within):
                            break
                        msg = str(df.at[j, "message"] or "").lower()
                        if any(term.lower() in msg for term in step["cooccur_terms"]):
                            ok2 = True
                            matched_idxs.append(j)
                            prev_idx = j
                            step_i += 1
                            cursor = j + 1
                            break
                        j += 1
                    if not ok2:
                        ok = False
                        break
                    else:
                        continue
                # Normal step: find next message that satisfies step
                found = False
                j = cursor
                while j < N:
                    if within and minutes_between(df.at[prev_idx, "datetime"], df.at[j, "datetime"]) > float(within):
                        break
                    if step_matches(step_i, j, prev_idx):
                        matched_idxs.append(j)
                        prev_idx = j
                        cursor = j + 1
                        found = True
                        break
                    j += 1
                if not found:
                    ok = False
                    break
                step_i += 1

            if ok and step_i == len(steps):
                event = {
                    "sequence_type": seq_name,
                    "score": score,
                    "start_index": int(matched_idxs[0]),
                    "end_index": int(matched_idxs[-1]),
                    "message_indices": [int(x) for x in matched_idxs],
                    "start_datetime": str(df.at[matched_idxs[0], "datetime"]),
                    "end_datetime": str(df.at[matched_idxs[-1], "datetime"]),
                    "participants": list({df.at[k, "sender"] for k in matched_idxs}),
                }
                seq_events.append(event)
                # Advance cursor beyond match to avoid redundant overlaps
                i = matched_idxs[-1] + 1
            else:
                i += 1

    return seq_events
