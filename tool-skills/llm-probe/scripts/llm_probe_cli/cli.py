#!/usr/bin/env python3
"""llm-probe CLI — scriptable client for the llm-probe service (deployed on
ovh-files, tailnet-only, http://100.91.190.107:8030 by default; override
with LLM_PROBE_URL). Every subcommand also takes --json for machine-readable
output — this is what the llm-probe SKILL.md tells agents to call.

`llm-probe tui` launches the interactive Textual app (llm_probe_cli.tui) for
human use instead of one-shot commands.

Byline: Claude Code · Sonnet 5 · 2026-08-27
"""
from __future__ import annotations

import json as json_mod
import sys
from typing import Optional

import typer
from rich.console import Console
from rich.table import Table

from .client import LLMProbeClient

app = typer.Typer(add_completion=False, no_args_is_help=True, pretty_exceptions_show_locals=False)
console = Console()
err_console = Console(stderr=True)


def _out(data, as_json: bool) -> None:
    if as_json:
        print(json_mod.dumps(data, indent=2, default=str))
    else:
        console.print(data)


@app.command()
def health(json: bool = typer.Option(False, "--json")):
    """Check the service is up and which providers have keys configured."""
    with LLMProbeClient() as c:
        _out(c.health(), json)


@app.command()
def providers(json: bool = typer.Option(False, "--json")):
    """List all known providers and whether each has a key configured."""
    with LLMProbeClient() as c:
        data = c.providers()
    if json:
        _out(data, True)
        return
    t = Table(title="Providers")
    t.add_column("provider"); t.add_column("configured"); t.add_column("base_url")
    for p in data:
        t.add_row(p["name"], "yes" if p["configured"] else "NO", p["base_url"])
    console.print(t)


@app.command()
def models(provider: str, json: bool = typer.Option(False, "--json")):
    """Live-fetch a provider's model catalog."""
    with LLMProbeClient() as c:
        try:
            data = c.models(provider)
        except Exception as e:
            err_console.print(f"[red]failed:[/red] {e}")
            raise typer.Exit(1)
    if json:
        _out(data, True)
        return
    for m in data:
        console.print(m.get("id", m))


@app.command()
def probes(json: bool = typer.Option(False, "--json")):
    """List the four named, scored probes and their exact prompts."""
    with LLMProbeClient() as c:
        data = c.probe_defs()
    if json:
        _out(data, True)
        return
    for name, d in data.items():
        console.print(f"[bold]{name}[/bold]: {d['description']}")
        console.print(f"  [dim]{d['prompt'][:120]}[/dim]")


@app.command()
def probe(
    provider: str, model: str, probe_name: str,
    max_tokens: int = typer.Option(500, "--max-tokens"),
    temperature: float = typer.Option(0, "--temperature"),
    reasoning_effort: Optional[str] = typer.Option(None, "--reasoning-effort", help="none|low|medium|high"),
    no_persist: bool = typer.Option(False, "--no-persist"),
    note: Optional[str] = typer.Option(None, "--note"),
    json: bool = typer.Option(False, "--json"),
):
    """Run one of the named scored probes (liveness/tool_use/summarization/instruction_following)."""
    with LLMProbeClient() as c:
        result = c.run_probe(provider, model, probe_name, max_tokens=max_tokens,
                              temperature=temperature, reasoning_effort=reasoning_effort,
                              persist=not no_persist, run_note=note)
    if json:
        _out(result, True)
        return
    ok = "[green]PASS[/green]" if result["ok"] else "[red]FAIL[/red]"
    console.print(f"{provider}/{model} :: {probe_name} -> {ok}  ({result.get('latency_s')}s)")
    if result.get("reasoning_overhead_tokens"):
        console.print(f"  [yellow]hidden reasoning tokens: {result['reasoning_overhead_tokens']}[/yellow]")
    for k in ("content", "hits", "missed", "reason", "args", "error"):
        if result.get(k) is not None:
            console.print(f"  {k}: {result[k]}")


@app.command()
def run(
    provider: str, model: str, prompt: str,
    max_tokens: int = typer.Option(500, "--max-tokens"),
    temperature: float = typer.Option(0, "--temperature"),
    reasoning_effort: Optional[str] = typer.Option(None, "--reasoning-effort", help="none|low|medium|high"),
    label: Optional[str] = typer.Option(None, "--label"),
    no_persist: bool = typer.Option(False, "--no-persist"),
    json: bool = typer.Option(False, "--json"),
):
    """Playground: run any free-form prompt against any provider/model."""
    with LLMProbeClient() as c:
        result = c.run_playground(provider, model, prompt, max_tokens=max_tokens,
                                   temperature=temperature, reasoning_effort=reasoning_effort,
                                   label=label, persist=not no_persist)
    if json:
        _out(result, True)
        return
    if not result["http_ok"]:
        err_console.print(f"[red]error ({result.get('status')}):[/red] {result.get('error')}")
        raise typer.Exit(1)
    console.print(result["content"] or "[dim](empty)[/dim]")
    usage = result.get("usage") or {}
    overhead = result.get("reasoning_overhead_tokens")
    console.print(f"\n[dim]{result['latency_s']}s · completion={usage.get('completion_tokens')} "
                  f"prompt={usage.get('prompt_tokens')} total={usage.get('total_tokens')}"
                  + (f" · hidden reasoning={overhead}" if overhead else "") + "[/dim]")


@app.command()
def history(
    provider: Optional[str] = None, model: Optional[str] = None,
    limit: int = 20, json: bool = typer.Option(False, "--json"),
):
    """Recent playground runs."""
    with LLMProbeClient() as c:
        data = c.playground_history(provider, model, limit)
    if json:
        _out(data, True)
        return
    t = Table(title="Playground history")
    for col in ("id", "provider", "model", "ok", "latency_s", "label", "created_at"):
        t.add_column(col)
    for r in data:
        t.add_row(str(r["id"]), r["provider"], r["model"], "OK" if r["ok"] else "FAIL",
                  str(r["latency_s"]), r.get("label") or "", str(r["created_at"])[:19])
    console.print(t)


@app.command()
def board(
    provider: Optional[str] = None,
    live_only: bool = typer.Option(False, "--live-only"),
    untested_only: bool = typer.Option(False, "--untested-only"),
    json: bool = typer.Option(False, "--json"),
):
    """The full per-model liveness + capability board."""
    with LLMProbeClient() as c:
        rows = c.board()
    if provider:
        rows = [r for r in rows if r["provider"] == provider]
    if live_only:
        rows = [r for r in rows if r["tier0_ok"]]
    if untested_only:
        rows = [r for r in rows if r["tier0_ok"] and "tool_use_ok" not in r]
    if json:
        _out(rows, True)
        return
    t = Table(title=f"Board ({len(rows)} models)")
    for col in ("provider", "model", "live", "tool_use", "summarize", "instr"):
        t.add_column(col)
    def cell(v):
        if v is None: return "[dim]—[/dim]"
        return "[green]OK[/green]" if v else "[red]FAIL[/red]"
    for r in rows:
        t.add_row(r["provider"], r["model"], cell(r["tier0_ok"]),
                   cell(r.get("tool_use_ok")), cell(r.get("summarization_ok")),
                   cell(r.get("instruction_following_ok")))
    console.print(t)


@app.command()
def summary(json: bool = typer.Option(False, "--json")):
    """Aggregate pass/fail counts across the whole board."""
    with LLMProbeClient() as c:
        data = c.summary()
    _out(data, json)


@app.command()
def tui():
    """Launch the interactive terminal app."""
    from .tui import LLMProbeApp
    LLMProbeApp().run()


if __name__ == "__main__":
    app()
