"""Run the existing LLM Probe UI beside its API without sharing provider secrets.

Inputs: supervisor mode and existing container settings. Outputs: UI process,
bounded health status or supervisor event acknowledgement. Effects: child
execution and container-supervisor lifecycle only; never host power control.
Pick for the consolidated image; API/provider/history behavior stays in its
existing modules and PostgreSQL schema.
"""
from __future__ import annotations

import os
from pathlib import Path
import signal
import sys
from typing import Mapping, TextIO
from urllib.request import urlopen


def ui_environment(settings: Mapping[str, str]) -> dict[str, str]:
    """Build a UI runtime environment without API credentials or database settings.

    Inputs: supervisor environment. Output: explicit Node runtime settings.
    Effects: none; pick before launching Next instead of inheriting API secrets.
    The public API URL is compiled as /api/probe; no secret is a build argument.
    """
    allowed = ("PATH", "LANG", "LC_ALL", "TZ", "HOME")
    result = {key: settings[key] for key in allowed if key in settings}
    result.update(NODE_ENV="production", HOSTNAME="0.0.0.0", PORT="3000")
    return result


def start_ui() -> None:
    """Replace the supervisor child with the unchanged Next standalone server.

    Inputs: compiled UI and sanitized environment. Output: Node process.
    Effects: exec only; pick for the bundle so supervisor receives real signals
    and provider credentials remain in the API process environment.
    """
    os.execve("/usr/local/bin/node", ["node", "/opt/llm-probe-ui/server.js"], ui_environment(os.environ))


def check_health() -> int:
    """Check API, UI and the same-origin proxy without calling any provider.

    Inputs: fixed loopback endpoints. Output: zero only for three HTTP200s.
    Effects: bounded local GETs; pick for bundle health instead of API-only
    liveness that could hide a failed UI or broken proxy path.
    """
    for url in ("http://127.0.0.1:8000/health", "http://127.0.0.1:3000/playground",
                "http://127.0.0.1:3000/api/probe/health"):
        try:
            with urlopen(url, timeout=3) as response:
                if response.status != 200:
                    return 1
        except Exception:
            return 1
    return 0


def fatal_listener(source: TextIO, target: TextIO) -> None:
    """Stop only the container supervisor after a child exhausts startup retries.

    Inputs: bounded supervisor event protocol. Output: required acknowledgement.
    Effects: SIGTERM to this process's supervisor parent only. Pick over leaving
    one half of the bundle alive after the other enters permanent FATAL state.
    """
    while True:
        target.write("READY\n")
        target.flush()
        line = source.readline(4096)
        if not line:
            return
        fields = dict(part.split(":", 1) for part in line.split())
        length = int(fields["len"])
        if not 0 <= length <= 65536 or len(source.read(length)) != length:
            raise ValueError("Invalid bounded supervisor event")
        target.write("RESULT 2\nOK")
        target.flush()
        if fields.get("eventname") == "PROCESS_STATE_FATAL":
            parent = os.getppid()
            if (parent != 1 or os.environ.get("SUPERVISOR_ENABLED") != "1"
                    or os.environ.get("SUPERVISOR_PROCESS_NAME") != "fatal-child"
                    or Path("/proc/1/comm").read_text().strip() != "supervisord"):
                raise ValueError("Not the container supervisor listener")
            os.kill(parent, signal.SIGTERM)
            return


def main() -> int:
    """Dispatch one explicit bundled-runtime mode.

    Inputs: one CLI mode. Output: exit status. Effects: mode-specific bounded
    actions; pick for the supervisor/healthcheck rather than a new application.
    """
    mode = sys.argv[1] if len(sys.argv) == 2 else ""
    if mode == "ui":
        start_ui()
        return 0
    if mode == "health":
        return check_health()
    if mode == "fatal-listener":
        fatal_listener(sys.stdin, sys.stdout)
        return 0
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
