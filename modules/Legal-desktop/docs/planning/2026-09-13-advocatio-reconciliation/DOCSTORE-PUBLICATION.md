# Advocatio Docstore publication

Posted and independently read back on 2026-09-13: **80 documents, notes and inventories**. This includes the planning package, routing documents, reconciliation/reference receipts, generated inventories and the private review notes produced during this task. Original third-party archives and extracted donor source were not uploaded as authored notes.

The existing Docstore CLI and dedicated named service were used. Documents under `docs/` use version capture where supported; other notes/large report inventories use governed document registration. Each record's text hash was checked after the write. The accompanying [record manifest](DOCSTORE-PUBLICATION.json) contains source paths, record/revision IDs and verification hashes. Its entries retain original owning paths; private note bodies remain outside the Git commit.

Roadmap logical key: `document:advocatio_de27d527b5894eafbbc6ae5e30787f60`.
Roadmap record: `docstore_document:dc53c8991b130561c0aeef06bbdd380946c33b0cb4fe37338a192cc0a4136ada`.

The publication receipts are also captured in Docstore, and a searchable Advocatio publication note points to the roadmap and manifest. The publication note records persistence findings; it does not mark every proposed design decision as approved.

Search embedding/index refresh was not run. This receipt establishes stored content and readback, independently of that pipeline. The commit contains the local roadmap and source reports so another model can continue from [ROADMAP.md](ROADMAP.md).
