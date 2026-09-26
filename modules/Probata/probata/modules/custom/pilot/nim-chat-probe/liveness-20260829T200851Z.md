# NVIDIA NIM chat-generation liveness — 2026-08-29T20:08:51+00:00

- Catalog: 83 models; chat candidates: 75
- Live: 19; unavailable (404/410): 47; inconclusive: 9
- First pass: 8.0s with 2 workers; retry: 20.0s serialized

## Live

| Model | Stage | Latency | Reply | Tokens |
|---|---:|---:|---|---:|
| `nvidia/nemotron-3.5-content-safety` | fast | 0.31s | User Safety: safe | 474/5 |
| `nvidia/llama-3.1-nemoguard-8b-content-safety` | fast | 0.39s | {"User Safety": "safe"} | 402/8 |
| `nvidia/riva-translate-4b-instruct-v2` | fast | 0.41s | Reply with exactly the word PONG. | 21/9 |
| `meta/llama-3.2-11b-vision-instruct` | fast | 0.42s | PONG | 43/3 |
| `nvidia/ising-calibration-1.5-31b` | fast | 0.44s | PONG | 21/3 |
| `nvidia/riva-translate-4b-instruct-v1.1` | fast | 0.69s | Reply with the exact word PONG. | 21/9 |
| `nvidia/nemotron-3.5-lightning-30b-a3b` | retry | 0.76s | Here's a thinking process:

1.  **Analyze User Input:** The user says "Reply with exactly the word PONG."
2.  **Identify | 24/64 |
| `openai/gpt-oss-20b` | fast | 0.86s | PONG | 73/52 |
| `nvidia/llama-3.1-nemotron-safety-guard-8b-v3` | fast | 0.97s | {"User Safety": "safe"} | 402/9 |
| `openai/gpt-oss-120b` | fast | 0.97s | PONG | 73/57 |
| `meta/llama-3.2-90b-vision-instruct` | fast | 1.06s | PONG | 43/3 |
| `nvidia/nemotron-3-nano-30b-a3b` | fast | 1.21s | The user asks: "Reply with exactly the word PONG." So we must output exactly "PONG". No extra punctuation, no extra spac | 24/64 |
| `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning` | fast | 1.63s | PONG | 24/58 |
| `moonshotai/kimi-k3` | fast | 2.18s | PONG | 94/35 |
| `nvidia/nemotron-3-super-120b-a12b` | fast | 3.61s | PONG | 24/43 |
| `minimaxai/minimax-m3` | retry | 3.87s | PONG | 171/3 |
| `deepseek-ai/deepseek-v4-flash-0731` | retry | 4.64s | PONG | 12/26 |
| `google/diffusiongemma-26b-a4b-it` | fast | 6.14s |  | 21/1 |
| `meta/muse-glimmer-30b` | retry | 11.21s | Reply with exactly the word PONG.

We need reply with exactly the word PONG. Probably just PONG. Exactly the word PONG.  | 64/64 |

## Unavailable or inconclusive

| Model | Classification | Status | Stage | Detail |
|---|---|---:|---:|---|
| `01-ai/yi-large` | unavailable | 404 | fast | Function '23bd454d-b225-49a3-8118-582a62fc51b8': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `adept/fuyu-8b` | unavailable | 404 | fast | Function 'e598bfc1-b058-41af-869d-556d3c7e1b48': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ai21labs/jamba-1.5-large-instruct` | unavailable | 404 | fast | Function '6497fc2b-7ff8-4019-8946-123dccbfc863': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `aisingapore/sea-lion-7b-instruct` | unavailable | 404 | fast | Function '02f84bf4-c1a1-489b-a9de-ac3e8dcdec14': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `bigcode/starcoder2-15b` | unavailable | 404 | fast | 404 page not found
 |
| `databricks/dbrx-instruct` | unavailable | 404 | fast | Function '3d6c2ff8-8bfc-4d10-8fd0-b7337288e869': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `deepseek-ai/deepseek-coder-6.7b-instruct` | unavailable | 404 | fast | Function 'e503b15c-62b0-4d69-b532-a88f0bfa2656': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `deepseek-ai/deepseek-v4-pro-0813` | inconclusive | -1 | retry | The read operation timed out |
| `google/codegemma-1.1-7b` | unavailable | 404 | fast | Function 'e2d298c5-204e-4213-b921-9f492cc9011b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/codegemma-7b` | unavailable | 404 | fast | Function '7dfc10a8-3cc4-448e-97c1-2213308dc222': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/deplot` | unavailable | 404 | fast | Function '784a8ca4-ea7d-4c93-bb46-ec027c3fae47': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-2b` | unavailable | 404 | fast | Function '04174188-f742-4069-9e72-d77c2b77d3cb': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-3-12b-it` | unavailable | 404 | fast | Function 'ee47df99-c92b-4dc9-b3a7-f3fb0f087b73': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-3-4b-it` | unavailable | 404 | fast | Function 'c322f327-55a3-4af3-a91f-c757e2b8b135': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-4-31b-it` | inconclusive | -1 | retry | The read operation timed out |
| `google/recurrentgemma-2b` | unavailable | 404 | fast | Function '2f495340-a99f-4b4b-89bd-1beb003dd896': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-3.0-3b-a800m-instruct` | unavailable | 404 | fast | Function '67324577-3f91-4aa6-b750-97468262530d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-3.0-8b-instruct` | unavailable | 404 | fast | Function '5a24a4f0-2d59-46b3-ac65-42307f2633d1': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-34b-code-instruct` | unavailable | 404 | fast | Function '4df48b4f-e3c5-4ade-82c7-c06b65e25d18': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-8b-code-instruct` | unavailable | 404 | fast | Function 'af7b6f03-f615-4c5f-86c6-388bd35cede0': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `meta/codellama-70b` | unavailable | 404 | fast | Function 'f6b06895-d073-4714-8bb2-26c09e9f6597': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `meta/llama-guard-4-12b` | inconclusive | -1 | retry | The read operation timed out |
| `meta/llama2-70b` | unavailable | 404 | fast | Function '2fddadfb-7e76-4c8a-9b82-f7d3fab94471': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `microsoft/kosmos-2` | unavailable | 404 | fast | Function '6018fed7-f227-48dc-99bc-3fd4264d5037': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `microsoft/phi-3-vision-128k-instruct` | unavailable | 404 | fast | Function '20f2537e-8593-4eb9-ad40-60eee3bbaa55': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `microsoft/phi-3.5-moe-instruct` | unavailable | 404 | fast | Function 'e6cab982-62f4-481e-9a7a-3dedb87dbd01': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/codestral-22b-instruct-v0.1` | unavailable | 404 | fast | Function '9a10b012-e6df-46fd-83b2-700dcbc75814': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-7b-instruct-v0.3` | unavailable | 404 | fast | Function 'cd89bd68-13e3-47a9-861e-9a62e6e14b05': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-large` | unavailable | 404 | fast | Function '767b5b9a-3f9d-4c1d-86e8-fa861988cee7': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-large-2-instruct` | unavailable | 404 | fast | Function '7fadd4de-e22a-48e4-90e9-f02ef14a74b9': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-nemotron` | inconclusive | -1 | retry | The read operation timed out |
| `mistralai/mixtral-8x22b-v0.1` | unavailable | 404 | fast | Function '39655fc1-9ebc-4b24-963e-6915ea6680de': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `moonshotai/kimi-k2.6` | unavailable | 404 | fast | Function '23d4f03a-b8a6-4adb-a183-7daa083a09cc': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nv-mistralai/mistral-nemo-12b-instruct` | unavailable | 404 | fast | Function 'f8c05193-d2e2-4f0f-bb4d-7ad70070002b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/ai-synthetic-video-detector` | inconclusive | 500 | retry | Internal error while making inference request |
| `nvidia/cosmos-reason2-8b` | unavailable | 404 | fast | Function 'e199b43b-6c62-4a63-9379-f60e1a953236': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.1-nemoguard-8b-topic-control` | inconclusive | 500 | retry | Error during inference of request chat-ba429c35e5fd40c2b4f856d16d85dc2e -- Encountered an error in forwardAsync function: [TensorRT-LLM][ERROR] CUDA runtime error in cudaMemcpyAsync(dst, src.data(), src.getSizeInBytes(), cudaMemcpyDefault,  |
| `nvidia/llama-3.1-nemotron-51b-instruct` | unavailable | 404 | fast | Function '5beba52c-65a9-4f46-8cd9-656689a1b205': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.1-nemotron-70b-instruct` | unavailable | 404 | fast | Function '9b96341b-9791-4db9-a00d-4e43aa192a39': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.1-nemotron-ultra-253b-v1` | unavailable | 404 | fast | Function '84bf12ff-edbd-4435-baea-0fa6a7453d2e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama3-chatqa-1.5-70b` | unavailable | 404 | fast | Function '46594287-38b9-481c-a37f-baa02f2d3ba1': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/mistral-nemo-minitron-8b-8k-instruct` | unavailable | 404 | fast | Function '5aa06dd2-0a02-4a5d-be4c-bf88e956965d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nemotron-3-ultra-550b-a55b` | inconclusive | -1 | retry | The read operation timed out |
| `nvidia/nemotron-4-340b-instruct` | unavailable | 404 | fast | Function 'b0fcd392-e905-4ab4-8eb9-aeae95c30b37': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nemotron-4-340b-reward` | unavailable | 404 | fast | Function 'c53ee0e9-bad9-4e09-b365-52c9d6b71254': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nemotron-nano-3-30b-a3b` | unavailable | 404 | fast | Model not found |
| `nvidia/nemotron-parse` | inconclusive | 400 | fast | Content cannot be a plain string. The model does not support text input. Content cannot be a plain string. The model does not support text input. |
| `nvidia/neva-22b` | unavailable | 404 | fast | Function 'bc205f8e-1740-40df-8d32-c4321763498a': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/riva-translate-4b-instruct` | unavailable | 404 | fast | Function 'f35337fa-b4dd-4996-bcba-5476ee01171d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/vila` | unavailable | 404 | fast | Function '1c8df143-2303-419b-8b28-b4dd82cfe113': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `poolside/laguna-xs-2.1` | inconclusive | 503 | retry | ResourceExhausted: Worker local total request limit reached (55/32) |
| `writer/palmyra-creative-122b` | unavailable | 404 | fast | Function '00bdd0a7-e38f-4423-9007-c4d8730a3f78': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `writer/palmyra-fin-70b-32k` | unavailable | 404 | fast | Function '316490c6-f1ed-41f9-9da8-3fa9e885653b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `writer/palmyra-med-70b` | unavailable | 404 | fast | Function 'aab71274-5281-4941-b0b8-20f339d1fc7e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `writer/palmyra-med-70b-32k` | unavailable | 404 | fast | Function 'd6faa974-3591-49a4-963d-97221d074b2e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `zyphra/zamba2-7b-instruct` | unavailable | 404 | fast | Function '8378ffb2-51b0-4140-9684-dda1889373e6': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
