# Rclone R2 credential recovery inventory — 2026-09-23

> Byline: Codex · GPT-6 · 2026-09-23. Read-only credential-location audit and exact-bucket probes. Credential values were not copied into this receipt.

## Outcome

No distinct, proven-entitled Cloudflare R2 credential was found for both `casebible-sorted` and `nexus`. The Probata Workbench Coolify application `xjbuo6drbwjfby75lalk8bk7` was not changed or redeployed. Its existing degraded R2 state remains open. This audit does not establish a scoped write grant for `nexus/workbench/staging/`.

## Redacted inventory

Fingerprints below are the first 16 hexadecimal characters of SHA-256 over either the access-key ID or the string `access-key-id|secret-access-key`. They identify equality within this audit; they are not credentials. The endpoint fingerprint is SHA-256 of the configured endpoint string.

| Configuration locations inspected | Remote | Type/provider | Access ID fingerprint | Pair fingerprint | Endpoint fingerprint | Finding |
|---|---|---|---|---|---|---|
| Windows `C:/Users/matts/.config/rclone/rclone.conf`; `AppData/Roaming/rclone/rclone.conf`; Scoop `current/rclone.conf` and `persist/rclone/rclone.conf`; two dated Scoop backups | `r2` | S3/Cloudflare | `54562ea7948d6df7` | `98b008534c228a91` | `b0894992236ca1ef` | One repeated credential. |
| `ovh-app:/var/lib/docker-plugins/rclone/config/rclone.conf`; `ovh-files:/root/.config/rclone/rclone.conf`, `/home/ubuntu/.config/rclone/rclone.conf`, `/opt/casebible/rclone.conf`, and one dated root backup | `r2` | S3/Cloudflare | `54562ea7948d6df7` | `98b008534c228a91` | `b0894992236ca1ef` | Same repeated credential. |
| Windows `C:/Users/matts/.config/rclone/rclone.conf` | `R@` | S3/Cloudflare | `e4eb1170d56537fd` | `7952ac9fe12db54f` | `fbaf55e92074b527` | Distinct saved pair. Saved endpoint is a nonexistent historical local `.env` path, not an S3 URL; `env_auth=true`, no `old_access_key_id`, no region, and invalid 5 KiB upload chunk size. |
| `ovh-files:/root/.config/rclone/rclone.conf` | `b2s3`; `b2native-full` | S3/Other; native Backblaze B2 | Distinct from Cloudflare | Distinct from Cloudflare | Backblaze endpoint / none | Backblaze credentials, not evidence of Cloudflare R2 entitlement. |

The Windows file-name search found the three active configuration locations plus Scoop's persisted file; the known dated backups were also inspected. The scoped server search covered the rclone plugin path and likely `/root`, `/home/ubuntu`, `/opt/casebible`, and `/data/probata` configuration locations. A final scoped search on `ion-control` found no rclone configuration at those paths. This is an inventory of plausible configured sources, not a claim about every file on every host.

## Exact-bucket request results

Read-only `rclone lsf` requests targeted the root of each required bucket. The repeated `r2` credential returned `NotEntitled` for `casebible-sorted` and `nexus`. The `R@` remote, invoked with its exact saved configuration, failed local validation for both buckets because its configured 5 KiB chunk size is below rclone's 5 MiB minimum; neither request reached Cloudflare. The historical `.env` endpoint path referenced by `R@` does not exist and was not created or sourced.

The distinct saved `R@` pair was then tested with command-line overrides for the actual Cloudflare account endpoint, valid 5 MiB chunk size, and `env_auth=false`. Both bucket requests returned `SignatureDoesNotMatch`; neither established identity or entitlement. Rclone's local S3 help states that `env_auth=true` only obtains runtime credentials when both configured key fields are blank; both `R@` fields are populated. No process `AWS_*`, `RCLONE_CONFIG_*`, or `R2_*` environment variables and no user `.aws/credentials` or `.aws/config` file were present in the checked Windows session. No object contents were printed or changed.

## Recovery boundary

The earlier [Workbench deployment receipt](2026-09-23-workbench-p0-deployment-receipt.md) identifies the mounted Workbench key as the repeated `r2` identity and records its 403 `NotEntitled` result. Replacing it with that same key or with the unverified `R@` pair would not constitute recovery. A Cloudflare R2 credential owner needs to provide or authorize a distinct key for the correct R2 account endpoint with Object Read on `casebible-sorted` and Object Read & Write on `nexus`. Exact list/read on both buckets and the staging write scope must be proved before a Coolify environment update. A verified key can then be installed through the Coolify API and checked on the live Tailnet Workbench surface.

No Coolify environment, application lifecycle, Cloudflare policy, R2 bucket, B2 bucket, source file, or host service was modified in this investigation.
