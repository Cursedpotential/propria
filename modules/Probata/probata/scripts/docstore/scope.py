"""The owner-authorized Docstore source roots, relative to the Propria monorepo root.

Owner ruling 2026-09-26: seven roots, not five. Vestigia and Family Court Workbench
are real modules with real docs, so they are indexed like the rest.
Byline: Claude Code / Opus 5 / 2026-09-26.

TWO THINGS THIS FILE GETS RIGHT ON PURPOSE -- do not "simplify" either one.

1. SOURCE ROOT IS A JUNCTION UNDER Propria/docs, NEVER A MODULE PATH.
   Every root below reads `docs/<project>`, a junction that Propria/docs holds into
   the owning module's docs directory. The previous version of this file hard-coded
   module paths (`Probata/probata/docs`, `Consignatio/docs`, `Legal-desktop/docs`);
   the 2026-09-20 move to `modules/*` invalidated all three, and this validator then
   rejected the desktop registry outright, which is what stopped the indexer from
   running anywhere but the container. The junction survives a module move; a module
   path does not. Keep reading through the junctions.

2. CANONICAL PREFIX IS DOCUMENT IDENTITY, AND THE TRAILING SLASH IS PART OF IT.
   The prefixes below are byte-for-byte what the live store was built with, trailing
   slash included. Changing one does not rename a document, it creates a different
   document and orphans the old one -- which is why validate_registry refuses a
   mismatch instead of accepting it. `vestigia` and `family-court-workbench` are new
   identities and take the same slash-terminated form.

Each value is (source_root, canonical_prefix).
"""

ROOTS = {
    'propria': ('docs', 'propria/docs/'),
    'probata': ('docs/probata', 'docs/'),
    'consignatio': ('docs/consignatio', 'consignatio/docs/'),
    'consignatio-intake': ('docs/consignatio-intake', 'consignatio/Intake/docs/'),
    'advocatio': ('docs/advocatio', 'advocatio/docs/'),
    'vestigia': ('docs/vestigia', 'vestigia/traceiq-rebuild/docs/'),
    'family-court-workbench': ('docs/family-court', 'family-court-workbench/docs/'),
}


def validate_registry(payload):
    active = [p for p in payload.get('projects', []) if p.get('registration_status') == 'active']
    if len(active) != len(ROOTS) or {p.get('project_id') for p in active} != set(ROOTS):
        raise ValueError(
            f'Docstore requires exactly the {len(ROOTS)} approved docs roots: '
            f'{", ".join(sorted(ROOTS))}'
        )
    for entry in active:
        root, prefix = ROOTS[entry['project_id']]
        if entry.get('source_root', '').replace('\\', '/').rstrip('/') != root:
            raise ValueError(
                f"Source root is outside approved docs scope: {entry['project_id']} "
                f"declares {entry.get('source_root')!r}, approved is {root!r} "
                f'(roots read through the Propria/docs junctions, not module paths)'
            )
        if entry.get('canonical_prefix') != prefix:
            raise ValueError(
                f"Canonical prefix change requires an identity migration: "
                f"{entry['project_id']} declares {entry.get('canonical_prefix')!r}, "
                f'the store was built with {prefix!r}'
            )
        if entry.get('ingestion_status') == 'excluded' or not entry.get('required', True):
            raise ValueError(f"All {len(ROOTS)} roots are required: {entry['project_id']} is not")
        exclusions = entry.get('excluded_patterns', [])
        if 'private/**' not in exclusions or '**/to_be_deleted/**' not in exclusions:
            raise ValueError(
                f"Private/quarantine exclusions are mandatory: {entry['project_id']} "
                f'must exclude both private/** and **/to_be_deleted/**'
            )
    return payload
