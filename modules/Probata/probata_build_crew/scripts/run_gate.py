"""Run ONLY the gatekeeper stage of the JSON crew against saved upstream outputs.

`crewai replay` does not support JSON-first crews (it spawns the classic-project `uv run replay` script), so when
the last stage fails after 40 minutes of research this script re-runs just `review_task` with the four saved task
outputs supplied as context. It loads the same crew.jsonc (same agent, tools, LLM chain) and writes the same
output_file.

Usage (from the project root, with .env in place):
    uv run python scripts/run_gate.py output/run-08-task1-constraints_brief.md output/run-08-task2-code_review.md \
        output/run-08-task3-gap_analysis.md output/run-08-task4-work_package_draft.md

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import io
import os
import sys
from pathlib import Path

from dotenv import load_dotenv


def main(argv: list[str]) -> int:
    if len(argv) != 4:
        print(__doc__)
        return 2
    root = Path(__file__).resolve().parents[1]
    os.chdir(root)
    load_dotenv(root / ".env", override=True)
    sys.path.insert(0, str(root))

    from crewai import Crew, Task
    from crewai.project.crew_loader import load_crew

    crew, defaults = load_crew(root / "crew.jsonc")
    review = next(t for t in crew.tasks if t.name == "review_task")
    gate_agent = review.agent
    labels = ["CONSTRAINTS BRIEF (task 1)", "CODE REVIEW (task 2)", "GAP ANALYSIS (task 3)", "WORK PACKAGE (task 4)"]
    docs = [io.open(p, encoding="utf-8").read() for p in argv]
    context = "\n\n".join(f"===== {lab} =====\n{doc}" for lab, doc in zip(labels, docs))

    description = review.description + (
        "\n\nThe four upstream documents follow verbatim. Treat them as the outputs of tasks 1-4.\n\n" + context
    )
    task = Task(
        name="review_task_gate_only",
        description=description,
        expected_output=review.expected_output,
        agent=gate_agent,
        markdown=True,
        output_file=review.output_file or "output/work-package.md",
    )
    gate_crew = Crew(agents=[gate_agent], tasks=[task], process=crew.process, verbose=True, memory=False)
    result = gate_crew.kickoff(inputs=dict(defaults))
    gate_text = str(result)
    gate_path = root / (review.output_file or "output/gatekeeper-review.md")
    gate_path.parent.mkdir(parents=True, exist_ok=True)
    io.open(gate_path, "w", encoding="utf-8", newline="\n").write(gate_text)

    from scripts.assemble_package import assemble  # deterministic stitching, no LLM copying

    assembled = assemble(docs[0], docs[1], docs[2], docs[3], gate_text)
    out = root / "output" / "work-package.md"
    io.open(out, "w", encoding="utf-8", newline="\n").write(assembled)
    print("\n===== GATE RESULT (head) =====")
    print(gate_text[:1500])
    print(f"\nassembled {out} ({len(assembled)} bytes); gate section at {gate_path} ({len(gate_text)} bytes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
