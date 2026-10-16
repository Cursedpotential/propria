"""Generate the TwelveLabs reference from its registered function docstrings.

Inputs: Python tool implementation. Output: docs/reference/TWELVELABS-TOOLS.md.
Side effects: writes that generated reference; no credentials or network needed.
Use after editing tool contracts to keep the catalog derived from code.
"""

from __future__ import annotations

import ast
from pathlib import Path


def main() -> None:
    """Extract tool identifiers and docstrings without importing the runtime."""
    root = Path(__file__).resolve().parents[1]
    source = root / "server/tools/extractors/twelvelabs.py"
    lines = [
        "<!-- Generated from server/tools/extractors/twelvelabs.py; do not edit manually. -->",
        "# TwelveLabs atomic tools",
        "",
        "Available through the existing tool-runtime registry, REST execution, and atomic_tools MCP directory.",
        "",
    ]
    for node in ast.parse(source.read_text(encoding="utf-8")).body:
        if not isinstance(node, ast.FunctionDef):
            continue
        for decorator in node.decorator_list:
            if (
                not isinstance(decorator, ast.Call)
                or not isinstance(decorator.func, ast.Name)
                or decorator.func.id != "register"
            ):
                continue
            values = {
                keyword.arg: ast.literal_eval(keyword.value)
                for keyword in decorator.keywords
                if keyword.arg in ("id", "capability")
            }
            lines.extend(
                [
                    f"## `{values['id']}`",
                    "",
                    f"Capability: `{values['capability']}`",
                    "",
                    ast.get_docstring(node) or "",
                    "",
                ]
            )
    target = root / "docs/reference/TWELVELABS-TOOLS.md"
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("\n".join(lines), encoding="utf-8")


if __name__ == "__main__":
    main()
