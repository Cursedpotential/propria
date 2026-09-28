---
tags: [probata, librechat, mongo, receipt, transfer]
---

# LibreChat Mongo data: ovh-app copy brought to ovh-files (2026-09-27 04:15–04:17 UTC)

> _Byline: Claude Code · Fable 5.1 · 2026-09-26 (EDT). Receipt for the transfer signed off by the supervisor at 23:59 EDT ("A": exactly this scope). Log entry: `docs/planning/2026-09-20-TODO.md`._

**Why.** LibreChat's Mongo data existed twice: ovh-files `/data/probata/volumes/librechat/mongo` (last write 2026-08-06 02:54 UTC) and ovh-app, same path (last write 2026-09-19 07:54 UTC). Same WiredTiger lineage (ident suffix `-6051849212766268511`) and the same collection/index file set; the ovh-app copy is the newer one. Both had shut down cleanly (`mongod.lock` 0 bytes). LibreChat now runs on ovh-files, so it boots from the newer copy.

**What moved.** Nothing was deleted and the ovh-app original was not touched.
1. ovh-files `mongo/` (399 files, 425,493,501 bytes) moved aside to `/data/probata/volumes/librechat/_superseded/mongo-20260806/`. Manifest: `superseded-ovh-files-20260806.sha256`.
2. ovh-app `mongo/` streamed to ovh-files over the tailnet: `tar --numeric-owner -cp | zstd -3` → `zstd -d | tar --numeric-owner -xp`, 52 s. No container mounted the source (0 containers, no `mongod` process on ovh-app).

**Verified after write, before first boot.**

| | ovh-app source | ovh-files copy |
|---|---|---|
| files | 403 | 403 |
| directories | 5 | 5 |
| file bytes | 531,925,818 | 531,925,818 |
| manifest sha256 | `0c9dd54a7675f953516be08d1fea87e9c03c3fffcf683af3b45c8e9f028d2840` | `0c9dd54a7675f953516be08d1fea87e9c03c3fffcf683af3b45c8e9f028d2840` |

The two manifests are byte-identical (`cmp`). Ownership kept: `999:999` (Mongo's user), `.mongodb` `999:0`. Manifest form: `find . -type f -print0 | LC_ALL=C sort -z | xargs -0 sha256sum`, run inside each `mongo/` directory.

**Left in place.** ovh-app `/data/probata/volumes/librechat/` (the original, now an unused copy) and ovh-files `_superseded/mongo-20260806/`. Only the owner deletes.
