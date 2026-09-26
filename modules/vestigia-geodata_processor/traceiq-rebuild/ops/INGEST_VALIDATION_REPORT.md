# Ingest Validation Report

Generated: 2026-07-24

## Per-chunk ingest proof

| chunk | source file | size | source records | DB records | match | hash |
|---|---|---:|---:|:---:|---:|
| 1 | `2024-03_2024-06.json.txt` | 1305007 | 1641 | 1641 | ✓ | ✓ |

| type | source count | DB count | match |
|---|---|---:|:---:|
| semanticSegments.visit | 626 | 626 | ✓ |
| semanticSegments.activity | 516 | 516 | ✓ |
| semanticSegments.timelinePath | 498 | 498 | ✓ |
| semanticSegments.timelineMemory | 1 | 1 | ✓ |

| 2 | `Copy of K- Timeline.json | variant A - K-named original` | 13795216 | 20161 | 20161 | ✓ | ✓ |

| type | source count | DB count | match |
|---|---|---:|:---:|
| semanticSegments.visit | 7236 | 7236 | ✓ |
| semanticSegments.activity | 6375 | 6375 | ✓ |
| semanticSegments.timelinePath | 6545 | 6545 | ✓ |
| semanticSegments.timelineMemory | 4 | 4 | ✓ |
| userLocationProfile | 1 | 1 | ✓ |

| 3 | `Records-1.json x4 mirrors)` | 13795202 | 20161 | 20161 | ✓ | ✓ |

| type | source count | DB count | match |
|---|---|---:|:---:|
| semanticSegments.visit | 7236 | 7236 | ✓ |
| semanticSegments.activity | 6375 | 6375 | ✓ |
| semanticSegments.timelinePath | 6545 | 6545 | ✓ |
| semanticSegments.timelineMemory | 4 | 4 | ✓ |
| userLocationProfile | 1 | 1 | ✓ |

## Working-layer reconciliation

- build_id: 3
- raw_count: 41963
- derived_count: 40320
- exception_count: 1643
- unaccounted: 0 (✓ PASS)

## Summary: PASS

