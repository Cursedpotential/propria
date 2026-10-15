"""Generate read-only ContextForge REST registrations from LLM Probes tool docstrings.

Inputs: none. Outputs: registration dictionaries for two fixed service routes.
Effects: none; registration is performed separately with authenticated gateway APIs.
Use these adapters to inspect existing probe data without starting provider calls.
"""


def tracked_models():
    """List the models already tracked by the personal LLM Probes service.

    Inputs: none. Outputs: existing tracked-model records, including provider and
    model identity. Side effects: reads stored records only; does not start a
    probe, contact a model provider, or change tracking. Choose this tool to see
    what is being tracked; choose results-summary for aggregate probe outcomes.
    """
    return '/providers/tracked-models/all'


def results_summary():
    """Read aggregate results from previously completed personal LLM probes.

    Inputs: none. Outputs: counts for tested models, liveness and capability
    outcomes. Side effects: reads existing results only; does not start probes
    or contact model providers. Choose this tool for aggregate outcomes rather
    than the tracked-model catalog; counts do not establish current model health.
    """
    return '/results/summary'


def registrations():
    """Build fixed-route REST tool definitions using their source docstrings.

    Inputs: none. Outputs: ContextForge ToolCreate dictionaries. Side effects:
    none. Use during registration; never accept caller-controlled upstream URLs.
    """
    base = 'http://100.91.190.107:8030'
    return [
        {
            'name': 'llm-probes-' + fn.__name__.replace('_', '-'),
            'description': fn.__doc__,
            'url': base + fn(),
            'integration_type': 'REST',
            'request_type': 'GET',
            'inputSchema': {'type': 'object', 'properties': {}, 'additionalProperties': False},
            'annotations': {'readOnlyHint': True, 'destructiveHint': False,
                            'idempotentHint': True, 'openWorldHint': False},
            'visibility': 'public',
            'expose_passthrough': False,
            'timeout_ms': 20000,
            'tags': ['propria', 'llm-probes', 'read-only'],
        }
        for fn in (tracked_models, results_summary)
    ]
