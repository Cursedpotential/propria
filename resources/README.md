# Resources

This directory owns shared source material that is reusable across Propria
projects but is not itself a product runtime. Expected categories include:

- `design/` — the shared surface-design contract and portable tokens;
- `libraries/` — Propria-owned reusable packages after dependency review;
- `vendors/` — controlled third-party sources with license and provenance files;
- `references/` — architecture/reference material that is safe and useful to version;
- `schemas/` — cross-product interchange contracts where one project is not the owner.

`resources/` is not a data lake and is not a dumping ground. Evidence, recovered
files, cloud mirrors, local databases, embeddings, indexes, model caches, and
generated outputs stay outside Git and are represented by manifests or connection
configuration without credentials.
