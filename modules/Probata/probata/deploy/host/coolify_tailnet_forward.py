#!/usr/bin/env python3
"""Adopt and verify the existing sole Coolify tailnet forwarder without creating duplicates.

Inputs: --check (default) or --ensure for the declared existing host helper.
Outputs: sanitized identity, route and HTTP receipt. Effects: check reads only;
ensure may start only the validated canonical container, never pull, create,
remove, rename or stop one. Choose for this unmanaged host helper; Coolify-owned
applications/services must use the Coolify management API instead.

Owner: 2026-10-15 infrastructure consolidation. The September 26 preserved
container formerly called broken proved to own the working route. Its replacement
was disconnected and is now preserved stopped with restart=no. Docker's retained
unless-stopped policy supplies boot persistence; this file is the durable declared
identity/preflight, not an additional long-running service.
"""
from __future__ import annotations

import argparse
import datetime
import json
import subprocess
from urllib.request import urlopen

SPEC = {
    "name": "coolify-tailnet-forward",
    "image_id": "sha256:24220ef2c80a2a421ea08e4624488e985330c421b6aa3329bae14b0933a1d403",
    "host_ip": "127.0.0.1",
    "host_port": "8001",
    "container_port": "8000/tcp",
    "target": "TCP:100.98.98.38:8000",
    "restart_policy": "unless-stopped",
}


def docker(*arguments: str) -> str:
    """Run a bounded Docker query or the one explicitly selected start operation.

    Inputs: exact argument values. Outputs: captured stdout. Effects: only start
    mutates runtime; choose instead of a shell or unmanaged duplicate docker run.
    """
    result = subprocess.run(["docker", *arguments], capture_output=True, timeout=10)
    if result.returncode:
        raise RuntimeError("Docker operation failed; inspect the declared helper privately")
    return result.stdout.decode()


def verify_configuration(value: dict) -> None:
    """Require the declared image, endpoint and restart policy before adopting a container.

    Input: private Docker inspection record. Output: none or failure. Effects:
    none; choose before any start so a name collision cannot redirect access.
    """
    host = value["HostConfig"]
    bindings = host.get("PortBindings", {}).get(SPEC["container_port"])
    expected_binding = [{"HostIp": SPEC["host_ip"], "HostPort": SPEC["host_port"]}]
    command = value["Config"].get("Cmd") or []
    if not (
        value["Name"] == "/" + SPEC["name"]
        and value["Image"] == SPEC["image_id"]
        and bindings == expected_binding
        and host.get("PortBindings") == {SPEC["container_port"]: expected_binding}
        and host["RestartPolicy"]["Name"] == SPEC["restart_policy"]
        and host["NetworkMode"] == "bridge"
        and SPEC["target"] in command
        and any(argument.startswith("TCP-LISTEN:8000,") for argument in command)
        and len(command) == 2
        and not (value["Config"].get("Labels") or {}).get("coolify.managed")
    ):
        raise RuntimeError("Canonical forwarder differs from the declared source; no action taken")


def main() -> int:
    """Check or start only the validated existing canonical forwarder and verify its route.

    Inputs: command-line mode. Outputs: status JSON. Effects: optional guarded
    start only. Choose to retain one helper; absence or contenders require review.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--ensure", action="store_true")
    mode.add_argument("--check", action="store_true")
    options = parser.parse_args()
    value = json.loads(docker("inspect", SPEC["name"]))[0]
    verify_configuration(value)
    contenders = docker("ps", "--no-trunc", "--filter", "publish=" + SPEC["host_port"], "--format", "{{.ID}}").splitlines()
    if any(container_id != value["Id"] for container_id in contenders):
        raise RuntimeError("Another live container publishes the declared port; no action taken")
    started = False
    if not value["State"]["Running"]:
        if not options.ensure:
            raise RuntimeError("Canonical forwarder is stopped; use explicit --ensure after review")
        docker("start", value["Id"])
        started = True
        value = json.loads(docker("inspect", SPEC["name"]))[0]
        verify_configuration(value)
    actual = value["NetworkSettings"].get("Ports", {}).get(SPEC["container_port"])
    if actual != [{"HostIp": SPEC["host_ip"], "HostPort": SPEC["host_port"]}]:
        raise RuntimeError("Canonical container has no actual declared port mapping")
    with urlopen("http://127.0.0.1:8001/", timeout=5) as response:
        status = response.status
    if status != 200:
        raise RuntimeError("Declared route did not return the expected HTTP status")
    print(json.dumps({"observed_at": datetime.datetime.now(datetime.UTC).isoformat(),
                      "name": SPEC["name"], "id": value["Id"], "pid": value["State"]["Pid"],
                      "image_id": value["Image"], "restart_policy": SPEC["restart_policy"],
                      "http_status": status, "started_by_this_call": started}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
