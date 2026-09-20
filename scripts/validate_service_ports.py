"""Validate classified backend ports and Tailscale-only client endpoints.

This is a static source gate. It does not inspect or modify the live host,
advertise Tailscale Services, bind ports, or perform network requests.
"""
from __future__ import annotations

import argparse
import ipaddress
import json
from pathlib import Path
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_REGISTRY = ROOT / "deploy/service-port-registry.json"


class RegistryError(ValueError):
    pass


def validate(path: Path) -> dict:
    value = json.loads(path.read_text(encoding="utf-8"))
    if value.get("schema_version") != 1:
        raise RegistryError("Unsupported service-port registry version")
    suffix = value.get("tailnet_dns_suffix")
    if not isinstance(suffix, str) or not suffix.startswith("."):
        raise RegistryError("Tailnet DNS suffix is invalid")
    families = value.get("families")
    products = value.get("products")
    if not isinstance(families, dict) or not isinstance(products, dict):
        raise RegistryError("Families and products are required")

    codes: set[str] = set()
    canonical_ports: set[int] = set()
    services: set[str] = set()
    endpoint_count = 0
    for product, definition in products.items():
        code = definition.get("code")
        if not isinstance(code, str) or len(code) != 2 or not code.isdigit() or code in codes:
            raise RegistryError(f"{product}: product code must be unique two digits")
        codes.add(code)
        endpoints = definition.get("endpoints")
        if not isinstance(endpoints, list) or not endpoints:
            raise RegistryError(f"{product}: at least one endpoint is required")
        for endpoint in endpoints:
            endpoint_count += 1
            family_name = endpoint.get("family")
            family = families.get(family_name)
            if not isinstance(family, dict):
                raise RegistryError(f"{product}: unknown family {family_name!r}")
            port = endpoint.get("canonical_backend_port")
            bounds = family.get("range")
            prefix = family.get("prefix")
            if (type(port) is not int or not isinstance(bounds, list) or len(bounds) != 2
                    or not bounds[0] <= port <= bounds[1] or f"{port:04d}"[:2] != prefix
                    or f"{port:04d}"[-2:] != code):
                raise RegistryError(f"{product}: canonical port violates family/code shape")
            if port in canonical_ports:
                raise RegistryError(f"{product}: duplicate canonical backend port")
            canonical_ports.add(port)
            transitional = endpoint.get("transitional_backend_ports", [])
            if not isinstance(transitional, list) or any(type(p) is not int for p in transitional):
                raise RegistryError(f"{product}: transitional ports must be integers")
            service = endpoint.get("service")
            if not isinstance(service, str) or not service.startswith("svc:") or service in services:
                raise RegistryError(f"{product}: invalid or duplicate Tailscale Service")
            services.add(service)
            parsed = urlsplit(endpoint.get("client_endpoint", ""))
            if parsed.scheme not in {"https", "wss", "postgresql"} or not parsed.hostname:
                raise RegistryError(f"{product}: invalid client endpoint")
            try:
                ipaddress.ip_address(parsed.hostname)
            except ValueError:
                pass
            else:
                raise RegistryError(f"{product}: client endpoint must not contain an IP")
            if not parsed.hostname.endswith(suffix):
                raise RegistryError(f"{product}: client endpoint is not a tailnet Service name")
            expected_client_port = family.get("client_port")
            actual_client_port = parsed.port or (443 if parsed.scheme in {"https", "wss"} else 5432)
            if actual_client_port != expected_client_port:
                raise RegistryError(f"{product}: client port differs from family standard")
    return {
        "ok": True,
        "products": len(products),
        "endpoints": endpoint_count,
        "canonical_ports": sorted(canonical_ports),
        "network_accessed": False,
        "live_state_modified": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("registry", nargs="?", type=Path, default=DEFAULT_REGISTRY)
    args = parser.parse_args()
    try:
        print(json.dumps(validate(args.registry), sort_keys=True))
        return 0
    except (OSError, json.JSONDecodeError, RegistryError) as exc:
        print(json.dumps({"ok": False, "error": str(exc)}))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
