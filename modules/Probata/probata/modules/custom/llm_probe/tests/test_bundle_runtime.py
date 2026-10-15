"""Verify consolidated LLM Probe lifecycle without providers or production state.

Inputs: fictional settings, local-response doubles and supervisor event frames.
Outputs: secret isolation and bounded lifecycle assertions. Effects: memory only;
pick before an isolated streaming/container smoke test on the managed devbox.
"""
from io import StringIO
import signal

import pytest

from llm_probe import bundle_runtime as runtime


def test_ui_environment_excludes_provider_database_and_encryption_keys():
    """Keep all API secret classes out of the Next runtime environment.

    Inputs: fictional credentials and runtime settings. Output: explicit UI
    allowlist; effects: none. Pick to verify same-container process isolation.
    """
    settings = {"PATH": "/bin", "LANG": "C.UTF-8", "PORT": "9999", "HOSTNAME": "container-id",
                "DATABASE_URL": "fictional-secret", "NVIDIA_API_KEY": "fictional-secret",
                "PROVIDER_KEY_PASSPHRASE": "fictional-secret", "JWT_SECRET": "fictional-secret",
                "NEXT_PUBLIC_LLM_PROBE_URL": "http://old-api.example"}
    actual = runtime.ui_environment(settings)
    assert actual == {"PATH": "/bin", "LANG": "C.UTF-8", "NODE_ENV": "production", "HOSTNAME": "0.0.0.0", "PORT": "3000"}
    assert "fictional-secret" not in actual.values()


def test_start_ui_executes_the_existing_standalone_server(monkeypatch):
    """Launch Next directly with its existing server and sanitized settings.

    Inputs: intercepted exec. Output: exact executable/arguments; effects:
    synthetic capture only. Pick to verify supervisor signal propagation.
    """
    captured = []
    monkeypatch.setattr(runtime.os, "execve", lambda *args: captured.append(args))
    runtime.start_ui()
    assert captured[0][:2] == ("/usr/local/bin/node", ["node", "/opt/llm-probe-ui/server.js"])
    assert captured[0][2]["HOSTNAME"] == "0.0.0.0"


class _Response:
    """Supply one synthetic bounded HTTP health response.

    Inputs: status. Output: context-managed response; effects: none. Pick to
    test bundle health without binding a service or requesting a provider.
    """

    def __init__(self, status):
        """Retain the synthetic status.

        Inputs: integer. Output: response; effects: none. Pick per health call.
        """
        self.status = status

    def __enter__(self):
        """Return this synthetic response.

        Inputs: context entry. Output: response; effects: none. Pick for GETs.
        """
        return self

    def __exit__(self, *args):
        """Close the synthetic response without suppressing exceptions.

        Inputs: context exit. Output: false; effects: none. Pick after GETs.
        """
        return False


def test_health_requires_api_ui_and_proxy_without_provider_calls(monkeypatch):
    """Require both runtimes and the browser API route to answer successfully.

    Inputs: three synthetic responses. Output: successful bounded health;
    effects: captured URLs only. Pick to prevent half-working healthy bundles.
    """
    calls = []

    def read(url, timeout):
        """Capture a local health GET.

        Inputs: URL/timeout. Output: synthetic200; effects: memory only.
        Pick to prove no provider or database history endpoint is requested.
        """
        calls.append((url, timeout))
        return _Response(200)

    monkeypatch.setattr(runtime, "urlopen", read)
    assert runtime.check_health() == 0
    assert calls == [("http://127.0.0.1:8000/health", 3), ("http://127.0.0.1:3000/playground", 3),
                     ("http://127.0.0.1:3000/api/probe/health", 3)]
    monkeypatch.setattr(runtime, "urlopen", lambda *args, **kwargs: _Response(503))
    assert runtime.check_health() == 1


def test_fatal_child_acknowledges_then_stops_only_verified_container_supervisor(monkeypatch):
    """Stop the bundle after fatal startup failure with no host-process target.

    Inputs: fictional supervisor event and verified-parent doubles. Output:
    protocol acknowledgement and one captured signal; effects: memory only.
    Pick to prevent supervisor keeping a permanently failed half-bundle alive.
    """
    monkeypatch.setenv("SUPERVISOR_ENABLED", "1")
    monkeypatch.setenv("SUPERVISOR_PROCESS_NAME", "fatal-child")
    monkeypatch.setattr(runtime.os, "getppid", lambda: 1)
    monkeypatch.setattr(runtime.Path, "read_text", lambda self: "supervisord\n")
    calls = []
    monkeypatch.setattr(runtime.os, "kill", lambda *args: calls.append(args))
    source, target = StringIO("eventname:PROCESS_STATE_FATAL len:3\nbad"), StringIO()
    runtime.fatal_listener(source, target)
    assert target.getvalue() == "READY\nRESULT 2\nOK"
    assert calls == [(1, signal.SIGTERM)]


@pytest.mark.parametrize("parent,command", [(123, "supervisord"), (1, "systemd")])
def test_fatal_listener_refuses_unverified_parent(monkeypatch, parent, command):
    """Refuse signaling outside the exact container supervisor arrangement.

    Inputs: wrong parent PID or init name. Output: explicit rejection and no
    signal; effects: memory only. Pick for accidental direct CLI invocation.
    """
    monkeypatch.setenv("SUPERVISOR_ENABLED", "1")
    monkeypatch.setenv("SUPERVISOR_PROCESS_NAME", "fatal-child")
    monkeypatch.setattr(runtime.os, "getppid", lambda: parent)
    monkeypatch.setattr(runtime.Path, "read_text", lambda self: command)
    monkeypatch.setattr(runtime.os, "kill", lambda *args: pytest.fail("unexpected signal"))
    with pytest.raises(ValueError, match="container supervisor"):
        runtime.fatal_listener(StringIO("eventname:PROCESS_STATE_FATAL len:0\n"), StringIO())


@pytest.mark.parametrize("frame", ["eventname:PROCESS_STATE_FATAL len:65537\n", "eventname:PROCESS_STATE_FATAL len:3\nx"])
def test_fatal_listener_rejects_unbounded_or_truncated_events(frame):
    """Reject oversized or truncated event payloads before any lifecycle action.

    Inputs: invalid protocol frame. Output: permanent error; effects: none.
    Pick for bounded supervisor event handling rather than unchecked reads.
    """
    with pytest.raises(ValueError, match="bounded supervisor event"):
        runtime.fatal_listener(StringIO(frame), StringIO())
