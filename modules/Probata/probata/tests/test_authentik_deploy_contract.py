"""Source contracts for Authentik provider and Workbench consumer manifests.

These tests are intentionally offline. Live Traefik routing, Authentik provider
creation, DNS, and Coolify deployment remain separate release gates.

Byline: Codex · GPT-5 · 2026-08-29
Byline amendment: Codex · GPT-5 · 2026-08-29 (official 2026.8 contract)
Byline amendment: Claude Code · Opus 5.5 · 2026-09-27 (edge via svc:authentik and tailnet-bound doors)
"""

from __future__ import annotations

from pathlib import Path

import yaml


AUTHENTIK_PATH = Path("deploy/authentik.yaml")
WORKBENCH_PATH = Path("deploy/workbench.yaml")
AUTHENTIK_IMAGE = (
    "ghcr.io/goauthentik/server:2026.8.0@sha256:7421753cfea67e89a6d295a1f0173ccea3866b33768c88dad90453b151cdcfd5"
)
AUTHENTIK_RUNTIME_IMAGE = "probata-authentik:2026.8.0"
AUTHENTIK_DOCKERFILE = Path("deploy/docker/authentik/Dockerfile")
AUTHENTIK_BLUEPRINT = Path("deploy/docker/authentik/blueprints/probata-workbench.yaml")
POSTGRES_IMAGE = (
    "docker.io/library/postgres:18-alpine@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2"
)
AUTHENTIK_EXACT_PROXY_SETTING = "${TRAEFIK_PROXY_CIDR:?exact Traefik proxy CIDR required}"
# Coolify 4.1.2 renders Compose's `:?message` form as the literal message when
# the variable is absent. Workbench uses plain substitution and its own auth
# boundary rejects an empty or malformed value at runtime.
WORKBENCH_EXACT_PROXY_SETTING = "${TRAEFIK_PROXY_CIDR}"


def _load(path: Path) -> dict:
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def _labels(service: dict) -> str:
    return "\n".join(str(label) for label in service.get("labels", []))


def _secret_mounts(service: dict) -> list[str]:
    return [mount for mount in service.get("volumes", []) if isinstance(mount, str) and "/run/secrets/" in mount]


class TestAuthentikProvider:
    def test_official_2026_8_service_shape(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        assert set(services) == {
            "authentik-postgres",
            "authentik-server",
            "authentik-worker",
        }
        assert services["authentik-server"]["command"] == "server"
        assert services["authentik-worker"]["command"] == "worker"

    def test_images_are_exact_tag_and_digest_pinned(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        for name in ("authentik-server", "authentik-worker"):
            assert services[name]["image"] == AUTHENTIK_RUNTIME_IMAGE
            assert services[name]["build"] == {
                "context": ".",
                "dockerfile": "deploy/docker/authentik/Dockerfile",
            }
        dockerfile = AUTHENTIK_DOCKERFILE.read_text(encoding="utf-8")
        assert f"FROM {AUTHENTIK_IMAGE}" in dockerfile
        assert services["authentik-postgres"]["image"] == POSTGRES_IMAGE

    def test_redis_removed_from_current_contract(self) -> None:
        compose = _load(AUTHENTIK_PATH)
        assert "authentik-redis" not in compose["services"]
        for service in compose["services"].values():
            assert all(not key.startswith("AUTHENTIK_REDIS__") for key in service.get("environment", {}))

    def test_official_authentik_environment_names_and_file_uris(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        expected = {
            "AUTHENTIK_POSTGRESQL__HOST": "authentik-postgres",
            "AUTHENTIK_POSTGRESQL__PORT": "5432",
            "AUTHENTIK_POSTGRESQL__USER": "authentik",
            "AUTHENTIK_POSTGRESQL__NAME": "authentik",
            "AUTHENTIK_POSTGRESQL__PASSWORD": ("file:///run/secrets/authentik/postgres-password"),
            "AUTHENTIK_SECRET_KEY": "file:///run/secrets/authentik/secret-key",
            "AUTHENTIK_LISTEN__TRUSTED_PROXY_CIDRS": AUTHENTIK_EXACT_PROXY_SETTING,
        }
        assert services["authentik-server"]["environment"] == expected
        assert services["authentik-worker"]["environment"] == {
            **expected,
            "AUTHENTIK_BOOTSTRAP_PASSWORD_HASH": "${AUTHENTIK_BOOTSTRAP_PASSWORD_HASH:-}",
        }
        text = AUTHENTIK_PATH.read_text(encoding="utf-8")
        assert "AUTHENTIK_POSTGRES__" not in text
        assert "AUTHENTIK_SECRET_KEY_FILE" not in text

    def test_owner_bootstrap_hash_is_worker_only_and_defaults_empty(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        assert "AUTHENTIK_BOOTSTRAP_PASSWORD_HASH" not in services["authentik-server"]["environment"]
        assert (
            services["authentik-worker"]["environment"]["AUTHENTIK_BOOTSTRAP_PASSWORD_HASH"]
            == "${AUTHENTIK_BOOTSTRAP_PASSWORD_HASH:-}"
        )
        text = AUTHENTIK_PATH.read_text(encoding="utf-8")
        assert "AUTHENTIK_BOOTSTRAP_PASSWORD:" not in text

    def test_secret_mounts_are_read_only(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        for name in services:
            for mount in _secret_mounts(services[name]):
                assert mount.endswith(":ro")
        for name in ("authentik-server", "authentik-worker"):
            assert len(_secret_mounts(services[name])) == 2

    def test_empty_host_templates_do_not_shadow_packaged_templates(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        for name in ("authentik-server", "authentik-worker"):
            assert all(
                not str(mount).split(":", maxsplit=1)[0].endswith("/templates")
                for mount in services[name].get("volumes", [])
            )

    def test_official_authentik_healthcheck_is_used(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        for name in ("authentik-server", "authentik-worker"):
            assert services[name]["healthcheck"]["test"] == [
                "CMD",
                "ak",
                "healthcheck",
            ]

    def test_postgres_gate_is_readiness_not_unrequired_extensions(self) -> None:
        postgres = _load(AUTHENTIK_PATH)["services"]["authentik-postgres"]
        command = " ".join(postgres["healthcheck"]["test"])
        assert "pg_isready" in command
        assert "pgcrypto" not in command
        assert "uuid-ossp" not in command
        assert all("docker-entrypoint-initdb.d" not in volume for volume in postgres["volumes"])

    def test_only_the_tailnet_door_is_published_and_no_docker_socket(self) -> None:
        # authentik-server publishes one port, bound to the tailnet address svc:authentik
        # forwards to (fail-closed to loopback when BIND_IP is unset). Postgres and the
        # worker publish nothing. Claude Code · Opus 5.5 · 2026-09-26.
        services = _load(AUTHENTIK_PATH)["services"]
        assert services["authentik-server"]["ports"] == ["${BIND_IP:-127.0.0.1}:9075:9000"]
        for name, service in services.items():
            if name != "authentik-server":
                assert not service.get("ports")
            assert all("0.0.0.0" not in str(binding) for binding in service.get("ports", []))
            assert "docker.sock" not in "\n".join(service.get("volumes", []))

    def test_no_docker_label_routing_or_fixed_edge_addresses(self) -> None:
        # Since 2026-09-27 Traefik reaches Authentik only through svc:authentik, routed by the
        # tracked Traefik file, so the socket peer is always the host's tailnet address. Docker
        # labels would reach Authentik over a Docker network whose proxy address drifts, and the
        # hand-made propria-edge network with pinned addresses is retired.
        # Claude Code · Opus 5.5 · 2026-09-27.
        compose = _load(AUTHENTIK_PATH)
        for service in compose["services"].values():
            assert "traefik." not in _labels(service)
            networks = service.get("networks", [])
            assert "propria-edge" not in networks
            if isinstance(networks, dict):
                assert all("ipv4_address" not in (cfg or {}) for cfg in networks.values())
        assert "propria-edge" not in compose["networks"]

    def test_blueprint_pins_single_app_provider_and_embedded_outpost(self) -> None:
        text = AUTHENTIK_BLUEPRINT.read_text(encoding="utf-8")
        assert "mode: forward_single" in text
        assert "external_host: https://workbench.int.mitechconsult.com" in text
        assert "name: authentik Embedded Outpost" in text
        assert "authentik_host: https://auth.int.mitechconsult.com/" in text
        assert "authentik_host_browser: https://auth.int.mitechconsult.com/" in text
        assert "- !KeyOf probata-workbench-provider" in text


class TestWorkbenchConsumer:
    def test_one_tailnet_bound_door_for_serve_and_traefik(self) -> None:
        # svc:workbench and Traefik both dial ${BIND_IP}:9071 (fail-closed to loopback).
        # Claude Code · Opus 5.5 · 2026-09-27.
        service = _load(WORKBENCH_PATH)["services"]["workbench"]
        assert service.get("ports") == ["${BIND_IP:-127.0.0.1}:9071:8020"]
        assert all("0.0.0.0" not in binding for binding in service["ports"])

    def test_publish_network_is_declared_so_the_trusted_peer_is_fixed(self) -> None:
        # Traefik's hop is masqueraded to the gateway of the network the port maps through;
        # that gateway (10.201.8.1) is TRAEFIK_PROXY_CIDR, so it must be declared, not pooled.
        compose = _load(WORKBENCH_PATH)
        service = compose["services"]["workbench"]
        assert service["networks"]["workbench-publish"] == {"gw_priority": 1}
        # Coolify 4.1.2 drops null-valued network entries; every entry must carry a mapping.
        assert all(isinstance(cfg, dict) for cfg in service["networks"].values())
        network = compose["networks"]["workbench-publish"]
        assert network["ipam"]["config"] == [{"subnet": "10.201.8.0/29", "gateway": "10.201.8.1"}]
        assert "propria-edge" not in compose["networks"]

    def test_exact_proxy_boundary_is_required_in_manifest(self) -> None:
        # ~~The Workbench uses the same `:?`-guarded value as Authentik.~~
        # CORRECTED 2026-09-07 (commit 16ac0fc, register section 14): Coolify
        # renders a `:?message` default as the LITERAL VALUE, so the guard
        # silently produced an unparseable CIDR string and every request 403'd
        # with "Authentication gateway not configured". The manifest now passes
        # the bare variable and auth.py owns fail-closed. Authentik's own
        # services keep EXACT_PROXY_SETTING; only this consumer changed.
        service = _load(WORKBENCH_PATH)["services"]["workbench"]
        assert service["environment"]["TRUSTED_AUTH_PROXY_CIDRS"] == "${TRAEFIK_PROXY_CIDR}"
        text = WORKBENCH_PATH.read_text(encoding="utf-8")
        assert 'TRUSTED_AUTH_PROXY_CIDRS: "10.0.0.0/8' not in text
        assert 'TRUSTED_AUTH_PROXY_CIDRS: "172.16.0.0/12' not in text

    def test_public_route_is_not_docker_labels(self) -> None:
        # workbench.int is the tracked Traefik file's router with Authentik's domain
        # forward-auth; docker labels would reach the Workbench over a Docker network whose
        # proxy address drifts. Claude Code · Opus 5.5 · 2026-09-27.
        service = _load(WORKBENCH_PATH)["services"]["workbench"]
        assert "traefik." not in _labels(service)

    def test_no_basic_auth_or_password_ingress_contract(self) -> None:
        service = _load(WORKBENCH_PATH)["services"]["workbench"]
        labels = _labels(service).lower()
        assert "basicauth" not in labels
        assert "OPENCODE_PASSWORD" not in service["environment"]


class TestSharedBoundary:
    def test_both_manifests_use_external_agno_network(self) -> None:
        # probata is a shared private network for app-to-app calls; Traefik reaches neither
        # app over it (see the edge notes in both manifests).
        for path in (AUTHENTIK_PATH, WORKBENCH_PATH):
            compose = _load(path)
            assert compose["networks"]["probata"]["external"] is True

    def test_no_literal_credentials_in_authentik_manifest(self) -> None:
        services = _load(AUTHENTIK_PATH)["services"]
        for service in services.values():
            for key, value in service.get("environment", {}).items():
                if key in {"AUTHENTIK_SECRET_KEY", "AUTHENTIK_POSTGRESQL__PASSWORD"}:
                    assert value.startswith("file:///run/secrets/authentik/")
                if key == "POSTGRES_PASSWORD_FILE":
                    assert value.startswith("/run/secrets/authentik/")

    def test_no_basic_auth_anywhere_in_provider_or_consumer(self) -> None:
        for path in (AUTHENTIK_PATH, WORKBENCH_PATH):
            compose = _load(path)
            for service in compose["services"].values():
                assert "basicauth" not in _labels(service).lower()
