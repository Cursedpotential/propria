"""Call a coolify-write plugin tool over stdio.

The desktop app cached a connection failure for this MCP server while its server.py was
being edited mid-session, so its tools are absent from this session. The server itself is
healthy. This drives the SAME plugin code over the same stdio protocol the app uses, so
the owner's rule holds: Coolify work goes through the plugin's tools, not hand-run ssh,
docker or curl. Nothing here reimplements a Coolify operation.

Usage:  python coolify_call.py <tool_name> '<json args>'
        python coolify_call.py --list

Byline: Claude Code · Opus 5 · 2026-09-28
"""
import json
import os
import subprocess
import sys

PLUGIN = r"E:\AI_Workspace\plugins\plugins\coolify-write"
PYTHON = PLUGIN + r"\.venv\Scripts\python.exe"
SERVER = PLUGIN + r"\scripts\server.py"


def call(requests):
    """Run an ordered list of JSON-RPC requests against one server process."""
    env = dict(os.environ)
    env.update(PYTHONUTF8="1", COOLIFY_ENV_FILE="C:/Users/matts/.secrets/coolify-ionos-api.env")
    lines = [json.dumps(r) for r in requests]
    proc = subprocess.run(
        [PYTHON, SERVER],
        input="\n".join(lines) + "\n",
        capture_output=True, text=True, env=env, timeout=600,
    )
    out = []
    for line in proc.stdout.splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            out.append(json.loads(line))
        except json.JSONDecodeError:
            pass
    if not out:
        sys.stderr.write(proc.stderr[-2000:])
    return out


def main():
    tool = sys.argv[1]
    args = json.loads(sys.argv[2]) if len(sys.argv) > 2 else {}
    init = {"jsonrpc": "2.0", "id": 1, "method": "initialize",
            "params": {"protocolVersion": "2024-11-05", "capabilities": {},
                       "clientInfo": {"name": "propria-session", "version": "0"}}}
    ready = {"jsonrpc": "2.0", "method": "notifications/initialized"}
    if tool == "--list":
        req = {"jsonrpc": "2.0", "id": 2, "method": "tools/list"}
    else:
        req = {"jsonrpc": "2.0", "id": 2, "method": "tools/call",
               "params": {"name": tool, "arguments": args}}

    for message in call([init, ready, req]):
        if message.get("id") != 2:
            continue
        if "error" in message:
            print("ERROR:", json.dumps(message["error"], indent=2))
            sys.exit(1)
        result = message.get("result", {})
        if tool == "--list":
            for entry in result.get("tools", []):
                print(f"{entry['name']}\t{(entry.get('description') or '').splitlines()[0][:110]}")
            return
        for block in result.get("content", []):
            if block.get("type") == "text":
                print(block["text"])
        if result.get("structuredContent") and not result.get("content"):
            print(json.dumps(result["structuredContent"], indent=2))
        return
    print("no response from server", file=sys.stderr)
    sys.exit(1)


if __name__ == "__main__":
    main()


def call_many(requests_spec, chunk=1, attempts=3):
    """Chunked wrapper: sends at most `chunk` calls per server process and retries stragglers.

    A single process given 38 calls answered exactly 19 of them and dropped the rest, which is
    how the migration produced 92, 84, 101 and 0 on consecutive runs. The dropped ones were the
    applications with the largest environments -- intake-engine, legal-workspace, propria-docstore,
    propria-portal, workbench -- so the limit is on output per process, not on call count.

    chunk=1 reads all 38 and is the default for that reason. The original code also used one
    process per call and was still unreliable; what makes this correct is the retry plus an
    explicit error per call, so a dropped answer can never masquerade as an empty one.
    """
    pending = list(requests_spec)
    out = {label: {"rows": [], "error": "no response"} for label, _, _ in requests_spec}
    for _ in range(attempts):
        if not pending:
            break
        for index in range(0, len(pending), chunk):
            for label, value in _call_batch(pending[index:index + chunk]).items():
                if not value["error"]:
                    out[label] = value
        pending = [row for row in pending if out[row[0]]["error"]]
    return out


def _call_batch(requests_spec):
    """Run several tool calls against ONE server process, returning {label: rows}.

    Spawning a process per call made the migration non-deterministic: consecutive runs collected
    92, 0, 84 and 101 secrets because some spawns failed and the caller treated a failure as an
    empty result. One process, and an explicit error per call, makes a partial read visible
    instead of silent.

    requests_spec: [(label, tool_name, arguments), ...]
    returns:       {label: {"rows": [...], "error": None | str}}
    """
    messages = [{"jsonrpc": "2.0", "id": 1, "method": "initialize",
                 "params": {"protocolVersion": "2024-11-05", "capabilities": {},
                            "clientInfo": {"name": "propria-session", "version": "0"}}},
                {"jsonrpc": "2.0", "method": "notifications/initialized"}]
    order = {}
    for index, (label, tool, arguments) in enumerate(requests_spec, start=2):
        order[index] = label
        messages.append({"jsonrpc": "2.0", "id": index, "method": "tools/call",
                         "params": {"name": tool, "arguments": arguments}})

    out = {label: {"rows": [], "error": "no response"} for label, _, _ in requests_spec}
    for message in call(messages):
        label = order.get(message.get("id"))
        if label is None:
            continue
        if "error" in message:
            out[label] = {"rows": [], "error": json.dumps(message["error"])[:200]}
            continue
        rows = []
        for block in message.get("result", {}).get("content", []):
            if block.get("type") != "text":
                continue
            try:
                rows.append(json.loads(block["text"]))
            except json.JSONDecodeError:
                pass
        out[label] = {"rows": rows, "error": None}
    return out
