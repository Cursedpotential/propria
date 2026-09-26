# NIM embedders — live probe 2026-08-26T12:12:14+00:00

> _Byline: nim-embed-probe · 2026-08-26_

- `/models` listed 83 models total
- endpoint: `https://integrate.api.nvidia.com/v1`

## WORKING

| model | dim | mode | batch of 4 (s) | in /models |
|---|---|---|---|---|
| `nvidia/llama-nemotron-embed-vl-1b-v2` | 2048 | asymmetric(input_type required) | 0.63 | yes |
| `nvidia/nemotron-3-embed-1b` | 2048 | symmetric | 0.43 | yes |

## NOT WORKING

| model | status | detail |
|---|---|---|
| `nvidia/embed-qa-4` | 404 | Function '09c64e32-2b65-4892-a285-2f585408d118': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.2-nemoretriever-1b-vlm-embed-v1` | 404 | Function '6cb5bc77-9adc-48ac-85e2-c0ebceda934f': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.2-nv-embedqa-1b-v1` | 404 | Function 'b51e8011-e772-4c9a-8b02-618e99ae4467': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nv-embedqa-mistral-7b-v2` | 404 | Function '6caf65cf-1c3a-4823-89a5-21b821640d99': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nvclip` | 404 | Function '3072eebf-b0f0-4318-a5b8-5a45cd035b95': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `snowflake/arctic-embed-l` | 404 | Function '1528a0ad-205a-46ac-a783-94e2372586a9': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nv-embed-v1` | 410 | The model 'nvidia/nv-embed-v1' has reached its end of life on 2026-08-25T09:00:00Z and is no longer available. |
| `baai/bge-m3` | 410 | The model 'baai/bge-m3' has reached its end of life on 2026-08-25T09:00:00Z and is no longer available. |
| `nvidia/nv-embedqa-e5-v5` | 410 | The model 'nvidia/nv-embedqa-e5-v5' has reached its end of life on 2026-08-25T09:00:00Z and is no longer available. |
| `nvidia/llama-3.2-nv-embedqa-1b-v2` | 410 | The model 'nvidia/llama-3.2-nv-embedqa-1b-v2' has reached its end of life on 2026-05-18T00:00:00Z and is no longer available. |

## Changes vs previous run

- (no previous run / no changes)
