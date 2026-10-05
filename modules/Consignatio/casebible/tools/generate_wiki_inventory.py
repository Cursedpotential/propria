#!/usr/bin/env python3
"""Generate the Case Vault tool wiki from manifests, source docstrings and live registries.

Inputs: plugin marketplace root, output folder, and optional live-registry flag.
Outputs: Obsidian Markdown, JSON inventory, CSV command index and source hashes.
Side effects: writes only the selected documentation output directory; reads registries.
Choose this for documentation discovery, never for indexing corpora or executing tools.
Byline: Codex · GPT-6 · 2026-10-04.
"""
from __future__ import annotations

import argparse
import ast
import csv
import hashlib
import importlib.util
import json
import re
import tomllib
from datetime import datetime, timezone
from pathlib import Path

IGNORE = {'.git', '.venv', 'venv', 'node_modules', '__pycache__', 'to_be_deleted', '_stale', '.codex-plugin'}
DATE = '2026-10-04'
BYLINE = 'Codex · GPT-6 · 2026-10-04 — generated from the cited sources.'


def origin(path: Path, line: int = 1) -> dict:
    """Identify source bytes and a pinpoint; input path/line, output citation, read only.

    Choose this for inventory evidence rather than inferring truth from a URL.
    """
    return {'path': path.as_posix(), 'line': line, 'sha256': hashlib.sha256(path.read_bytes()).hexdigest()}


def frontmatter(path: Path) -> dict:
    """Read descriptive Markdown metadata; input note, output fields, no writes.

    Choose for skill/command identity, not runtime invocation validation.
    """
    text = path.read_text(encoding='utf-8-sig')
    if not text.startswith('---'):
        return {'description': next((x.lstrip('# ') for x in text.splitlines() if x.strip()), '')}
    block = text.split('---', 2)[1]
    try:
        import yaml
        value = yaml.safe_load(block)
        return value if isinstance(value, dict) else {}
    except Exception as exc:
        result = {'metadata_parse_note': f'Frontmatter fallback: {type(exc).__name__}'}
        for key in ('name', 'description', 'argument-hint'):
            match = re.search(rf'^{re.escape(key)}:\s*(.*?)(?=\n[\w-]+:|\Z)', block, re.M | re.S)
            if match:
                result[key] = ' '.join(match.group(1).strip().strip('"\'').split())
        return result


def cell(value) -> str:
    """Escape a table cell; input arbitrary value, output Markdown text, no side effects.

    Choose for short inventory rows; full descriptions stay in the JSON and detail pages.
    """
    return ' '.join(str(value if value is not None and value != '' else '—').split()).replace('|', '\\|').replace('`', "'")


def citation(value: dict) -> str:
    """Render a source locator; input source identity, output link and hash, no writes.

    Choose for per-item verification references.
    """
    return f"[{Path(value['path']).name}:{value['line']}](<{value['path']}:{value['line']}>) · SHA-256 `{value['sha256']}`"


def note(title: str, body: str, status: str = 'SOURCE_VERIFIED') -> str:
    """Wrap an Obsidian note; input title/body/status, output Markdown, no writes.

    Choose for generated wiki pages; human guides may supply their own frontmatter.
    """
    return (f'---\ntitle: {json.dumps(title)}\ntype: tool-reference\nstatus: {status}\n'
            f'date: {DATE}\ngenerated_by: "Codex / GPT-6"\nrevision: 1\n'
            'tags: [propria, tools, wiki]\n---\n\n'
            f'# {title}\n\n> _Byline: {BYLINE}_\n\n{body}\n')


def python_surface(path: Path) -> dict:
    """Extract Python tool declarations and CLI flags without importing application code.

    Inputs: source path. Outputs: descriptions, tool signatures, flags and citations.
    Side effects: none. Choose for discovery when executing unknown scripts would be unsafe.
    """
    tree = ast.parse(path.read_text(encoding='utf-8-sig'))
    tools, flags, arguments, subcommands = [], [], [], []
    for node in ast.walk(tree):
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and node.func.attr == 'add_argument':
            names = [a.value for a in node.args if isinstance(a, ast.Constant) and isinstance(a.value, str)]
            flags.extend(names)
            contract = {'names': names, 'source_line': node.lineno}
            for keyword in node.keywords:
                if keyword.arg in {'help', 'required', 'nargs', 'choices', 'action', 'dest'}:
                    try:
                        contract[keyword.arg] = ast.literal_eval(keyword.value)
                    except (ValueError, TypeError):
                        contract[keyword.arg] = 'computed in source'
                elif keyword.arg == 'type':
                    contract['type'] = ast.unparse(keyword.value)
            arguments.append(contract)
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and node.func.attr == 'add_parser' and node.args:
            if isinstance(node.args[0], ast.Constant) and isinstance(node.args[0].value, str):
                item = {'name': node.args[0].value, 'source_line': node.lineno}
                for keyword in node.keywords:
                    if keyword.arg in {'help', 'description'} and isinstance(keyword.value, ast.Constant):
                        item[keyword.arg] = keyword.value.value
                subcommands.append(item)
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            decorated = [ast.unparse(d.func if isinstance(d, ast.Call) else d) for d in node.decorator_list]
            if any(d.endswith('.tool') or d == 'tool' for d in decorated):
                name = node.name
                for decorator in node.decorator_list:
                    if isinstance(decorator, ast.Call):
                        for keyword in decorator.keywords:
                            if keyword.arg == 'name' and isinstance(keyword.value, ast.Constant):
                                name = keyword.value.value
                parameters = [{'name': a.arg, 'type': ast.unparse(a.annotation) if a.annotation else 'not declared'}
                              for a in (*node.args.posonlyargs, *node.args.args, *node.args.kwonlyargs) if a.arg not in {'self', 'cls'}]
                tools.append({'name': name, 'description': ast.get_docstring(node) or 'Missing docstring.',
                              'parameters': parameters, 'source': origin(path, node.lineno), 'status': 'source declaration; invocation not tested'})
        # The Search stdio server builds its explicit MCP catalog with tool(name, description, ...).
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id == 'tool' and len(node.args) >= 2:
            if all(isinstance(a, ast.Constant) and isinstance(a.value, str) for a in node.args[:2]):
                tools.append({'name': node.args[0].value, 'description': node.args[1].value,
                              'parameters': [], 'source': origin(path, node.lineno), 'status': 'source catalog; invocation not tested'})
    return {'description': ast.get_docstring(tree) or 'Missing module docstring.', 'tools': tools,
            'flags': sorted(set(flags)), 'arguments': arguments, 'subcommands': subcommands, 'source': origin(path),
            'entry_point': any(isinstance(n, ast.If) and '__name__' in ast.unparse(n.test) for n in tree.body)}


def plugin_inventory(root: Path, registration: dict, installed: dict) -> dict:
    """Inventory one plugin from its declared source, excluding runtime and retired trees.

    Inputs: marketplace root, registration, installed-name sets. Outputs: exposed surfaces.
    Side effects: source reads only. Choose for catalog discovery, not service-health claims.
    """
    base = (root / registration['source']).resolve()
    result = {'name': registration['name'], 'version': registration.get('version'),
              'description': registration.get('description', ''), 'source_root': base.as_posix(),
              'registered': registration.get('registered', True),
              'installed_in': [app for app, names in installed.items() if registration['name'] in names],
              'skills': [], 'commands': [], 'agents': [], 'mcp_tools': [], 'mcp_servers': [], 'scripts': [], 'cli_entries': [], 'errors': []}
    if not base.exists():
        result['errors'].append('Declared source directory missing.'); return result
    for kind, pattern in [('skills', 'skills/**/SKILL.md'), ('commands', 'commands/**/*.md'), ('agents', 'agents/**/*.md')]:
        for path in sorted(base.glob(pattern)):
            if any(part in IGNORE for part in path.relative_to(base).parts):
                continue
            meta = frontmatter(path)
            if meta.get('metadata_parse_note'):
                result['errors'].append(f"{path.relative_to(base).as_posix()}: {meta['metadata_parse_note']}")
            result[kind].append({'name': meta.get('name') or (path.parent.name if kind == 'skills' else path.stem),
                                 'description': meta.get('description', 'Description not supplied.'),
                                 'argument_hint': meta.get('argument-hint'), 'source': origin(path),
                                 'invocation': f"/{registration['name']}:{path.stem}" if kind == 'commands' else None})
    for filename in ('.mcp.json', '.codex-plugin/mcp.json'):
        path = base / filename
        if path.exists():
            try:
                config = json.loads(path.read_text(encoding='utf-8-sig'))
                for name, server in config.get('mcpServers', {}).items():
                    result['mcp_servers'].append({'name': name, 'transport': server.get('type') or ('stdio' if 'command' in server else 'http'),
                                                 'configuration_names': sorted(server.get('env', {}).keys()),
                                                 'source': origin(path), 'status': 'configured; health not inferred'})
            except (ValueError, TypeError) as exc:
                result['errors'].append(f'{filename}: {type(exc).__name__}')
    for path in sorted(base.rglob('*.py')):
        relative = path.relative_to(base)
        if any(part in IGNORE or part.startswith('test') for part in relative.parts):
            continue
        try:
            surface = python_surface(path)
        except (SyntaxError, UnicodeError) as exc:
            result['errors'].append(f'{relative.as_posix()}: {type(exc).__name__}'); continue
        result['mcp_tools'].extend(surface.pop('tools'))
        if surface['flags'] or surface['entry_point'] or relative.parts[0] in {'tools', 'scripts', 'hooks'}:
            surface.update(name=relative.as_posix(), classification='CLI candidate' if surface['flags'] else 'helper or hook; not established as a human command',
                           invocation=f'python "{path.as_posix()}" --help' if surface['flags'] else None,
                           status='source inspected; execute only when the documented owning runtime permits')
            result['scripts'].append(surface)
    for path in sorted(base.rglob('*.cmd')):
        if not any(part in IGNORE for part in path.relative_to(base).parts):
            result['cli_entries'].append({'name': path.name, 'invocation': f'& "{path.as_posix()}" --help', 'source': origin(path),
                                          'status': 'launcher present; help/runtime proof is recorded separately'})
    for path in sorted(base.rglob('pyproject.toml')):
        if any(part in IGNORE for part in path.relative_to(base).parts):
            continue
        try:
            config = tomllib.loads(path.read_text(encoding='utf-8'))
            for name, target in config.get('project', {}).get('scripts', {}).items():
                result['cli_entries'].append({'name': name, 'target': target, 'invocation': f'{name} --help',
                                              'source': origin(path), 'status': 'entry point declared; installation/PATH not inferred'})
        except ValueError as exc:
            result['errors'].append(f'{path.name}: {type(exc).__name__}')
    return result


def live_inventory(root: Path) -> dict:
    """Read existing gateway and ContextForge registries using the toolbox adapter.

    Input: marketplace root. Output: allowlisted public metadata and explicit failures.
    Side effects: authenticated GETs only; credential values never reach the inventory.
    Choose instead of duplicating credential handling or inventing another tool registry.
    """
    path = root / 'plugins/propria-toolbox/scripts/generate_catalog.py'
    spec = importlib.util.spec_from_file_location('existing_toolbox_catalog', path)
    adapter = importlib.util.module_from_spec(spec); spec.loader.exec_module(adapter)
    result = {'source': origin(path), 'checked_at': datetime.now(timezone.utc).isoformat(), 'gateway_tools': [], 'servers': [], 'contextforge_tools': [], 'errors': []}
    try:
        values = adapter.fetch_gateway()
        fields = {'id', 'name', 'description', 'capability', 'formats', 'side_effect', 'execution_policy', 'input_schema', 'inputSchema', 'output_schema', 'outputSchema', 'version', 'provenance', 'tool_version', 'contract_version', 'input_schema_version', 'output_schema_version', 'quality'}
        result['gateway_tools'] = [{k: v for k, v in item.items() if k in fields} for item in values]
    except (Exception, SystemExit) as exc:
        result['errors'].append(f'tool gateway GET /tools failed: {type(exc).__name__}')
    try:
        token = adapter.cf_token()
        servers = adapter.cf_get('/servers?include_inactive=true', token)
        tools = adapter.cf_get('/tools?include_inactive=true&limit=0', token)
        server_fields = {'id', 'name', 'description', 'associatedTools', 'isActive', 'enabled'}
        tool_fields = {'id', 'name', 'description', 'inputSchema', 'input_schema', 'outputSchema', 'annotations', 'enabled', 'gatewayId', 'gateway_id', 'originalName', 'original_name'}
        result['servers'] = [{k: v for k, v in item.items() if k in server_fields} for item in servers]
        result['contextforge_tools'] = [{k: v for k, v in item.items() if k in tool_fields} for item in tools]
    except (Exception, SystemExit) as exc:
        result['errors'].append(f'ContextForge registry GET failed: {type(exc).__name__}')
    return result


def write_plugin_page(output: Path, plugin: dict) -> None:
    """Write one complete plugin reference from extracted source descriptions.

    Inputs: output folder, plugin inventory. Output: Markdown page.
    Side effects: documentation file write only. Choose for readable per-plugin detail.
    """
    body = [plugin['description'], '', f"Source: `{plugin['source_root']}`. Version: `{plugin['version']}`.",
            f"Registered: `{plugin['registered']}`. Installed manifests: {', '.join(plugin['installed_in']) or 'not found in inspected manifests'}.",
            '', '## How to invoke it', '',
            'Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.',
            '', 'Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.', '']
    for kind in ('commands', 'skills', 'agents', 'cli_entries', 'scripts', 'mcp_tools', 'mcp_servers'):
        body.extend([f"## {kind.replace('_', ' ').title()}", ''])
        if not plugin[kind]:
            body.extend(['No entries found in the inspected declarations.', '']); continue
        for item in plugin[kind]:
            body.extend([f"### `{item['name']}`", '', str(item.get('description') or item.get('classification') or item.get('transport') or ''), ''])
            if item.get('invocation'):
                body.extend(['```text', item['invocation'], '```', ''])
            if item.get('argument_hint'):
                body.extend([f"Arguments: `{item['argument_hint']}`", ''])
            if item.get('flags'):
                body.extend(['Declared arguments: ' + ', '.join(f'`{f}`' for f in item['flags']), ''])
            if item.get('subcommands'):
                body.extend(['Declared subcommands:', '', '| Command | Source help |', '|---|---|'] +
                            [f"| `{s['name']}` | {cell(s.get('help') or s.get('description'))} |" for s in item['subcommands']] + [''])
            if item.get('arguments'):
                body.extend(['Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):', '',
                             '| Argument | Type/action | Required | Choices | Meaning |', '|---|---|---|---|---|'] +
                            [f"| {cell(', '.join(a['names']))} | {cell(a.get('type') or a.get('action'))} | {cell(a.get('required', 'positional' if a['names'] and not a['names'][0].startswith('-') else False))} | {cell(a.get('choices'))} | {cell(a.get('help'))} |" for a in item['arguments']] + [''])
            if item.get('parameters'):
                body.extend(['| Parameter | Declared type |', '|---|---|'] + [f"| `{p['name']}` | {cell(p['type'])} |" for p in item['parameters']] + [''])
            if item.get('status'):
                body.extend([f"Validation: {item['status']}.", ''])
            body.extend([f"Source: {citation(item['source'])}", ''])
    if plugin['errors']:
        body.extend(['## Incomplete checks', ''] + [f'- {e}' for e in plugin['errors']])
    if plugin.get('federated_tools'):
        body.extend(['', '## Existing hosted MCP exposure', '',
                     'The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.', ''] +
                    [f"- `{name}`" for name in plugin['federated_tools']])
    body.extend(['', 'Back to [[Code/wiki/plugin-inventory|Plugin inventory]].'])
    (output / 'plugins' / f"{plugin['name']}.md").write_text(note(plugin['name'], '\n'.join(body)), encoding='utf-8')


def registry_pages(output: Path, registry: dict) -> None:
    """Render live tool arguments and effects without invoking any registered tool.

    Inputs: wiki output and sanitized registry. Outputs: registry Markdown pages.
    Side effects: documentation writes. Choose for service tools beyond plugin-local code.
    """
    server_body = ['These are existing virtual servers returned by ContextForge. Attach the required server in your MCP client; its tool names are listed below. Enabled registration does not prove every upstream call works.', '']
    for server in registry['servers']:
        server_body.extend([f"## `{server['name']}`", '', str(server.get('description', '')), '', f"Server ID: `{server.get('id')}`. Enabled: `{server.get('enabled')}`.", '',
                            *[f"- `{name}`" for name in server.get('associatedTools', [])], ''])
    (output / 'mcp-servers.md').write_text(note('MCP servers and exposed tools', '\n'.join(server_body)), encoding='utf-8')
    for category, key in [('atomic-tools', 'gateway_tools'), ('contextforge-tools', 'contextforge_tools')]:
        values = registry[key]
        body = [f"Read from the existing registry at `{registry['checked_at']}`. {len(values)} entries.", '',
                'Registry presence is not a successful tool invocation. Preserve the declared side effects and execution policy when selecting a tool.', '',
                'Gateway tools are callable through the attached `atomic-tools` MCP server or its documented authenticated HTTP run endpoint. ContextForge tools are callable by an MCP client attached to a virtual server that exposes the named tool.', '',
                'Use the exact tool name and supply the required fields shown in its schema. A placeholder is not a valid real source or workflow ID.', '']
        if category == 'atomic-tools':
            body.extend(['## Browse before executing', '',
                         'ContextForge exposes one directory tool, `atomic-tools-atomic-tools`, over the catalog. Read-only examples:', '',
                         '```json', '{"path":""}', '{"path":"messages"}', '{"path":"messages.sms-xml"}', '```', '',
                         'To execute, add `run` containing the source locator and tool options, matching the contract returned by the browse call:', '',
                         '```json', '{"path":"messages.sms-xml","run":{"source_ref":"b2://salem-data/<actual-object-key>","args":{}}}', '```', '',
                         'This last example is a payload template, not an executed job. A real locator and the tool-specific options must be supplied. The gateway HTTP equivalent is authenticated `POST /tools/{id}/run` with `{source_ref,args}`.', '',
                         'The ContextForge atomic-tool description still reports 43 tools, while the directly read gateway reports 55. This is an observed description-freshness gap; the live gateway inventory is the source for the 55 listed IDs.', ''])
        for item in values:
            name = item.get('name') or item.get('id')
            body.extend([f'## `{name}`', '', str(item.get('description') or 'Description missing from registry.'), ''])
            for field in ('id', 'capability', 'formats', 'side_effect', 'execution_policy', 'enabled'):
                if field in item:
                    body.extend([f"{field}: `{item[field]}`", ''])
            if category == 'contextforge-tools':
                names = [s['name'] for s in registry['servers'] if name in s.get('associatedTools', [])]
                body.extend([f"Exposed by: {', '.join('`'+n+'`' for n in names) or 'no virtual-server association returned'}; see [[Code/wiki/mcp-servers]].", ''])
            schema = item.get('inputSchema') or item.get('input_schema')
            if schema:
                body.extend(['Input contract:', '', '```json', json.dumps(schema, ensure_ascii=False, indent=2), '```', ''])
            else:
                body.extend(['Input schema was not returned by this registry. Discover the tool schema through MCP before calling it; no arguments are invented here.', ''])
            body.extend([f"Source: `assets/tool-inventory.json` → `live_registry.{key}` → `{name}`; registry adapter {citation(registry['source'])}.", ''])
        body.extend(['Back to [[Code/wiki/plugin-inventory|Plugin inventory]].'])
        (output / f'{category}.md').write_text(note(category, '\n'.join(body)), encoding='utf-8')


def main() -> None:
    """Generate a source-backed wiki inventory with optional live registry discovery.

    Inputs: CLI marketplace/output/live flags. Outputs: notes, CSV, JSON, SHA-256 manifest.
    Side effects: writes the named output only; no tool execution or catalog mutations.
    Choose as the reproducible refresh entry point for this documentation inventory.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--marketplace', type=Path, default=Path('E:/AI_Workspace/plugins'))
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--live', action='store_true')
    parser.add_argument('--docstore-schemas', type=Path, help='Recorded read-only docstore_capabilities discovery JSON.')
    args = parser.parse_args(); output = args.output.resolve(); output.mkdir(parents=True, exist_ok=True)
    (output / 'assets').mkdir(exist_ok=True); (output / 'plugins').mkdir(exist_ok=True)
    marketplace_path = args.marketplace / '.claude-plugin/marketplace.json'
    registered = json.loads(marketplace_path.read_text(encoding='utf-8'))['plugins']
    known = {item['name'] for item in registered}
    for path in sorted((args.marketplace / 'plugins').glob('*/.claude-plugin/plugin.json')):
        if any(part in IGNORE for part in path.relative_to(args.marketplace).parts):
            continue
        meta = json.loads(path.read_text(encoding='utf-8-sig'))
        if meta['name'] not in known:
            registered.append({**meta, 'source': path.parent.parent.relative_to(args.marketplace).as_posix(), 'registered': False})
    installed = {}
    for app in ('.claude', '.codex'):
        path = Path.home() / app / 'plugins/installed_plugins.json'
        if path.exists():
            installed[app] = {name.split('@')[0] for name in json.loads(path.read_text(encoding='utf-8')).get('plugins', {})}
    plugins = [plugin_inventory(args.marketplace, item, installed) for item in sorted(registered, key=lambda p: p['name'])]
    registry = live_inventory(args.marketplace) if args.live else {'checked_at': None, 'gateway_tools': [], 'servers': [], 'contextforge_tools': [], 'errors': ['Live registry lookup not requested.']}
    for plugin in plugins:
        server_name = 'atomic-tools' if plugin['name'] == 'propria-toolbox' else plugin['name']
        plugin['federated_tools'] = sorted({name for server in registry['servers'] if server['name'] == server_name
                                            for name in server.get('associatedTools', [])})
    inventory = {'generated_at': datetime.now(timezone.utc).isoformat(), 'owner_date': DATE, 'byline': BYLINE,
                 'scope': 'registered Propria marketplace, additional direct plugin manifests, local exposed declarations and optional existing live registries; excludes retired copies and third-party cache trees',
                 'marketplace_source': origin(marketplace_path), 'plugins': plugins, 'live_registry': registry}
    if args.docstore_schemas:
        discovered = json.loads(args.docstore_schemas.read_text(encoding='utf-8'))
        inventory['docstore_operations'] = discovered
        operations_body = ['Docstore exposes five MCP entry points and discovers the following operations through `docstore_capabilities`.', '',
                           'Read each operation schema before calling it. Invoke with `docstore_query`, supplying `operation`, `arguments`, and `mode`. Read mode rejects mutations; write mode retains governance checks.', '',
                           '```json', '{"operation":"docstore_revision_state","arguments":{"document_key":"document:casebible_catalog_guide_20261004"},"mode":"read"}', '```', '',
                           f"Discovery evidence: {citation(origin(args.docstore_schemas))}", '']
        for operation in discovered['operations']:
            operations_body.extend([f"## `{operation['operation']}`", '', operation.get('description', ''), '',
                                    f"Read only: `{operation.get('read_only')}`. Capability group: `{operation.get('group')}`.", '',
                                    '```json', json.dumps(operation.get('input_schema', {}), ensure_ascii=False, indent=2), '```', ''])
        (output / 'docstore-operations.md').write_text(note('Docstore operations', '\n'.join(operations_body)), encoding='utf-8')
    (output / 'assets/tool-inventory.json').write_text(json.dumps(inventory, ensure_ascii=False, indent=2), encoding='utf-8')
    rows = []
    for plugin in plugins:
        write_plugin_page(output, plugin)
        for kind in ('commands', 'cli_entries', 'scripts'):
            for item in plugin[kind]:
                if item.get('invocation'):
                    rows.append({'plugin': plugin['name'], 'kind': kind, 'name': item['name'], 'invocation': item['invocation'],
                                 'description': item.get('description') or item.get('status', ''), 'source': item['source']['path'],
                                 'line': item['source']['line'], 'sha256': item['source']['sha256'], 'validation': item.get('status') or 'slash-command source declaration',
                                 'generated_by': BYLINE, 'owner_date': DATE})
    with (output / 'assets/human-commands.csv').open('w', encoding='utf-8', newline='') as handle:
        writer = csv.DictWriter(handle, fieldnames=['plugin', 'kind', 'name', 'invocation', 'description', 'source', 'line', 'sha256', 'validation', 'generated_by', 'owner_date'])
        writer.writeheader(); writer.writerows(rows)
    if registry.get('source'):
        registry_pages(output, registry)
    totals = {kind: sum(len(p[kind]) for p in plugins) for kind in ('skills', 'commands', 'agents', 'mcp_tools', 'scripts', 'cli_entries')}
    body = [f"{len(plugins)} plugin sources inspected: {sum(p['registered'] for p in plugins)} marketplace registrations and {sum(not p['registered'] for p in plugins)} additional direct manifests.", '',
            '## Choose an entry point', '',
            '- [[Code/wiki/case-bible-search|Search Case Bible content]]',
            '- [[Code/wiki/search-and-recall|Terminal commands for code search and memory recall]]',
            '- [[Code/wiki/atomic-tools|Atomic tools from the existing gateway]]',
            '- [[Code/wiki/contextforge-tools|Tools exposed through ContextForge]]', '',
            '- [[Code/wiki/mcp-servers|Virtual servers and tool associations]]',
            '- [[Code/wiki/docstore-operations|Docstore operation schemas]]', '',
            f"Declared surfaces: {totals['skills']} skills, {totals['commands']} slash commands, {totals['agents']} agents, {totals['mcp_tools']} local MCP declarations, {totals['scripts']} Python helpers/CLI candidates and {totals['cli_entries']} launcher/entry-point declarations.", '',
            f"Live registries returned {len(registry['gateway_tools'])} gateway tools, {len(registry['servers'])} virtual servers and {len(registry['contextforge_tools'])} ContextForge tools. These counts overlap local declarations and must not be added into a unique-tool total.", '',
            '| Plugin | Version | Skills | Slash commands | MCP declarations | CLI entries |', '|---|---|---:|---:|---:|---:|']
    for p in plugins:
        body.append(f"| [[Code/wiki/plugins/{p['name']}\\|{p['name']}]] | {cell(p['version'])} | {len(p['skills'])} | {len(p['commands'])} | {len(p['mcp_tools'])} | {len(p['cli_entries'])} |")
    body.extend(['', '## Complete machine-readable inventory', '',
                 '[Tool inventory JSON](assets/tool-inventory.json) contains full descriptions, parameters and source identities. [Human commands CSV](assets/human-commands.csv) is filterable by plugin and command type.', '',
                 '## Refresh and validation', '',
                 'Generated descriptions come from source docstrings/metadata or existing registries. Update those sources first, then regenerate. Missing docstrings and absent input contracts remain explicit findings.', '',
                 f'Source generator: {citation(origin(Path(__file__).resolve()))}', '',
                 '```powershell', f'python "{Path(__file__).resolve().as_posix()}" --output "<wiki-output-folder>" --live', '```', '',
                 'This is a bounded documentation inventory, not a new operational catalog or index. Source inspection does not validate credentials, live service health, semantic relevance or every mutation command. The search guide records the commands tested read-only.', '',
                 f'Marketplace evidence: {citation(inventory["marketplace_source"])}'])
    if registry['errors']:
        body.extend(['', '## Incomplete registry checks', ''] + [f'- {e}' for e in registry['errors']])
    (output / 'plugin-inventory.md').write_text(note('Custom plugins and tools', '\n'.join(body)), encoding='utf-8')
    print(json.dumps({'plugins': len(plugins), **totals, 'human_command_rows': len(rows), 'gateway_tools': len(registry['gateway_tools']),
                      'contextforge_tools': len(registry['contextforge_tools']), 'errors': registry['errors']}))


if __name__ == '__main__':
    main()
