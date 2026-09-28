# Receipt: the hand-made Homepage portal, as it ran on 2026-09-26

> _Byline: Claude Code · Opus 5.5 · 2026-09-26._

Byte-for-byte copies of the configuration the two hand-made Homepage containers on ovh-app served
on the night of 2026-09-26, taken before they were replaced by the declared portal
(`modules/Probata/probata/deploy/portal.yaml` + `deploy/portal/`). None of this was ever in git.
The copies are in the folder `2026-09-26-handmade-homepage/` beside this file; their MD5s match
the live files read from the host at 22:15 and again at 00:30 EDT.

| MD5 | File in `2026-09-26-handmade-homepage/` | Host path |
|---|---|---|
| `f1588512108b024fab281a32b630c651` | `dashboards-compose.yml` | `/data/dashboards/compose.yml` (defines `homepage` only) |
| `39f514ab1aa94125c33a3af9a82f18ab` | `tailnet/services.yaml` | `/data/dashboards/homepage/services.yaml` |
| `740866fe830d92efe923e3359ea23893` | `tailnet/settings.yaml` | `/data/dashboards/homepage/settings.yaml` |
| `718e0c7f9a6c34c9931ebc362f261ccf` | `public/services.yaml` | `/data/dashboards/homepage-public/services.yaml` |
| `43cb0fb6c993e5b6394a8bdd9f049417` | `public/settings.yaml` | `/data/dashboards/homepage-public/settings.yaml` |
| `4810544b3229acadbddabcc9cc74f921` | `*/widgets.yaml` | identical in both |
| `d8d40ac05bd0c315f751d6dc00e62027` | `*/bookmarks.yaml` | identical in both |
| `d3cd9150fe25c87593a59061ce6c17d9` | `*/custom.css` | identical in both |
| `ebd09eb77cad4cd373f20b9400612123` | `*/custom.js` | identical in both |
| `2cdd6259f1852818248e8e58bd94a916` | `*/docker.yaml` | identical in both (local Docker socket) |

## The containers

Both ran `ghcr.io/gethomepage/homepage:latest`, which resolved to v1.13.2
(`sha256:a0b71c8e757298d02560186bab9fbe3fc2d375c523a62cc1019177b37e48aa28`), restart
`unless-stopped`, network `dashboards_default`, with the host Docker socket mounted read-only.

- `homepage` — compose project `dashboards`, `/data/dashboards/compose.yml`. Port
  `100.72.169.40:3010->3000`, config bind `/data/dashboards/homepage:/app/config`,
  `HOMEPAGE_ALLOWED_HOSTS=100.72.169.40:3010,homepage.tilapia-skilift.ts.net`.
- `homepage-public` — no compose labels, no file anywhere that recreates it; started by hand. Its
  settings, read back with `docker inspect`, are equivalent to:

  ```
  docker run -d --name homepage-public --restart unless-stopped --network dashboards_default \
    -p 100.72.169.40:3012:3000 \
    -e HOMEPAGE_ALLOWED_HOSTS=100.72.169.40:3012,homepage.int.mitechconsult.com \
    -v /data/dashboards/homepage-public:/app/config \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    ghcr.io/gethomepage/homepage:latest
  ```

Defects this configuration carried (audit of 2026-09-26 and the before screenshots): the public
instance listed Coolify, Temporal, n8n, ContextForge, Portkey, Infisical and the portal editor,
linked Advocatio to the tailnet-only `legal.tilapia-skilift.ts.net`, and pointed two tiles at the
same `workbench.int/.../schemas` page; Filestash had no site monitor; the three-column Live board
left large empty bands and pushed the app buttons 2,400 px down the page; and a first-time browser
on the public instance got Homepage's build-time page (title "Homepage", no layout) because
nothing re-rendered its cached index after the container was recreated.
