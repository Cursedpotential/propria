// Byline: OpenAI Codex · GPT-5 · 2026-09-12
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ContextStrip } from "./context-strip";

afterEach(cleanup);

describe("ContextStrip", () => {
  it("states the bounded Family Court product and authority in text", () => {
    render(<ContextStrip />);
    const context = screen.getByRole("region", { name: "Workspace context" });
    expect(context).toHaveTextContent("Family Court Console");
    expect(context).toHaveTextContent("Local case workspace");
    expect(context).toHaveTextContent("Planning aid");
    expect(context).not.toHaveTextContent("advocatio Legal Workdesk");
  });
});
