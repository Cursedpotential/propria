"""The five owner-authorized source roots, relative to Propria."""
ROOTS = {
    'propria': ('docs', 'propria/docs/'),
    'probata': ('Probata/probata/docs', 'docs/'),
    'consignatio': ('Consignatio/docs', 'consignatio/docs/'),
    'consignatio-intake': ('Consignatio/Intake/docs', 'consignatio/Intake/docs/'),
    'advocatio': ('Legal-desktop/docs', 'advocatio/docs/'),
}


def validate_registry(payload):
    active = [p for p in payload.get('projects', []) if p.get('registration_status') == 'active']
    if len(active) != 5 or {p.get('project_id') for p in active} != set(ROOTS):
        raise ValueError('Docstore requires exactly the five approved docs roots')
    for entry in active:
        root, prefix = ROOTS[entry['project_id']]
        if entry.get('source_root', '').replace('\\', '/').rstrip('/') != root:
            raise ValueError('Source root is outside approved docs scope')
        if entry.get('canonical_prefix') != prefix:
            raise ValueError('Canonical prefix change requires an identity migration')
        if entry.get('ingestion_status') == 'excluded' or not entry.get('required', True):
            raise ValueError('All five roots are required')
        exclusions = entry.get('excluded_patterns', [])
        if 'private/**' not in exclusions or '**/to_be_deleted/**' not in exclusions:
            raise ValueError('Private/quarantine exclusions are mandatory')
    return payload
