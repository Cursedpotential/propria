"""Static contract for the per-service tsnet identities (owner directive 2026-09-07).

The directive is "APIs without networking endpoints; create host names; block
from the regular internet; bind to TS only". These assertions pin the parts of
that a manifest can get wrong silently:

  * every HTTP-serving Go service declares a tsnet identity with a CANONICAL
    hostname from docs/NAMING.md section 2 — not a Coolify app name, not a host
    name;
  * the auth key arrives as a read-only FILE mount, never an environment value;
  * tsnet node state is a persistent per-service bind under /data/probata/tsnet,
    or the node re-registers on every deploy and its Service FQDN churns;
  * the rollout flag defaults FALSE on the two live services, because
    modules/engine/** is a Coolify watch path for both — a code or manifest
    default of true would let a push move a live service onto a tailnet
    identity whose auth key does not exist yet;
  * nothing anywhere publishes 0.0.0.0.

Byline: Claude Code subagent · Opus 5 · 2026-09-07.
"""

from __future__ import annotations

from pathlib import Path

import yaml

ROOT = Path(__file__).parents[1]
DEPLOY = ROOT / "deploy"

TSNET_STATE_CONTAINER = "/data/tsnet"
TSNET_STATE_HOST_ROOT = "/data/probata/tsnet"
TSNET_AUTHKEY_CONTAINER = "/run/secrets/tsnet-authkey"

# service manifest -> (compose service name, canonical component name)
TSNET_SERVICES = {
    "parser-activity-runtime.yaml": ("parser-activity-runtime", "parser-runtime"),
    "proffer-starter.yaml": ("proffer-starter", "proffer-starter"),
    "workbench.yaml": ("tsnet-front", "workbench"),
}

# The two manifests whose Coolify apps are LIVE today and whose watch paths
# include modules/engine/**: a push redeploys them, so their flag must default
# off and their legacy listener must still be configured.
LIVE_STAGED = ("parser-activity-runtime.yaml", "proffer-starter.yaml")


def _compose(name: str) -> dict:
    return yaml.safe_load((DEPLOY / name).read_text(encoding="utf-8"))


def _service(name: str) -> dict:
    compose_service, _ = TSNET_SERVICES[name]
    return _compose(name)["services"][compose_service]


def test_every_tsnet_service_declares_a_canonical_hostname_and_service_name() -> None:
    for manifest, (_, canonical) in TSNET_SERVICES.items():
        env = _service(manifest)["environment"]
        assert env["TSNET_HOSTNAME"].endswith(f":-{canonical}}}"), (
            f"{manifest}: TSNET_HOSTNAME must default to the canonical component "
            f"name {canonical!r} from docs/NAMING.md section 2"
        )
        assert env["TSNET_SERVICE"].endswith(f":-svc:{canonical}}}"), (
            f"{manifest}: TSNET_SERVICE must default to svc:{canonical}"
        )
        assert "tag:docker" in env["TSNET_TAGS"], (
            f"{manifest}: a Tailscale Service needs a tag-based identity (D-134)"
        )


def test_auth_keys_are_read_only_file_mounts_and_never_environment_values() -> None:
    for manifest in TSNET_SERVICES:
        service = _service(manifest)
        env = service["environment"]
        assert env["TSNET_AUTHKEY_FILE"] == TSNET_AUTHKEY_CONTAINER
        assert not any(
            key.startswith("TSNET_AUTHKEY") and key != "TSNET_AUTHKEY_FILE" for key in env
        ), f"{manifest}: the auth key value must never be an environment variable"
        mounts = [m for m in service["volumes"] if isinstance(m, str)]
        authkey = [m for m in mounts if m.endswith(f"{TSNET_AUTHKEY_CONTAINER}:ro")]
        assert len(authkey) == 1, f"{manifest}: exactly one read-only auth key mount"
        assert authkey[0].startswith("/data/probata/secrets/"), (
            f"{manifest}: the auth key lives under the host secret root"
        )


def test_tsnet_state_is_a_persistent_per_service_bind_under_the_host_root() -> None:
    for manifest, (_, canonical) in TSNET_SERVICES.items():
        service = _service(manifest)
        assert service["environment"]["TSNET_STATE_DIR"] == TSNET_STATE_CONTAINER
        expected = f"{TSNET_STATE_HOST_ROOT}/{canonical}:{TSNET_STATE_CONTAINER}"
        mounts = [m for m in service["volumes"] if isinstance(m, str)]
        assert expected in mounts, (
            f"{manifest}: tsnet state must be the persistent bind {expected} — "
            "without it the node re-registers on every deploy"
        )


def test_the_rollout_flag_defaults_off_on_the_two_live_services() -> None:
    for manifest in LIVE_STAGED:
        flag = _service(manifest)["environment"]["TSNET_LISTENER_ENABLED"]
        assert flag.endswith(":-false}"), (
            f"{manifest}: modules/engine/** is a Coolify watch path for this app, so a "
            "push redeploys it. The flag must default false; the flip is a deliberate "
            "per-service deploy act after host prep."
        )


def test_the_legacy_listener_is_still_configured_so_the_flag_off_path_works() -> None:
    """D-127: a flag may move when a gate asserts, never whether it exists."""
    parser = _service("parser-activity-runtime.yaml")
    starter = _service("proffer-starter.yaml")
    assert parser["environment"]["PARSER_ACTIVITY_ADDR"] == ":8090"
    assert parser["ports"] == ["${BIND_IP:-127.0.0.1}:8090:8090"]
    assert starter["environment"]["REFERENCE_STARTER_ADDR"] == "100.91.190.107:8091"


def test_the_workbench_front_has_no_listener_but_its_tsnet_one() -> None:
    front = _service("workbench.yaml")
    # No published port and no network of its own: it shares the Workbench's
    # namespace so the proxy peer is loopback, which auth.py already trusts.
    assert "ports" not in front
    assert "networks" not in front
    assert front["network_mode"] == "service:workbench"
    assert front["environment"]["TSNET_FRONT_UPSTREAM"].endswith(":-http://127.0.0.1:8020}")
    # Repo-root build context: Coolify builds with --project-directory <repo root>.
    assert front["build"]["context"] == "."
    assert front["build"]["dockerfile"] == "deploy/docker/tsnet-front/Dockerfile"


def test_no_deploy_manifest_publishes_a_wildcard_address() -> None:
    offenders: list[str] = []
    for manifest in sorted(DEPLOY.glob("*.yaml")):
        try:
            compose = yaml.safe_load(manifest.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:  # a malformed manifest is its own failure
            offenders.append(f"{manifest.name}: unparseable: {exc}")
            continue
        if not isinstance(compose, dict):
            continue
        for name, service in (compose.get("services") or {}).items():
            if not isinstance(service, dict):
                continue
            for port in service.get("ports") or []:
                if isinstance(port, str) and port.startswith("0.0.0.0:"):
                    offenders.append(f"{manifest.name}:{name}:{port}")
    assert offenders == [], f"wildcard publishes are internet-facing: {offenders}"


def test_the_tsnet_front_image_declares_no_exposed_port() -> None:
    dockerfile = (DEPLOY / "docker/tsnet-front/Dockerfile").read_text(encoding="utf-8")
    assert "./cmd/tsnet-front" in dockerfile
    instructions = [
        line.strip() for line in dockerfile.splitlines() if not line.lstrip().startswith("#")
    ]
    assert not any(line.upper().startswith("EXPOSE") for line in instructions), (
        "the front's only listener is its tsnet one; an EXPOSE advertises a socket it "
        "does not have"
    )
    assert f"TSNET_STATE_DIR={TSNET_STATE_CONTAINER}" in dockerfile
