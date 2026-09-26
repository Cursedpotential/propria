"""llm-probe TUI — interactive terminal client for the llm-probe service.
Three tabs: Playground (compose a prompt, pick provider/model/params, run it
live), Board (the full liveness+capability grid), History (past playground
runs). All network calls run in a worker thread so the UI never blocks.

Run via: python -m llm_probe_cli.cli tui   (or directly: python -m llm_probe_cli.tui)

Byline: Claude Code · Sonnet 5 · 2026-08-27
"""
from __future__ import annotations

from typing import Optional

from textual.app import App, ComposeResult
from textual.containers import Horizontal, Vertical, VerticalScroll
from textual.widgets import (
    Button, DataTable, Footer, Header, Input, Label, Select, Static, TabbedContent,
    TabPane, TextArea,
)
from textual.worker import Worker, WorkerState

from .client import LLMProbeClient, base_url

REASONING_OPTIONS = [("(unset)", "unset"), ("none", "none"), ("low", "low"), ("medium", "medium"), ("high", "high")]


class PlaygroundPane(Vertical):
    def compose(self) -> ComposeResult:
        with Horizontal(classes="row"):
            yield Select([], id="provider_select", prompt="provider")
            yield Input(placeholder="model id", id="model_input")
        yield TextArea(id="prompt_area", text="Are you operational? Reply YES or NO.")
        with Horizontal(classes="row"):
            yield Input(placeholder="max_tokens", value="500", id="max_tokens_input")
            yield Input(placeholder="temperature", value="0", id="temperature_input")
            yield Select(REASONING_OPTIONS, id="reasoning_select", value="unset")
            yield Button("Run", id="run_button", variant="primary")
        yield Static("", id="playground_status")
        with VerticalScroll(id="playground_output_scroll"):
            yield Static("", id="playground_output")


class BoardPane(Vertical):
    def compose(self) -> ComposeResult:
        with Horizontal(classes="row"):
            yield Select([], id="board_provider_select", prompt="filter provider", allow_blank=True)
            yield Button("Refresh", id="board_refresh")
            yield Static("", id="board_status")
        yield DataTable(id="board_table")


class HistoryPane(Vertical):
    def compose(self) -> ComposeResult:
        with Horizontal(classes="row"):
            yield Button("Refresh", id="history_refresh")
            yield Static("", id="history_status")
        yield DataTable(id="history_table")


class LLMProbeApp(App):
    CSS = """
    .row { height: auto; margin-bottom: 1; }
    .row > Select { width: 1fr; margin-right: 1; }
    .row > Input { width: 1fr; margin-right: 1; }
    #prompt_area { height: 8; margin-bottom: 1; }
    #playground_output_scroll { height: 1fr; border: round $accent; }
    #playground_output { padding: 1; }
    DataTable { height: 1fr; }
    """
    TITLE = "llm-probe"
    BINDINGS = [("q", "quit", "Quit"), ("r", "refresh_current", "Refresh")]

    def __init__(self):
        super().__init__()
        self.client = LLMProbeClient()
        self._providers: list[str] = []

    def compose(self) -> ComposeResult:
        yield Header()
        with TabbedContent(initial="playground"):
            with TabPane("Playground", id="playground"):
                yield PlaygroundPane()
            with TabPane("Board", id="board"):
                yield BoardPane()
            with TabPane("History", id="history"):
                yield HistoryPane()
        yield Footer()

    def on_mount(self) -> None:
        self.sub_title = base_url()
        self.query_one("#board_table", DataTable).add_columns("provider", "model", "live", "tool_use", "summarize", "instr")
        self.query_one("#history_table", DataTable).add_columns("id", "provider", "model", "ok", "latency", "label", "when")
        self.run_worker(self._load_providers, thread=True, name="load_providers")

    # ---- data loading (all run in worker threads) ----

    def _load_providers(self) -> list[str]:
        provs = self.client.providers()
        return [p["name"] for p in provs if p["configured"]]

    def _load_board(self) -> list[dict]:
        return self.client.board()

    def _load_history(self) -> list[dict]:
        return self.client.playground_history(limit=50)

    def _do_run(self, provider: str, model: str, prompt: str, max_tokens: int,
                temperature: float, reasoning_effort: Optional[str]) -> dict:
        return self.client.run_playground(provider, model, prompt, max_tokens=max_tokens,
                                           temperature=temperature, reasoning_effort=reasoning_effort,
                                           label="tui")

    def on_worker_state_changed(self, event: Worker.StateChanged) -> None:
        if event.state != WorkerState.SUCCESS:
            return
        name = event.worker.name
        result = event.worker.result

        if name == "load_providers":
            self._providers = result
            opts = [(p, p) for p in result]
            self.query_one("#provider_select", Select).set_options(opts)
            self.query_one("#board_provider_select", Select).set_options(opts)

        elif name == "run_playground":
            status = self.query_one("#playground_status", Static)
            out = self.query_one("#playground_output", Static)
            if not result.get("http_ok"):
                status.update(f"[red]FAILED[/red] status={result.get('status')}")
                out.update(str(result.get("error")))
            else:
                usage = result.get("usage") or {}
                overhead = result.get("reasoning_overhead_tokens")
                extra = f" | hidden reasoning tokens: {overhead}" if overhead else ""
                status.update(
                    f"[green]OK[/green] {result['latency_s']}s | "
                    f"completion={usage.get('completion_tokens')} prompt={usage.get('prompt_tokens')} "
                    f"total={usage.get('total_tokens')}{extra}"
                )
                out.update(result.get("content") or "(empty response)")

        elif name == "load_board":
            table = self.query_one("#board_table", DataTable)
            table.clear()
            def cell(v):
                if v is None: return "—"
                return "OK" if v else "FAIL"
            for r in result:
                table.add_row(r["provider"], r["model"], cell(r["tier0_ok"]),
                              cell(r.get("tool_use_ok")), cell(r.get("summarization_ok")),
                              cell(r.get("instruction_following_ok")))
            self.query_one("#board_status", Static).update(f"{len(result)} models")

        elif name == "load_history":
            table = self.query_one("#history_table", DataTable)
            table.clear()
            for r in result:
                table.add_row(str(r["id"]), r["provider"], r["model"], "OK" if r["ok"] else "FAIL",
                              str(r["latency_s"]), r.get("label") or "", str(r["created_at"])[:19])
            self.query_one("#history_status", Static).update(f"{len(result)} runs")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "run_button":
            provider = self.query_one("#provider_select", Select).value
            model = self.query_one("#model_input", Input).value.strip()
            prompt = self.query_one("#prompt_area", TextArea).text.strip()
            if not provider or provider is Select.BLANK or not model or not prompt:
                self.query_one("#playground_status", Static).update("[yellow]need provider + model + prompt[/yellow]")
                return
            try:
                max_tokens = int(self.query_one("#max_tokens_input", Input).value or "500")
                temperature = float(self.query_one("#temperature_input", Input).value or "0")
            except ValueError:
                self.query_one("#playground_status", Static).update("[red]max_tokens/temperature must be numbers[/red]")
                return
            reasoning = self.query_one("#reasoning_select", Select).value
            reasoning = None if reasoning in (None, "unset", Select.BLANK) else reasoning
            self.query_one("#playground_status", Static).update("running…")
            self.run_worker(
                lambda: self._do_run(provider, model, prompt, max_tokens, temperature, reasoning),
                thread=True, name="run_playground",
            )
        elif event.button.id == "board_refresh":
            self.query_one("#board_status", Static).update("loading…")
            self.run_worker(self._load_board, thread=True, name="load_board")
        elif event.button.id == "history_refresh":
            self.query_one("#history_status", Static).update("loading…")
            self.run_worker(self._load_history, thread=True, name="load_history")

    def action_refresh_current(self) -> None:
        tabs = self.query_one(TabbedContent)
        if tabs.active == "board":
            self.run_worker(self._load_board, thread=True, name="load_board")
        elif tabs.active == "history":
            self.run_worker(self._load_history, thread=True, name="load_history")

    def on_unmount(self) -> None:
        self.client.close()


if __name__ == "__main__":
    LLMProbeApp().run()
